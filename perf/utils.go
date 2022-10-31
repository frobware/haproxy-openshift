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

func mustResolveHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	return hostname
}

func mustResolveHostIP() string {
	// TODO; we want anything but 127.0.0.1 || ::1 returned.
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		log.Fatal(err)
	}
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
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

func fetchCertficates(uri string) (*CertificateBundle, error) {
	url := fmt.Sprintf("%s/certs?json=1", uri)
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
		var certBundle CertificateBundle
		if err := json.Unmarshal(body, &certBundle); err != nil {
			return nil, err
		}
		return &certBundle, nil
	}
	return nil, fmt.Errorf("request failed %v", resp.StatusCode)
}
