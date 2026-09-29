package api

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/events"
)

func fp(token string) string { return tokenFingerprint(token) }

func TestSessionOwnerIsolation(t *testing.T) {
	s := NewServer()
	ownerA, ownerB := fp("tok-A"), fp("tok-B")
	if err := s.persistSessionStart("sess-1", ownerA); err != nil {
		t.Fatalf("persist: %v", err)
	}
	if _, err := s.loadSessionState("sess-1", ownerA); err != nil {
		t.Fatalf("owner read: %v", err)
	}
	if _, err := s.loadSessionState("sess-1", ownerB); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-owner read: got %v, want not-found", err)
	}
}

func TestIdempotencyLedger(t *testing.T) {
	s := NewServer()
	if _, dup := s.checkIdem("k1", fp("A")); dup {
		t.Fatal("unseen key reported dup")
	}
	s.storeIdem("k1", fp("A"), "s1")
	e, dup := s.checkIdem("k1", fp("A"))
	if !dup || e.done || e.sessionID != "s1" {
		t.Fatalf("stored entry wrong: %+v %v", e, dup)
	}
	if _, dup := s.checkIdem("k1", fp("B")); dup {
		t.Fatal("cross-owner key visible")
	}
	s.finishIdem("s1", "completed")
	e, dup = s.checkIdem("k1", fp("A"))
	if !dup || !e.done || e.status != "completed" {
		t.Fatalf("finished entry wrong: %+v %v", e, dup)
	}
}

func TestMultiTokenAuth(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "")
	t.Setenv("API_AUTH_TOKENS", "tok-A, tok-B")
	s := NewServer()
	for _, tok := range []string{"tok-A", "tok-B"} {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("token %s: got %d", tok, w.Code)
		}
	}
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("Authorization", "Bearer nope")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("bad token: got %d, want 401", w.Code)
	}
}

// TestStaticPathsBypassAuth: non-/api/ paths (the --web frontend shell and
// its assets) must load without a bearer token, because a browser navigation
// cannot attach an Authorization header. Every /api/* endpoint stays gated.
func TestStaticPathsBypassAuth(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "tok-A")
	t.Setenv("API_AUTH_TOKENS", "")
	s := NewServer()
	if err := s.ServeStatic(stubStaticFS{}); err != nil {
		t.Fatalf("ServeStatic: %v", err)
	}

	for _, path := range []string{"/", "/index.html", "/assets/app.js"} {
		// Served through a live test server so net/http's canonicalization
		// redirects (/index.html -> ./) are followed like a browser would.
		srv := httptest.NewServer(s.router)
		resp, err := http.Get(srv.URL + path)
		srv.Close()
		if err != nil {
			t.Fatalf("static %s without token: %v", path, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("static %s without token: got %d, want 200", path, resp.StatusCode)
		}
	}

	// The API surface is unchanged: no token still means 401.
	for _, path := range []string{"/api/v1/health", "/api/v1/sessions/x", "/api/v1/stream/x"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("api %s without token: got %d, want 401", path, w.Code)
		}
	}
}

// stubStaticFS serves a fixed body for any path, standing in for the
// embedded frontend in this auth-focused test.
type stubStaticFS struct{}

func (stubStaticFS) Open(name string) (http.File, error) {
	// net/http requires "/" to be a directory and resolves index.html
	// inside it; every other path serves the stub body directly.
	if name == "/" || name == "" || name == "." {
		return stubStaticDir{}, nil
	}
	return &stubStaticFile{
		Reader: bytes.NewReader([]byte("<html>stub</html>")),
		name:   strings.TrimPrefix(name, "/"),
	}, nil
}

type stubStaticFile struct {
	*bytes.Reader
	name string
}

func (f *stubStaticFile) Close() error { return nil }
func (f *stubStaticFile) Stat() (fs.FileInfo, error) {
	return stubStaticInfo{name: f.name, size: int64(f.Len())}, nil
}
func (f *stubStaticFile) Readdir(int) ([]fs.FileInfo, error) {
	return nil, errors.New("not a directory")
}

type stubStaticDir struct{}

func (stubStaticDir) Close() error { return nil }
func (stubStaticDir) Read([]byte) (int, error) {
	return 0, errors.New("is a directory")
}
func (stubStaticDir) Seek(int64, int) (int64, error) {
	return 0, errors.New("is a directory")
}
func (stubStaticDir) Stat() (fs.FileInfo, error) {
	return stubStaticInfo{name: "/", size: 0, dir: true}, nil
}
func (stubStaticDir) Readdir(int) ([]fs.FileInfo, error) {
	return []fs.FileInfo{}, nil
}

type stubStaticInfo struct {
	name string
	size int64
	dir  bool
}

func (i stubStaticInfo) Name() string       { return i.name }
func (i stubStaticInfo) Size() int64        { return i.size }
func (i stubStaticInfo) Mode() fs.FileMode  { return 0o444 }
func (i stubStaticInfo) ModTime() time.Time { return time.Time{} }
func (i stubStaticInfo) IsDir() bool        { return i.dir }
func (i stubStaticInfo) Sys() any           { return nil }

func TestSSEUnknownSession404(t *testing.T) {
	t.Setenv("API_AUTH_TOKEN", "tok-A")
	t.Setenv("API_AUTH_TOKENS", "")
	s := NewServer()
	req := httptest.NewRequest("GET", "/api/v1/stream/ghost", nil)
	req.Header.Set("Authorization", "Bearer tok-A")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown stream: got %d, want 404", w.Code)
	}
}

func TestSSECrossOwner404(t *testing.T) {
	t.Setenv("API_AUTH_TOKENS", "tok-A, tok-B")
	t.Setenv("API_AUTH_TOKEN", "")
	s := NewServer()
	if err := s.persistSessionStart("sess-x", fp("tok-A")); err != nil {
		t.Fatalf("persist: %v", err)
	}
	req := httptest.NewRequest("GET", "/api/v1/stream/sess-x", nil)
	req.Header.Set("Authorization", "Bearer tok-B")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-owner stream: got %d, want 404", w.Code)
	}
}

func TestPassSSEEvent(t *testing.T) {
	if !passSSEEvent("s1", events.Event{SessionID: "s1"}) {
		t.Fatal("own session dropped")
	}
	if passSSEEvent("s1", events.Event{SessionID: "s2"}) {
		t.Fatal("foreign session passed")
	}
	if passSSEEvent("s1", events.Event{}) {
		t.Fatal("untagged event passed")
	}
}
