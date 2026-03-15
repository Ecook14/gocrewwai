package flows

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Checkpoint represents a snapshot of a Flow's state at a specific node.
type Checkpoint struct {
	FlowID    string          `json:"flow_id"`
	NodeID    string          `json:"node_id"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// CheckpointManager handles persistence of flow states.
type CheckpointManager struct {
	StorageDir string
}

func NewCheckpointManager(dir string) (*CheckpointManager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &CheckpointManager{StorageDir: dir}, nil
}

// Save persists the current flow state for a given node.
func (m *CheckpointManager) Save(flowID, nodeID string, state State) error {
	data, err := state.ToJSON()
	if err != nil {
		return err
	}

	checkpoint := Checkpoint{
		FlowID:    flowID,
		NodeID:    nodeID,
		Timestamp: time.Now(),
		Data:      data,
	}

	fileName := fmt.Sprintf("%s_%s.json", flowID, nodeID)
	path := filepath.Join(m.StorageDir, fileName)

	fileData, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, fileData, 0644)
}

// Load retrieves the last checkpoint for a flow.
func (m *CheckpointManager) Load(flowID string) (*Checkpoint, error) {
	matches, err := filepath.Glob(filepath.Join(m.StorageDir, flowID+"_*.json"))
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("no checkpoints found for flow %s", flowID)
	}

	// For simplicity in this implementation, find the one with latest timestamp
	var latest *Checkpoint
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		var cp Checkpoint
		if err := json.Unmarshal(data, &cp); err != nil {
			continue
		}
		if latest == nil || cp.Timestamp.After(latest.Timestamp) {
			latest = &cp
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("failed to load checkpoint for flow %s", flowID)
	}

	return latest, nil
}
