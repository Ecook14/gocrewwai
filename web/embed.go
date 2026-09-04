// Package web provides stub static-asset serving for the API server.
// The full React/Vite dashboard lives at https://gocrewwai-ui.vercel.app
// (Cloud UI), so the in-binary dashboard is intentionally a no-op FS.
package web

import "io/fs"

// GetFS returns an empty in-memory filesystem. The Visual Builder lives
// at gocrewwai-ui.vercel.app and is the canonical UI for the Cloud.
func GetFS() fs.FS {
	return nil
}
