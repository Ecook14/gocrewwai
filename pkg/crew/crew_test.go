package crew

import (
	"context"
	"testing"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/tasks"
)

func TestCrewKickoffSequential(t *testing.T) {
	agent := &agents.Agent{Role: "Researcher"}
	task1 := &tasks.Task{Description: "Find Data", Agent: agent}
	task2 := &tasks.Task{Description: "Analyze Data", Agent: agent}

	system := Crew{
		Agents:  []*agents.Agent{agent},
		Tasks:   []*tasks.Task{task1, task2},
		Process: Sequential,
	}

	result, err := system.Kickoff(context.Background())
	if err != nil {
		t.Fatalf("Kickoff failed: %v", err)
	}

	expectedResult := "Task executed successfully by Researcher"
	if resultStr, ok := result.(string); ok {
		if resultStr != expectedResult {
			t.Errorf("expected '%s', got '%s'", expectedResult, resultStr)
		}
	} else {
		t.Errorf("Expected result sequence to be castable as string format")
	}

	if !task1.Processed || !task2.Processed {
		t.Error("expected all tasks to be marked Processed")
	}
}

func TestCrewKickoffInvalid(t *testing.T) {
	system := Crew{
		Process: Sequential,
	}

	_, err := system.Kickoff(context.Background())
	if err == nil {
		t.Error("expected error for zero tasks, got nil")
	}
}

func TestCrewKickoffHierarchical(t *testing.T) {
	agent := &agents.Agent{Role: "ParallelWorker"}
	task1 := &tasks.Task{Description: "Sync 1", Agent: agent}
	task2 := &tasks.Task{Description: "Sync 2", Agent: agent}

	system := Crew{
		Agents:  []*agents.Agent{agent},
		Tasks:   []*tasks.Task{task1, task2},
		Process: Hierarchical,
	}

	result, err := system.Kickoff(context.Background())
	if err != nil {
		t.Fatalf("Hierarchical Kickoff failed: %v", err)
	}

	if resultSlice, ok := result.([]interface{}); ok {
		if len(resultSlice) != 2 {
			t.Errorf("Expected 2 results from parallel execution, got %d", len(resultSlice))
		}
	} else {
		t.Errorf("Expected result sequence to be castable as []interface{} format")
	}
}

func TestCrewKickoffCancellation(t *testing.T) {
	agent := &agents.Agent{Role: "SlowWorker"}
	task := &tasks.Task{Description: "Long running", Agent: agent}

	system := Crew{
		Agents:  []*agents.Agent{agent},
		Tasks:   []*tasks.Task{task},
		Process: Sequential,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := system.Kickoff(ctx)
	if err == nil || err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}
