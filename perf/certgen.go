package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

type CertificateBundle struct {
	LeafCertPEM   string
	LeafKeyPEM    string
	RootCACertPEM string
	RootCAKeyPEM  string
}

// MarshalKeyToDERBlock converts the key to a string representation
// (SEC 1, ASN.1 DER form) suitable for dropping into TLS key.
func MarshalKeyToDERBlock(key *ecdsa.PrivateKey) ([]byte, error) {
	data, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal private key: %v", err)
	}

	buf := &bytes.Buffer{}
	if err := pem.Encode(buf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: data}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// MarshalCertToPEMBlock encodes derBytes to PEM format.
func MarshalCertToPEMBlock(derBytes []byte) ([]byte, error) {
	buf := &bytes.Buffer{}

	if err := pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return nil, fmt.Errorf("failed to encode cert data: %v", err)
	}

	return buf.Bytes(), nil
}

// CreateTLSCerts generates self-signed certificates suitable for
// client/server tls.Config .
func CreateTLSCerts(name pkix.Name, notBefore, notAfter time.Time, alternateNames ...string) (*CertificateBundle, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %v", err)
	}

	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDSA key: %v", err)
	}

	rootCA := x509.Certificate{
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		NotAfter:              notAfter,
		NotBefore:             notBefore,
		SerialNumber:          serialNumber,
		Subject:               name,
	}

	rootCAPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %v", err)
	}

	rootCABytes, err := x509.CreateCertificate(rand.Reader, &rootCA, &rootCA, &rootCAPrivateKey.PublicKey, rootCAPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create rooot certificate: %v\n", err)
	}

	cert := x509.Certificate{
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		NotAfter:              notAfter,
		NotBefore:             notBefore,
		SerialNumber:          serialNumber,
		Subject:               name,
	}

	for _, host := range alternateNames {
		if ip := net.ParseIP(host); ip != nil {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		} else {
			cert.DNSNames = append(cert.DNSNames, host)
		}
	}

	certPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDSA key: %v", err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, &rootCA, &certPrivateKey.PublicKey, rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create leaf certificate: %v", err)
	}

	// convert all keys/certs to PEM format for convenience.

	caPEM, err := MarshalCertToPEMBlock(rootCABytes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall root certificate: %v", err)
	}

	caPrivateKeyPEM, err := MarshalKeyToDERBlock(rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall root private key: %v", err)
	}

	leafCertPEM, err := MarshalCertToPEMBlock(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall leaf certificate: %v", err)
	}

	leafKeyPEM, err := MarshalKeyToDERBlock(certPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall leaf private key: %v", err)
	}

	return &CertificateBundle{
		LeafCertPEM:   string(leafCertPEM),
		LeafKeyPEM:    string(leafKeyPEM),
		RootCACertPEM: string(caPEM),
		RootCAKeyPEM:  string(caPrivateKeyPEM),
	}, nil
}
