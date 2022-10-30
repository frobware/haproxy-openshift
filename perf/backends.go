package main

import (
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	ChildBackendListenAddress      = "CHILD_BACKEND_LISEN_ADDRESS"
	ChildBackendEnvName            = "CHILD_BACKEND_NAME"
	ChildBackendTrafficTypeEnvName = "CHILD_BACKEND_TERMINATION_TYPE"
)

var CertName = pkix.Name{
	Organization: []string{"Cert Gen Company"},
	CommonName:   "Common Name",
}

func serveBackendMetadata(backendsByTrafficType BackendsByTrafficType, port int, postNotifier func(b BoundBackend)) {
	// Provide synchronous access to the asynchronously registered
	// port number for a backend.
	var registeredBackends sync.Map

	printBackendsForType := func(w io.Writer, t TrafficType) error {
		for _, b := range backendsByTrafficType[t] {
			port, ok := registeredBackends.Load(b.Name)
			if !ok {
				panic("missing port for" + b.Name)
			}
			if _, err := io.WriteString(w, fmt.Sprintf("%v %v %v %v\n", b.HostAddr, b.Name, port, b.TrafficType)); err != nil {
				return err
			}
		}
		return nil
	}

	mux := http.NewServeMux()

	var mu sync.Mutex

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method != "POST" {
			panic("unexpected: " + r.Method)
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var x BoundBackend
		if err := decoder.Decode(&x); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		registeredBackends.Store(x.Name, x)
		postNotifier(x)
	})

	mux.HandleFunc("/backends", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["json"]; ok {
			var boundBackendsByTrafficType = BoundBackendsByTrafficType{}

			for _, t := range AllTrafficTypes {
				for _, b := range backendsByTrafficType[t] {
					x, ok := registeredBackends.Load(b.Name)
					if !ok {
						panic("missing port for" + b.Name)
					}
					boundBackend := x.(BoundBackend)
					boundBackendsByTrafficType[t] = append(boundBackendsByTrafficType[t],
						BoundBackend{
							Backend:       b,
							Port:          boundBackend.Port,
							ListenAddress: boundBackend.ListenAddress,
						})
				}
			}
			data, err := json.MarshalIndent(boundBackendsByTrafficType, "", "  ")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if _, err := io.WriteString(w, string(data)); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			for _, t := range AllTrafficTypes {
				if err := printBackendsForType(w, t); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				}
			}
		}
	})

	mux.HandleFunc("/backends/edge", func(w http.ResponseWriter, r *http.Request) {
		if err := printBackendsForType(w, EdgeTraffic); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	mux.HandleFunc("/backends/http", func(w http.ResponseWriter, r *http.Request) {
		if err := printBackendsForType(w, HTTPTraffic); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	mux.HandleFunc("/backends/passthrough", func(w http.ResponseWriter, r *http.Request) {
		if err := printBackendsForType(w, PassthroughTraffic); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	mux.HandleFunc("/backends/reencrypt", func(w http.ResponseWriter, r *http.Request) {
		if err := printBackendsForType(w, ReencryptTraffic); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})

	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", port), mux); err != nil {
		log.Fatal(err)
	}
}

type CertificatePath struct {
	Domain  string
	RootCA  string
	TLSCert string
	TLSKey  string
}

func CertificatePaths(certDir string) CertificatePath {
	return CertificatePath{
		Domain:  path.Join(certDir, "domain.pem"),
		RootCA:  path.Join(certDir, "rootCA.pem"),
		TLSKey:  path.Join(certDir, "tls.key"),
		TLSCert: path.Join(certDir, "tls.crt"),
	}
}

func writeCertificates(dir string, certBundle *CertificateBundle) (*CertificatePath, error) {
	certDir := path.Join(dir, "certs")

	if err := os.RemoveAll(certDir); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(certDir, 0755); err != nil {
		return nil, err
	}

	serverCert := certBundle.LeafCertPEM + "\n" + certBundle.LeafKeyPEM + "\n" + certBundle.RootCACertPEM
	serverCert = strings.TrimSuffix(serverCert, "\n")

	certPath := CertificatePaths(certDir)

	if err := createFile(certPath.Domain, []byte(serverCert)); err != nil {
		return nil, err
	}
	if err := createFile(certPath.RootCA, []byte(certBundle.RootCACertPEM)); err != nil {
		return nil, err
	}

	if err := createFile(certPath.TLSCert, []byte(certBundle.LeafCertPEM)); err != nil {
		return nil, err
	}

	if err := createFile(certPath.TLSKey, []byte(certBundle.LeafKeyPEM)); err != nil {
		return nil, err
	}

	return &certPath, nil
}

func createNewTLSCertificates(certDir string, hosts ...string) (*CertificatePath, error) {
	certBundle, err := GenerateCerts(CertName, time.Now(), time.Now().AddDate(1, 0, 0), hosts...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificates: %v", err)
	}
	return writeCertificates(certDir, certBundle)
}

func (c *ServeBackendsCmd) Run(p *ProgramCtx) error {
	certPaths, err := createNewTLSCertificates(p.OutputDir, Hostname(), "localhost", "localhost.localdomain")
	if err != nil {
		return err
	}

	fmt.Println(certPaths)

	hostIPAddr := HostIPAddress()
	backendsByTrafficType := BackendsByTrafficType{}

	for _, t := range AllTrafficTypes {
		for i := 0; i < p.Nbackends; i++ {
			backend := Backend{
				HostAddr:    hostIPAddr.String(),
				Name:        fmt.Sprintf("%s-%v-%v", p.HostPrefix, t, i),
				TrafficType: t,
			}
			backendsByTrafficType[t] = append(backendsByTrafficType[t], backend)
		}
	}

	log.SetPrefix(fmt.Sprintf("[P %v] %v ", os.Getpid(), HostIPAddress()))

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGCHLD)
		log.Println(<-sigc)
		os.Exit(1)
	}()

	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	defer func(r *os.File) { _ = r.Close() }(r)
	defer func(w *os.File) { _ = w.Close() }(w)

	var backendsReady = make(chan bool)
	var backendsRegistered = 0

	go serveBackendMetadata(backendsByTrafficType, p.Port, func(b BoundBackend) {
		backendsRegistered += 1
		if backendsRegistered == len(backendsByTrafficType)*p.Nbackends {
			backendsReady <- true
			return
		}
	})

	log.Printf("starting %d backends for traffic types: %v\n",
		len(AllTrafficTypes)*p.Nbackends/len(AllTrafficTypes),
		AllTrafficTypes)

	for t, backends := range backendsByTrafficType {
		for _, backend := range backends {
			childEnv := []string{
				fmt.Sprintf("%s=%v", ChildBackendEnvName, backend.Name),
				fmt.Sprintf("%s=%v", ChildBackendListenAddress, c.ListenAddress),
				fmt.Sprintf("%s=%v", ChildBackendTrafficTypeEnvName, t),
			}
			// We want to be a child of the current
			// process so the following fork/exec needs to
			// change the current program arguments so
			// that the exec, and subsequent command line
			// parsing, ensures we call serve-backend
			// (singular) and not server-backends
			// (plural). Otherwise we'll end up back here.
			newArgs := os.Args[:1]
			newArgs = append(newArgs, "serve-backend")
			args := append(newArgs, fmt.Sprintf("#%v %v", backend.Name, c.ListenAddress))
			if _, err := syscall.ForkExec(args[0], args, &syscall.ProcAttr{
				Env:   append(os.Environ(), childEnv...),
				Files: []uintptr{0, 1, 2, r.Fd()},
			}); err != nil {
				return err
			}
		}
	}

	<-backendsReady
	log.Printf("%d backends registered", backendsRegistered)
	log.Printf("metadata server running at http://%s:%v/backends\n", Hostname(), p.Port)
	<-p.Context.Done()
	return nil
}
