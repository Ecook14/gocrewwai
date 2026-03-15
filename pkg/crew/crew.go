package crew

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/core"
	//"github.com/Ecook14/gocrewwai/pkg/dashboard"
	"github.com/Ecook14/gocrewwai/pkg/delegation"
	crewErrors "github.com/Ecook14/gocrewwai/pkg/errors"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/events"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/Ecook14/gocrewwai/pkg/training"
	"os"
	"time"
)

var defaultLogger = slog.Default()

// ProcessType defines the execution mode for a Crew.
type ProcessType string

const (
	Sequential   ProcessType = "sequential"
	Hierarchical ProcessType = "hierarchical"
	Consensual   ProcessType = "consensual"
	Graph        ProcessType = "graph"
	Reflective   ProcessType = "reflective"
	StateMachine ProcessType = "state_machine"
)

// CrewOption defines a functional option for configuring a Crew.
type CrewOption func(*Crew)

// CrewConfig defines the parameters for creating a new Crew in a declarative style.
type CrewConfig struct {
	Agents         []core.Agent
	Tasks          []*tasks.Task
	Process        ProcessType
	Verbose        bool
	ManagerLLM     llm.Client
	ManagerAgent   core.Agent
	OnTaskComplete func(taskIndex int, result interface{})
	OnTaskError    func(taskIndex int, err error)
	StateFile      string
	OutputLogFile  string
	MaxCycles      int
	MaxConcurrency int
	MaxRPM         int
	Planning       bool
	PlanningLLM    llm.Client
	KnowledgeSources []memory.KnowledgeSource
	Stream         bool
	TrainingDir    string // Directory for training iteration data
	TestLLM        llm.Client // LLM used for evaluating test runs
}

func WithProcess(p ProcessType) CrewOption {
	return func(c *Crew) { c.Process = p }
}

func WithVerbose(v bool) CrewOption {
	return func(c *Crew) { c.Verbose = v }
}

func WithMaxConcurrency(max int) CrewOption {
	return func(c *Crew) { c.MaxConcurrency = max }
}

func WithPlanning(v bool) CrewOption {
	return func(c *Crew) { c.Planning = v }
}

func WithManager(m core.Agent) CrewOption {
	return func(c *Crew) { c.ManagerAgent = m }
}

func WithStateFile(path string) CrewOption {
	return func(c *Crew) { c.StateFile = path }
}

