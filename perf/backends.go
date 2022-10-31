package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"
)

type Backend struct {
	Name        string      `json:"name"`
	TrafficType TrafficType `json:"traffic_type"`
}

type BoundBackend struct {
	Backend

	ListenAddress string `json:"listen_address"`
	Port          int    `json:"port"`
}

type BackendsByTrafficType map[TrafficType][]Backend
type BoundBackendsByTrafficType map[TrafficType][]BoundBackend

const (
	ChildBackendListenAddress      = "CHILD_BACKEND_LISTEN_ADDRESS"
	ChildBackendEnvName            = "CHILD_BACKEND_NAME"
	ChildBackendTrafficTypeEnvName = "CHILD_BACKEND_TERMINATION_TYPE"
)

func serveBackendMetadata(certBundle *CertificateBundle, backendsByTrafficType BackendsByTrafficType, port int, postNotifier func(b BoundBackend)) {
	// Provide synchronous access to the asynchronously registered
	// port number for a backend.
	var registeredBackends sync.Map

	printBackendsForType := func(w io.Writer, t TrafficType) error {
		for _, b := range backendsByTrafficType[t] {
			obj, ok := registeredBackends.Load(b.Name)
			if !ok {
				panic("missing port for" + b.Name)
			}
			boundBackend, ok := obj.(BoundBackend)
			if !ok {
				panic(fmt.Sprintf("unexpected type: %T", obj))
			}
			if _, err := io.WriteString(w, fmt.Sprintf("%v %v %v\n", b.Name, boundBackend.ListenAddress, boundBackend.Port)); err != nil {
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
			http.Error(w, r.Method, http.StatusBadRequest)
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var boundBackend BoundBackend
		if err := decoder.Decode(&boundBackend); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		registeredBackends.Store(boundBackend.Name, boundBackend)
		postNotifier(boundBackend)
	})

	mux.HandleFunc("/certs", func(w http.ResponseWriter, r *http.Request) {
		data, err := json.MarshalIndent(certBundle, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := io.WriteString(w, string(data)); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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

func (c *ServeBackendsCmd) Run(p *ProgramCtx) error {
	if err := os.RemoveAll(path.Join(p.OutputDir, "certs")); err != nil {
		return err
	}

	var subjectAltNames = []string{
		mustResolveHostname(),
		mustResolveHostIP(),
		c.ListenAddress,
		"localhost",
		"127.0.0.1",
		"::1",
	}

	backendsByTrafficType := BackendsByTrafficType{}

	for _, t := range AllTrafficTypes {
		for i := 0; i < p.Nbackends; i++ {
			backend := Backend{
				Name:        fmt.Sprintf("%s-%v-%v", p.HostPrefix, t, i),
				TrafficType: t,
			}
			backendsByTrafficType[t] = append(backendsByTrafficType[t], backend)
			subjectAltNames = append(subjectAltNames, backend.Name)
		}
	}

	log.SetPrefix(fmt.Sprintf("[P %v] %v ", os.Getpid(), mustResolveHostIP()))

	go func() {
		// if a backend exits then exit everything as
		// something unexpected occurred.
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

	certBundle, err := CreateTLSCerts(time.Now(), time.Now().AddDate(1, 0, 0), subjectAltNames...)
	if err != nil {
		return fmt.Errorf("failed to generate certificates: %v", err)
	}

	// var b []byte = make([]byte, 1)
	// fmt.Println("write certs")
	// os.Stdin.Read(b)

	if _, err := writeCertificates(path.Join(p.OutputDir, "certs"), certBundle); err != nil {
		return err
	}

	go serveBackendMetadata(certBundle, backendsByTrafficType, p.Port, func(b BoundBackend) {
		backendsRegistered += 1
		if backendsRegistered == len(backendsByTrafficType)*p.Nbackends {
			backendsReady <- true
			return
		}
	})

	log.Printf("starting %d backends for traffic types: %v\n",
		len(AllTrafficTypes)*p.Nbackends/len(AllTrafficTypes),
		AllTrafficTypes)

	listenAddress := c.ListenAddress
	if listenAddress == "" || listenAddress == "127.0.0.1" || listenAddress == "::1" {
		listenAddress = mustResolveHostIP()
	}

	for _, backends := range backendsByTrafficType {
		for _, backend := range backends {
			newArgs := os.Args[:1]
			newArgs = append(newArgs, []string{
				"serve-backend",
				fmt.Sprintf("--listen-address=%s", listenAddress),
				fmt.Sprintf("--name=%s", backend.Name),
				fmt.Sprintf("--traffic-type=%s", backend.TrafficType),
			}...)
			if _, err := syscall.ForkExec(os.Args[0], newArgs, &syscall.ProcAttr{
				Files: []uintptr{0, 1, 2, r.Fd()},
			}); err != nil {
				return err
			}
		}
	}

	<-backendsReady
	log.Printf("%d backends registered", backendsRegistered)
	log.Printf("metadata server running at http://%s:%v/backends\n", mustResolveHostname(), p.Port)
	<-p.Context.Done()
	return nil
}
