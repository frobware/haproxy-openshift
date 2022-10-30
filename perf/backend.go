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
	listenAddr, found := os.LookupEnv(ChildBackendListenAddress)
	if !found {
		return fmt.Errorf("%q not found in environment", ChildBackendListenAddress)
	}

	backendName, found := os.LookupEnv(ChildBackendEnvName)
	if !found {
		return fmt.Errorf("%q not found in environment", ChildBackendEnvName)
	}

	trafficType, found := os.LookupEnv(ChildBackendTrafficTypeEnvName)
	if !found {
		return fmt.Errorf("%q not found in environment", ChildBackendTrafficTypeEnvName)
	}

	log.SetPrefix(fmt.Sprintf("[c %v %s] ", os.Getpid(), backendName))

	var t = ParseTrafficType(trafficType)

	l, err := net.Listen("tcp", fmt.Sprintf("%v:0", listenAddr))
	if err != nil {
		return err
	}

	certs := CertificatePaths(path.Join(p.Globals.OutputDir, "certs"))

	go func() {
		switch t {
		case HTTPTraffic, EdgeTraffic:
			if err := http.Serve(l, http.FileServer(http.FS(BackendFS))); err != nil {
				log.Fatal(err)
			}
		default:
			if err := http.ServeTLS(l, http.FileServer(http.FS(BackendFS)), certs.TLSCert, certs.TLSKey); err != nil {
				log.Fatal(err)
			}
		}
	}()

	boundBackend := BoundBackend{
		Backend: Backend{
			HostAddr:    HostIPAddress().String(),
			Name:        backendName,
			TrafficType: t,
		},
		ListenAddress: listenAddr,
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

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST failed for %+v; Status=%v\n", boundBackend, resp.Status)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	//gspt.SetProcTitle(fmt.Sprintf("%s %v %v", boundBackend.Name, listenAddr, boundBackend.Port))

	SetProcessName(fmt.Sprintf("%s %v", boundBackend.Name, boundBackend.Port))

	os.NewFile(3, "<pipe>").Read(make([]byte, 1))
	os.Exit(2)

	return nil
}
