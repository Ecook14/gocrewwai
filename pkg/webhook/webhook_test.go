package webhook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNotify_RoundTrip(t *testing.T) {
	t.Setenv("GOCREW_ALLOW_PRIVATE_URLS", "1")
	var gotBody []byte
	var gotSig, gotTs, gotEvent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		gotBody = body
		gotSig, gotTs, gotEvent = r.Header.Get("X-Gocrew-Signature"), r.Header.Get("X-Gocrew-Timestamp"), r.Header.Get("X-Gocrew-Event")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n, err := NewNotifier(srv.URL, "s3cret")
	if err != nil {
		t.Fatalf("notifier: %v", err)
	}
	if err := n.Notify(context.Background(), Payload{Event: "kickoff.completed", Status: "ok"}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if gotEvent != "kickoff.completed" {
		t.Errorf("event = %q", gotEvent)
	}
	if err := VerifySignature("s3cret", gotBody, gotSig, gotTs, 5*time.Minute); err != nil {
		t.Errorf("verify: %v", err)
	}
	if err := VerifySignature("wrong", gotBody, gotSig, gotTs, 5*time.Minute); err == nil {
		t.Error("wrong secret accepted")
	}
}

func TestNewNotifier_BlockedURL(t *testing.T) {
	os.Unsetenv("GOCREW_ALLOW_PRIVATE_URLS")
	if _, err := NewNotifier("http://169.254.169.254/x", "s"); err == nil {
		t.Error("SSRF target accepted")
	}
	if _, err := NewNotifier("ftp://example.com/x", "s"); err == nil {
		t.Error("non-http scheme accepted")
	}
}

func TestNotify_RedirectBlocked(t *testing.T) {
	t.Setenv("GOCREW_ALLOW_PRIVATE_URLS", "1")
	// Evil receiver redirects to link-local metadata. Delivery must fail
	// rather than follow the redirect past the SSRF gate.
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/", http.StatusFound)
	}))
	defer evil.Close()
	n, err := NewNotifier(evil.URL, "s3cret")
	if err != nil {
		t.Fatalf("notifier: %v", err)
	}
	// Production posture for the delivery leg: no private-URL exception,
	// so the redirect hop must hit the SSRF gate.
	t.Setenv("GOCREW_ALLOW_PRIVATE_URLS", "")
	err = n.Notify(context.Background(), Payload{Event: "kickoff.completed", Status: "ok"})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected SSRF-gate error following redirect to metadata IP, got: %v", err)
	}
}

func TestVerifySignature_Stale(t *testing.T) {
	body := []byte(`{"event":"x"}`)
	sig := "sha256=" + Sign("s", body)
	old := "1"
	if err := VerifySignature("s", body, sig, old, 5*time.Minute); err == nil {
		t.Error("stale timestamp accepted")
	}
	if !strings.HasPrefix(sig, "sha256=") {
		t.Error("signature format")
	}
}
