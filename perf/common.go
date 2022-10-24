package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
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

	// //go:embed tls.crt
	// tlsCert string

	// //go:embed tls.key
	// tlsKey string
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

func mustWriteString(b *bytes.Buffer, s string) {
	if _, err := io.WriteString(b, s); err != nil {
		panic(err)
	}
}

func mustResolveCurrentHost() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	net.LookupIP(hostname)
	return hostname
}

// TODO fix me; we want anything but localhost returned.
func getOutboundIPAddr() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

func createFile(path string, data []byte) error {
	dirname := filepath.Dir(path)
	if err := os.MkdirAll(dirname, 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return f.Close()
}
