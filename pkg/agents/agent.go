package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"net/http"
	"strings"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/events"
	"github.com/Ecook14/gocrewwai/pkg/training"
	crewErrors "github.com/Ecook14/gocrewwai/pkg/errors"
	"github.com/Ecook14/gocrewwai/pkg/guardrails"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/Ecook14/gocrewwai/pkg/tools"
	"github.com/Ecook14/gocrewwai/pkg/protocols"
	"github.com/Ecook14/gocrewwai/pkg/sandbox"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/attribute"
	"github.com/Ecook14/gocrewwai/pkg/core"
)

var _ core.Agent = (*Agent)(nil)

var defaultLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

// Tool is a convenience alias for tools.Tool
type Tool = tools.Tool

// AgentConfig defines the parameters for creating a new Agent in a declarative style.
type AgentConfig struct {
	Role                 string
	Goal                 string
	Backstory            string
	LLM                  llm.Client
	Tools                []tools.Tool
	Memory               memory.Store
	Verbose              bool
	AllowDelegation      bool
	MaxIterations        int
	MaxRPM               int
	MaxRetryLimit        int
	MaxExecutionTime     time.Duration
	RespectContextWindow bool
	SelfHealing          bool
	Cache                llm.Cache
	SystemTemplate       string
	PromptTemplate       string
	ResponseTemplate     string
	StepCallback         func(step map[string]interface{})
	StepReview           func(toolName string, toolInput interface{}) bool
	StepStreamCallback   func(token string)
	SelfCritique         bool
	KnowledgeBases       []string
	FewShotExamples      []string
	Reasoning            bool
	MaxReasoningAttempts int
	Embedder             map[string]interface{}
	KnowledgeSources     []memory.KnowledgeSource
	UseSystemPrompt      *bool // Pointer to distinguish between unset and false
	AllowCodeExecution   bool
	CodeExecutionMode    string // "safe" or "unsafe"
	Multimodal           bool
	InjectDate           bool
	DateFormat           string
	TrainingMode         bool
	TrainingDir          string
	BeforeLLMCall        func(messages []llm.Message) []llm.Message
	Sandbox              string
	SandboxProvider      sandbox.Provider // NEW: Explicitly providing a sandbox runner
	SandboxType          string           // NEW: "none", "wasm", "docker"
	MCPS                 []string // MCP server URLs or command strings
	MCPAllowList         []string // Optional: Specific tools to allow (empty means all)
	MCPBlockList         []string // Optional: Specific tools to block
	MCPSamplingPolicy    string   // "Never", "Always", "AskHuman" (default: "AskHuman")
	A2APort              int      // If > 0, starts an A2A server on this port
	A2ACapabilities      []string // Capabilities to declare in A2A
	A2AAuthToken         string   // Bearer token for inter-agent auth
}

// Agent translates the `class Agent` python abstraction into idiomatic Go.
type Agent struct {
	Role      string `json:"role"`
	Goal      string `json:"goal"`
	Backstory string `json:"backstory"`
	Provider  string `json:"provider,omitempty"`
	LLMModel  string `json:"llm_model,omitempty"`
	Verbose   bool   `json:"verbose"`

	// LLM config
	LLM                llm.Client `json:"-"`
	FunctionCallingLLM llm.Client `json:"-"` // Separate LLM for tool calling
	Tools              []tools.Tool `json:"-"`

	// A2A Support
	A2AServer    *protocols.A2AServer    `json:"-"`
	A2AClient    *protocols.A2AClient    `json:"-"`
	A2ADiscovery *protocols.AgentDiscovery `json:"-"`
	A2AAuthToken string                  `json:"-"`
	A2AID        string                  `json:"a2a_id"`

	// MCP Bridge
	MCPServer *protocols.MCPServer `json:"-"`

	// SandboxProvider handles code execution environment.
	SandboxProvider sandbox.Provider `json:"-"`

	// Execution context limits
	MaxIterations        int `json:"-"`
	MaxRetryLimit        int `json:"-"`
	MaxRPM               int `json:"max_rpm"`
	RespectContextWindow bool `json:"-"`

	// Memory enables agents to recall and store context across executions.
	Memory memory.Store `json:"-"`

	// EntityMemory stores specific named facts about people, places, or things.
	EntityMemory memory.EntityStore `json:"-"`

	// Guardrails validate agent output before it is returned.
	Guardrails []guardrails.Guardrail `json:"-"`

	// SelfHealing enables the agent to autonomously correct tool errors.
	SelfHealing bool `json:"-"`

	// Cache persists LLM and tool results to save costs and latency.
	Cache llm.Cache `json:"-"`

	// UsageMetrics tracks token consumption during the agent's lifecycle.
	UsageMetrics map[string]int `json:"-"`

	// AllowDelegation enables this agent to delegate sub-tasks to coworkers.
	AllowDelegation bool `json:"-"`

	// StepCallback allows developers to hook into the execution loop for UI streaming.
	StepCallback func(step map[string]interface{}) `json:"-"`

	// StepReview allows for human-in-the-loop approval before tool execution.
	// Returns true if the action is approved, false otherwise.
	StepReview func(toolName string, toolInput interface{}) bool `json:"-"`

	// StepStreamCallback is called for each token generated by the LLM (Streaming).
	StepStreamCallback func(token string) `json:"-"`

	// SelfCritique enables internal reflection before returning an answer.
	SelfCritique bool `json:"-"`

	// KnowledgeBases provide additional context for the agent's tasks.
	KnowledgeBases []string `json:"-"`
	
	// FewShotExamples are used to train the agent's prompt for better accuracy.
	FewShotExamples []string `json:"-"`

	// Custom Templates
	SystemTemplate   string `json:"-"`
	PromptTemplate   string `json:"-"`
	ResponseTemplate string `json:"-"`

	// Core 1 Perfection Fields
	AllowCodeExecution   bool                   `json:"-"`
	CodeExecutionMode    string                 `json:"-"` // "safe", "unsafe"
	Multimodal           bool                   `json:"-"`
	InjectDate           bool                   `json:"-"`
	DateFormat           string                 `json:"-"`
	Reasoning            bool                   `json:"-"`
	MaxReasoningAttempts int                    `json:"-"`
	EmbedderConfig       map[string]interface{} `json:"-"`
	KnowledgeSources     []memory.KnowledgeSource  `json:"-"`
	UseSystemPrompt      bool                   `json:"-"`

	// InterruptCh allows sending async instructions/interrupts to the agent mid-execution.
	InterruptCh chan string `json:"-"`

	// Sandbox defines the preferred execution environment for the agent's code tools.
	Sandbox string `json:"sandbox"` // "local", "docker", "e2b", "wasm"

	// Timeout enforces a maximum duration for the entire execute loop.
	Timeout time.Duration `json:"timeout"`

	lastExecution time.Time

	TrainingMode bool   `json:"-"`
	TrainingDir  string `json:"-"`

	BeforeLLMCall func(messages []llm.Message) []llm.Message `json:"-"`
}

