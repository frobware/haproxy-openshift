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

func writeCertificates(dir string, certBundle *Certificates) (*CertStore, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	certPath := certStore(dir)

	domainPEM := strings.Join([]string{
		strings.TrimSuffix(certBundle.LeafCertPEM, "\n"),
		strings.TrimSuffix(certBundle.LeafKeyPEM, "\n"),
		strings.TrimSuffix(certBundle.RootCACertPEM, "\n"),
	}, "\n")

	if err := createFile(certPath.DomainFile, []byte(strings.TrimSuffix(domainPEM, "\n"))); err != nil {
		return nil, err
	}

	if err := createFile(certPath.RootCAFile, []byte(certBundle.RootCACertPEM)); err != nil {
		return nil, err
	}

	if err := createFile(certPath.RootCAKeyFile, []byte(certBundle.RootCAKeyPEM)); err != nil {
		return nil, err
	}

	if err := createFile(certPath.TLSCertFile, []byte(certBundle.LeafCertPEM)); err != nil {
		return nil, err
	}

	if err := createFile(certPath.TLSKeyFile, []byte(certBundle.LeafKeyPEM)); err != nil {
		return nil, err
	}

	return &certPath, nil
}