func NewCrew(agents []core.Agent, tasks []*tasks.Task, opts ...CrewOption) *Crew {
	c := &Crew{
		Agents:       agents,
		Tasks:        tasks,
		Process:      Sequential,
		UsageMetrics: make(map[string]int),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// New creates a new Crew using a declarative configuration struct.
func New(cfg CrewConfig) *Crew {
	return &Crew{
		Agents:         cfg.Agents,
		Tasks:          cfg.Tasks,
		Process:        cfg.Process,
		Verbose:        cfg.Verbose,
		ManagerLLM:     cfg.ManagerLLM,
		ManagerAgent:   cfg.ManagerAgent,
		OnTaskComplete: cfg.OnTaskComplete,
		OnTaskError:    cfg.OnTaskError,
		StateFile:      cfg.StateFile,
		OutputLogFile:  cfg.OutputLogFile,
		MaxCycles:      cfg.MaxCycles,
		MaxConcurrency: cfg.MaxConcurrency,
		MaxRPM:         cfg.MaxRPM,
		Planning:       cfg.Planning,
		PlanningLLM:    cfg.PlanningLLM,
		KnowledgeSources: cfg.KnowledgeSources,
		Stream:         cfg.Stream,
		TrainingDir:    cfg.TrainingDir,
		TestLLM:        cfg.TestLLM,
		UsageMetrics:   make(map[string]int),
	}
}

// Crew ...
type Crew struct {
	Agents  []core.Agent
	Tasks   []*tasks.Task
	Process ProcessType

	Verbose bool

	// ManagerLLM allows binding a specific LLM for the manager agent in hierarchical/consensual mode.
	ManagerLLM llm.Client

	// ManagerAgent allows providing a custom manager agent for orchestration.
	ManagerAgent core.Agent

	// Shared Configuration
	MaxRPM         int
	MaxConcurrency int
	MaxCycles      int

	// Callbacks
	OnTaskComplete func(taskIndex int, result interface{}) `json:"-"`
	OnTaskError    func(taskIndex int, err error)          `json:"-"`
	StepCallback   func(step map[string]interface{})       `json:"-"` // Called per agent step for collaboration tracking
	TaskCallback   func(taskIndex int, output interface{})  `json:"-"` // Called when any task produces output

	// Persistence & Logging
	StateFile     string
	OutputLogFile string

	// Elite Features
	Planning      bool
	PlanningLLM   llm.Client
	KnowledgeSources []memory.KnowledgeSource
	Stream        bool
	TrainingDir   string
	TestLLM       llm.Client

	// Execution Tracking
	UsageMetrics map[string]int
	TrainingMode bool // Internal flag when executing c.Train()

	staticSyncDone bool
}

// Train runs the crew for n iterations with human feedback enabled to improve agent performance.
func (c *Crew) Train(ctx context.Context, iterations int, inputs map[string]interface{}) error {
	if iterations <= 0 {
		return fmt.Errorf("iterations must be positive")
	}

	events.GlobalBus.Publish(events.Event{
		Type:   events.CrewTrainStarted,
		Source: "Crew",
		Payload: map[string]interface{}{
			"iterations": iterations,
		},
	})

	c.TrainingMode = true
	defer func() { c.TrainingMode = false }()

	for i := 0; i < iterations; i++ {
		slog.Info(fmt.Sprintf("🏋️ Training Iteration %d/%d", i+1, iterations))
		
		// Run a normal kickoff but with training flags enabled internally
		_, err := c.Kickoff(ctx)
		if err != nil {
			events.GlobalBus.Publish(events.Event{
				Type:   events.CrewTrainFailed,
				Source: "Crew",
				Error:  err,
			})
			return err
		}
	}

	// Consolidate training data
	if c.TrainingDir != "" {
		store := training.NewStore(c.TrainingDir)
		for _, agent := range c.Agents {
			data, err := store.LoadAgentData(agent.GetRole())
			if err == nil {
				training.ConsolidateFeedback(data)
				store.SaveAgentData(agent.GetRole(), data)
			}
		}
	}

	events.GlobalBus.Publish(events.Event{
		Type:   events.CrewTrainCompleted,
		Source: "Crew",
	})

	return nil
}

// Kickoff starts the execution process based on the process type.
func (c *Crew) Kickoff(ctx context.Context) (interface{}, error) { 
	ctx, span := telemetry.Tracer.Start(ctx, "Crew.Kickoff")
	defer span.End()

	// Publish system event
	events.GlobalBus.Publish(events.Event{
		Type:   events.CrewKickoffStarted,
		Source: "Crew",
		Payload: map[string]interface{}{
			"process_type": string(c.Process),
			"num_tasks":    len(c.Tasks),
			"num_agents":   len(c.Agents),
		},
	})

	slog.Info("🚀 Crew Kickoff Initiated",
		slog.String("process_type", string(c.Process)),
		slog.Int("num_tasks", len(c.Tasks)),
		slog.Int("num_agents", len(c.Agents)))

	// Sync training settings to agents
	for _, a := range c.Agents {
		if local, ok := a.(*agents.Agent); ok {
			if c.TrainingMode {
				local.TrainingMode = true
			}
			if c.TrainingDir != "" {
				local.TrainingDir = c.TrainingDir
			}
		}
	}

	// Register existing entities with DynamicRegistry for Dashboard visibility (ONLY ONCE)
	if !c.staticSyncDone {
		for _, a := range c.Agents {
			telemetry.GlobalDynamicRegistry.AddAgent(a, false)
		}
		for _, t := range c.Tasks {
			telemetry.GlobalDynamicRegistry.AddTask(t, false)
		}
		c.staticSyncDone = true
	}

	if len(c.Tasks) == 0 {
		return "", crewErrors.ErrNoTasks // Changed return value
	}
	if len(c.Agents) == 0 {
		return "", crewErrors.ErrNoAgents
	}

	// Load state if a StateFile is provided and exists
	if c.StateFile != "" {
		if _, err := os.Stat(c.StateFile); err == nil {
			if c.Verbose {
				defaultLogger.Info("📍 Resuming Crew from Checkpoint", slog.String("file", c.StateFile))
			}
			if err := c.LoadState(c.StateFile); err != nil {
				defaultLogger.Warn("⚠️ Failed to load state file", slog.String("error", err.Error()))
			}
		}
	}

	if c.Verbose {
		slog.Info("Starting Crew Execution", slog.String("process", string(c.Process)))
	}

	// Dynamic UI Control Check
	if err := telemetry.GlobalExecutionController.WaitIfPaused(ctx); err != nil {
		return nil, err
	}

	// 1. Core 3 Perfection: Enforce Global MaxRPM
	if c.MaxRPM > 0 {
		for _, agent := range c.Agents {
			if agent.GetMaxRPM() == 0 || agent.GetMaxRPM() > c.MaxRPM {
				agent.SetMaxRPM(c.MaxRPM)
			}
		}
	}

	// Phase 2: Advanced Planning
	if c.Planning {
		if err := c.runPlanningPhase(ctx); err != nil {
			return nil, fmt.Errorf("planning phase failed: %w", err)
		}
	}

	// Initialize Delegation Tools for agents that allow it
	c.InjectDelegationTools()

	var result interface{}
	var err error

	switch c.Process {
	case Sequential:
		result, err = c.executeSequential(ctx)
		result = fmt.Sprintf("%v", result)
	case Hierarchical:
		result, err = c.executeHierarchical(ctx)
		result = fmt.Sprintf("%v", result)
	case Consensual:
		result, err = c.executeConsensual(ctx)
	case Graph:
		result, err = c.executeGraph(ctx)
	case Reflective:
		result, err = c.executeReflective(ctx)
	case StateMachine:
		result, err = c.executeStateMachine(ctx)
	default:
		err = fmt.Errorf("%w: %s", crewErrors.ErrUnsupportedProcess, c.Process)
	}

	if err != nil {
		events.GlobalBus.Publish(events.Event{
			Type:   events.CrewKickoffFailed,
			Source: "Crew",
			Error:  err,
		})
		return nil, err
	}

	events.GlobalBus.Publish(events.Event{
		Type:   events.CrewKickoffCompleted,
		Source: "Crew",
		Payload: map[string]interface{}{
			"result": result,
		},
	})
	return result, nil
}

// Replay restarts the crew execution from a specific task ID or name.
// It resets the 'Processed' status of the target task and all downstream tasks.
func (c *Crew) Replay(ctx context.Context, taskID string) (interface{}, error) {
	slog.Info("🔄 Replaying Crew Execution", slog.String("start_task", taskID))
	
	found := false
	for _, t := range c.Tasks {
		if t.Name == taskID || t.Description == taskID {
			found = true
		}
		if found {
			t.Processed = false
			t.Failed = false
			t.Error = nil
		}
	}
	
	if !found {
		return nil, fmt.Errorf("task not found for replay: %s", taskID)
	}
	
	return c.Kickoff(ctx)
}




// executeSequential executes tasks one by one in order, piping context between them.
// Tasks marked with AsyncExecution=true are dispatched in the background via TaskFuture
// and their results are collected after all sequential tasks complete.
func (c *Crew) executeSequential(ctx context.Context) (interface{}, error) {
	var finalResult interface{}
	var asyncFutures []*asyncEntry

	for i := 0; i < len(c.Tasks); i++ {
		task := c.Tasks[i]
		// Skip if already processed
		if task.Processed {
			continue
		}

		// Cooldown delay between tasks to prevent bursty rate limits (Elite Tier Reliability)
		if i > 0 {
			time.Sleep(2 * time.Second)
		}

		// Context check before task execution
		select {
		case <-ctx.Done():
			return finalResult, ctx.Err()
		default:
		}

		if c.Verbose {
			defaultLogger.Info("Executing Task", slog.Int("index", i+1), slog.String("description", task.Description))
		}

		// Dynamic UI Control Check before task
		if err := telemetry.GlobalExecutionController.WaitIfPaused(ctx); err != nil {
			return finalResult, err
		}

		// Pipe previous task output into current task's context
		if i > 0 && c.Tasks[i-1].Processed && c.Tasks[i-1].Output != nil {
			if task.Context == nil {
				task.Context = make([]*tasks.Task, 0)
			}
			// Add the previous task as context if not already included
			alreadyIncluded := false
			for _, ctxTask := range task.Context {
				if ctxTask == c.Tasks[i-1] {
					alreadyIncluded = true
					break
				}
			}
			if !alreadyIncluded {
				task.Context = append(task.Context, c.Tasks[i-1])
			}
		}

		if task.AsyncExecution {
			// Dispatch async task — result collected later via TaskFuture
			if c.Verbose {
				defaultLogger.Info("⚡ Dispatching async task", slog.Int("index", i+1))
			}
			future := c.dispatchAsyncTask(ctx, i+1, task)
			asyncFutures = append(asyncFutures, &asyncEntry{
				index:  i,
				task:   task,
				future: future,
			})
		} else {
			result, err := task.Execute(ctx)
			if err != nil {
				task.Failed = true
				task.Error = err
				task.Processed = true // Prevent infinite re-run
				taskErr := crewErrors.NewTaskError(i+1, task.Description, err)
				if c.OnTaskError != nil {
					c.OnTaskError(i+1, taskErr)
				}
				slog.Error("Task Failed", slog.Int("index", i+1), slog.String("error", err.Error()))
				return finalResult, taskErr
			}
			task.Processed = true
			task.Output = result
			finalResult = result
			if c.OnTaskComplete != nil {
				c.OnTaskComplete(i+1, result)
			}
		}
	}

	// Collect all async task results
	if len(asyncFutures) > 0 {
		if c.Verbose {
			defaultLogger.Info("⏳ Waiting for async tasks to complete", slog.Int("count", len(asyncFutures)))
		}
		for _, entry := range asyncFutures {
			result, err := entry.future.Result()
			if err != nil {
				taskErr := crewErrors.NewTaskError(entry.index+1, entry.task.Description, err)
				if c.OnTaskError != nil {
					c.OnTaskError(entry.index+1, taskErr)
				}
				// Continue collecting remaining — don't fail the whole crew on one async error
				defaultLogger.Warn("Async task failed", slog.Int("index", entry.index+1), slog.Any("error", err))
				continue
			}
			// Update the task's output so downstream can reference it
			entry.task.Processed = true
			entry.task.Output = result
			finalResult = result
			if c.Verbose {
				defaultLogger.Info("✅ Async task collected", slog.Int("index", entry.index+1))
			}
		}
	}

	return finalResult, nil
}

// asyncEntry pairs a task with its future for later collection.
type asyncEntry struct {
	index  int
	task   *tasks.Task
	future *TaskFuture
}

// executeHierarchical implements the Manager Agent delegation pattern.
// The manager coordinates parallel task execution and aggregates results.
func (c *Crew) executeHierarchical(ctx context.Context) (interface{}, error) {
	if c.Verbose {
		defaultLogger.Info("Initiating Hierarchical (Manager Driven) Execution")
	}

	// Construct or use the provided manager agent
	var orchestrator *agents.ManagerAgent
	if c.ManagerAgent != nil {
		if m, ok := c.ManagerAgent.(*agents.ManagerAgent); ok {
			orchestrator = m
			orchestrator.ManagedAgents = c.Agents
		} else if local, ok := c.ManagerAgent.(*agents.Agent); ok {
			orchestrator = &agents.ManagerAgent{Agent: *local, ManagedAgents: c.Agents}
		}
	}
	
	if orchestrator == nil {
		model := c.ManagerLLM
		if model == nil && len(c.Agents) > 0 {
			if local, ok := c.Agents[0].(*agents.Agent); ok {
				model = local.LLM
			}
		}
		orchestrator = agents.NewManagerAgent(model, c.Agents)
		orchestrator.Verbose = c.Verbose
	}

	// Results slice to grow as tasks grow
	finalResults := make([]interface{}, 0, len(c.Tasks))

	concurrency := c.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 5 // Safe default to prevent rate limits
	}
	sem := make(chan struct{}, concurrency)

	maxReplans := 5 // Safety limit to avoid infinite loops
	replanCount := 0

	for {
		var wg sync.WaitGroup
		var pendingTasks []*tasks.Task
		var pendingIndices []int

		for i, t := range c.Tasks {
			if !t.Processed {
				pendingTasks = append(pendingTasks, t)
				pendingIndices = append(pendingIndices, i)
			}
		}

		// Grow results array if c.Tasks grew during replanning
		for len(finalResults) < len(c.Tasks) {
			finalResults = append(finalResults, nil)
		}

		if len(pendingTasks) == 0 {
			break // All tasks processed
		}

		errCh := make(chan error, len(pendingTasks))

		for i, pTask := range pendingTasks {
			wg.Add(1)
			go func(index int, task *tasks.Task) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				default:
				}

				assignedAgent, err := orchestrator.DelegateTask(ctx, task.Description)
				if err != nil {
					if task.Agent == nil {
						errCh <- fmt.Errorf("task delegation failed and no default agent assigned: %w", err)
						return
					}
					assignedAgent = task.Agent
				}
				task.Agent = assignedAgent

				if c.Verbose {
					defaultLogger.Info("Manager Delegating Task",
						slog.Int("index", index+1),
						slog.String("assignee", strings.Clone(task.Agent.GetRole())))
				}

				if local, ok := task.Agent.(*agents.Agent); ok && local.StepCallback != nil {
					local.StepCallback(map[string]interface{}{"status": "delegated_by_manager"})
				}

				res, err := task.Execute(ctx)
				if err != nil {
					task.Failed = true
					task.Error = err
					task.Processed = true
					taskErr := crewErrors.NewTaskError(index+1, task.Description, err)
					errCh <- taskErr
					if c.OnTaskError != nil {
						c.OnTaskError(index+1, taskErr)
					}
					slog.Error("Task Failed in Hierarchical", slog.Int("index", index+1), slog.String("error", err.Error()))
					return
				}

				task.Processed = true
				task.Output = res // Store output on the task object natively

				// Populate finalResults safely using a local scoped lock if needed,
				// but since indices are unique per goroutine in this round, direct assignment is safe.
				finalResults[index] = res

				if c.OnTaskComplete != nil {
					c.OnTaskComplete(index+1, res)
				}
			}(pendingIndices[i], pTask)
		}

		wg.Wait()
		close(errCh)

		for err := range errCh {
			if err != nil {
				return nil, err
			}
		}

		// ---------------------------------------------------------
		// DYNAMIC RE-PLANNING STAGE
		// ---------------------------------------------------------
		if c.Verbose {
			defaultLogger.Info("🔍 Manager evaluating plan for potential re-routing")
		}
		
		planContext := "CURRENT STATUS:\n"
		for i, t := range c.Tasks {
			status := "Pending"
			if t.Processed {
				status = "Completed"
			}
			planContext += fmt.Sprintf("Task %d: %s [%s]\n", i+1, t.Description, status)
		}

		replanPrompt := planContext + "\n\nAs the Manager, review the completed tasks. Should we add any new follow-up tasks or modify the existing plan based on current results? " +
			"If yes, describe the new tasks cleanly. If no and the goals are met, respond with exactly 'PLAN_STABLE'."
		
		decision, err := orchestrator.Execute(ctx, replanPrompt, nil)
		if err == nil {
			decisionStr := fmt.Sprintf("%v", decision)
			if !strings.Contains(strings.ToUpper(decisionStr), "PLAN_STABLE") && replanCount < maxReplans {
				if c.Verbose {
					defaultLogger.Info("🔄 Manager INITIATED RE-PLANNING", slog.String("decision", decisionStr))
				}
				
				// Elite Pattern: Dynamic Re-Planning native injection.
				newTask := &tasks.Task{
					Description: "Follow-up execution based on manager refinement: " + decisionStr,
					Agent:       &orchestrator.Agent,
				}
				
				c.Tasks = append(c.Tasks, newTask)
				replanCount++
				continue // Trigger outer loop to process the newly appended task natively
			}
		}
		
		break // The plan is stable or we hit the replan limit
	}

	// 4. Final Aggregation and Metric Sync
	if c.UsageMetrics == nil {
		c.UsageMetrics = make(map[string]int)
	}
	for _, a := range c.Agents {
		for k, v := range a.GetUsageMetrics() {
			c.UsageMetrics[k] += v
		}
	}

	if c.Verbose {
		defaultLogger.Info("Hierarchical parallel block complete. Manager aggregating.",
			slog.Int("prompt_tokens", c.UsageMetrics["prompt_tokens"]),
			slog.Int("completion_tokens", c.UsageMetrics["completion_tokens"]))
	}

	// Manager synthesis
	if orchestrator.LLM != nil {
		var sb fmt.Stringer = &resultAggregator{results: finalResults, tasks: c.Tasks}
		synthesisInput := fmt.Sprintf(
			"You are aggregating results from %d parallel worker tasks. "+
				"Please provide a coherent, well-structured final summary.\n\n%s",
			len(finalResults), sb)

		synthesized, err := orchestrator.Execute(ctx, synthesisInput, nil)
		if err != nil {
			if c.Verbose {
				defaultLogger.Warn("Manager synthesis failed, returning raw results", slog.String("error", err.Error()))
			}
			return sb.String(), nil
		}

		// Sync manager metrics too
		for k, v := range orchestrator.GetUsageMetrics() {
			c.UsageMetrics[k] += v
		}

		return synthesized, nil
	}

	return finalResults, nil
}

