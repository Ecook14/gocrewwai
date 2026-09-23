// Package web provides embedded static-asset serving for the API server.
// The Visual Builder lives in web/src/ and is embedded below; the canonical
// cloud-hosted UI is https://gocrewwai-ui.vercel.app
package web

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed src
var srcFS embed.FS

// GetFS returns an http.FileSystem for the embedded UI source.
// Returns a non-nil FS even when src/ is empty.
func GetFS() http.FileSystem {
	sub, err := fs.Sub(srcFS, "src")
	if err != nil {
		// If embed failed (e.g. src/ empty), return an empty FS so the
		// server still starts — serving nothing rather than panicking.
		log.Printf("web: embed src/ failed (%v), serving empty FS", err)
		return http.FS(os.DirFS("."))
	}
	return http.FS(sub)
}
