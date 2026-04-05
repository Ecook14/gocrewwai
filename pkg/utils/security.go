package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	//"strings"
)

// ValidatePath checks if a given path is within the allowed 'chroot' directory.
func ValidatePath(path string, chroot string) (string, error) {
	if chroot == "" {
		chroot = "." // Default to current directory
	}

	absChroot, err := filepath.Abs(chroot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve chroot directory: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Ensure the path is within the chroot
	if !strings.HasPrefix(absPath, absChroot) {
		return "", fmt.Errorf("security violation: path %s is outside allowed directory %s", path, chroot)
	}

	return absPath, nil
}
