package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
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

func (c *ServeBackendsCmd) spawnBackend(ctx context.Context, backend Backend) error {
	newArgs := []string{
		"serve-backend",
		fmt.Sprintf("--name=%s", backend.Name),
		fmt.Sprintf("--traffic-type=%s", backend.TrafficType),
	}
	if c.ListenAddress != "127.0.0.1" {
		newArgs = append(newArgs, fmt.Sprintf("--listen-address=%s", c.ListenAddress))
	}
	cmd := exec.Command(os.Args[0], newArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func (c *ServeBackendsCmd) Run(p *ProgramCtx) error {
	log.SetPrefix(fmt.Sprintf("[P %v] %v ", os.Getpid(), mustResolveHostIP()))

	if err := os.RemoveAll(path.Join(p.OutputDir, "certs")); err != nil {
		return err
	}

	var (
		backendsByTrafficType = BackendsByTrafficType{}
		backendsReady         = make(chan bool)
		backendsRegistered    = 0
		boundBackends         sync.Map
		registerHandlerLock   sync.Mutex
	)

	mux := http.NewServeMux()

	httpServer := &http.Server{
		Handler:      mux,
		Addr:         fmt.Sprintf("0.0.0.0:%v", p.Port),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	g, gCtx := errgroup.WithContext(p.Context)
	chldSignalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGCHLD)
	defer stop()

	g.Go(func() error {
		select {
		case <-chldSignalCtx.Done():
			return fmt.Errorf("a backend died")
		case <-gCtx.Done():
			return nil
		}
	})

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		registerHandlerLock.Lock()
		defer registerHandlerLock.Unlock()
		if backendsRegistered == len(backendsByTrafficType)*p.Nbackends {
			http.Error(w, "unexpected registration", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, r.Method, http.StatusBadRequest)
			return
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var boundBackend BoundBackend
		if err := decoder.Decode(&boundBackend); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		boundBackends.Store(boundBackend.Name, boundBackend)
		backendsRegistered += 1
		if backendsRegistered == len(backendsByTrafficType)*p.Nbackends {
			backendsReady <- true
		}
	})

	g.Go(func() error {
		return httpServer.ListenAndServe()
	})

	g.Go(func() error {
		<-gCtx.Done()
		httpServer.SetKeepAlivesEnabled(false)
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), httpServer.WriteTimeout)
		defer shutdownRelease()
		return httpServer.Shutdown(shutdownCtx)
	})

	var subjectAlternateNames = []string{
		mustResolveHostname(),
		mustResolveHostIP(),
		c.ListenAddress,
		"localhost",
		"127.0.0.1",
		"::1",
	}

	for _, t := range AllTrafficTypes {
		for i := 0; i < p.Nbackends; i++ {
			backend := Backend{
				Name:        fmt.Sprintf("%s-%v-%v", p.HostPrefix, t, i),
				TrafficType: t,
			}
			backendsByTrafficType[t] = append(backendsByTrafficType[t], backend)
			subjectAlternateNames = append(subjectAlternateNames, backend.Name)
		}
	}

	// Create certificates after we know all the backend names.
	certBundle, err := CreateTLSCerts(time.Now(), time.Now().AddDate(1, 0, 0), subjectAlternateNames...)
	if err != nil {
		return fmt.Errorf("failed to generate certificates: %v", err)
	}

	if _, err := writeCertificates(path.Join(p.OutputDir, "certs"), certBundle); err != nil {
		return err
	}

	for t, backends := range backendsByTrafficType {
		log.Printf("starting %d %s backend(s)\n", p.Nbackends, t)
		for _, backend := range backends {
			if err := c.spawnBackend(p.Context, backend); err != nil {
				return err
			}
		}
	}

	select {
	case <-backendsReady:
		log.Printf("%d backend(s) processes registered", len(AllTrafficTypes)*p.Nbackends)
	case <-gCtx.Done():
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("timeout waiting for backends to register")
	}

	printBackendsForType := func(w io.Writer, t TrafficType) error {
		for _, b := range backendsByTrafficType[t] {
			obj, ok := boundBackends.Load(b.Name)
			if !ok {
				panic("missing backend registration for" + b.Name)
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
		if _, ok := r.URL.Query()["json"]; !ok {
			for _, t := range AllTrafficTypes {
				if err := printBackendsForType(w, t); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				}
			}
			return
		}

		var boundBackendsByTrafficType = BoundBackendsByTrafficType{}

		for _, t := range AllTrafficTypes {
			for _, b := range backendsByTrafficType[t] {
				obj, ok := boundBackends.Load(b.Name)
				if !ok {
					panic("missing registration for" + b.Name)
				}
				boundBackend, ok := obj.(BoundBackend)
				if !ok {
					panic(fmt.Sprintf("unexpected type: %T", obj))
				}
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
	})

	log.Printf("metadata server available at http://%s:%v/backends", mustResolveHostname(), p.Port)

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
