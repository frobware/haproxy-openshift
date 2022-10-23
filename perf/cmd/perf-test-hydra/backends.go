package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/frobware/haproxy-openshift/perf/pkg/termination"
)

type Backend struct {
	HostAddr    string                  `json:"host_addr"`
	Name        string                  `json:"name"`
	Port        int                     `json:"port"`
	TrafficType termination.TrafficType `json:"traffic_type"`
}

type BackendsByTrafficType map[termination.TrafficType][]Backend

const (
	ChildBackendEnvName            = "CHILD_BACKEND_NAME"
	ChildBackendTrafficTypeEnvName = "CHILD_BACKEND_TERMINATION_TYPE"
)

var (
	//go:embed *.html
	htmlFS embed.FS

	//go:embed tls.crt
	tlsCert string

	//go:embed tls.key
	tlsKey string
)

func mustCreateTemporaryFile(data []byte) string {
	f, err := os.CreateTemp("", strconv.Itoa(os.Getpid()))
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write(data)
	if err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	return f.Name()
}

func tlsTemporaryKeyFile() string {
	return mustCreateTemporaryFile([]byte(tlsKey))
}

func tlsTemporaryCertFile() string {
	return mustCreateTemporaryFile([]byte(tlsCert))
}

func mustResolveCurrentHost() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	return hostname
}

func startMetadataServer(backendsByType BackendsByTrafficType, port int) {
	var mu sync.Mutex

	printBackendsForType := func(w io.Writer, t termination.TrafficType) error {
		for _, b := range backendsByType[t] {
			if _, err := io.WriteString(w, fmt.Sprintf("%v %v %v %v\n", b.Name, b.HostAddr, b.Port, b.TrafficType)); err != nil {
				return err
			}
		}
		return nil
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/backends", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case "POST":
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			var x Backend
			if err := decoder.Decode(&x); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			words := strings.Split(x.Name, "-")
			if len(words) != 4 {
				http.Error(w, "expected 4 words", http.StatusBadRequest)
				return
			}
			i, err := strconv.ParseInt(words[3], 10, 64)
			if err != nil {
				http.Error(w, "expected 4 words", http.StatusBadRequest)
				return
			}
			backendsByType[x.TrafficType][i].Port = x.Port
		default:
			for _, t := range termination.AllTerminationTypes[:] {
				printBackendsForType(w, t)
			}
		}
	})

	mux.HandleFunc("/backends/edge", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		printBackendsForType(w, termination.Edge)
	})

	mux.HandleFunc("/backends/http", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		printBackendsForType(w, termination.HTTP)
	})

	mux.HandleFunc("/backends/passthrough", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		printBackendsForType(w, termination.Passthrough)
	})

	mux.HandleFunc("/backends/reencrypt", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		printBackendsForType(w, termination.Reencrypt)
	})

	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", port), mux); err != nil {
		log.Fatal(err)
	}
}

func startBackends(ctx context.Context, backendsByType BackendsByTrafficType, port int) error {
	log.SetPrefix(fmt.Sprintf("[P %v] ", os.Getpid()))
	log.Printf("pid: %d, ppid: %d, args: %s", os.Getpid(), os.Getppid(), os.Args)

	go func() {
		sigc := make(chan os.Signal, 1)
		// if any child exits then consider that fatal for the
		// parent too.
		signal.Notify(sigc, syscall.SIGCHLD)
		log.Println(<-sigc)
		os.Exit(1)
	}()

	go startMetadataServer(backendsByType, port)

	var children []int

	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	defer r.Close()
	defer w.Close()

	for t, backends := range backendsByType {
		for _, backend := range backends {
			childEnv := []string{
				fmt.Sprintf("CHILD_ID=%v", backend.Name),
				fmt.Sprintf("%s=%v", ChildBackendEnvName, backend.Name),
				fmt.Sprintf("%s=%v", ChildBackendTrafficTypeEnvName, t),
			}
			os.Args[1] = "serve-backend"
			args := append(os.Args, fmt.Sprintf("#%v", backend.Name))
			child, err := syscall.ForkExec(args[0], args, &syscall.ProcAttr{
				Env:   append(os.Environ(), childEnv...),
				Files: []uintptr{0, 1, 2, r.Fd()},
			})
			if err != nil {
				return err
			}
			if child != 0 {
				children = append(children, child)
			}
		}
	}

	<-ctx.Done()
	return nil
}

func serveBackend(name, trafficType string, port int) error {
	log.SetPrefix(fmt.Sprintf("[c %v %s] ", os.Getpid(), name))

	var t termination.TrafficType = termination.ParseTrafficType(trafficType)

	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return err
	}

	// log.Printf("%s %v\n", name, l.Addr().(*net.TCPAddr).Port)

	tlsCertFile := tlsTemporaryCertFile()
	defer os.Remove(tlsCertFile)

	tlsKeyFile := tlsTemporaryKeyFile()
	defer os.Remove(tlsKeyFile)

	go func() {
		switch t {
		case termination.HTTP, termination.Edge:
			if err := http.Serve(l, http.FileServer(http.FS(htmlFS))); err != nil {
				log.Fatal(err)
			}
		default:
			if err := http.ServeTLS(l, http.FileServer(http.FS(htmlFS)), tlsCertFile, tlsKeyFile); err != nil {
				log.Fatal(err)
			}
		}
	}()

	backend := Backend{
		HostAddr:    "127.0.0.1",
		Name:        name,
		Port:        l.Addr().(*net.TCPAddr).Port,
		TrafficType: t,
	}

	jsonValue, err := json.Marshal(backend)
	if err != nil {
		panic(err)
	}

	var (
		resp    *http.Response
		retries int = 17
	)

	for retries > 0 {
		resp, err = http.Post(fmt.Sprintf("http://127.0.0.1:%d/backends", port), "application/json", bytes.NewBuffer(jsonValue))
		if err != nil {
			retries -= 1
		} else {
			break
		}
		time.Sleep(42 * time.Millisecond)
	}

	if resp != nil {
		defer resp.Body.Close()
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
	}

	os.NewFile(3, "<pipe>").Read(make([]byte, 1))
	os.Exit(2)

	return nil
}
