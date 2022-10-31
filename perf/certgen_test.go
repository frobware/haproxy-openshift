package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCertGen(t *testing.T) {
	certBundle, err := CreateTLSCerts(time.Now(), time.Now().AddDate(1, 0, 0),
		mustResolveHostname(),
		"localhost",
		"127.0.0.1",
		"::1")

	if err != nil {
		t.Fatalf("failed to generate certificates: %v", err)
	}

	serverCert, err := tls.X509KeyPair([]byte(certBundle.LeafCertPEM), []byte(certBundle.LeafKeyPEM))
	if err != nil {
		t.Fatalf("failed to create key pair: %v", err)
	}

	serverTLSConf := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "success!")
	}))

	server.TLS = serverTLSConf
	server.StartTLS()
	defer server.Close()

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM([]byte(certBundle.RootCACertPEM))

	clientTLSConf := &tls.Config{
		RootCAs: certpool,
	}

	transport := &http.Transport{
		TLSClientConfig: clientTLSConf,
	}
	http := http.Client{
		Transport: transport,
	}

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	body := strings.TrimSpace(string(respBodyBytes[:]))
	if body != "success!" {
		t.Fatalf(`expected "success", got %q`, body)
	}
}
