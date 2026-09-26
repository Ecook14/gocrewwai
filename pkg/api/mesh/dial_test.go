package mesh

import (
	"testing"
)

func TestResolveTransportCAWins(t *testing.T) {
	t.Setenv("MESH_TLS_CA", "/tmp/ca.pem")
	t.Setenv("MESH_INSECURE", "1")
	useTLS, skip, err := ResolveTransport("10.0.0.5:50051")
	if err != nil || !useTLS || skip {
		t.Fatalf("CA must win: %v %v %v", useTLS, skip, err)
	}
}

func TestResolveTransportLoopback(t *testing.T) {
	t.Setenv("MESH_TLS_CA", "")
	t.Setenv("MESH_INSECURE", "")
	for _, addr := range []string{"localhost:50051", "127.0.0.1:50051", "127.0.1.20:9", "[::1]:50051"} {
		useTLS, skip, err := ResolveTransport(addr)
		if err != nil || !useTLS || !skip {
			t.Fatalf("%s: got %v %v %v", addr, useTLS, skip, err)
		}
	}
}

func TestResolveTransportRemoteRefuses(t *testing.T) {
	t.Setenv("MESH_TLS_CA", "")
	t.Setenv("MESH_INSECURE", "")
	if _, _, err := ResolveTransport("10.0.0.5:50051"); err == nil {
		t.Fatal("remote plaintext without CA must fail closed")
	}
}

func TestResolveTransportExplicitInsecure(t *testing.T) {
	t.Setenv("MESH_TLS_CA", "")
	t.Setenv("MESH_INSECURE", "1")
	useTLS, _, err := ResolveTransport("10.0.0.5:50051")
	if err != nil || useTLS {
		t.Fatalf("explicit insecure: %v %v", useTLS, err)
	}
}
