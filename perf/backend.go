package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"time"
)

//go:embed *.html
var BackendFS embed.FS

func (c *ServeBackendCmd) Run(p *ProgramCtx) error {
	log.SetPrefix(fmt.Sprintf("[c %v %s] ", os.Getpid(), c.Name))

	var t = mustParseTrafficType(string(c.TrafficType))

	listenAddress := c.ListenAddress
	if listenAddress == "" || listenAddress == "127.0.0.1" || listenAddress == "::1" {
		listenAddress = "0.0.0.0"
	}

	l, err := net.Listen("tcp", fmt.Sprintf("%v:0", listenAddress))
	if err != nil {
		return err
	}

	certs := certStore(path.Join(p.Globals.OutputDir, "certs"))

	go func() {
		switch t {
		case HTTPTraffic, EdgeTraffic:
			if err := http.Serve(l, http.FileServer(http.FS(BackendFS))); err != nil {
				log.Fatal(err)
			}
		default:
			if err := http.ServeTLS(l, http.FileServer(http.FS(BackendFS)), certs.TLSCertFile, certs.TLSKeyFile); err != nil {
				log.Fatal(err)
			}
		}
	}()

	if listenAddress == "0.0.0.0" {
		// we're going to advertise this address so make it
		// discoverable/routable.
		listenAddress = mustResolveHostIP()
	}

	boundBackend := BoundBackend{
		Backend: Backend{
			Name:        c.Name,
			TrafficType: t,
		},
		ListenAddress: listenAddress,
		Port:          l.Addr().(*net.TCPAddr).Port,
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
			time.Sleep(time.Duration(42) * time.Millisecond)
		} else {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("POST failed for %+v: %v\n", boundBackend, err)
	}

	defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST failed for %+v; Status=%v\n", boundBackend, resp.Status)
	}

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	_, _ = os.NewFile(3, "<pipe>").Read(make([]byte, 1))
	os.Exit(0) // the parent has exited we should too.

	return nil
}
