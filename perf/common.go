package main

import (
	"embed"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

type Backend struct {
	HostAddr    string      `json:"host_addr"`
	Name        string      `json:"name"`
	TrafficType TrafficType `json:"traffic_type"`
}

type BoundBackend struct {
	Backend

	Port int `json:"port"`
}

func (b BoundBackend) URL() string {
	return fmt.Sprintf("%s://%s:%v/1024.html", b.TrafficType.Scheme(), b.HostAddr, b.Port)
}

type BackendsByTrafficType map[TrafficType][]Backend

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
	net.LookupIP(hostname)
	return hostname
}

// TODO fix me
func getOutboundIPAddr() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
