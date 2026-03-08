package crew

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ---------------------------------------------------------------------------
// State Management — Checkpointing, Versioning, Auto-Save
// ---------------------------------------------------------------------------

// CheckpointManager handles automatic state persistence for crew executions.
type CheckpointManager struct {
	BaseDir         string
	AutoSaveEnabled bool
	MaxCheckpoints  int // Maximum number of checkpoint files to keep
}

// Checkpoint represents a point-in-time snapshot of execution state.
type Checkpoint struct {
	ID          string                 `json:"id"`
	CrewID      string                 `json:"crew_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     int                    `json:"version"`
	TaskIndex   int                    `json:"task_index"`   // Current task being executed
	TaskResults map[string]string      `json:"task_results"` // Completed task results
	State       map[string]interface{} `json:"state"`        // Arbitrary state data
	Status      string                 `json:"status"`       // "in_progress", "completed", "failed"
	Error       string                 `json:"error,omitempty"`
}

// NewCheckpointManager creates a checkpoint manager.
func NewCheckpointManager(baseDir string) *CheckpointManager {
	if baseDir == "" {
		baseDir = ".crew_checkpoints"
	}
	os.MkdirAll(baseDir, 0755)
	return &CheckpointManager{
		BaseDir:         baseDir,
		AutoSaveEnabled: true,
		MaxCheckpoints:  10,
	}
}

// Save writes a checkpoint to disk.
func (cm *CheckpointManager) Save(cp *Checkpoint) error {
	cp.Timestamp = time.Now()
	cp.Version++

	filename := fmt.Sprintf("checkpoint_%s_%d.json", cp.CrewID, cp.Timestamp.UnixMilli())
	path := filepath.Join(cm.BaseDir, filename)

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	// Also save as "latest" for easy resume
	latestPath := filepath.Join(cm.BaseDir, fmt.Sprintf("checkpoint_%s_latest.json", cp.CrewID))
	if err := os.WriteFile(latestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write latest checkpoint: %w", err)
	}

	// Cleanup old checkpoints
	cm.cleanup(cp.CrewID)

	return nil
}

// LoadLatest reads the most recent checkpoint for a crew.
func (cm *CheckpointManager) LoadLatest(crewID string) (*Checkpoint, error) {
	path := filepath.Join(cm.BaseDir, fmt.Sprintf("checkpoint_%s_latest.json", crewID))
	return cm.loadFromFile(path)
}

// LoadByID reads a specific checkpoint.
func (cm *CheckpointManager) LoadByID(crewID string, timestamp int64) (*Checkpoint, error) {
	pattern := filepath.Join(cm.BaseDir, fmt.Sprintf("checkpoint_%s_%d.json", crewID, timestamp))
	return cm.loadFromFile(pattern)
}

func (cm *CheckpointManager) loadFromFile(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint: %w", err)
	}

	return &cp, nil
}

// ListCheckpoints returns all checkpoints for a crew, newest first.
func (cm *CheckpointManager) ListCheckpoints(crewID string) ([]*Checkpoint, error) {
	pattern := filepath.Join(cm.BaseDir, fmt.Sprintf("checkpoint_%s_*.json", crewID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var checkpoints []*Checkpoint
	for _, path := range matches {
		if filepath.Base(path) == fmt.Sprintf("checkpoint_%s_latest.json", crewID) {
			continue // Skip the "latest" symlink
		}
		cp, err := cm.loadFromFile(path)
		if err != nil {
			continue
		}
		checkpoints = append(checkpoints, cp)
	}

	return checkpoints, nil
}

func (cm *CheckpointManager) cleanup(crewID string) {
	checkpoints, err := cm.ListCheckpoints(crewID)
	if err != nil || len(checkpoints) <= cm.MaxCheckpoints {
		return
	}

	// Remove oldest checkpoints
	for i := cm.MaxCheckpoints; i < len(checkpoints); i++ {
		filename := fmt.Sprintf("checkpoint_%s_%d.json", crewID, checkpoints[i].Timestamp.UnixMilli())
		os.Remove(filepath.Join(cm.BaseDir, filename))
	}
}

// SaveOnFailure creates a checkpoint when an error occurs, enabling resume.
func (cm *CheckpointManager) SaveOnFailure(crewID string, taskIndex int, results map[string]string, state map[string]interface{}, err error) {
	cp := &Checkpoint{
		ID:          fmt.Sprintf("%s_fail_%d", crewID, time.Now().UnixMilli()),
		CrewID:      crewID,
		TaskIndex:   taskIndex,
		TaskResults: results,
		State:       state,
		Status:      "failed",
		Error:       err.Error(),
	}
	cm.Save(cp)
}
