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

	for retries > 0 {
		resp, err = http.Post(fmt.Sprintf("http://127.0.0.1:%d/register", p.Port), "application/json", bytes.NewBuffer(jsonValue))
		if err != nil {
			retries -= 1
			log.Printf("#%v retries remaining", retries)
			time.Sleep(250 * time.Millisecond)
		} else {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("POST failed for %+v: %v\n", boundBackend, err)
	}

	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST failed for %+v; Status=%v\n", boundBackend, resp.Status)
	}

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
