package mesh

import (
	"testing"
)

// A remote DNS name carrying a "127." prefix must not receive the loopback
// trust exception (CWE-295).
func TestResolveTransportPrefixImpostor(t *testing.T) {
	t.Setenv("MESH_TLS_CA", "")
	t.Setenv("MESH_INSECURE", "")
	for _, addr := range []string{"127.evil.com:50051", "127.0.0.1.evil.com:50051", "10.0.0.5:50051"} {
		if _, _, err := ResolveTransport(addr); err == nil {
			t.Errorf("%s: must fail closed without CA", addr)
		}
	}
}

func TestResolveTransportLiteralLoopbackKept(t *testing.T) {
	t.Setenv("MESH_TLS_CA", "")
	t.Setenv("MESH_INSECURE", "")
	for _, addr := range []string{"127.0.0.1:50051", "127.0.1.20:9", "[::1]:50051"} {
		useTLS, skip, err := ResolveTransport(addr)
		if err != nil || !useTLS || !skip {
			t.Errorf("%s: got %v %v %v", addr, useTLS, skip, err)
		}
	}
}
