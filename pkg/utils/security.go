package utils

import (
	"errors"
	"fmt"
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
