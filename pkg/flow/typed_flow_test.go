package flow

import (
	"context"
	"testing"
)

type counterState struct {
	Count int `json:"count"`
}

func TestTypedFlow_PersistResume(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	persist := NewJSONFilePersistence(dir)
	incr := func(ctx context.Context, s counterState) (counterState, error) {
		s.Count++
		return s, nil
	}

	first := NewTypedFlow(counterState{}).WithPersistence("resume-test", persist)
	first.AddNode(incr)
	out, err := first.Kickoff(ctx)
	if err != nil {
		t.Fatalf("first kickoff: %v", err)
	}
	if out.Count != 1 {
		t.Fatalf("first kickoff count = %d, want 1", out.Count)
	}

	// A new flow with the same ID must restore Count=1, then run the node.
	second := NewTypedFlow(counterState{}).WithPersistence("resume-test", persist)
	second.AddNode(incr)
	out, err = second.Kickoff(ctx)
	if err != nil {
		t.Fatalf("second kickoff: %v", err)
	}
	if out.Count != 2 {
		t.Fatalf("resumed kickoff count = %d, want 2 (restore + 1 node)", out.Count)
	}
}

func TestTypedFlow_CtxCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := NewTypedFlow(counterState{})
	f.AddNode(func(ctx context.Context, s counterState) (counterState, error) {
		s.Count++
		return s, nil
	})
	if _, err := f.Kickoff(ctx); err == nil {
		t.Fatal("expected context error for cancelled ctx")
	}
}
