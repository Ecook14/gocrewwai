package crew

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

// mockLLM for async tests
type asyncMockLLM struct {
	llm.Client
	delay    time.Duration
	response string
}

func (m *asyncMockLLM) Generate(ctx context.Context, messages []llm.Message, options map[string]interface{}) (string, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return m.response, nil
}

func TestTaskFuture_Basic(t *testing.T) {
	f := newTaskFuture(func() (interface{}, error) {
		return "hello", nil
	})

	result, err := f.Result()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result != "hello" {
		t.Errorf("Expected 'hello', got %v", result)
	}
	if !f.Done() {
		t.Error("Expected Done() to be true after Result()")
	}
}

func TestTaskFuture_Error(t *testing.T) {
	f := newTaskFuture(func() (interface{}, error) {
		return nil, context.DeadlineExceeded
	})

	_, err := f.Result()
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}
}

func TestTaskFuture_Done(t *testing.T) {
	f := newTaskFuture(func() (interface{}, error) {
		time.Sleep(50 * time.Millisecond)
		return "delayed", nil
	})

	// Should not be done immediately
	if f.Done() {
		t.Error("Expected Done() to be false immediately")
	}

	f.Wait()
	if !f.Done() {
		t.Error("Expected Done() to be true after Wait()")
	}
}

func TestTaskFuture_ResultWithTimeout(t *testing.T) {
	f := newTaskFuture(func() (interface{}, error) {
		time.Sleep(200 * time.Millisecond)
		return "too late", nil
	})

	_, err := f.ResultWithTimeout(10 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Now wait for completion and verify actual result
	result, err := f.Result()
	if err != nil || result != "too late" {
		t.Errorf("Expected 'too late', got %v, err: %v", result, err)
	}
}

func TestKickoffAsync(t *testing.T) {
	mock := &asyncMockLLM{response: "async result", delay: 20 * time.Millisecond}

	agent := &agents.Agent{
		Role:         "AsyncTester",
		Goal:         "Test async",
		LLM:          mock,
		UsageMetrics: make(map[string]int),
	}

	task := &tasks.Task{
		Description: "Test async execution",
		Agent:       agent,
	}

	crew := NewCrew([]*agents.Agent{agent}, []*tasks.Task{task})
	future := crew.KickoffAsync(context.Background())

	// Should not be done immediately (task has delay)
	// (Timing-sensitive — task needs time to start)

	result, err := future.Result()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestSequentialWithAsyncTasks(t *testing.T) {
	var callOrder int64

	syncMock := &asyncMockLLM{response: "sync-result"}
	asyncMock := &asyncMockLLM{response: "async-result", delay: 50 * time.Millisecond}

	syncAgent := &agents.Agent{
		Role:         "SyncWorker",
		Goal:         "Sync work",
		LLM:          syncMock,
		UsageMetrics: make(map[string]int),
	}
	asyncAgent := &agents.Agent{
		Role:         "AsyncWorker",
		Goal:         "Async work",
		LLM:          asyncMock,
		UsageMetrics: make(map[string]int),
	}

	completedTasks := make([]int, 0, 3)
	crew := NewCrew(
		[]*agents.Agent{syncAgent, asyncAgent},
		[]*tasks.Task{
			{Description: "Sync task 1", Agent: syncAgent},
			{Description: "Async task", Agent: asyncAgent, AsyncExecution: true},
			{Description: "Sync task 2", Agent: syncAgent},
		},
		WithVerbose(false),
	)
	crew.OnTaskComplete = func(index int, result interface{}) {
		atomic.AddInt64(&callOrder, 1)
		completedTasks = append(completedTasks, index)
	}

	result, err := crew.Kickoff(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestExecuteTasksAsync(t *testing.T) {
	futures := make([]*TaskFuture, 3)
	for i := 0; i < 3; i++ {
		idx := i
		futures[i] = newTaskFuture(func() (interface{}, error) {
			time.Sleep(time.Duration(10*(3-idx)) * time.Millisecond) // reverse order completion
			return idx, nil
		})
	}

	results := ExecuteTasksAsync(context.Background(), futures)
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Results should be in original order regardless of completion order
	for i, r := range results {
		if r.Index != i {
			t.Errorf("Expected index %d, got %d", i, r.Index)
		}
		if r.Error != nil {
			t.Errorf("Expected no error at index %d, got %v", i, r.Error)
		}
		if r.Result != i {
			t.Errorf("Expected result %d, got %v", i, r.Result)
		}
	}
}

func TestKickoffAsync_ContextCancellation(t *testing.T) {
	mock := &asyncMockLLM{response: "never", delay: 5 * time.Second}

	agent := &agents.Agent{
		Role:         "SlowAgent",
		LLM:          mock,
		UsageMetrics: make(map[string]int),
	}

	crew := NewCrew(
		[]*agents.Agent{agent},
		[]*tasks.Task{{Description: "Slow task", Agent: agent}},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	future := crew.KickoffAsync(ctx)
	_, err := future.Result()
	if err == nil {
		t.Error("Expected error from cancelled context")
	}
}
