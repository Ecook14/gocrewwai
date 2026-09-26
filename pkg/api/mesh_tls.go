package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// ephemeralServerCert generates a 24h self-signed ECDSA certificate for the
// mesh server when the operator provides none. Encryption without identity
// trust: safe against passive eavesdropping, not against active MITM —
// clients should still pin MESH_TLS_CA in hostile networks. Returns PEM
// blocks and the SPKI fingerprint for log correlation.
func ephemeralServerCert() (certPEM, keyPEM []byte, fingerprint string, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, "", fmt.Errorf("mesh: ephemeral key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, "", fmt.Errorf("mesh: serial: %w", err)
	}
	host, _ := os.Hostname()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "gocrewwai-mesh-ephemeral"},
		NotBefore:    time.Now().Add(-5 * time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	if host != "" {
		tmpl.DNSNames = append(tmpl.DNSNames, host)
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, "", fmt.Errorf("mesh: ephemeral cert: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, "", fmt.Errorf("mesh: marshal key: %w", err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	sum := sha256.Sum256(der)
	return certPEM, keyPEM, hex.EncodeToString(sum[:])[:16], nil
}

// serverTLSCreds resolves the server TLS credentials: files win, otherwise
// an ephemeral self-signed cert, unless MESH_INSECURE=1 explicitly opts out.
func serverTLSCreds(certFile, keyFile, clientCAFile string) (*tls.Config, string, error) {
	if certFile == "" {
		certFile = os.Getenv("MESH_TLS_CERT")
	}
	if keyFile == "" {
		keyFile = os.Getenv("MESH_TLS_KEY")
	}
	if clientCAFile == "" {
		clientCAFile = os.Getenv("MESH_TLS_CLIENT_CA")
	}
	if certFile != "" && keyFile != "" {
		cfg, err := LoadTLSConfig(MeshServerCredentialConfig{
			EnableTLS:    true,
			CertFile:     certFile,
			KeyFile:      keyFile,
			ClientCAFile: clientCAFile,
		})
		return cfg, "file", err
	}
	if os.Getenv("MESH_INSECURE") == "1" {
		return nil, "insecure", nil
	}
	certPEM, keyPEM, fp, err := ephemeralServerCert()
	if err != nil {
		return nil, "", err
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, "", fmt.Errorf("mesh: ephemeral keypair: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, "ephemeral:" + fp, nil
}
