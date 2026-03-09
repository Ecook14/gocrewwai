// Package gocrew provides a unified, ergonomic SDK for the Gocrewwai framework.
//
// Instead of importing multiple sub-packages, users can simply:
//
//	import "github.com/Ecook14/gocrewwai/gocrew"
//
// And then use all core AND intermediate types directly:
//
//	agent := gocrew.NewAgent(gocrew.AgentConfig{...})
//	task := gocrew.NewTask(gocrew.TaskConfig{...})
//	crew := gocrew.NewCrew(gocrew.CrewConfig{...})
//	mem := gocrew.NewMemory(store, llmClient, nil)
//	f := gocrew.NewFlow(nil)
package gocrew

import (
	"context"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/events"
	"github.com/Ecook14/gocrewwai/pkg/files"
	"github.com/Ecook14/gocrewwai/pkg/flow"
	"github.com/Ecook14/gocrewwai/pkg/knowledge"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/testing"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

// ============================================================
// Core Type Aliases
// ============================================================

// Agent Types
type Agent = agents.Agent
type AgentConfig = agents.AgentConfig

// Task Types
type Task = tasks.Task
type TaskConfig = tasks.TaskConfig

// Crew Types
type Crew = crew.Crew
type CrewConfig = crew.CrewConfig

// LLM Types
type LLMClient = llm.Client
type LLMMessage = llm.Message
type LLMOptions = llm.Options

// Tool Types
type Tool = tools.Tool
type BaseTool = tools.BaseTool
type ArgSchema = tools.ArgSchema

// ============================================================
// Intermediate Concept Type Aliases
// ============================================================

// Memory Types
type MemoryStore = memory.Store
type KnowledgeSource = memory.KnowledgeSource
type UnifiedMemory = memory.UnifiedMemory
type UnifiedMemoryConfig = memory.UnifiedMemoryConfig
type MemoryScope = memory.MemoryScope
type MemorySlice = memory.MemorySlice
type RememberOptions = memory.RememberOptions
type RecallOptions = memory.RecallOptions
type ScoredMemory = memory.ScoredMemory
type RecallDepth = memory.RecallDepth

// Memory Recall Depth Constants
const (
	RecallShallow = memory.RecallShallow
	RecallDeep    = memory.RecallDeep
)

// File Types
type File = files.File
type FileType = files.FileType
type FileMode = files.FileMode
type FileBytes = files.FileBytes

// File Mode Constants
const (
	ModeStrict FileMode = files.ModeStrict
	ModeAuto   FileMode = files.ModeAuto
	ModeWarn   FileMode = files.ModeWarn
	ModeChunk  FileMode = files.ModeChunk
)

// Flow Types
type Flow = flow.Flow
type FlowState = flow.State
type FlowNode = flow.Node
type FlowRouter = flow.Router
type FlowPersistence = flow.FlowPersistence
type PersistentFlow = flow.PersistentFlow
type HumanFeedbackConfig = flow.HumanFeedbackConfig
type TypedFlow[T any] = flow.TypedFlow[T]
type TypedNode[T any] = flow.TypedNode[T]

// Knowledge Types
type KnowledgeConfig = knowledge.Config
type KnowledgeEvent = knowledge.Event
type KnowledgeEventType = knowledge.EventType
type StringSource = knowledge.StringSource
type TextFileSource = knowledge.TextFileSource
type PDFSource = knowledge.PDFSource
type CSVSource = knowledge.CSVSource
type JSONSource = knowledge.JSONSource
type URLSource = knowledge.URLSource

// Advanced/Mastery Types
type Event = events.Event
type EventType = events.EventType
type TestResult = testing.TestResult
type PerformanceSuite = testing.PerformanceSuite

// Global Bus Access
var GlobalBus = events.GlobalBus

// Process Type Constants
const (
	Sequential   = crew.Sequential
	Hierarchical = crew.Hierarchical
	Consensual   = crew.Consensual
	Graph        = crew.Graph
	Reflective   = crew.Reflective
	StateMachine = crew.StateMachine
)

// ============================================================
// Core Constructors
// ============================================================

// NewAgent creates a new Agent using a declarative config.
func NewAgent(cfg AgentConfig) *Agent {
	return agents.New(cfg)
}

// NewTask creates a new Task using a declarative config.
func NewTask(cfg TaskConfig) *Task {
	return tasks.New(cfg)
}

// NewCrew creates a new Crew using a declarative config.
func NewCrew(cfg CrewConfig) *Crew {
	return crew.New(cfg)
}

// Kickoff is a convenience function that creates and immediately executes a crew.
func Kickoff(ctx context.Context, cfg CrewConfig) (interface{}, error) {
	c := crew.New(cfg)
	return c.Kickoff(ctx)
}

// ============================================================
// Memory Constructors
// ============================================================

// NewMemory creates a unified memory instance with Remember/Recall/Forget API.
func NewMemory(store MemoryStore, llmClient LLMClient, cfg *UnifiedMemoryConfig) *UnifiedMemory {
	return memory.NewUnifiedMemory(store, llmClient, cfg)
}

// ============================================================
// File Constructors
// ============================================================

// ImageFile creates a file handle for an image (path or URL).
func ImageFile(source string, mode ...FileMode) File { return files.ImageFile(source, mode...) }

// PDFFile creates a file handle for a PDF.
func PDFFile(source string, mode ...FileMode) File { return files.PDFFile(source, mode...) }

// AudioFile creates a file handle for audio content.
func AudioFile(source string, mode ...FileMode) File { return files.AudioFile(source, mode...) }

// VideoFile creates a file handle for video content.
func VideoFile(source string, mode ...FileMode) File { return files.VideoFile(source, mode...) }

// TextFile creates a file handle for text content.
func TextFile(source string, mode ...FileMode) File { return files.TextFile(source, mode...) }

// NewFile auto-detects file type from extension.
func NewFile(source string, mode ...FileMode) File { return files.NewFile(source, mode...) }

// FromBytes creates a file from raw bytes.
func FromBytes(fb FileBytes, mode ...FileMode) File { return files.FromBytes(fb, mode...) }

// ValidateFile checks if a file is compatible with a provider.
func ValidateFile(file File, provider string) error { return files.ValidateFile(file, provider) }

// ============================================================
// Flow Constructors
// ============================================================

// NewFlow creates a new workflow orchestration flow.
func NewFlow(initialState FlowState) *Flow {
	return flow.NewFlow(initialState)
}

// NewPersistentFlow creates a flow that auto-persists state after each node.
func NewPersistentFlow(flowID string, persistence FlowPersistence, initial FlowState) *PersistentFlow {
	return flow.NewPersistentFlow(flowID, persistence, initial)
}

// NewJSONFilePersistence creates a file-based flow persistence backend.
func NewJSONFilePersistence(dir string) *flow.JSONFilePersistence {
	return flow.NewJSONFilePersistence(dir)
}

// NewTypedFlow creates a new type-safe flow with generic state management.
func NewTypedFlow[T any](initial T) *TypedFlow[T] {
	return flow.NewTypedFlow(initial)
}

// ============================================================
// Knowledge Constructors
// ============================================================

// DefaultKnowledgeConfig returns the default knowledge configuration.
func DefaultKnowledgeConfig() KnowledgeConfig {
	return knowledge.DefaultConfig()
}

// ============================================================
// LLM Constructors
// ============================================================

// NewOpenAI creates an OpenAI LLM client.
func NewOpenAI(apiKey, model string) *llm.OpenAIClient {
	return llm.NewOpenAIClient(apiKey)
}

// NewOpenRouter creates an OpenRouter LLM client.
func NewOpenRouter(apiKey, model string) *llm.OpenRouterClient {
	return llm.NewOpenRouterClient(apiKey, model)
}

// NewAnthropic creates an Anthropic/Claude LLM client.
func NewAnthropic(apiKey, model string) *llm.AnthropicClient {
	return llm.NewAnthropicClient(apiKey, model)
}

// NewGemini creates a Google Gemini LLM client.
func NewGemini(apiKey, model string) *llm.GeminiClient {
	return llm.NewGeminiClient(apiKey, model)
}

// NewGroq creates a Groq LLM client.
func NewGroq(apiKey, model string) *llm.GroqClient {
	return llm.NewGroqClient(apiKey, model)
}

// ============================================================
// Tool Constructors
// ============================================================

// NewSearchWebTool creates a web search tool.
func NewSearchWebTool() *tools.SearchWebTool {
	return tools.NewSearchWebTool()
}

// NewShellTool creates a shell execution tool.
func NewShellTool(opts ...func(*tools.ShellTool)) *tools.ShellTool {
	return tools.NewShellTool(opts...)
}

// NewCodeInterpreter creates a code execution tool.
func NewCodeInterpreter(safe bool) *tools.CodeInterpreterTool {
	return tools.NewCodeInterpreterTool(tools.WithSafeMode(safe))
}

// NewFileReadTool creates a file reading tool.
func NewFileReadTool() tools.Tool {
	return tools.NewFileReadTool()
}

// NewFileWriteTool creates a file writing tool.
func NewFileWriteTool() tools.Tool {
	return tools.NewFileWriteTool()
}

// ============================================================
// LLM Option Helpers
// ============================================================

// Float64 creates a *float64 for LLM options.
func Float64(v float64) *float64 { return llm.Float64(v) }

// Int creates a *int for LLM options.
func Int(v int) *int { return llm.Int(v) }