// resultAggregator formats task results for the manager's synthesis prompt.
type resultAggregator struct {
	results []interface{}
	tasks   []*tasks.Task
}

func (ra *resultAggregator) String() string {
	var sb string
	for i, res := range ra.results {
		desc := "Unknown Task"
		if i < len(ra.tasks) {
			desc = ra.tasks[i].Description
		}
		sb += fmt.Sprintf("--- Task %d: %s ---\nResult: %v\n\n", i+1, desc, res)
	}
	return sb
}

// executeConsensual runs the same task across all agents in parallel and uses a manager
// to synthesize a singular "consensus" result from all outputs.
func (c *Crew) executeConsensual(ctx context.Context) (string, error) {
	if len(c.Tasks) == 0 {
		return "", fmt.Errorf("consensus requires at least one task")
	}

	// For consensus, we typically run the *first* task across *all* agents
	mainTask := c.Tasks[0]

	if c.Verbose {
		defaultLogger.Info("Initiating Consensual Execution (Multi-Agent Agreement)",
			slog.String("task", mainTask.Description),
			slog.Int("agents", len(c.Agents)))
	}

	var wg sync.WaitGroup
	results := make([]string, len(c.Agents))
	errCh := make(chan error, len(c.Agents))

	concurrency := c.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 5 
	}
	sem := make(chan struct{}, concurrency)

	for i, agent := range c.Agents {
		wg.Add(1)
		go func(idx int, a core.Agent) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			// Execute task with this specific agent
			res, err := a.Execute(ctx, mainTask.Description, nil)
			if err != nil {
				errCh <- fmt.Errorf("agent %s failed: %w", a.GetRole(), err)
				return
			}
			results[idx] = fmt.Sprintf("Agent: %s\nOutput: %v", a.GetRole(), res)
		}(i, agent)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return "", err
		}
	}

	// Consolidate into a manager synthesis prompt
	var orchestrator *agents.ManagerAgent
	if c.ManagerAgent != nil {
		if m, ok := c.ManagerAgent.(*agents.ManagerAgent); ok {
			orchestrator = m
			orchestrator.ManagedAgents = c.Agents
		} else if local, ok := c.ManagerAgent.(*agents.Agent); ok {
			orchestrator = &agents.ManagerAgent{Agent: *local, ManagedAgents: c.Agents}
		}
	}
	
	if orchestrator == nil {
		model := c.ManagerLLM
		if model == nil && len(c.Agents) > 0 {
			if local, ok := c.Agents[0].(*agents.Agent); ok {
				model = local.LLM
			}
		}
		orchestrator = agents.NewManagerAgent(model, c.Agents)
	}

	synthesisPrompt := "You are a Consensus Manager. Below are results from multiple agents on the same task. " +
		"Analyze all responses and provide the single most accurate, consensus-driven final answer.\n\n"
	for _, res := range results {
		synthesisPrompt += res + "\n\n"
	}

	finalAnswer, err := orchestrator.Execute(ctx, synthesisPrompt, nil)
	
	// Update Metrics (Aggressively sync even if synthesis failed partially)
	if c.UsageMetrics == nil {
		c.UsageMetrics = make(map[string]int)
	}
	for _, a := range c.Agents {
		for k, v := range a.GetUsageMetrics() {
			c.UsageMetrics[k] += v
		}
	}
	if orchestrator != nil {
		for k, v := range orchestrator.GetUsageMetrics() {
			c.UsageMetrics[k] += v
		}
	}

	if err != nil {
		return "", fmt.Errorf("consensus synthesis failed: %w", err)
	}

	return fmt.Sprintf("%v", finalAnswer), nil
}

