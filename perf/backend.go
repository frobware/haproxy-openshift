package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/erikdubbelboer/gspt"
)

//go:embed *.html
var BackendFS embed.FS

func (c *ServeBackendCmd) Run(p *ProgramCtx) error {
	backendName, found := os.LookupEnv(ChildBackendEnvName)
	if !found {
		return fmt.Errorf("%q not found in environment", ChildBackendEnvName)
	}

	trafficType, found := os.LookupEnv(ChildBackendTrafficTypeEnvName)
	if !found {
		return fmt.Errorf("%q not found in environment", ChildBackendTrafficTypeEnvName)
	}

	log.SetPrefix(fmt.Sprintf("[c %v %s] ", os.Getpid(), backendName))

	var t TrafficType = ParseTrafficType(trafficType)

	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return err
	}

	go func() {
		switch t {
		case HTTPTraffic, EdgeTraffic:
			if err := http.Serve(l, http.FileServer(http.FS(BackendFS))); err != nil {
				log.Fatal(err)
			}
		default:
			if err := http.ServeTLS(l, http.FileServer(http.FS(BackendFS)), p.TLSCert, p.TLSKey); err != nil {
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
		Port: l.Addr().(*net.TCPAddr).Port,
	}

	jsonValue, err := json.Marshal(boundBackend)
	if err != nil {
		return err
	}

	var (
		resp    *http.Response
		retries int = 17
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

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("POST failed for %+v; Status=%v\n", boundBackend, resp.Status)
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	gspt.SetProcTitle(fmt.Sprintf("%s %v", boundBackend.Name, boundBackend.Port))

	os.NewFile(3, "<pipe>").Read(make([]byte, 1))
	os.Exit(2)

	return nil
}
