package crew

import (
	"encoding/json"
	"os"
)
// CrewState represents the serialized state of a crew for checkpointing.
type CrewState struct {
	CurrentTaskIndex int                      `json:"current_task_index"`
	TaskOutputs      map[int]interface{}      `json:"task_outputs"`
	UsageMetrics     map[string]int           `json:"usage_metrics"`
	TaskCycles       map[int]int              `json:"task_cycles"`
}

func (c *Crew) SaveState(path string, taskIndex int) error {
	state := CrewState{
		CurrentTaskIndex: taskIndex,
		TaskOutputs:      make(map[int]interface{}),
		UsageMetrics:     c.UsageMetrics,
		TaskCycles:       make(map[int]int),
	}

	for i, t := range c.Tasks {
		if t.Processed {
			state.TaskOutputs[i] = t.Output
		}
		state.TaskCycles[i] = t.CycleCount
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Crew) LoadState(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var state CrewState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	c.UsageMetrics = state.UsageMetrics
	for i, output := range state.TaskOutputs {
		if i < len(c.Tasks) {
			c.Tasks[i].Processed = true
			c.Tasks[i].Output = output
		}
	}
	for i, cycles := range state.TaskCycles {
		if i < len(c.Tasks) {
			c.Tasks[i].CycleCount = cycles
		}
	}

	return nil
}
