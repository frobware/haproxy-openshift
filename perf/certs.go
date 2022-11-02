package main

import (
	"os"
	"path"
	"strings"
)

type CertStore struct {
	DomainFile    string
	RootCAFile    string
	RootCAKeyFile string
	TLSCertFile   string
	TLSKeyFile    string
}

func certStore(certDir string) CertStore {
	return CertStore{
		DomainFile:    path.Join(certDir, "domain.pem"),
		RootCAFile:    path.Join(certDir, "rootCA.pem"),
		RootCAKeyFile: path.Join(certDir, "rootCA-key.pem"),
		TLSKeyFile:    path.Join(certDir, "tls.key"),
		TLSCertFile:   path.Join(certDir, "tls.crt"),
	}
}

func writeCertificates(dir string, certs *Certificates) (*CertStore, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	certPath := certStore(dir)

	domainPEM := strings.Join([]string{
		strings.TrimSuffix(certs.LeafCertPEM, "\n"),
		strings.TrimSuffix(certs.LeafKeyPEM, "\n"),
		strings.TrimSuffix(certs.RootCACertPEM, "\n"),
	}, "\n")

	for _, cert := range []struct {
		filename string
		pemData  string
	}{
		{certPath.DomainFile, domainPEM},
		{certPath.RootCAFile, certs.RootCACertPEM},
		{certPath.RootCAKeyFile, certs.RootCAKeyPEM},
		{certPath.TLSCertFile, certs.LeafCertPEM},
		{certPath.TLSKeyFile, certs.LeafKeyPEM},
	} {
		if err := createFile(cert.filename, []byte(strings.TrimSuffix(cert.pemData, "\n"))); err != nil {
			return nil, err
		}
	}

	return &certPath, nil
}