// executeGraph refactored to support cycles via task reset.
func (c *Crew) executeGraph(ctx context.Context) (string, error) {
	if len(c.Tasks) == 0 {
		return "", nil
	}

	if c.Verbose {
		defaultLogger.Info("Initiating Elite Graph Execution (Supports Cycles)")
	}

	// Track processing state
	processed := make(map[*tasks.Task]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCh := make(chan error, len(c.Tasks)*2)

	concurrency := c.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 5 
	}
	sem := make(chan struct{}, concurrency)

	maxGlobalCycles := c.MaxCycles
	if maxGlobalCycles <= 0 {
		maxGlobalCycles = 100
	}

	for globalIter := 0; globalIter < maxGlobalCycles; globalIter++ {
		mu.Lock()
		var readyTasks []*tasks.Task
		allDone := true
		for _, t := range c.Tasks {
			if !processed[t] {
				allDone = false
				depsMet := true
				for _, dep := range t.Dependencies {
					if !processed[dep] {
						depsMet = false
						break
					}
				}
				if depsMet {
					readyTasks = append(readyTasks, t)
				}
			}
		}
		mu.Unlock()

		if allDone {
			break
		}

		if len(readyTasks) == 0 {
			return "", fmt.Errorf("deadlock or unresolved cyclic dependency in graph")
		}

		// Parallel launch
		for _, t := range readyTasks {
			mu.Lock()
			processed[t] = true
			mu.Unlock()

			wg.Add(1)
			go func(task *tasks.Task) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				res, err := task.Execute(ctx)
				if err != nil {
					errCh <- err
					return
				}

				// ELITE: Check for feedback loop
				if task.OutputCondition != nil && task.NextPaths != nil {
					path := task.OutputCondition(res)
					if next, ok := task.NextPaths[path]; ok {
						if next == task || contains(task.Dependencies, next) {
							// Handle Cycle: Mark tasks as NOT processed to trigger re-execution
							mu.Lock()
							processed[next] = false
							if c.Verbose {
								defaultLogger.Info("🔄 Graph Cycle Triggered", slog.String("target", next.Description))
							}
							mu.Unlock()
						}
					}
				}
			}(t)
		}
		wg.Wait()

		select {
		case err := <-errCh:
			return "", err
		default:
		}
	}

	lastTask := c.Tasks[len(c.Tasks)-1]
	return fmt.Sprintf("%v", lastTask.Output), nil
}

