package main

import (
	"log"
	"net"
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
