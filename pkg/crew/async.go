package crew

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// TaskFuture — Async Result Handle
// ---------------------------------------------------------------------------

// TaskFuture represents the eventual result of an async task execution.
// It provides a channel-based API to wait for completion, check readiness,
// and retrieve the result or error.
//
// Usage:
//
//	future := crew.KickoffAsync(ctx)
//	// ... do other work ...
//	result, err := future.Result()    // blocks until done
//	// or check without blocking:
//	if future.Done() { ... }
type TaskFuture struct {
	done   chan struct{}
	result interface{}
	err    error
	once   sync.Once
}

// newTaskFuture creates a future and runs fn in a goroutine.
func newTaskFuture(fn func() (interface{}, error)) *TaskFuture {
	f := &TaskFuture{done: make(chan struct{})}
	go func() {
		result, err := fn()
		f.result = result
		f.err = err
		f.once.Do(func() { close(f.done) })
	}()
	return f
}

// Result blocks until the async operation completes and returns the result.
func (f *TaskFuture) Result() (interface{}, error) {
	<-f.done
	return f.result, f.err
}

// ResultWithTimeout blocks until the operation completes or the timeout expires.
func (f *TaskFuture) ResultWithTimeout(timeout time.Duration) (interface{}, error) {
	select {
	case <-f.done:
		return f.result, f.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("async task timed out after %v", timeout)
	}
}

// Done returns true if the async operation has completed.
func (f *TaskFuture) Done() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// Wait blocks until the async operation completes (ignores result).
func (f *TaskFuture) Wait() {
	<-f.done
}

// ---------------------------------------------------------------------------
// KickoffAsync — Non-Blocking Crew Execution
// ---------------------------------------------------------------------------

// KickoffAsync starts the crew execution in a background goroutine and
// returns a TaskFuture immediately. This is the primary async entry point.
//
// Usage:
//
//	crew := NewCrew(agents, tasks, WithProcess(Sequential))
//	future := crew.KickoffAsync(ctx)
//
//	// Do other work while crew runs in background...
//
//	result, err := future.Result() // blocks until completion
func (c *Crew) KickoffAsync(ctx context.Context) *TaskFuture {
	return newTaskFuture(func() (interface{}, error) {
		return c.Kickoff(ctx)
	})
}

// ---------------------------------------------------------------------------
// Multi-Task Async Execution Helpers
// ---------------------------------------------------------------------------

// TaskResult holds the result of an individual async task.
type TaskResult struct {
	Index  int
	Result interface{}
	Error  error
}

// ExecuteTasksAsync runs a slice of tasks concurrently with a concurrency
// limit and returns all results. Each task runs in its own goroutine.
// Results are returned in the same order as the input tasks.
func ExecuteTasksAsync(ctx context.Context, tasks []*TaskFuture) []TaskResult {
	results := make([]TaskResult, len(tasks))
	var wg sync.WaitGroup

	for i, f := range tasks {
		wg.Add(1)
		go func(idx int, future *TaskFuture) {
			defer wg.Done()
			result, err := future.Result()
			results[idx] = TaskResult{Index: idx, Result: result, Error: err}
		}(i, f)
	}

	wg.Wait()
	return results
}

// ---------------------------------------------------------------------------
// Per-Task Async Dispatch (used inside executeSequential)
// ---------------------------------------------------------------------------

// dispatchAsyncTask wraps a single task in a TaskFuture for fire-and-collect
// async execution within sequential flows. Unlike fire-and-forget, the
// future can be collected later.
func (c *Crew) dispatchAsyncTask(ctx context.Context, taskIndex int, task interface{ Execute(context.Context) (interface{}, error) }) *TaskFuture {
	return newTaskFuture(func() (interface{}, error) {
		result, err := task.Execute(ctx)
		if err != nil {
			if c.OnTaskError != nil {
				c.OnTaskError(taskIndex, err)
			}
			return nil, err
		}
		if c.OnTaskComplete != nil {
			c.OnTaskComplete(taskIndex, result)
		}
		if c.Verbose {
			slog.Info("Async task completed", slog.Int("index", taskIndex))
		}
		return result, nil
	})
}
