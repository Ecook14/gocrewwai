package utils

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// MaxCleanComponents is a rough bound on how deep a cleaned path may be,
// used to reject degenerate traversals before we even touch the filesystem.
const MaxCleanComponents = 256

// ValidatePath checks if a given path is within the allowed 'chroot' directory.
//
// Security notes:
//   - Symlinks: this function does NOT canonicalize symlinks (no os.Stat /
//     os.Readlink). Callers that operate on a path that may be a symlink must
//     re-check the resolved target themselves (e.g. by opening with O_NOFOLLOW
//     or calling filepath.EvalSymlinks on the already-open file descriptor).
//   - chroot defaults: an empty chroot is treated as "." which is resolved to
//     the *current working directory*. This is acceptable only when the process
//     CWD is already the intended root. For absolute, unambiguous roots pass an
//     absolute path (e.g. "/var/data/chroot").
func ValidatePath(path string, chroot string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}

	if chroot == "" {
		// Default to CWD. If you need a hard absolute root, pass one.
		chroot = "."
	}

	// Clean first so that trivial ".." segments are collapsed before we
	// measure depth and resolve absolutes. Clean does NOT follow symlinks.
	cleanPath := filepath.Clean(path)

	// Reject absurdly deep clean paths (reflects a traversal attempt or a
	// misbehaving caller) early, before any I/O.
	parts := strings.Split(filepath.ToSlash(cleanPath), string(filepath.Separator))
	// Filter out empty strings from leading/trailing separators
	cleanParts := 0
	for _, p := range parts {
		if p != "" {
			cleanParts++
		}
	}
	if cleanParts > MaxCleanComponents {
		return "", fmt.Errorf("path too deep (%d components)", cleanParts)
	}

	absChroot, err := filepath.Abs(chroot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve chroot directory: %w", err)
	}
	// Ensure the chroot itself is clean so HasPrefix comparisons are stable.
	absChroot = filepath.Clean(absChroot)

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	absPath = filepath.Clean(absPath)

	// Ensure the path is within the chroot.
	if !strings.HasPrefix(absPath, absChroot) {
		return "", fmt.Errorf("security violation: path %s is outside allowed directory %s", path, chroot)
	}

	// Post-check: the path must be exactly the chroot or a descendant.
	// Without this, a chroot of "/data" would also accept "/dataX/foo".
	if absPath != absChroot && !strings.HasPrefix(absPath, absChroot+string(filepath.Separator)) {
		return "", fmt.Errorf("security violation: path %s is outside allowed directory %s", path, chroot)
	}

	return absPath, nil
}

// FileWriteSanitize is called by file_write.go after filepath.Clean to verify
// the cleaned path still lands inside the chroot. This catches cases where
// Clean strips a leading ".." but the resulting absolute path escapes because
// the input was something like "foo/../../../etc/passwd" resolved from a CWD
// outside the root.
func FileWriteSanitize(cleanedPath, chroot string) (string, error) {
	return ValidatePath(cleanedPath, chroot)
}

// ValidatePathResolved is the symlink-hardened variant. It first runs
// ValidatePath, then resolves symlinks on the existing prefix via
// filepath.EvalSymlinks and re-checks containment. Callers opening files that
// may be symlinks must use this (or O_NOFOLLOW) — plain ValidatePath does not
// follow symlinks by design.
func ValidatePathResolved(path, chroot string) (string, error) {
	absPath, err := ValidatePath(path, chroot)
	if err != nil {
		return "", err
	}
	absChroot, err := filepath.Abs(chroot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve chroot directory: %w", err)
	}
	absChroot = filepath.Clean(absChroot)
	// Resolve the longest existing prefix so non-existent leaf files still validate.
	target := absPath
	for {
		if _, statErr := os.Lstat(target); statErr == nil {
			break
		}
		parent := filepath.Dir(target)
		if parent == target {
			break
		}
		target = parent
		if len(target) < len(absChroot) {
			break
		}
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		// If nothing exists yet, fall back to the lexical check.
		return absPath, nil
	}
	resolved = filepath.Clean(resolved)
	if resolved != absChroot && !strings.HasPrefix(resolved, absChroot+string(filepath.Separator)) {
		return "", fmt.Errorf("security violation: symlink target %s escapes allowed directory %s", path, chroot)
	}
	return absPath, nil
}

// MaxURLBytes bounds outbound fetch sizes for callers.
const MaxURLBytes = 10 << 20 // 10MB

// ValidateURL rejects SSRF targets: non-http(s) schemes, embedded credentials,
// localhost/loopback, link-local metadata endpoints, and private-network IPs
// resolved via DNS. Returns the parsed URL on success.
//
// Test/dev exception: when GOCREW_ALLOW_PRIVATE_URLS=1, loopback and private
// IPs are permitted (for httptest servers). Never set in production.
func ValidateURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil || u.Host == "" {
		return nil, fmt.Errorf("invalid URL: %s", raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	if u.User != nil {
		return nil, errors.New("URL must not contain credentials")
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("invalid URL host: %s", raw)
	}
	allowPrivate := os.Getenv("GOCREW_ALLOW_PRIVATE_URLS") == "1"
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "metadata.google.internal" {
		if !allowPrivate {
			return nil, fmt.Errorf("blocked URL host %q", host)
		}
		return u, nil
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
			if !allowPrivate {
				return nil, fmt.Errorf("blocked private/link-local IP %q", host)
			}
		}
		return u, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return nil, fmt.Errorf("unable to resolve URL host %q", host)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
			if !allowPrivate {
				return nil, fmt.Errorf("blocked URL host %q resolves to private address", host)
			}
		}
	}
	return u, nil
}
