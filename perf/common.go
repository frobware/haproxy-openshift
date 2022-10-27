package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
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

type BackendsByTrafficType map[TrafficType][]Backend
type BoundBackendsByTrafficType map[TrafficType][]BoundBackend

func (b BoundBackend) URL() string {
	return fmt.Sprintf("%s://%s:%v/1024.html", b.TrafficType.Scheme(), b.HostAddr, b.Port)
}

func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	return hostname
}

func HostIPAddress() net.IP {
	// TODO; we want anything but 127.0.0.1 || ::1 returned.
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		panic(err)
	}
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
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

func fetchAllBackendMetadata(uri string) (BoundBackendsByTrafficType, error) {
	url := fmt.Sprintf("%s/backends?json=1", uri)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var backendsByType BoundBackendsByTrafficType
		if err := json.Unmarshal(body, &backendsByType); err != nil {
			return nil, err
		}
		return backendsByType, nil
	}
	return nil, fmt.Errorf("request failed %v", resp.StatusCode)
}
