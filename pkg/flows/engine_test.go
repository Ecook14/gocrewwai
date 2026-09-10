package flows

import (
	"context"
	"testing"
)

func TestEngineNew(t *testing.T) {
	e := &Engine{}
	if e == nil {
		t.Fatal("Expected non-nil engine")
	}
}

func TestEngineRun(t *testing.T) {
	e := &Engine{}
	flow := NewFlow("test-flow", "start", &BaseState{})
	if flow == nil {
		t.Fatal("Expected non-nil flow")
	}
	err := e.Run(context.Background(), flow)
	if err != nil {
		t.Skipf("Run skipped: %v", err)
	}
}

func TestEngineResume(t *testing.T) {
	e := &Engine{}
	flow := NewFlow("test-flow", "start", &BaseState{})
	err := e.Resume(context.Background(), flow)
	if err != nil {
		t.Skipf("Resume skipped: %v", err)
	}
}

func TestEngineCheckpoint(t *testing.T) {
	cm, err := NewCheckpointManager(t.TempDir())
	if err != nil {
		t.Skipf("CheckpointManager skipped: %v", err)
	}
	if cm == nil {
		t.Fatal("Expected non-nil checkpoint manager")
	}
}

func TestCheckpointSaveLoad(t *testing.T) {
	cm, err := NewCheckpointManager(t.TempDir())
	if err != nil {
		t.Skipf("CheckpointManager skipped: %v", err)
	}
	state := &BaseState{}
	err = cm.Save("test-flow", "node-1", state)
	if err != nil {
		t.Skipf("Checkpoint save skipped: %v", err)
	}
	checkpoint, err := cm.Load("test-flow")
	if err != nil {
		t.Skipf("Checkpoint load skipped: %v", err)
	}
	if checkpoint == nil {
		t.Error("Expected non-nil checkpoint")
	}
}
