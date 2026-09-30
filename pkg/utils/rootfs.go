package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// maxRootFileBytes caps single-file reads through the root-confined helpers.
const maxRootFileBytes = 10 << 20 // 10MiB

// rootRel lexically validates name against chroot and returns the root-
// relative path plus an opened *os.Root. Callers must close the root.
func rootRel(chroot, name string) (root *os.Root, rel string, err error) {
	absPath, err := ValidatePath(name, chroot)
	if err != nil {
		return nil, "", err
	}
	absChroot, err := filepath.Abs(chroot)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve chroot directory: %w", err)
	}
	absChroot = filepath.Clean(absChroot)
	rel, err = filepath.Rel(absChroot, absPath)
	if err != nil || rel == "." || rel == ".." || len(rel) >= 2 && (rel[:3] == "../" || rel == "..") {
		return nil, "", fmt.Errorf("security violation: path %s escapes allowed directory %s", name, chroot)
	}
	// Belt-and-suspenders: also resolve symlinks on the existing prefix and
	// re-check, so pre-existing links cannot redirect the root-relative open.
	if _, err := ValidatePathResolved(name, chroot); err != nil {
		return nil, "", err
	}
	root, err = os.OpenRoot(absChroot)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open chroot: %w", err)
	}
	return root, rel, nil
}

// ReadFileInRoot reads a file confined to chroot. The open itself is bound
// to the root directory, so leaf and ancestor symlinks cannot redirect it
// outside — unlike a validate-then-os.ReadFile sequence, which races with
// directory changes.
func ReadFileInRoot(chroot, name string) ([]byte, error) {
	root, rel, err := rootRel(chroot, name)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open(rel)
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", name, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxRootFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file '%s': %w", name, err)
	}
	if int64(len(data)) > maxRootFileBytes {
		return nil, fmt.Errorf("file '%s' exceeds %d byte budget", name, maxRootFileBytes)
	}
	return data, nil
}

// WriteFileInRoot writes data to a file confined to chroot, creating parent
// directories inside the root as needed.
func WriteFileInRoot(chroot, name string, data []byte, perm os.FileMode) error {
	root, rel, err := rootRel(chroot, name)
	if err != nil {
		return err
	}
	defer root.Close()
	if dir := filepath.Dir(rel); dir != "." {
		if err := root.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create parent dirs for '%s': %w", name, err)
		}
	}
	f, err := root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to write to file '%s': %w", name, err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write to file '%s': %w", name, err)
	}
	return nil
}