// executeReflective runs tasks sequentially but with a mandatory "Manager Review" 
// stage for each task output. If the manager rejects, the agent must retry.
func (c *Crew) executeReflective(ctx context.Context) (string, error) {
	var finalResult string
	
	var orchestrator *agents.ManagerAgent
	if c.ManagerAgent != nil {
		if m, ok := c.ManagerAgent.(*agents.ManagerAgent); ok {
			orchestrator = m
			orchestrator.ManagedAgents = c.Agents
		} else if local, ok := c.ManagerAgent.(*agents.Agent); ok {
			orchestrator = &agents.ManagerAgent{Agent: *local, ManagedAgents: c.Agents}
		}
	}
	
	if orchestrator == nil {
		model := c.ManagerLLM
		if model == nil && len(c.Agents) > 0 {
			if local, ok := c.Agents[0].(*agents.Agent); ok {
				model = local.LLM
			}
		}
		orchestrator = agents.NewManagerAgent(model, c.Agents)
	}

	for i, task := range c.Tasks {
		if c.Verbose {
			defaultLogger.Info("Executing Reflective Task", slog.Int("index", i+1))
		}

		result, err := task.Execute(ctx)
		if err != nil {
			return "", err
		}

		// Manager Review Stage
		reviewPrompt := fmt.Sprintf("Please review the following task output for accuracy and quality.\nTask: %s\nOutput: %v\n\nRespond with 'APPROVED' if it is satisfactory, or provide constructive feedback for improvement.", task.Description, result)
		
		maxReviewRetries := 2
		for j := 0; j < maxReviewRetries; j++ {
			review, err := orchestrator.Execute(ctx, reviewPrompt, nil) // Corrected to use orchestrator and reviewPrompt, and capture err
			if err != nil {
				return "", fmt.Errorf("manager review failed: %w", err)
			}

			reviewStr := fmt.Sprintf("%v", review)
			if strings.Contains(strings.ToUpper(reviewStr), "APPROVED") {
				if c.Verbose {
					defaultLogger.Info("✅ Manager APPROVED task output", slog.Int("task", i+1))
				}
				break
			}

			if j == maxReviewRetries-1 {
				if c.Verbose {
					defaultLogger.Warn("⚠️ Manager gave feedback but max review retries reached", slog.Int("task", i+1))
				}
				break
			}

			if c.Verbose {
				defaultLogger.Info("🔄 Manager REQUESTED REVISION", slog.Int("task", i+1), slog.String("feedback", reviewStr))
			}

			// Feed back into the task and execute again
			task.Description += "\n\nMANAGER FEEDBACK: " + reviewStr
			result, err = task.Execute(ctx)
			if err != nil {
				return "", err
			}
		}
		
		finalResult = fmt.Sprintf("%v", result)
	}

	return finalResult, nil
}

