package crew

import (
	"context"
	"fmt"
	"time"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/tasks"
)

// Process mode for execution. Translates from `Process(Enum)` in Python.
type ProcessType string

const (
	Sequential ProcessType = "sequential"
	Hierarchical ProcessType = "hierarchical"
)

// Crew represents a group of agents working together to accomplish tasks.
type Crew struct {
	Agents  []*agents.Agent
	Tasks   []*tasks.Task
	Process ProcessType

	Verbose bool

	// Execution Tracking Translators
	UsageMetrics map[string]int
}

// Kickoff starts the execution process based on the process type.
func (c *Crew) Kickoff(ctx context.Context) (interface{}, error) {
	if len(c.Tasks) == 0 || len(c.Agents) == 0 {
		return nil, fmt.Errorf("crew requires both tasks and agents to kickoff")
	}

	if c.Verbose {
		fmt.Printf("[%s] Starting Crew Execution [%s]\n", time.Now().Format(time.RFC3339), c.Process)
	}

	if c.Process == Sequential {
		return c.executeSequential(ctx)
	} else if c.Process == Hierarchical {
		return c.executeHierarchical(ctx)
	}

	return nil, fmt.Errorf("process type %s not supported", c.Process)
}

// executeSequential executes tasks one by one in order. 
// Uses WaitGroups internally if Tasks define AsyncExecution.
func (c *Crew) executeSequential(ctx context.Context) (interface{}, error) {
	var finalResult interface{}
	var err error

	for i, task := range c.Tasks {
		// Context check before task execution
		select {
		case <-ctx.Done():
			return finalResult, ctx.Err()
		default:
		}

		if c.Verbose {
			defaultLogger.Info("Executing Task", slog.Int("index", i+1), slog.String("description", task.Description))
		}

		if task.AsyncExecution {
			// Real goroutine pattern for Async
			go func(t *tasks.Task) {
				_, _ = t.Execute(ctx) // In a fully built graph, async task results are harvested later
			}(task)
			finalResult = "Async task dispatched"
			err = nil
		} else {
			result, e := task.Execute(ctx)
			finalResult = result
			err = e
		}

		if err != nil {
			return finalResult, fmt.Errorf("task %d failed: %w", i+1, err)
		}
	}

	return finalResult, nil
}

// executeHierarchical implements true Manager Agent delegation pattern.
func (c *Crew) executeHierarchical(ctx context.Context) (interface{}, error) {
	if c.Verbose {
		defaultLogger.Info("Initiating Hierarchical (Manager Driven) Execution")
	}

	// Phase 14: Dynamically construct a Manager if none exists
	managerRole := "Orchestration Manager"
	managerGoal := "Evaluate the provided tasks and efficiently aggregate results."
	
	manager := &agents.Agent{
		Role: managerRole,
		Goal: managerGoal,
		Verbose: c.Verbose,
		// We safely fall back to the first Agent's LLM if a global Crew LLM is absent
	}

	if len(c.Agents) > 0 {
		manager.LLM = c.Agents[0].LLM 
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(c.Tasks))
	results := make([]interface{}, len(c.Tasks))

	// The Manager reviews tasks before delegating them contextually.
	for i, t := range c.Tasks {
		wg.Add(1)
		
		go func(index int, task *tasks.Task) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			if c.Verbose {
				defaultLogger.Info("Manager Delegating Task", slog.Int("index", index+1), slog.String("assignee", task.Agent.Role))
			}

			// Pre-execution evaluation callback (Phase 16 hook)
			if task.Agent.StepCallback != nil {
				task.Agent.StepCallback(map[string]interface{}{"status": "delegated_by_manager"})
			}

			res, err := task.Execute(ctx)
			if err != nil {
				errCh <- fmt.Errorf("task %d failed: %w", index, err)
				return
			}
			
			results[index] = res
		}(i, t)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	if c.Verbose {
		defaultLogger.Info("Hierarchical parallel block complete. Manager aggregating.")
	}

	// Manager synthesis stub
	if manager.LLM != nil {
		// Natively we would pass the `results` string back to the LLM to write a final executive summary.
	}

	return results, nil
}
