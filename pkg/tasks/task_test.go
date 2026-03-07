package tasks

import (
	"context"
	"testing"

	"github.com/Ecook14/crewai-go/pkg/agents"
)

func TestTaskExecute(t *testing.T) {
	agent := &agents.Agent{
		Role: "Writer",
	}

	task := Task{
		Description: "Write a short blog post",
		Agent:       agent,
	}

	result, err := task.Execute(context.Background())
	if err != nil {
		t.Fatalf("task.Execute failed: %v", err)
	}

	if !task.Processed {
		t.Error("task.Processed expected true, got false")
	}

	expectedResult := "Task executed successfully by Writer"
	
	// Assert string compatibility
	if resultStr, ok := result.(string); ok {
		if resultStr != expectedResult {
			t.Errorf("expected '%s', got '%s'", expectedResult, resultStr)
		}
	} else {
		t.Errorf("expected return wrapper to be parsable as a string")
	}
}

func TestTaskAsyncFlag(t *testing.T) {
	agent := &agents.Agent{Role: "AsyncWorker"}
	task := &Task{
		Description:    "Background Job",
		Agent:          agent,
		AsyncExecution: true,
	}

	if !task.AsyncExecution {
		t.Error("Expected AsyncExecution to be true")
	}

	res, err := task.Execute(context.Background())
	if err != nil {
		t.Fatalf("task execution failed: %v", err)
	}

	// Just checking the stub parses
	if _, ok := res.(string); ok {
		if !task.Processed {
			t.Error("Async task flag should map to Processed state upon channel completion")
		}
	}
}

func TestTaskExecuteNoAgent(t *testing.T) {
	task := Task{
		Description: "No agent task",
	}

	_, err := task.Execute(context.Background())
	if err == nil {
		t.Error("expected error for nil agent, got nil")
	}
}