// executeStateMachine handles explicit state transitions and cycles.
func (c *Crew) executeStateMachine(ctx context.Context) (string, error) {
	if len(c.Tasks) == 0 {
		return "", nil
	}

	if c.Verbose {
		defaultLogger.Info("Initiating State Machine Execution")
	}

	currentTask := c.Tasks[0]
	maxGlobalCycles := c.MaxCycles
	if maxGlobalCycles <= 0 {
		maxGlobalCycles = 50
	}

	for i := 0; i < maxGlobalCycles; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		if c.Verbose {
			defaultLogger.Info("StateMachine executing task", slog.String("description", currentTask.Description))
		}

		result, err := currentTask.Execute(ctx)
		if err != nil {
			return "", err
		}

		if c.OnTaskComplete != nil {
			c.OnTaskComplete(-1, result)
		}

		// Determine next state
		if currentTask.OutputCondition != nil && currentTask.NextPaths != nil {
			path := currentTask.OutputCondition(result)
			next, ok := currentTask.NextPaths[path]
			if ok {
				if next == currentTask {
					currentTask.CycleCount++
					if currentTask.MaxCycles > 0 && currentTask.CycleCount > currentTask.MaxCycles {
						return "", fmt.Errorf("task cycle limit exceeded for: %s", currentTask.Description)
					}
				}
				currentTask = next
				continue
			}
		}

		// If no transition, check if there's a next task in the slice or finish
		found := false
		for idx, t := range c.Tasks {
			if t == currentTask {
				if idx+1 < len(c.Tasks) {
					currentTask = c.Tasks[idx+1]
					found = true
					break
				}
			}
		}

		if !found {
			return fmt.Sprintf("%v", currentTask.Output), nil
		}
	}

	return "", fmt.Errorf("global state machine cycle limit reached")
}

