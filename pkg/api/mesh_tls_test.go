package api

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"
)

func TestEphemeralCertParses(t *testing.T) {
	certPEM, keyPEM, fp, err := ephemeralServerCert()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(fp) == 0 || len(keyPEM) == 0 {
		t.Fatal("empty outputs")
	}
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if time.Until(cert.NotAfter) > 25*time.Hour {
		t.Fatal("cert lifetime too long")
	}
	foundLocalhost := false
	for _, dns := range cert.DNSNames {
		if dns == "localhost" {
			foundLocalhost = true
		}
	}
	if !foundLocalhost {
		t.Fatalf("missing localhost SAN: %v", cert.DNSNames)
	}
}

func TestMeshServerStopBeforeStart(t *testing.T) {
	s := NewMeshServer()
	s.Stop() // must not panic
}

func TestMeshServerStartStop(t *testing.T) {
	t.Setenv("MESH_INSECURE", "1")
	s := NewMeshServer()
	done := make(chan error, 1)
	go func() { done <- s.Start("18051") }()
	time.Sleep(500 * time.Millisecond)
	s.Stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
	StopMeshServers() // registry path must not panic
}
