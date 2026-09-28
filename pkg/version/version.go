// Package version exposes the single source of truth for the build version.
//
// The value is meant to be overridden at link time so binaries and release
// tags can never disagree:
//
//	go build -ldflags "-X github.com/Ecook14/gocrewwai/pkg/version.Version=v1.0.0"
//
// When built without that flag (local dev, `go run`, `go test`) Version
// reports "dev" rather than a stale release number, so a local build is never
// mistaken for a shipped one.
package version

import "strings"

// Version is overridden at link time via -ldflags -X. Keep the "dev" default.
var Version = "dev"

// Commit is set the same way (full SHA preferred). Optional.
var Commit = "unknown"

// Display returns the version for user-facing output with any leading "v"
// removed, so an unset build shows "dev" rather than "vdev".
func Display() string {
	s := Version
	if s == "dev" || s == "" {
		return "dev"
	}
	return strings.TrimPrefix(s, "v")
}

// String returns a human-readable version string for CLI and banner output.
func String() string {
	if Commit != "" && Commit != "unknown" {
		return Display() + " (" + shortCommit() + ")"
	}
	return Display()
}

func shortCommit() string { return shortCommitOf(Commit) }

func shortCommitOf(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}
