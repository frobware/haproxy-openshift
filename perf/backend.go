package main

import (
	"bytes"
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

type ServeBackendCmd struct {
	Args []string `arg:""`
}

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

	// log.Printf("%s %v\n", name, l.Addr().(*net.TCPAddr).Port)

	tlsCertFile := tlsTemporaryCertFile()
	defer os.Remove(tlsCertFile)

	tlsKeyFile := tlsTemporaryKeyFile()
	defer os.Remove(tlsKeyFile)

	go func() {
		switch t {
		case HTTP, Edge:
			if err := http.Serve(l, http.FileServer(http.FS(htmlFS))); err != nil {
				log.Fatal(err)
			}
		default:
			if err := http.ServeTLS(l, http.FileServer(http.FS(htmlFS)), tlsCertFile, tlsKeyFile); err != nil {
				log.Fatal(err)
			}
		}
	}()

	boundBackend := BoundBackend{
		Backend: Backend{
			HostAddr:    "127.0.0.1",
			Name:        backendName,
			TrafficType: t,
		},
		Port: l.Addr().(*net.TCPAddr).Port,
	}

	jsonValue, err := json.Marshal(boundBackend)
	if err != nil {
		log.Fatal(err)
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
		log.Fatalf("POST failed for %+v: %v\n", boundBackend, err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("POST failed for %+v; Status=%v\n", boundBackend, resp.Status)
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	gspt.SetProcTitle(fmt.Sprintf("%s %v", boundBackend.Name, boundBackend.Port))

	os.NewFile(3, "<pipe>").Read(make([]byte, 1))
	os.Exit(2)

	return nil
}
