package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePathResolvedDanglingSymlink(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "dangling")
	// Existing symlink whose target does not exist outside the root.
	if err := os.Symlink(filepath.Join(t.TempDir(), "nope"), link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if _, err := ValidatePathResolved(link, root); err == nil {
		t.Error("existing dangling symlink must fail closed")
	}
}

func TestValidatePathResolvedNewFileAllowed(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "newdir", "file.txt")
	got, err := ValidatePathResolved(p, root)
	if err != nil {
		t.Fatalf("new file beneath safe parent must validate: %v", err)
	}
	if got == "" {
		t.Error("expected resolved path")
	}
	if _, err := ValidatePathResolved(filepath.Join(root, "..", "escape"), root); err == nil {
		t.Error("traversal must be rejected")
	}
}

func TestFindMatchingBlockBudgets(t *testing.T) {
	if _, _, ok := FindMatchingBlock("abc", ""); ok {
		t.Error("empty search must not match")
	}
	if _, _, ok := FindMatchingBlock("abc", string(make([]byte, maxMatchTargetBytes+1))); ok {
		t.Error("oversize target must not match")
	}
	start, end, ok := FindMatchingBlock("  hello  \nworld\n", "hello\nworld")
	if !ok || start != 0 || end <= 0 {
		t.Errorf("normalized match failed: %d %d %v", start, end, ok)
	}
}
