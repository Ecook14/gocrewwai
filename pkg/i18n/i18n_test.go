package i18n_test

import (
	"testing"
	"github.com/Ecook14/gocrewwai/pkg/i18n"
)

func TestI18N_Retrieve(t *testing.T) {
	i, err := i18n.NewI18N("en")
	if err != nil {
		t.Fatalf("failed to create i18n: %v", err)
	}

	// Test slice retrieval
	obs := i.Slice("observation")
	if obs == "" || obs == "[slices:observation not found]" {
		t.Errorf("unexpected observation: %s", obs)
	}

	// Test nested retrieval (Process)
	rolePlaying := i.Slice("role_playing")
	processed := i.Process(rolePlaying, map[string]string{
		"role": "Researcher",
		"backstory": "You focus on AI research.",
		"goal": "Write a report.",
	})

	if !testing.Short() {
		t.Logf("Processed role playing: %s", processed)
	}

	if !contains(processed, "Researcher") || !contains(processed, "AI research") {
		t.Errorf("processing failed, result: %s", processed)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s[:len(substr)] == substr || contains(s[1:], substr))
}
