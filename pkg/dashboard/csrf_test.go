package dashboard

import (
	"net/http"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://localhost:8080/api/delete", nil)
	req.Host = "localhost:8080"
	if !sameOrigin(req) {
		t.Error("no Origin/Referer (curl) must be allowed")
	}
	req.Header.Set("Origin", "http://localhost:8080")
	if !sameOrigin(req) {
		t.Error("matching Origin must be allowed")
	}
	req.Header.Set("Origin", "http://evil.com")
	if sameOrigin(req) {
		t.Error("cross-origin must be rejected")
	}
	req.Header.Del("Origin")
	req.Header.Set("Referer", "http://evil.com/x")
	if sameOrigin(req) {
		t.Error("cross-origin Referer must be rejected")
	}
}
