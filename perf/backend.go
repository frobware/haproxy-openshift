package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"golang.org/x/sync/errgroup"
)

//go:embed *.html
var BackendFS embed.FS
var listener net.Listener

func debugServerHandler(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("Backend Address: %s\n", listener.Addr().String())
	w.Write([]byte(msg))
	w.Write([]byte("Request Headers:\n"))
	for name, values := range r.Header {
		// Loop over all values for the name.
		for _, value := range values {
			w.Write([]byte(fmt.Sprintf("> %s: %s\n", name, value)))
		}
	}
}

func (c *ServeBackendCmd) Run(p *ProgramCtx) error {
	log.SetPrefix(fmt.Sprintf("[c %v %v %s] ", os.Getpid(), mustResolveHostIP(), c.Name))

	var t = mustParseTrafficType(string(c.TrafficType))

	listenAddress := c.ListenAddress
	if listenAddress == "" || listenAddress == "127.0.0.1" || listenAddress == "::1" {
		listenAddress = "0.0.0.0"
	}

	var err error
	listener, err = net.Listen("tcp", fmt.Sprintf("%v:0", listenAddress))
	if err != nil {
		return err
	}

	certs := certStore(path.Join(p.Globals.OutputDir, "certs"))

	var serverHandler http.Handler
	// If debug is on, use our debug handler to dump request and server information
	if c.InfoServer {
		serverHandler = http.HandlerFunc(debugServerHandler)
	} else {
		serverHandler = http.FileServer(http.FS(BackendFS))
	}

	httpServer := &http.Server{
		Handler:      serverHandler,
		Addr:         fmt.Sprintf("%v:%v", listenAddress, p.Port),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	g, gCtx := errgroup.WithContext(p.Context)

	g.Go(func() error {
		switch t {
		case HTTPTraffic, EdgeTraffic:
			return httpServer.Serve(listener)
		default:
			return httpServer.ServeTLS(listener, certs.DomainFile, certs.TLSKeyFile)
		}
	})

	g.Go(func() error {
		<-gCtx.Done()
		httpServer.SetKeepAlivesEnabled(false)
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), httpServer.WriteTimeout)
		defer shutdownRelease()
		return httpServer.Shutdown(shutdownCtx)
	})

	if listenAddress == "0.0.0.0" {
		listenAddress = mustResolveHostIP()
	}

	boundBackend := BoundBackend{
		Backend: Backend{
			Name:        c.Name,
			TrafficType: t,
		},
		ListenAddress: listenAddress,
		Port:          listener.Addr().(*net.TCPAddr).Port,
	}

	jsonValue, err := json.Marshal(boundBackend)
	if err != nil {
		return err
	}

	var (
		resp    *http.Response
		retries = 17
	)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/register", p.Port)

	for retries > 0 {
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonValue))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		request.Close = true
		if resp, err = client.Do(request); err == nil {
			break
		}
		retries -= 1
		log.Printf("#%v retries remaining", retries)
		time.Sleep(250 * time.Millisecond)
	}

	if err != nil {
		return fmt.Errorf("POST failed for %+v: %v", boundBackend, err)
	}

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return err
	}

	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed for %+v; Status=%v", boundBackend, resp.Status)
	}

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
