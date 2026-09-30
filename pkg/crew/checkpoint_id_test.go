package crew

import (
	"strings"
	"testing"
)

func TestValidCheckpointID(t *testing.T) {
	valid := []string{"sess_123", "abc-XYZ.9", "a"}
	for _, id := range valid {
		if !ValidCheckpointID(id) {
			t.Errorf("ValidCheckpointID(%q) = false, want true", id)
		}
	}
	invalid := []string{"", "../escape", "a/b", "a\\b", "a:b", "a?b", "a#b", "a%b", ".", "..", strings.Repeat("x", 129)}
	for _, id := range invalid {
		if ValidCheckpointID(id) {
			t.Errorf("ValidCheckpointID(%q) = true, want false", id)
		}
	}
}

func TestCheckpointJoinBaseContainment(t *testing.T) {
	cm := NewCheckpointManager(t.TempDir())
	p, err := cm.joinBase("checkpoint_abc_latest.json")
	if err != nil {
		t.Fatalf("joinBase valid name: %v", err)
	}
	if !strings.HasPrefix(p, cm.BaseDir) && !strings.HasPrefix(p, "/") {
		t.Fatalf("joinBase returned unexpected path %q", p)
	}
	if _, err := cm.joinBase("../../escape.json"); err == nil {
		t.Fatal("joinBase traversal name must fail")
	}
	// Dots without separators are a harmless literal filename.
	if _, err := cm.joinBase("checkpoint_.._1.json"); err != nil {
		t.Fatalf("joinBase dotted filename should stay contained: %v", err)
	}
}
