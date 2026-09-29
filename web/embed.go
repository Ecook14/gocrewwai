//go:build !webdist

// Package web provides embedded static-asset serving for the API server.
// The Visual Builder is a Vite + React application that lives in web/src/.
// It is NOT embedded here: browsers cannot execute the raw .tsx source, and
// the project has no build step in the release pipeline yet, so there is no
// dist/ directory to embed. Serving src/ directly produced a directory
// listing of source files rather than a working app.
//
// Until a build step lands, GetFS serves an explicit "not built" page so the
// --web flag is honest instead of silently broken.
package web

import (
	"bytes"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

const notBuiltHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Gocrewwai Visual Builder &mdash; not built</title>
<style>
  :root { color-scheme: light dark; }
  body { font: 16px/1.6 ui-sans-serif, system-ui, sans-serif; margin: 0;
         display: grid; place-items: center; min-height: 100vh; }
  main { max-width: 42rem; padding: 2rem; }
  h1 { font-size: 1.4rem; margin: 0 0 .5rem; }
  code { background: rgba(127,127,127,.18); padding: .15em .4em; border-radius: 4px; }
  .muted { opacity: .7; font-size: .9rem; }
</style>
</head>
<body>
<main>
  <h1>Visual Builder is not built in this release</h1>
  <p>The Visual Builder is a Vite + React app under <code>web/src/</code>. It has no build
     step in the release pipeline yet, so this binary has no compiled assets to serve.</p>
  <p class="muted">Two things work today instead:<br>
     &bull; <code>gocrew kickoff --ui</code> &mdash; the live control panel on
         <code>:8080/web-ui</code><br>
     &bull; the REST API on <code>:8080</code> (see the README)</p>
</main>
</body>
</html>
`

const indexPage = "index.html"

// GetFS returns an http.FileSystem serving the "not built" placeholder.
// The root is a virtual directory whose sole entry is index.html, which is
// what net/http expects: mapping "/" straight to a file is rejected with
// "attempting to traverse a non-directory". It never returns nil, so server
// startup is safe regardless.
func GetFS() http.FileSystem {
	return singleFileFS{body: []byte(notBuiltHTML), mod: time.Unix(0, 0)}
}

// singleFileFS is a read-only FS exposing one virtual directory that contains
// a single file. Any path inside it resolves to the same body, so a deep link
// like /builder/agents still lands on the page rather than a listing.
type singleFileFS struct {
	body []byte
	mod  time.Time
}

func (f singleFileFS) Open(name string) (http.File, error) {
	clean := path.Clean("/" + name)
	base := path.Base(clean)
	isRoot := base == "/" || base == "."
	if isRoot {
		return &dirFile{name: "/", mod: f.mod}, nil
	}
	// The one real file, plus any directory ancestor needed for traversal.
	if base == indexPage || !strings.HasSuffix(clean, "/") {
		return &singleFile{Reader: bytes.NewReader(f.body), name: base, size: int64(len(f.body)), mod: f.mod}, nil
	}
	return &dirFile{name: base, mod: f.mod}, nil
}

type singleFile struct {
	*bytes.Reader
	name string
	size int64
	mod  time.Time
}

func (f *singleFile) Stat() (fs.FileInfo, error) {
	return fileInfo{name: f.name, size: f.size, mod: f.mod}, nil
}
func (f *singleFile) Close() error { return nil }
func (f *singleFile) Readdir(int) ([]fs.FileInfo, error) {
	return nil, errors.New("not a directory")
}

// dirFile satisfies http.File for virtual directories.
type dirFile struct {
	name string
	mod  time.Time
}

func (f *dirFile) Stat() (fs.FileInfo, error) {
	return fileInfo{name: f.name, dir: true, mod: f.mod}, nil
}
func (f *dirFile) Read([]byte) (int, error) {
	return 0, errors.New("is a directory")
}
func (f *dirFile) Seek(int64, int) (int64, error) {
	return 0, errors.New("is a directory")
}
func (f *dirFile) Close() error                       { return nil }
func (f *dirFile) Readdir(int) ([]fs.FileInfo, error) { return []fs.FileInfo{}, nil }

type fileInfo struct {
	name string
	size int64
	dir  bool
	mod  time.Time
}

func (i fileInfo) Name() string { return i.name }
func (i fileInfo) Size() int64  { return i.size }
func (i fileInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}
func (i fileInfo) ModTime() time.Time { return i.mod }
func (i fileInfo) IsDir() bool        { return i.dir }
func (i fileInfo) Sys() any           { return nil }

var (
	_ http.FileSystem = singleFileFS{}
	_ http.File       = (*singleFile)(nil)
	_ http.File       = (*dirFile)(nil)
	_ fs.FileInfo     = fileInfo{}
)
