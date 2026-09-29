//go:build webdist

package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var distFS embed.FS

// GetFS returns the compiled Visual Builder assets. This file only builds
// with -tags webdist, which requires web/dist to exist at compile time —
// i.e. after `npm run build` in web/. Clean checkouts (no dist/) fall back
// to the honest placeholder in embed.go.
func GetFS() http.FileSystem {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// Unreachable in practice: the embed directive guarantees dist/.
		// Fall through to an empty FS rather than panicking the server.
		return http.FS(emptyFS{})
	}
	return http.FS(sub)
}

// emptyFS serves nothing. Only reachable if the embed above is somehow
// empty; it keeps ServeStatic from receiving a nil filesystem.
type emptyFS struct{}

func (emptyFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}