func contains(tasks []*tasks.Task, t *tasks.Task) bool {
	for _, item := range tasks {
		if item == t {
			return true
		}
	}
	return false
}

func (c *Crew) runPlanningPhase(ctx context.Context) error {
	if c.Verbose {
		defaultLogger.Info("🗺️  Initiating Strategic Planning Phase")
	}

	// 1. Setup Manager
	var orchestrator *agents.ManagerAgent
	if c.ManagerAgent != nil {
		if m, ok := c.ManagerAgent.(*agents.ManagerAgent); ok {
			orchestrator = m
			orchestrator.ManagedAgents = c.Agents
		} else if local, ok := c.ManagerAgent.(*agents.Agent); ok {
			orchestrator = &agents.ManagerAgent{Agent: *local, ManagedAgents: c.Agents}
		}
	}
	
	if orchestrator == nil {
		model := c.PlanningLLM
		if model == nil {
			model = c.ManagerLLM
		}
		if model == nil && len(c.Agents) > 0 {
			if local, ok := c.Agents[0].(*agents.Agent); ok {
				model = local.LLM
			}
		}
		orchestrator = agents.NewManagerAgent(model, c.Agents)
		orchestrator.Verbose = c.Verbose
	}

	// 2. Prepare Task List for Planner
	var tasksList strings.Builder
	for i, t := range c.Tasks {
		tasksList.WriteString(fmt.Sprintf("Task %d: %s\n", i+1, t.Description))
	}

	// 3. Generate Plan
	plan, err := orchestrator.GeneratePlan(ctx, tasksList.String())
	if err != nil {
		return err
	}

	if c.Verbose {
		defaultLogger.Info("✅ Strategic Plan Generated", slog.String("plan", plan))
	}

	// 4. Inject Plan into all Task contexts
	for _, t := range c.Tasks {
		t.Description = fmt.Sprintf("[STRATEGIC PLAN]\n%s\n\n[TASK DESCRIPTION]\n%s", plan, t.Description)
	}

	return nil
}

