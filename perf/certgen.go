package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

type Certificates struct {
	LeafCertPEM   string
	LeafKeyPEM    string
	RootCACertPEM string
	RootCAKeyPEM  string
}

// CreateTLSCerts generates self-signed certificates suitable for
// client/server tls.Config.
func CreateTLSCerts(notBefore, notAfter time.Time, alternateNames ...string) (*Certificates, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %v", err)
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %v", err)
	}

	ca := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"perf development certificate"},
			OrganizationalUnit: []string{"perf dept"},
			CommonName:         "perf",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		MaxPathLenZero:        true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, &ca, &ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create rooot certificate: %v\n", err)
	}

	caPEM := new(bytes.Buffer)
	if err := pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}); err != nil {
		return nil, err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	if err := pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	}); err != nil {
		return nil, err
	}

	// server certificate
	cert := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"perf development certificate"},
			OrganizationalUnit: []string{"perf dept"},
		},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	for _, host := range alternateNames {
		if ip := net.ParseIP(host); ip != nil {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		} else {
			cert.DNSNames = append(cert.DNSNames, host)
		}
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %v", err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, &ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create leaf certificate: %v", err)
	}

	certPEM := new(bytes.Buffer)
	if err := pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return nil, err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	if err := pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	}); err != nil {
		return nil, err
	}

	return &Certificates{
		LeafCertPEM:   certPEM.String(),
		LeafKeyPEM:    certPrivKeyPEM.String(),
		RootCACertPEM: caPEM.String(),
		RootCAKeyPEM:  caPrivKeyPEM.String(),
	}, nil
}
