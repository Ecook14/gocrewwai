package training

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	s := NewStore(t.TempDir())
	in := &AgentTrainingData{
		Iterations: []IterationData{
			{InitialOutput: "a", HumanFeedback: "be clearer", ImprovedOutput: "b"},
		},
	}
	if err := s.SaveAgentData("researcher", in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.LoadAgentData("researcher")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Iterations) != 1 || got.Iterations[0].HumanFeedback != "be clearer" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	s := NewStore(t.TempDir())
	got, err := s.LoadAgentData("ghost")
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if len(got.Iterations) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

func TestRolePathTraversalRejected(t *testing.T) {
	s := NewStore(t.TempDir())
	for _, role := range []string{"../escape", "..\\escape", "a/b", "", "sub/../../x"} {
		if err := s.SaveAgentData(role, &AgentTrainingData{}); err == nil {
			t.Fatalf("save accepted role %q", role)
		}
		if _, err := s.LoadAgentData(role); err == nil {
			t.Fatalf("load accepted role %q", role)
		}
	}
	// No stray files outside the store dir.
	matches, _ := filepath.Glob(filepath.Join(s.Dir, "*.json"))
	if len(matches) != 0 {
		t.Fatalf("stray files: %v", matches)
	}
}

func TestConsolidateFeedback(t *testing.T) {
	d := &AgentTrainingData{}
	ConsolidateFeedback(d) // empty: no-op, no panic
	if len(d.Suggestions) != 0 {
		t.Fatal("empty input produced suggestions")
	}
	d.Iterations = []IterationData{
		{HumanFeedback: "x"},
		{HumanFeedback: "x"},
		{HumanFeedback: "y"},
		{HumanFeedback: ""},
	}
	ConsolidateFeedback(d)
	if len(d.Suggestions) != 2 {
		t.Fatalf("suggestions = %v, want 2 unique", d.Suggestions)
	}
	if d.Summary == "" {
		t.Fatal("empty summary")
	}
}