// RunCreatorMode enters a continuous polling loop, executing any new tasks staged via the UI.
// This is an Elite Tier developer feature for building long-running AI orchestration services.
func (c *Crew) RunCreatorMode(ctx context.Context) error {
	slog.Info("✅ Engine is now in 'Creator Mode'. Active polling for UI-staged entities...")

	// Setup synchronization callbacks for Dashboard result persistence
	c.OnTaskComplete = func(index int, result interface{}) {
		// Sync the current task's result back to the Dashboard's master list
		if index > 0 && index <= len(c.Tasks) {
			task := c.Tasks[index-1]
			telemetry.GlobalDynamicRegistry.SyncTaskResult(task.Description, task.Agent.GetRole(), result, nil, true)
		}
	}
	c.OnTaskError = func(index int, err error) {
		if index > 0 && index <= len(c.Tasks) {
			task := c.Tasks[index-1]
			telemetry.GlobalDynamicRegistry.SyncTaskResult(task.Description, task.Agent.GetRole(), nil, err, true)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Check for new tasks every 2 seconds
			newTasks := telemetry.GlobalDynamicRegistry.PullTasks()
			newAgents := telemetry.GlobalDynamicRegistry.PullAgents()

			if len(newTasks) > 0 || len(newAgents) > 0 {
				slog.Info("📥 Dynamic changes detected! Injecting into live engine loop...")
				
				// Inject Agents first so tasks can bind to them
				for _, a := range newAgents {
					if agent, ok := a.(core.Agent); ok {
						updated := false
						// Overwrite existing agent if role matches (to capture UI updates)
						for i, existing := range c.Agents {
							slog.Debug("Comparing agent roles", slog.String("existing", existing.GetRole()), slog.String("new", agent.GetRole()))
							if strings.TrimSpace(strings.ToLower(existing.GetRole())) == strings.TrimSpace(strings.ToLower(agent.GetRole())) {
								c.Agents[i] = agent
								slog.Info("📥 Dynamic Agent updated within engine", slog.String("role", agent.GetRole()))
								updated = true
								break
							}
						}
						// Otherwise append as a brand new agent
						if !updated {
							c.Agents = append(c.Agents, agent)
							slog.Info("📥 Dynamic Agent injected", slog.String("role", agent.GetRole()))
						}
					}
				}

				// Inject Tasks
				for _, t := range newTasks {
					var task *tasks.Task
					if tm, ok := t.(map[string]interface{}); ok {
						desc, _ := tm["description"].(string)
						role, _ := tm["agent_role"].(string)
						task = &tasks.Task{
							Description: desc,
							AgentRole:   role,
						}
					} else if tp, ok := t.(*tasks.Task); ok {
						task = tp
					}

					if task != nil {
						// Late bind agent if needed
						if task.Agent == nil {
							// Try to match by role (case-insensitive)
							if task.AgentRole != "" {
								for _, a := range c.Agents {
									if strings.EqualFold(strings.TrimSpace(a.GetRole()), strings.TrimSpace(task.AgentRole)) {
										task.Agent = a
										break
									}
								}
							}
							
							// Fallback: If still nil, assign to the first available agent
							if task.Agent == nil && len(c.Agents) > 0 {
								slog.Warn("⚠️ No agent match found for task. Falling back to first available agent.", 
									slog.String("task", task.Description), 
									slog.String("requested_role", task.AgentRole),
									slog.String("assigned_role", c.Agents[0].GetRole()))
								task.Agent = c.Agents[0]
							}
						}

						if task.Agent == nil {
							slog.Error("❌ Task Injection Failed: No agents available to assign", slog.String("task", task.Description))
							continue
						}

						// Deduplication
						isDuplicate := false
						for _, existing := range c.Tasks {
							if existing == task || (existing.Description == task.Description && existing.Agent == task.Agent) {
								isDuplicate = true
								break
							}
						}

						if isDuplicate {
							continue
						}

						slog.Info("📥 Dynamic Task injected", 
							slog.String("description", task.Description),
							slog.String("assigned_agent", task.Agent.GetRole()))
						c.Tasks = append(c.Tasks, task)
					}
				}
				
				// Full Sync: Add existing tasks to registry if missing (e.g. Task 1 from kickoff)
				for _, t := range c.Tasks {
					telemetry.GlobalDynamicRegistry.AddTask(t, false)
				}

				// Respect UI Process Type updates
				uiProcess := telemetry.GlobalDynamicRegistry.GetProcessType()
				if uiProcess != "" {
					c.Process = ProcessType(uiProcess)
				}

				// Proactively refresh delegation tools for the new team composition
				c.InjectDelegationTools()

				// Re-run the crew logic
				if _, err := c.Kickoff(ctx); err != nil {
					slog.Error("Dynamic Execution Error", slog.Any("error", err))
				}
			}
			time.Sleep(2 * time.Second)
		}
	}
}

// InjectDelegationTools ensures all agents with AllowDelegation: true have
// access to their current coworkers via DelegateWork and AskQuestion tools.
func (c *Crew) InjectDelegationTools() {
	for _, agent := range c.Agents {
		// Only inject tools for local agents
		localAgent, ok := agent.(*agents.Agent)
		if !ok || !localAgent.AllowDelegation {
			continue
		}

		coworkers := make([]core.Agent, 0)
		for _, other := range c.Agents {
			if other != agent {
				coworkers = append(coworkers, other)
			}
		}

			if len(coworkers) > 0 {
				// Remove existing delegation tools to avoid duplicates and update coworkers
				newTools := make([]agents.Tool, 0)
				for _, t := range localAgent.Tools {
					if t.Name() != "DelegateWork" && t.Name() != "AskQuestion" {
						newTools = append(newTools, t)
					}
				}
				localAgent.Tools = newTools
 
				// Inject fresh tools with current coworker list
				localAgent.Tools = append(localAgent.Tools, delegation.NewDelegateWorkTool(coworkers))
				localAgent.Tools = append(localAgent.Tools, delegation.NewAskQuestionTool(coworkers))

				if c.Verbose {
					defaultLogger.Info("🔁 Delegation tools refreshed",
						slog.String("agent", agent.GetRole()),
						slog.Int("coworkers", len(coworkers)))
				}
			}
	}
}

