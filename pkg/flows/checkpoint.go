package flows

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Checkpoint represents a snapshot of a Flow's state at a specific node.
type Checkpoint struct {
	FlowID    string          `json:"flow_id"`
	NodeID    string          `json:"node_id"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// MaxCheckpointBytes bounds a single checkpoint file (memory-DoS guard).
const MaxCheckpointBytes = 10 << 20 // 10MB

// CheckpointManager handles persistence of flow states.
type CheckpointManager struct {
	StorageDir string
}

func NewCheckpointManager(dir string) (*CheckpointManager, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &CheckpointManager{StorageDir: dir}, nil
}

// cleanCheckpointID rejects IDs that could escape the storage dir.
func cleanCheckpointID(id string) (string, error) {
	if id == "" || len(id) > 128 {
		return "", fmt.Errorf("flows: invalid checkpoint ID %q", id)
	}
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") {
		return "", fmt.Errorf("flows: invalid checkpoint ID %q", id)
	}
	if filepath.Clean(id) != id {
		return "", fmt.Errorf("flows: invalid checkpoint ID %q", id)
	}
	return id, nil
}

// atomicWriteFile writes via temp-file + rename so crashes never leave partial state.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-checkpoint-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Save persists the current flow state for a given node.
func (m *CheckpointManager) Save(flowID, nodeID string, state State) error {
	flowID, err := cleanCheckpointID(flowID)
	if err != nil {
		return err
	}
	nodeID, err = cleanCheckpointID(nodeID)
	if err != nil {
		return err
	}
	data, err := state.ToJSON()
	if err != nil {
		return err
	}
	if len(data) > MaxCheckpointBytes {
		return fmt.Errorf("flows: checkpoint state exceeds %d bytes", MaxCheckpointBytes)
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

	return atomicWriteFile(path, fileData, 0600)
}

// Load retrieves the last checkpoint for a flow.
func (m *CheckpointManager) Load(flowID string) (*Checkpoint, error) {
	flowID, err := cleanCheckpointID(flowID)
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(m.StorageDir, flowID+"_*.json"))
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("no checkpoints found for flow %s", flowID)
	}

	// For simplicity in this implementation, find the one with latest timestamp
	var latest *Checkpoint
	for _, match := range matches {
		if info, err := os.Stat(match); err != nil || info.Size() > MaxCheckpointBytes {
			continue
		}
		f, err := os.Open(match)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(f, MaxCheckpointBytes+1))
		f.Close()
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