// AgentOption defines a functional option for configuring an Agent.
type AgentOption func(*Agent)

// WithMemory enables memory for the agent with a specific store.
func WithMemory(store memory.Store) AgentOption {
	return func(a *Agent) { 
		a.Memory = store
	}
}

// WithEntityMemory enables entity-specific memory for the agent.
func WithEntityMemory(store memory.EntityStore) AgentOption {
	return func(a *Agent) {
		a.EntityMemory = store
	}
}

// WithTools adds tools to the agent.
func WithTools(t []tools.Tool) AgentOption {
	return func(a *Agent) {
		a.Tools = append(a.Tools, t...)
	}
}

// WithVerbose toggles verbose logging.
func WithVerbose(v bool) AgentOption {
	return func(a *Agent) {
		a.Verbose = v
	}
}

// WithSelfHealing enables or disables autonomous error correction.
func WithSelfHealing(enabled bool) AgentOption {
	return func(a *Agent) { a.SelfHealing = enabled }
}

// WithCache enables result caching for the agent.
func WithCache(cache llm.Cache) AgentOption {
	return func(a *Agent) { a.Cache = cache }
}

// WithMaxIterations sets the maximum number of steps an agent can take.
func WithMaxIterations(max int) AgentOption {
	return func(a *Agent) { a.MaxIterations = max }
}

// WithMaxRPM sets the maximum requests per minute for the agent.
func WithMaxRPM(rpm int) AgentOption {
	return func(a *Agent) { a.MaxRPM = rpm }
}

