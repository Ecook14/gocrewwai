//go:build webdist

package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDistServesBuiltApp verifies the embedded production bundle. This test
// only compiles with -tags webdist (which requires web/dist from
// `npm run build`); the placeholder tests in embed_test.go cover the
// default no-dist build.
func TestDistServesBuiltApp(t *testing.T) {
	srv := httptest.NewServer(http.FileServer(GetFS()))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	html := string(body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(html, `<div id="root">`) {
		t.Error("dist/index.html does not contain the React root; wrong file embedded?")
	}
	if strings.Contains(html, "not built") {
		t.Error("served the placeholder instead of the built app")
	}

	// Every local asset referenced by index.html (script src, stylesheet href)
	// must resolve inside the embedded FS, or the app loads as a blank page.
	for _, ref := range assetRefs(t, html) {
		resp, err := http.Get(srv.URL + ref)
		if err != nil {
			t.Fatalf("GET %s: %v", ref, err)
		}
		n, _ := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("asset %s status = %d, want 200", ref, resp.StatusCode)
		}
		if n == 0 {
			t.Errorf("asset %s is empty", ref)
		}
	}
}

// assetRefs extracts same-origin /assets/... references from built HTML.
func assetRefs(t *testing.T, html string) []string {
	t.Helper()
	var out []string
	for _, prefix := range []string{`src="`, `href="`} {
		rest := html
		for {
			i := strings.Index(rest, prefix)
			if i < 0 {
				break
			}
			rest = rest[i+len(prefix):]
			j := strings.Index(rest, `"`)
			if j < 0 {
				break
			}
			ref := rest[:j]
			rest = rest[j+1:]
			if strings.HasPrefix(ref, "/assets/") {
				out = append(out, ref)
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("no /assets/ references found in dist/index.html")
	}
	return out
}
