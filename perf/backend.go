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
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

//go:embed *.html
var BackendFS embed.FS

func (c *ServeBackendCmd) Run(p *ProgramCtx) error {
	log.SetPrefix(fmt.Sprintf("[c %v %v %s] ", os.Getpid(), mustResolveHostIP(), c.Name))

	var t = mustParseTrafficType(string(c.TrafficType))

	listenAddress := c.ListenAddress
	if listenAddress == "" || listenAddress == "127.0.0.1" || listenAddress == "::1" {
		listenAddress = "0.0.0.0"
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%v:0", listenAddress))
	if err != nil {
		return err
	}

	certs := certStore(path.Join(p.Globals.OutputDir, "certs"))

	httpServer := &http.Server{
		Handler:      http.FileServer(http.FS(BackendFS)),
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

	for {
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonValue))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		request.Close = true
		if resp, err = client.Do(request); err == nil {
			break
		}
		if errors.Is(err, syscall.ECONNREFUSED) {
			// Perhaps the parent crashed (e.g., cannot
			// spawn OS thread due to lack of resources,
			// or other). If we cannot connect there's no
			// point retrying.
			return fmt.Errorf("%s: %w", url, err)
		}
		retries -= 1
		if retries == 0 {
			return fmt.Errorf("backend registration %+v failed: %v", boundBackend, err)
		}
		time.Sleep(250 * time.Millisecond)
	}

	_, err = io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed for %+v; Status=%v", boundBackend, resp.Status)
	}

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
