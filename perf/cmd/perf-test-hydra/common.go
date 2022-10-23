package main

import (
	"embed"
	"log"
	"os"
	"strconv"

	"github.com/frobware/haproxy-openshift/perf/pkg/termination"
)

type Backend struct {
	HostAddr    string                  `json:"host_addr"`
	Name        string                  `json:"name"`
	TrafficType termination.TrafficType `json:"traffic_type"`
}

type BoundBackend struct {
	Backend

	Port int `json:"port"`
}

type BackendsByTrafficType map[termination.TrafficType][]Backend

type ServeBackendCmd struct {
	Args []string `arg:""`
}

type ServeBackendsCmd struct{}

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
