package pubkey

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	"google.golang.org/grpc/credentials"
)

func GeneratePriv() ([]byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	return privBytes, nil
}

func Cert(privBytes []byte) ([]byte, error) {
	priv, err := x509.ParsePKCS1PrivateKey(privBytes)
	if err != nil {
		return nil, fmt.Errorf("x509.ParsePKCS1PrivateKey: %v", err)
	}
	notBefore := time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC)
	notAfter := time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"server"},
		IsCA:                  true,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}
	return certBytes, nil
}

func ServerCreds(privBytes, certBytes []byte) (credentials.TransportCredentials, error) {
	priv, err := x509.ParsePKCS1PrivateKey(privBytes)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}
	cert := &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  priv,
		Leaf:        leaf,
	}
	return credentials.NewServerTLSFromCert(cert), nil
}

func ClientCreds(certBytes []byte) (credentials.TransportCredentials, error) {
	leaf, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	config := &tls.Config{
		RootCAs: pool,
	}
	return credentials.NewTLS(config), nil
}
