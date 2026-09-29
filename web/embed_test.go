package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// get serves path through http.FileServer and follows the canonicalization
// redirects net/http issues (e.g. /index.html -> ./), returning the final
// status and body.
func get(t *testing.T, path string) (int, string) {
	t.Helper()
	srv := httptest.NewServer(http.FileServer(GetFS()))
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return nil },
	}
	resp, err := client.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body %s: %v", path, err)
	}
	return resp.StatusCode, string(b)
}

// The Visual Builder has no build step yet, so GetFS must never serve the raw
// .tsx source. Regression: serving src/ produced a directory listing of
// App.tsx / components/ / main.tsx instead of an app.
func TestGetFSNeverServesSourceListing(t *testing.T) {
	for _, p := range []string{"/", "/index.html", "/App.tsx", "/components/", "/main.tsx", "/anything"} {
		code, body := get(t, p)
		if code != http.StatusOK {
			t.Errorf("path %q: status = %d, want 200", p, code)
			continue
		}
		if strings.Contains(body, "<a href=") {
			t.Errorf("path %q: served a directory listing instead of the placeholder:\n%s", p, body)
		}
		if !strings.Contains(body, "not built") {
			t.Errorf("path %q: placeholder text missing, got:\n%s", p, body)
		}
	}
}

func TestGetFSNeverReturnsNil(t *testing.T) {
	if GetFS() == nil {
		t.Fatal("GetFS() = nil; server startup would break")
	}
}

func TestRootIsADirectory(t *testing.T) {
	// net/http rejects a FileSystem that maps "/" to a file with
	// "attempting to traverse a non-directory" (500). The root must be a
	// directory containing index.html.
	f, err := GetFS().Open("/")
	if err != nil {
		t.Fatalf("Open(/): %v", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("Open(\"/\") is not a directory; http.FileServer will return 500")
	}
}

func TestSingleFileSupportsSeekReadAndStat(t *testing.T) {
	f, err := GetFS().Open("/index.html")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.IsDir() {
		t.Error("IsDir() = true, want false")
	}
	if info.Name() != "index.html" {
		t.Errorf("Name() = %q, want index.html", info.Name())
	}
	if info.Size() <= 0 {
		t.Error("Size() <= 0")
	}

	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(b) == 0 {
		t.Error("read 0 bytes from placeholder")
	}
	if _, err := f.Readdir(1); err == nil {
		t.Error("Readdir on a file should error")
	}
}