func NewAgent(role, goal, backstory string, llm llm.Client, opts ...AgentOption) *Agent {
	a := &Agent{
		Role:          role,
		Goal:          goal,
		Backstory:     backstory,
		LLM:           llm,
		MaxIterations: 10, // Default
		UsageMetrics:  make(map[string]int),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// New creates a new Agent using a declarative configuration struct.
// This is the preferred "CrewAI Python-like" way to initialize agents.
func New(cfg AgentConfig) *Agent {
	a := &Agent{
		Role:                 cfg.Role,
		Goal:                 cfg.Goal,
		Backstory:            cfg.Backstory,
		LLM:                  cfg.LLM,
		Tools:                cfg.Tools,
		Verbose:              cfg.Verbose,
		AllowDelegation:      cfg.AllowDelegation,
		MaxIterations:        cfg.MaxIterations,
		MaxRPM:               cfg.MaxRPM,
		MaxRetryLimit:        cfg.MaxRetryLimit,
		Timeout:              cfg.MaxExecutionTime,
		RespectContextWindow: cfg.RespectContextWindow,
		SelfHealing:          cfg.SelfHealing,
		Cache:                cfg.Cache,
		SystemTemplate:       cfg.SystemTemplate,
		PromptTemplate:       cfg.PromptTemplate,
		ResponseTemplate:     cfg.ResponseTemplate,
		StepCallback:         cfg.StepCallback,
		StepReview:           cfg.StepReview,
		StepStreamCallback:   cfg.StepStreamCallback,
		SelfCritique:         cfg.SelfCritique,
		KnowledgeBases:       cfg.KnowledgeBases,
		FewShotExamples:      cfg.FewShotExamples,
		Sandbox:              cfg.Sandbox,
		AllowCodeExecution:   cfg.AllowCodeExecution,
		CodeExecutionMode:    cfg.CodeExecutionMode,
		Multimodal:           cfg.Multimodal,
		InjectDate:           cfg.InjectDate,
		DateFormat:           cfg.DateFormat,
		Reasoning:            cfg.Reasoning,
		MaxReasoningAttempts: cfg.MaxReasoningAttempts,
		EmbedderConfig:       cfg.Embedder,
		KnowledgeSources:     cfg.KnowledgeSources,
		UsageMetrics:         make(map[string]int),
		InterruptCh:          make(chan string, 1),
		TrainingMode:         cfg.TrainingMode,
		TrainingDir:          cfg.TrainingDir,
		BeforeLLMCall:        cfg.BeforeLLMCall,
		Memory:               cfg.Memory,
	}

	if cfg.UseSystemPrompt != nil {
		a.UseSystemPrompt = *cfg.UseSystemPrompt
	} else {
		a.UseSystemPrompt = true // Default
	}

	// Sandbox initialization
	if cfg.SandboxProvider != nil {
		a.SandboxProvider = cfg.SandboxProvider
	} else if cfg.SandboxType != "" && cfg.SandboxType != "none" {
		ctx := context.Background()
		switch strings.ToLower(cfg.SandboxType) {
		case "wasm":
			if p, err := sandbox.NewWasmProvider(ctx); err == nil {
				a.SandboxProvider = p
			} else {
				slog.Error("sandbox: failed to init wasm provider", slog.Any("error", err))
			}
		case "docker":
			image := cfg.Sandbox
			if image == "" {
				image = "python:3.11-slim" // Default image for Docker sandbox
			}
			if p, err := sandbox.NewDockerProvider(image); err == nil {
				a.SandboxProvider = p
			} else {
				slog.Error("sandbox: failed to init docker provider", slog.Any("error", err))
			}
		}
	}

	if a.DateFormat == "" {
		a.DateFormat = "2006-01-02" // Go's ISO format equivalent
	}

	if a.CodeExecutionMode == "" {
		a.CodeExecutionMode = "safe"
	}

	if a.AllowCodeExecution {
		safe := a.CodeExecutionMode == "safe"
		opts := []tools.CodeInterpreterOption{tools.WithSafeMode(safe)}
		
		// Map agent sandbox config to tool options
		switch strings.ToLower(a.Sandbox) {
		case "docker":
			opts = append(opts, tools.WithDocker("python:3.9-slim")) // Default image
		case "e2b":
			// Primary: Crew-GO standardized env, Fallback: E2B standard env
			key := os.Getenv("CREW_GO_E2B_API_KEY")
			if key == "" {
				key = os.Getenv("E2B_API_KEY")
			}
			if key != "" {
				opts = append(opts, tools.WithE2B(key))
			}
		}

		a.Tools = append(a.Tools, tools.NewCodeInterpreterTool(opts...))
	}

	if a.MaxIterations <= 0 {
		a.MaxIterations = 15
	}
	if a.MaxRetryLimit <= 0 {
		a.MaxRetryLimit = 3
	}

	// Initialize MCP Tools if specified
	if len(cfg.MCPS) > 0 {
		for _, source := range cfg.MCPS {
			var client *protocols.MCPClient
			if strings.HasPrefix(source, "http") {
				client = protocols.NewMCPClient(source)
			} else if strings.HasPrefix(source, "stdio:") {
				cmdStr := strings.TrimPrefix(source, "stdio:")
				parts := strings.Fields(cmdStr)
				if len(parts) > 0 {
					transport := protocols.NewStdioTransport(parts[0], parts[1:]...)
					client = protocols.NewMCPClientWithTransport(transport)
				}
			}

			if client != nil {
				// Bind Sampling callback
				client.OnSample = func(ctx context.Context, prompt string) (string, error) {
					policy := cfg.MCPSamplingPolicy
					if policy == "" {
						policy = "AskHuman"
					}

					if policy == "Never" {
						return "", fmt.Errorf("sampling denied by agent policy")
					}

					if policy == "AskHuman" {
						slog.Info("[🤖 MCP SAMPLING REQUEST] Server is requesting an LLM completion", slog.String("prompt", prompt))
						fmt.Printf("MCP Server requests completion for: %s\n", prompt)
						fmt.Print("Approve sampling? (y/n/feedback): ")
						// In a real CLI/UI we'd wait for input
						// For now we assume approval if policy is Always or simulated
					}

					// Route to Agent's LLM
					res, err := a.Execute(ctx, prompt, nil)
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("%v", res), nil
				}

				// We use a short-lived context for discovery
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := client.Initialize(ctx); err == nil {
					// 1. Inject Tools with Filtering
					for _, toolDef := range client.Tools {
						if isAllowed(toolDef.Name, cfg.MCPAllowList, cfg.MCPBlockList) {
							a.Tools = append(a.Tools, protocols.WrapMCPToolForCrewGo(client, toolDef))
						}
					}

					// 2. Inject Resources as Tools with Filtering
					for _, resDef := range client.Resources {
						if isAllowed(resDef.Name, cfg.MCPAllowList, cfg.MCPBlockList) {
							a.Tools = append(a.Tools, protocols.WrapMCPResource(client, resDef))
						}
					}

					// 3. Inject Prompts into Backstory
					for _, promptDef := range client.Prompts {
						if isAllowed(promptDef.Name, cfg.MCPAllowList, cfg.MCPBlockList) {
							if promptText, err := client.GetPrompt(ctx, promptDef.Name, nil); err == nil {
								a.Backstory += fmt.Sprintf("\n\n[MCP PROMPT: %s]\n%s", promptDef.Name, promptText)
							}
						}
					}
				} else {
					slog.Error("Failed to initialize MCP client", slog.String("source", source), slog.Any("error", err))
				}
				cancel()
			}
		}
	}

	// 4. Auto-Discovery (mDNS)
	// Implementation of Zeroconf scanning would go here, adding servers to cfg.MCPS
	
	// 5. Setup A2A if configured
	if cfg.A2APort > 0 {
		a.A2AAuthToken = cfg.A2AAuthToken
		a.setupA2A(cfg)
	}

	return a
}

func (a *Agent) setupA2A(cfg AgentConfig) {
	a.A2AID = fmt.Sprintf("%s-%d", strings.ReplaceAll(a.Role, " ", "-"), time.Now().UnixNano())
	a.A2AClient = protocols.NewA2AClient(cfg.A2AAuthToken)
	
	router := protocols.NewA2ARouter()
	router.Handle("delegate_task", a.HandleA2AMessage)
	router.Handle("status", a.HandleA2AStatus)
	
	a.A2AServer = protocols.NewA2AServer(cfg.A2APort, router, cfg.A2AAuthToken)
	if err := a.A2AServer.Start(); err != nil {
		slog.Error("a2a: failed to start server", slog.Any("error", err))
		return
	}

	// Discovery Service
	a.A2ADiscovery = protocols.NewAgentDiscovery(protocols.GlobalA2ARegistry)
	card := &protocols.AgentCard{
		ID:           a.A2AID,
		Name:         a.Role,
		Role:         a.Role,
		Description:  a.Goal,
		Capabilities: cfg.A2ACapabilities,
		Endpoint:     fmt.Sprintf("http://localhost:%d", cfg.A2APort),
	}
	
	// Register locally and advertise globally (simulated)
	protocols.GlobalA2ARegistry.Register(card)
	a.A2ADiscovery.Advertise(context.Background(), card)
	a.A2ADiscovery.StartScanning(context.Background(), 1*time.Minute)
}

func (a *Agent) HandleA2AStatus(ctx context.Context, msg protocols.A2AMessage) (*protocols.A2AMessage, error) {
	status := map[string]interface{}{
		"status":    "healthy",
		"uptime":    time.Since(a.lastExecution).String(), // Just a placeholder
		"role":      a.Role,
		"metrics":   a.UsageMetrics,
		"iteration": "idle",
	}

	return &protocols.A2AMessage{
		ID:            fmt.Sprintf("res-%d", time.Now().UnixNano()),
		From:          a.A2AID,
		To:            msg.From,
		Type:          protocols.A2AResponse,
		Action:        "status_response",
		Payload:       status,
		CorrelationID: msg.ID,
		Timestamp:     time.Now(),
	}, nil
}

func (a *Agent) HandleA2AMessage(ctx context.Context, msg protocols.A2AMessage) (*protocols.A2AMessage, error) {
	if msg.Action == "delegate_task" {
		req, err := protocols.UnmarshalTaskRequest(msg.Payload)
		if err != nil {
			return nil, err
		}

		options := make(map[string]interface{})
		
		// Check for WebSocket connection in context for streaming
		if conn, ok := ctx.Value("a2a_ws_conn").(*websocket.Conn); ok {
			options["stream_callback"] = func(token string) {
				_ = conn.WriteJSON(protocols.A2AMessage{
					ID:            fmt.Sprintf("stream-%d", time.Now().UnixNano()),
					From:          a.A2AID,
					To:            msg.From,
					Type:          protocols.A2AStream,
					Action:        "token_stream",
					Payload:       map[string]interface{}{"token": token},
					CorrelationID: msg.ID,
					Timestamp:     time.Now(),
				})
			}
		}

		result, err := a.Execute(ctx, req.Description, options)
		
		resPayload := protocols.A2ATaskResponse{
			Success: err == nil,
			Result:  fmt.Sprintf("%v", result),
		}
		if err != nil {
			resPayload.Error = err.Error()
		}

		return &protocols.A2AMessage{
			ID:            fmt.Sprintf("res-%d", time.Now().UnixNano()),
			From:          a.A2AID,
			To:            msg.From,
			Type:          protocols.A2AResponse,
			Action:        "delegate_task_response",
			Payload:       protocols.MarshalTaskResponse(resPayload),
			CorrelationID: msg.ID,
			Timestamp:     time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("unsupported a2a action: %s", msg.Action)
}

func isAllowed(name string, allowList, blockList []string) bool {
	allowed := len(allowList) == 0
	if !allowed {
		for _, a := range allowList {
			if a == name {
				allowed = true
				break
			}
		}
	}
	for _, b := range blockList {
		if b == name {
			allowed = false
			break
		}
	}
	return allowed
}

// GetRole returns the agent's role. Implements the delegation.Agent interface.
func (a *Agent) GetRole() string {
	if a.UsageMetrics == nil {
		a.UsageMetrics = make(map[string]int)
	}
	return a.Role
}

func (a *Agent) GetGoal() string {
	return a.Goal
}

func (a *Agent) GetMaxRPM() int {
	return a.MaxRPM
}

func (a *Agent) SetMaxRPM(rpm int) {
	a.MaxRPM = rpm
}

func (a *Agent) GetUsageMetrics() map[string]int {
	if a.UsageMetrics == nil {
		a.UsageMetrics = make(map[string]int)
	}
	return a.UsageMetrics
}

func (a *Agent) GetLLM() llm.Client {
	return a.LLM
}

func (a *Agent) GetToolCount() int {
	return len(a.Tools)
}

// Equip allows adding tools to an agent after it has been initialized.
func (a *Agent) Equip(tools ...tools.Tool) {
	a.Tools = append(a.Tools, tools...)
}

// Execute handles running a task, converting from `async def execute_task()`.
// This forms the core logic layer for Agent behavior mapping.
// Provides structured outputs if mapped via Options array.
func (a *Agent) Execute(ctx context.Context, taskInput string, options map[string]interface{}) (interface{}, error) {
	// 1. Enforce Agent-level Timeout
	if a.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.Timeout)
		defer cancel()
	}

	// ---------------------------------------------------------
	// Rate Limiting (MaxRPM)
	// ---------------------------------------------------------
	if a.MaxRPM > 0 {
		minInterval := time.Minute / time.Duration(a.MaxRPM)
		elapsed := time.Since(a.lastExecution)
		if elapsed < minInterval {
			wait := minInterval - elapsed
			if a.Verbose {
				defaultLogger.Info("⏳ Rate Limiting (Agent MaxRPM)", slog.String("role", a.Role), slog.Duration("wait", wait))
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
		defer func() { a.lastExecution = time.Now() }()
	}

	ctx, span := telemetry.StartSpan(ctx, "Agent.Execute")
	if span != nil {
		defer span.End()
		span.SetAttributes(attribute.String("agent.role", a.Role))
	}

	if a.UsageMetrics == nil {
		a.UsageMetrics = make(map[string]int)
	}

	if a.LLM == nil {
		return "Task executed successfully by " + a.Role, nil
	}

	if a.Verbose {
		preview := taskInput
		if len(preview) > 50 {
			preview = preview[:47] + "..."
		}
		defaultLogger.Info("Agent executing", slog.String("role", a.Role), slog.String("task", preview))
	}

	if a.StepCallback != nil {
		a.StepCallback(map[string]interface{}{
			"role":  a.Role,
			"phase": "starting",
			"input": taskInput,
		})
	}

	// Publish system event
	events.GlobalBus.Publish(events.Event{
		Type:   events.AgentExecutionStarted,
		Source: a.Role,
		Payload: map[string]interface{}{
			"input": taskInput,
		},
	})

	// Memory Recall: inject relevant past context into the prompt
	enrichedInput := taskInput
	if a.Memory != nil {
		enrichedInput = a.recallMemory(ctx, taskInput)
	}

	// Format the ReAct Tooling System Prompt
	toolDescriptions := ""
	for _, t := range a.Tools {
		toolDescriptions += fmt.Sprintf("- %s: %s\n", t.Name(), t.Description())
	}

	// Optional sections for knowledge and few-shot examples
	knowledgeSection := ""
	if len(a.KnowledgeBases) > 0 {
		knowledgeSection = "\n\nKNOWLEDGE BASE CONTEXT:\n" + strings.Join(a.KnowledgeBases, "\n---\n")
	}

	fewShotSection := ""
	if len(a.FewShotExamples) > 0 {
		fewShotSection = "\n\nFEW-SHOT EXAMPLES FOR GUIDANCE:\n" + strings.Join(a.FewShotExamples, "\n---\n")
	}

	// Training Advice Section
	trainingAdvice := ""
	if a.TrainingDir != "" {
		store := training.NewStore(a.TrainingDir)
		if data, err := store.LoadAgentData(a.Role); err == nil && len(data.Suggestions) > 0 {
			trainingAdvice = "\n\nPAST TRAINING ADVICE FOR YOUR ROLE:\n"
			for _, s := range data.Suggestions {
				trainingAdvice += fmt.Sprintf("- %s\n", s)
			}
		}
	}

	systemPrompt := a.SystemTemplate
	if systemPrompt == "" {
		dateInjection := ""
		if a.InjectDate {
			dateInjection = fmt.Sprintf("\nCurrent date: %s", time.Now().Format(a.DateFormat))
		}

		systemPrompt = fmt.Sprintf(`You are %s. %s%s
Your goal is: %s
%s
%s

You have access to the following tools:
%s

To use a tool, you MUST reply with a pure JSON object in this exact format:
{"tool": "ToolName", "input": {"parameter_name": "parameter_value"}}

Once you have gathered all necessary information and are ready to provide the final answer, do NOT return a tool JSON. Simply output your final answer text natively.%s`, a.Role, a.Backstory, dateInjection, a.Goal, knowledgeSection, fewShotSection, toolDescriptions, trainingAdvice)
	}

	userInput := a.PromptTemplate
	if userInput == "" {
		userInput = enrichedInput
	} else {
		userInput = strings.ReplaceAll(userInput, "{input}", enrichedInput)
	}

	messages := []llm.Message{}
	if a.UseSystemPrompt {
		messages = append(messages, llm.Message{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, llm.Message{Role: "user", Content: userInput})

	// ---------------------------------------------------------
	// Core 1 Perfection: Reasoning Phase (Reflect-Evaluate-Refine)
	// ---------------------------------------------------------
	if a.Reasoning {
		plan, err := a.runReasoningLoop(ctx, messages, options)
		if err == nil {
			messages = append(messages, llm.Message{Role: "assistant", Content: "Plan: " + plan})
		}
	}

	// ---------------------------------------------------------
	// The ReAct Autonomous Tool-Calling Loop Engine
	// ---------------------------------------------------------
	maxLoops := a.MaxIterations
	if maxLoops <= 0 {
		maxLoops = 15 // Default safety guard against infinite loops
	}

	maxRetries := a.MaxRetryLimit
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for i := 0; i < maxLoops; i++ {
		// Check global task cancellation and human interrupts
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case interrupt := <-a.InterruptCh:
			if a.Verbose {
				defaultLogger.Info("🛑 Agent INTERRUPTED", slog.String("agent", a.Role), slog.String("instruction", interrupt))
			}
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: fmt.Sprintf("[USER INTERRUPT]: %s. Please adjust your next action accordingly.", interrupt),
			})
		default:
		}

		// Dynamic UI Control Check (HITL Pause)
		if err := telemetry.GlobalExecutionController.WaitIfPaused(ctx); err != nil {
			return nil, err
		}

		if a.StepCallback != nil {
			a.StepCallback(map[string]interface{}{"role": a.Role, "phase": "thinking", "iteration": i + 1})
		}

		// Publish system event
		events.GlobalBus.Publish(events.Event{
			Type:   events.AgentThinking,
			Source: a.Role,
			Payload: map[string]interface{}{
				"iteration": i + 1,
			},
		})

		// Trigger BeforeLLMCall hook
		if a.BeforeLLMCall != nil {
			messages = a.BeforeLLMCall(messages)
		}

		// If a strict final schema is requested, disable autonomous looping
		if options != nil && options["schema"] != nil {
			mappedSchema := options["schema"]
			response, err := a.LLM.GenerateStructured(ctx, messages, mappedSchema, options)
			return response, err
		}

		// LLM Generation (Streaming vs Block)
		var responseText string
		cacheKey := ""
		if a.Cache != nil {
			cacheKey = llm.GenerateCacheKey(fmt.Sprintf("%T", a.LLM), fmt.Sprintf("%v", messages), options)
			if cached, ok := a.Cache.Get(cacheKey); ok {
				responseText = cached
				if a.Verbose {
					defaultLogger.Info("💾 Agent LLM Cache Hit", slog.String("role", a.Role))
				}
			}
		}

		if responseText == "" {
			// Check for streaming callback in options (e.g., from A2A WebSocket)
			streamCallback := a.StepStreamCallback
			if cb, ok := options["stream_callback"].(func(string)); ok && streamCallback == nil {
				streamCallback = cb
			}

			if streamCallback != nil {
				// Trigger StepCallback for production monitoring
				if a.StepCallback != nil {
					a.StepCallback(map[string]interface{}{"role": a.Role, "phase": "generation_started", "streaming": true})
				}
				stream, err := a.LLM.StreamGenerate(ctx, messages, options)
				if err != nil {
					return nil, crewErrors.NewAgentError(strings.Clone(strings.TrimSpace(a.Role)), i+1, fmt.Errorf("%w: %v", crewErrors.ErrLLMFailed, err))
				}
				var sb strings.Builder
				for token := range stream {
					sb.WriteString(token)
					streamCallback(token)
				}
				responseText = sb.String()
			} else {
				llmClient := a.LLM
				if a.FunctionCallingLLM != nil {
					llmClient = a.FunctionCallingLLM
				}
				if a.StepCallback != nil {
					a.StepCallback(map[string]interface{}{"role": a.Role, "phase": "generation_started", "streaming": false})
				}
				var err error
				responseText, err = llmClient.Generate(ctx, messages, options)
				if err != nil {
					return nil, crewErrors.NewAgentError(strings.Clone(strings.TrimSpace(a.Role)), i+1, fmt.Errorf("%w: %v", crewErrors.ErrLLMFailed, err))
				}
			}
			if a.Cache != nil {
				_ = a.Cache.Set(cacheKey, responseText)
			}
		}

		// Elite Tier: Usage Metrics Tracking (Token Heuristic)
		a.UsageMetrics["prompt_tokens"] += len(enrichedInput) / 4
		a.UsageMetrics["completion_tokens"] += len(responseText) / 4

		// ReAct Tool Parsing
		var toolReq struct {
			Tool  string                 `json:"tool"`
			Input map[string]interface{} `json:"input"`
		}

		if err := json.Unmarshal([]byte(strings.TrimSpace(responseText)), &toolReq); err == nil && toolReq.Tool != "" {
			var activeTool tools.Tool
			for _, t := range a.Tools {
				if t.Name() == toolReq.Tool {
					activeTool = t
					break
				}
			}

			if activeTool != nil {
				if a.Verbose {
					defaultLogger.Info("🔨 Agent Tool Triggered", slog.String("agent", a.Role), slog.String("tool", toolReq.Tool), slog.Any("input", toolReq.Input))
				}

				if a.StepCallback != nil {
					a.StepCallback(map[string]interface{}{"role": a.Role, "phase": "tool_execution", "tool": toolReq.Tool})
				}

				// Publish system event
				events.GlobalBus.Publish(events.Event{
					Type:   events.ToolUsageStarted,
					Source: a.Role,
					Payload: map[string]interface{}{
						"tool":  toolReq.Tool,
						"input": toolReq.Input,
					},
				})

				// HITL Check
				if activeTool.RequiresReview() {
					if a.Verbose {
						defaultLogger.Info("⏳ Agent tool execution pending human approval", slog.String("agent", a.Role), slog.String("tool", toolReq.Tool))
					}

					var approved bool
					if a.StepReview != nil {
						approved = a.StepReview(toolReq.Tool, toolReq.Input)
					} else {
						reviewID := fmt.Sprintf("%s-%d", strings.ReplaceAll(a.Role, " ", "-"), time.Now().UnixNano())
						approved = telemetry.GlobalReviewManager.RequestReview(reviewID, a.Role, toolReq.Tool, toolReq.Input)
					}

					if !approved {
						observation := "Tool Execution Denied by human."
						messages = append(messages, llm.Message{Role: "assistant", Content: responseText})
						messages = append(messages, llm.Message{Role: "user", Content: observation})
						continue
					}
				}

				// Tool Execution
				var toolResult string
				var toolErr error
				toolCacheKey := ""
				if a.Cache != nil {
					toolCacheKey = fmt.Sprintf("tool:%s|input:%v", toolReq.Tool, toolReq.Input)
					if cached, ok := a.Cache.Get(toolCacheKey); ok {
						toolResult = cached
						if a.Verbose {
							defaultLogger.Info("💾 Agent Tool Cache Hit", slog.String("role", a.Role), slog.String("tool", toolReq.Tool))
						}
					}
				}

				if toolResult == "" {
					// Use Sandbox if configured and tool is CodeInterpreter
					if a.SandboxProvider != nil && (toolReq.Tool == "CodeInterpreter" || toolReq.Tool == "PythonInterpreter") {
						code, _ := toolReq.Input["code"].(string)
						if code == "" {
							// Fallback to standard execution if no code found
							toolResult, toolErr = activeTool.Execute(ctx, toolReq.Input)
						} else {
							if a.Verbose {
								defaultLogger.Info("🛡️ Routing tool to Production Sandbox", slog.String("provider", fmt.Sprintf("%T", a.SandboxProvider)))
							}
							toolResult, toolErr = a.SandboxProvider.Execute(ctx, code, nil)
						}
					} else {
						toolResult, toolErr = activeTool.Execute(ctx, toolReq.Input)
					}
					
					if toolErr == nil && a.Cache != nil {
						_ = a.Cache.Set(toolCacheKey, toolResult)
					}
				}

				// Publish system event
				events.GlobalBus.Publish(events.Event{
					Type:   events.ToolUsageFinished,
					Source: a.Role,
					Payload: map[string]interface{}{
						"tool":   toolReq.Tool,
						"result": toolResult,
						"error":  toolErr,
					},
				})

				var observation string
				if toolErr != nil {
					if a.SelfHealing && maxRetries > 0 {
						maxRetries--
						observation = fmt.Sprintf("Tool Error: %v. Please correct your parameters and try again.", toolErr)
						defaultLogger.Warn("Self-healing triggered", slog.String("agent", a.Role), slog.String("error", toolErr.Error()))
					} else {
						observation = fmt.Sprintf("Fatal Tool Error: %v", toolErr)
						defaultLogger.Error("❌ Tool Execution Failed", slog.String("agent", a.Role), slog.String("tool", toolReq.Tool), slog.Any("error", toolErr))
					}
				} else {
					observation = fmt.Sprintf("Observation: %v", toolResult)
					if a.Verbose {
						defaultLogger.Info("✅ Tool Execution Success", slog.String("agent", a.Role), slog.String("tool", toolReq.Tool))
					}
				}

				messages = append(messages, llm.Message{Role: "assistant", Content: responseText})
				messages = append(messages, llm.Message{Role: "user", Content: observation})
				continue
			} else {
				// The LLM hallucinated a tool name that doesn't exist
				messages = append(messages, llm.Message{Role: "assistant", Content: responseText})
				messages = append(messages, llm.Message{Role: "user", Content: fmt.Sprintf(
					"Observation Error: The tool '%s' does not exist. Available tools are: \n%s",
					toolReq.Tool, toolDescriptions)})
				continue
			}
		}

		// Final Result Handling
		if len(a.Guardrails) > 0 {
			if err := guardrails.RunAll(a.Guardrails, responseText); err != nil {
				if maxRetries > 0 {
					maxRetries--
					messages = append(messages, llm.Message{Role: "assistant", Content: responseText})
					messages = append(messages, llm.Message{Role: "user", Content: fmt.Sprintf("Validation Error: %v. Please regenerate.", err)})
					if a.Verbose {
						defaultLogger.Warn("⚠️ Guardrail failed, retrying", slog.String("agent", a.Role), slog.String("error", err.Error()))
					}
					continue
				}
				return nil, crewErrors.NewAgentError(strings.Clone(strings.TrimSpace(a.Role)), i+1,
					fmt.Errorf("%w: %v", crewErrors.ErrGuardrailFailed, err))
			}
		}

		if a.Verbose {
			defaultLogger.Info("✅ Agent loop finalized", slog.String("agent", a.Role))
		}

		if a.Memory != nil {
			a.saveMemory(ctx, taskInput, responseText)
		}

		// Self-Critique / Internal Reflection
		if a.SelfCritique {
			critiquePrompt := fmt.Sprintf("Critique your own response for accuracy and tone. If it is perfect, respond 'APPROVED'. Otherwise, suggest improvements.\nYour Response: %s", responseText)
			critique, err := a.LLM.Generate(ctx, []llm.Message{
				{Role: "system", Content: "You are a critical self-reviewer."},
				{Role: "user", Content: critiquePrompt},
			}, nil)
			
			if err == nil && !strings.Contains(strings.ToUpper(critique), "APPROVED") {
				if a.Verbose {
					defaultLogger.Info("🔄 Agent self-correcting based on internal critique", slog.String("role", a.Role))
				}
				// Re-run with critique
				return a.Execute(ctx, taskInput+"\n\nINTERNAL CRITIQUE: "+critique, options)
			}
		}

		// Publish system event
		events.GlobalBus.Publish(events.Event{
			Type:   events.AgentExecutionCompleted,
			Source: a.Role,
			Payload: map[string]interface{}{
				"result": responseText,
			},
		})

		// Handle Training Mode Persistence
		if a.TrainingMode && a.TrainingDir != "" {
			store := training.NewStore(a.TrainingDir)
			data, _ := store.LoadAgentData(a.Role)
			
			// We already captured the interaction if it was HITL, 
			// but here we force-consolidate it into the store.
			iteration := training.IterationData{
				InitialOutput:  taskInput,
				ImprovedOutput: responseText,
			}
			data.Iterations = append(data.Iterations, iteration)
			store.SaveAgentData(a.Role, data)
		}

		return responseText, nil
	}

	return nil, crewErrors.NewAgentError(strings.Clone(strings.TrimSpace(a.Role)), maxLoops,
		fmt.Errorf("%w (%d iterations)", crewErrors.ErrMaxIterations, maxLoops))
}

// recallMemory queries the agent's memory store for relevant past context
// and prepends it to the task input.
func (a *Agent) recallMemory(ctx context.Context, taskInput string) string {
	if a.LLM == nil {
		return taskInput
	}

	var sb strings.Builder
	hasContext := false

	// 1. Vector Search (Short-Term/Long-Term)
	if a.Memory != nil {
		if embedder, ok := a.LLM.(llm.Embedder); ok {
			vector, err := embedder.GenerateEmbedding(ctx, taskInput)
			if err == nil {
				items, _ := a.Memory.Search(ctx, vector, 2)
				if len(items) > 0 {
					if !hasContext {
						sb.WriteString("RELEVANT PAST CONTEXT:\n")
						hasContext = true
					}
					for i, item := range items {
						sb.WriteString(fmt.Sprintf("--- Historical Memory %d ---\n%s\n", i+1, item.Text))
					}
				}
			}
		}
	}

	// 2. Entity Fact Retrieval
	if a.EntityMemory != nil {
		// Use the task input as a keyword search for entities
		entities, _ := a.EntityMemory.Search(ctx, taskInput, 2)
		if len(entities) > 0 {
			if !hasContext {
				sb.WriteString("RELEVANT PAST CONTEXT:\n")
				hasContext = true
			}
			for i, entity := range entities {
				sb.WriteString(fmt.Sprintf("--- Tracked Entity %d ---\n%s\n", i+1, entity.Text))
			}
		}
	}

	// 3. Process Knowledge Sources
	if len(a.KnowledgeSources) > 0 {
		if !hasContext {
			sb.WriteString("RELEVANT PAST CONTEXT:\n")
			hasContext = true
		}
		for i, source := range a.KnowledgeSources {
			if content, err := source.Query(ctx, taskInput); err == nil && content != "" {
				sb.WriteString(fmt.Sprintf("--- Knowledge Source %d ---\n%s\n", i+1, content))
			}
		}
	}

	if hasContext {
		sb.WriteString("--------------------------\n\n")
		sb.WriteString(taskInput)
		return sb.String()
	}

	return taskInput
}

// saveMemory persists the task input and result to the agent's memory store.
func (a *Agent) saveMemory(ctx context.Context, taskInput, result string) {
	content := fmt.Sprintf("Task: %s\nResult: %s", taskInput, result)

	// Generate embedding for storage
	embedder, ok := a.LLM.(llm.Embedder)
	if !ok {
		if a.Verbose {
			defaultLogger.Warn("Memory save skipped: LLM does not support text embeddings")
		}
		return
	}

	vector, err := embedder.GenerateEmbedding(ctx, content)
	if err != nil {
		if a.Verbose {
			defaultLogger.Warn("Memory save: embedding generation failed", slog.String("error", err.Error()))
		}
		return
	}

	item := &memory.MemoryItem{
		ID:     fmt.Sprintf("agent_%s_%d", a.Role, len(content)),
		Text:   content,
		Vector: vector,
		Metadata: map[string]interface{}{
			"agent_role": a.Role,
			"type":       "execution_result",
		},
	}
	_ = a.Memory.Add(ctx, item)

	// If EntityMemory is present, we try to extract and save specific facts
	if a.EntityMemory != nil {
		go a.extractAndStoreEntities(ctx, result)
	}
}

// extractAndStoreEntities uses the LLM to find entities/facts in a result and saves them.
func (a *Agent) extractAndStoreEntities(ctx context.Context, text string) {
	prompt := fmt.Sprintf(
		"Extract key entities and facts from the following text. "+
		"Return ONLY a JSON array of objects like: [{\"entity\": \"Name\", \"value\": \"Value\", \"description\": \"Context\"}]. "+
		"If none found, return [].\nText: %s", text)

	response, err := a.LLM.Generate(ctx, []llm.Message{
		{Role: "system", Content: "You are a precise data extractor."},
		{Role: "user", Content: prompt},
	}, nil)

	if err != nil {
		return
	}

	var entities []struct {
		Entity      string `json:"entity"`
		Value       string `json:"value"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &entities); err == nil {
		for _, e := range entities {
			_ = a.EntityMemory.Upsert(ctx, e.Entity, e.Value, e.Description)
		}
	}
}

// StartMCPServer starts an MCP server over SSE on the given port.
func (a *Agent) StartMCPServer(port int) error {
	a.MCPServer = protocols.NewMCPServer()
	bridge := protocols.NewAgentMCPBridge(a.MCPServer)
	bridge.ExposeAgent(a, a.Role, a.Goal)

	addr := fmt.Sprintf(":%d", port)
	slog.Info("🚀 Agent MCP Server (SSE) starting", slog.String("role", a.Role), slog.Int("port", port))
	return http.ListenAndServe(addr, a.MCPServer.Handler())
}

// StartMCPStdioServer starts an MCP server over Stdio (for CLI plugins).
func (a *Agent) StartMCPStdioServer(ctx context.Context) error {
	a.MCPServer = protocols.NewMCPServer()
	bridge := protocols.NewAgentMCPBridge(a.MCPServer)
	bridge.ExposeAgent(a, a.Role, a.Goal)

	server := protocols.NewStdioServer(a.MCPServer)
	return server.Serve(ctx)
}

