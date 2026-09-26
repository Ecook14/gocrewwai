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
//
// Gocrewwai v0.9.0 — 30 core packages, 57 built-in tools, 7 LLM providers,
// 12 memory store types, 6 orchestration modes, full OTEL observability,
// MCP+A2A+WebMCP protocols, Docker+WASM sandboxing.
package gocrew

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/delegation"
	"github.com/Ecook14/gocrewwai/pkg/events"
	"github.com/Ecook14/gocrewwai/pkg/files"
	"github.com/Ecook14/gocrewwai/pkg/flow"
	"github.com/Ecook14/gocrewwai/pkg/flows"
	"github.com/Ecook14/gocrewwai/pkg/guardrails"
	"github.com/Ecook14/gocrewwai/pkg/i18n"
	"github.com/Ecook14/gocrewwai/pkg/knowledge"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/sandbox"
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
type CoreAgent = core.Agent
type Guardrail = guardrails.Guardrail

// Task Types
type Task = tasks.Task
type TaskConfig = tasks.TaskConfig

// Crew Types
type Crew = crew.Crew
type CrewConfig = crew.CrewConfig
type TrainingFeedback = crew.TrainingFeedback

// LLM Types
type LLMClient = llm.Client
type LLMMessage = llm.Message
type LLMOptions = llm.GenerateOptions

// Function calling types (bridges tools to LLM function-calling interface)
type ToolDefinition = llm.ToolDefinition
type ToolChoice = llm.ToolChoice

// Tool Types
type Tool = tools.Tool
type BaseTool = tools.BaseTool
type ArgSchema = tools.ArgSchema

// ============================================================
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
type TypedFlow[T any] = flow.TypedFlow[T]
type TypedNode[T any] = flow.TypedNode[T]
type HumanFeedbackConfig = flow.HumanFeedbackConfig
type RouterNode = flow.RouterNode
type Route = flow.Route
type Predicate = flow.Predicate

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
type DirectorySource = knowledge.DirectorySource
type IngestionEngine = knowledge.IngestionEngine

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

// NewAgent creates a new Agent using a declarative config or builder.
func NewAgent(cfg AgentConfig) *Agent {
	return agents.New(cfg)
}

// NewAgentBuilder returns a fluent agent builder. Useful for complex agent construction.
func NewAgentBuilder() *agents.AgentBuilder {
	return agents.NewAgentBuilder()
}

// NewTask creates a new Task using a declarative config or builder.
func NewTask(cfg TaskConfig) *Task {
	return tasks.New(cfg)
}

// NewTaskBuilder returns a fluent task builder. Useful for complex task construction.
func NewTaskBuilder() *tasks.TaskBuilder {
	return tasks.NewTaskBuilder()
}

// NewCrew creates a new Crew using a declarative config or builder.
func NewCrew(cfg CrewConfig) *Crew {
	return crew.New(cfg)
}

// NewCrewBuilder returns a fluent crew builder. Useful for complex crew construction.
func NewCrewBuilder() *crew.CrewBuilder {
	return crew.NewCrewBuilder()
}

// Kickoff is a convenience function that creates and immediately executes a crew.
func Kickoff(ctx context.Context, cfg CrewConfig) (interface{}, error) {
	c := crew.New(cfg)
	return c.Kickoff(ctx)
}

// GetOutput securely translates the raw interface{} Output on a task
// into a strongly typed pointer. Use this after a crew completes to
// unpack structured JSON results.
func GetOutput[T any](task *Task) *T {
	if task == nil || task.Output == nil {
		return nil
	}
	raw, ok := task.Output.([]byte)
	if ok && raw != nil {
		var t T
		if jsonErr := json.Unmarshal(raw, &t); jsonErr == nil {
			return &t
		}
	}
	rawAny, ok := task.Output.(map[string]interface{})
	if ok && rawAny != nil {
		b, _ := json.Marshal(rawAny)
		var t T
		if jsonErr := json.Unmarshal(b, &t); jsonErr == nil {
			return &t
		}
	}
	// Fallback: try direct type assertion for simple types
	var t T
	val := reflect.ValueOf(&t).Elem()
	src := reflect.ValueOf(task.Output)
	if src.Type().ConvertibleTo(val.Type()) {
		val.Set(src.Convert(val.Type()))
		return &t
	}
	return nil
}

// ============================================================
// Memory Constructors
// ============================================================

// NewMemory creates a unified memory store backed by the given store, embedder, and config.
// This is the primary entry point for memory in gocrew.
func NewMemory(store memory.Store, embedder llm.Client, cfg *memory.UnifiedMemoryConfig) *UnifiedMemory {
	if cfg == nil {
		cfg = &memory.UnifiedMemoryConfig{}
	}
	return memory.NewUnifiedMemory(store, embedder, cfg)
}

// NewLongTermMemory creates a long-term memory with the given backing store and embedder.
func NewLongTermMemory(store memory.Store, embedder llm.Client) *memory.LongTermMemory {
	return memory.NewLongTermMemory(store, embedder)
}

// NewSQLiteStore initializes a new SQLite database for persistent memory.
func NewSQLiteStore(dbPath string) (*memory.SQLiteStore, error) {
	return memory.NewSQLiteStore(dbPath)
}

// NewPDFTool creates a PDF reading tool with an optional chroot directory.
// Pass an empty string for chroot to allow reading any file.
func NewPDFTool(chroot ...string) tools.Tool {
	path := ""
	if len(chroot) > 0 {
		path = chroot[0]
	}
	return tools.NewPDFReadTool(path)
}

// NewChroot creates a chrooted file reader for the given workspace directory.
// This is equivalent to NewFileReadTool with the chroot path.
func NewChroot(workspace string) tools.Tool {
	return tools.NewFileReadTool(workspace)
}

// NewI18N creates an internationalization handler for the given language.
func NewI18N(lang string) (*i18n.I18N, error) {
	return i18n.NewI18N(lang)
}

// ============================================================
// Crew Option Wrappers (from pkg/crew)
// ============================================================

// WithProcess sets the crew orchestration process type.
func WithProcess(p crew.ProcessType) crew.CrewOption {
	return crew.WithProcess(p)
}

// WithManager sets the manager agent for hierarchical crews.
func WithManager(m core.Agent) crew.CrewOption {
	return crew.WithManager(m)
}

// ============================================================
// Sandbox Helpers
// ============================================================
func Remember(m *UnifiedMemory, ctx context.Context, text string) {
	m.Remember(ctx, text, nil)
}

// Recall retrieves relevant memories by query.
func Recall(m *UnifiedMemory, ctx context.Context, query string, opts ...RecallOptions) []ScoredMemory {
	scored, _ := m.Recall(ctx, query, nil)
	return scored
}

// Forget removes a memory by ID from the store.
func Forget(m *UnifiedMemory, ctx context.Context, id string) error {
	return m.Forget(ctx, id)
}

// ============================================================
// File Constructors
// ============================================================

// ImageFile creates a file handle for an image (path or URL).
func ImageFile(source string, mode ...FileMode) File {
	return files.ImageFile(source, mode...)
}

// PDFFile creates a file handle for a PDF.
func PDFFile(source string, mode ...FileMode) File {
	return files.PDFFile(source, mode...)
}

// AudioFile creates a file handle for audio content.
func AudioFile(source string, mode ...FileMode) File {
	return files.AudioFile(source, mode...)
}

// VideoFile creates a file handle for video content.
func VideoFile(source string, mode ...FileMode) File {
	return files.VideoFile(source, mode...)
}

// TextFile creates a file handle for text content.
func TextFile(source string, mode ...FileMode) File {
	return files.TextFile(source, mode...)
}

// NewFile auto-detects file type from extension.
func NewFile(source string, mode ...FileMode) File {
	return files.NewFile(source, mode...)
}

// FromBytes creates a file from raw bytes.
func FromBytes(fb FileBytes, mode ...FileMode) File {
	return files.FromBytes(fb, mode...)
}

// ValidateFile checks if a file is compatible with a provider.
func ValidateFile(file File, provider string) error {
	return files.ValidateFile(file, provider)
}

// ============================================================
// Flow Constructors
// ============================================================

// NewFlow creates a new workflow orchestration flow.
func NewFlow(initialState FlowState) *Flow {
	return flow.NewFlow(initialState)
}

// NewPersistentFlow creates a flow that auto-persists state after each node.
func NewPersistentFlow(flowID string, persistence FlowPersistence, initial FlowState) *PersistentFlow {
	if persistence == nil {
		return flow.NewPersistentFlow(flowID, flow.NewJSONFilePersistence("."), initial)
	}
	return flow.NewPersistentFlow(flowID, persistence, initial)
}

// AddNode adds a node to a Flow (convenience wrapper).
func AddNode(f *Flow, n FlowNode) {
	f.AddNode(n)
}

// AddParallelNodes executes nodes concurrently and merges their state outputs.
// Each node runs in its own goroutine; all results (including errors) are collected.
// If any node fails, the merged state from successful nodes is returned along with
// a combined error listing all failures.
func AddParallelNodes(f *Flow, nodes []FlowNode) {
	merged := func(ctx context.Context, state flow.State) (flow.State, error) {
		if len(nodes) == 0 {
			return state, nil
		}
		type result struct {
			state flow.State
			err   error
		}
		results := make([]result, len(nodes))
		var wg sync.WaitGroup
		for i, n := range nodes {
			wg.Add(1)
			go func(idx int, node FlowNode) {
				defer wg.Done()
				s, err := node(ctx, state)
				results[idx] = result{state: s, err: err}
			}(i, n)
		}
		wg.Wait()
		merged := make(flow.State)
		var errs []error
		for _, r := range results {
			if r.err != nil {
				errs = append(errs, r.err)
			} else {
				for k, v := range r.state {
					merged[k] = v
				}
			}
		}
		if len(errs) > 0 {
			return merged, fmt.Errorf("parallel nodes failed: %v", errs)
		}
		return merged, nil
	}
	f.AddNode(merged)
}

// AddRouter adds a router node to a Flow (convenience wrapper).
// The conditionFn selects which branch to execute based on state content.
func AddRouter(f *Flow, conditionFn func(state flow.State) string, branches map[string]FlowNode) {
	router := func(ctx context.Context, state flow.State) (flow.State, error) {
		branchName := conditionFn(state)
		node, ok := branches[branchName]
		if !ok {
			return state, fmt.Errorf("no branch found for '%s'", branchName)
		}
		return node(ctx, state)
	}
	f.AddNode(router)
}

// SetPersistence configures persistence for a PersistentFlow (convenience wrapper).
// Only works on flows created with NewPersistentFlow.
func SetPersistence(f *PersistentFlow, persistence FlowPersistence) *PersistentFlow {
	f.SetPersistence(persistence)
	return f
}

// NewJSONFilePersistence creates a file-based flow persistence backend.
func NewJSONFilePersistence(dir string) *flow.JSONFilePersistence {
	return flow.NewJSONFilePersistence(dir)
}

// NewTypedFlow creates a new type-safe flow with generic state management.
func NewTypedFlow[T any](initial T) *TypedFlow[T] {
	return flow.NewTypedFlow(initial)
}

// DefaultKnowledgeConfig returns the default knowledge configuration.
func DefaultKnowledgeConfig() KnowledgeConfig {
	return knowledge.DefaultConfig()
}

// NewPDFSource creates a PDF knowledge source from one or more file paths.
func NewPDFSource(filePaths ...string) *PDFSource {
	return knowledge.NewPDFSource(filePaths...)
}

// NewURLSource creates a URL knowledge source from one or more URLs.
func NewURLSource(urls ...string) *URLSource {
	return knowledge.NewURLSource(urls...)
}

// NewTextSource creates a text knowledge source from raw string content.
func NewTextSource(content string, label string) *StringSource {
	return knowledge.NewTextSource(content, label)
}

// NewDirectorySource creates a directory knowledge source that scans files matching a pattern.
func NewDirectorySource(path string, pattern string) *DirectorySource {
	return knowledge.NewDirectorySource(path, pattern)
}

// NewCSVSource creates a CSV knowledge source from one or more file paths.
func NewCSVSource(filePaths ...string) *CSVSource {
	return knowledge.NewCSVSource(filePaths...)
}

// NewJSONSource creates a JSON knowledge source from one or more file paths.
func NewJSONSource(filePaths ...string) *JSONSource {
	return knowledge.NewJSONSource(filePaths...)
}

// ============================================================
// LLM Constructors
// ============================================================

// NewOpenAI creates an OpenAI LLM client.
func NewOpenAI(apiKey, model string) *llm.OpenAIClient {
	client := llm.NewOpenAIClient(apiKey)
	if model != "" {
		client.Model = model
	}
	return client
}

// NewAnthropic creates an Anthropic LLM client.
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

// NewOpenRouter creates an OpenRouter LLM client.
func NewOpenRouter(apiKey, model string) *llm.OpenRouterClient {
	return llm.NewOpenRouterClient(apiKey, model)
}

// NewOllama creates an Ollama local LLM client.
func NewOllama(model string, baseURL ...string) *llm.OllamaClient {
	return llm.NewOllamaClient(model, baseURL...)
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

// NewCodeInterpreterTool creates a code execution tool with default options.
func NewCodeInterpreterTool() *tools.CodeInterpreterTool {
	return tools.NewCodeInterpreterTool()
}

// NewFileReadTool creates a file reading tool.
func NewFileReadTool(chroot ...string) tools.Tool {
	path := ""
	if len(chroot) > 0 {
		path = chroot[0]
	}
	return tools.NewFileReadTool(path)
}

// NewFileWriteTool creates a file writing tool.
func NewFileWriteTool(chroot ...string) tools.Tool {
	path := ""
	if len(chroot) > 0 {
		path = chroot[0]
	}
	return tools.NewFileWriteTool(path)
}

// NewFileEditTool creates a file editing tool.
func NewFileEditTool(chroot ...string) tools.Tool {
	path := ""
	if len(chroot) > 0 {
		path = chroot[0]
	}
	return tools.NewFileEditTool(path)
}

// NewDirectoryTool creates a directory traversal tool.
func NewDirectoryTool(root string, depth int, allowAbs bool) tools.Tool {
	return tools.NewDirectoryTool(root, depth, allowAbs)
}

// NewAskHumanTool creates a tool for requesting human approval.
func NewAskHumanTool(enabled bool, opts ...func(*tools.AskHumanTool)) tools.Tool {
	return tools.NewAskHumanTool(enabled, opts...)
}

// NewExaTool creates an Exa AI search tool.
func NewExaTool(apiKey string) tools.Tool {
	return tools.NewExaTool(apiKey)
}

// NewArxivTool creates an Arxiv search tool.
func NewArxivTool() tools.Tool {
	return tools.NewArxivTool()
}

// NewWikipediaTool creates a Wikipedia search tool.
func NewWikipediaTool() tools.Tool {
	return tools.NewWikipediaTool()
}

// NewSerperTool creates a Serper AI search tool.
func NewSerperTool(apiKey string) tools.Tool {
	return tools.NewSerperTool(apiKey)
}

// NewScraperTool creates a web scraper tool.
func NewScraperTool() tools.Tool {
	return tools.NewScraperTool()
}

// NewDiscordTool creates a Discord integration tool.
func NewDiscordTool(botToken, channelID string) *tools.DiscordTool {
	return tools.NewDiscordTool(botToken, channelID)
}

// NewGitHubTool creates a GitHub integration tool.
func NewGitHubTool(token string) *tools.GitHubTool {
	return tools.NewGitHubTool(token)
}

// NewSlackTool creates a Slack integration tool.
func NewSlackTool(token string) *tools.SlackTool {
	return tools.NewSlackTool(token)
}

// NewBrowserTool creates an automated browser navigation tool.
func NewBrowserTool() tools.Tool {
	return tools.NewBrowserTool()
}

// NewAskQuestionTool creates a tool that pauses for a human answer.
func NewAskQuestionTool() *tools.AskQuestionTool {
	return tools.NewAskQuestionTool()
}

// NewCodeSandboxTool creates an isolated code execution sandbox tool.
func NewCodeSandboxTool() *tools.CodeSandboxTool {
	return tools.NewCodeSandboxTool()
}

// NewDateTimeTool creates a date/time utility tool.
func NewDateTimeTool() *tools.DateTimeTool {
	return tools.NewDateTimeTool()
}

// NewDelegateWorkTool creates a delegation tool for the given coworkers.
func NewDelegateWorkTool(coworkers []CoreAgent) *delegation.DelegateWorkTool {
	return delegation.NewDelegateWorkTool(coworkers)
}

// NewElasticsearchTool creates an Elasticsearch query tool.
func NewElasticsearchTool(baseURL string, opts ...func(*tools.ElasticsearchTool)) *tools.ElasticsearchTool {
	return tools.NewElasticsearchTool(baseURL, opts...)
}

// NewFailoverClient creates an LLM client that falls back to secondary.
func NewFailoverClient(primary, secondary llm.Client) *llm.FailoverClient {
	return llm.NewFailoverClient(primary, secondary, nil)
}

// NewGoogleSheetsTool creates a Google Sheets integration tool.
func NewGoogleSheetsTool(token string) *tools.GoogleSheetsTool {
	return tools.NewGoogleSheetsTool(token)
}

// NewHubSpotTool creates a HubSpot CRM integration tool.
func NewHubSpotTool(token string) *tools.HubSpotTool {
	return tools.NewHubSpotTool(token)
}

// NewJiraTool creates a Jira integration tool.
func NewJiraTool(baseURL, email, token string) *tools.JiraTool {
	return tools.NewJiraTool(baseURL, email, token)
}

// NewJSONParseTool creates a JSON file parsing tool.
func NewJSONParseTool(chroot string) *tools.JSONParseTool {
	return tools.NewJSONParseTool(chroot)
}

// NewJSONTool creates a JSON utility tool.
func NewJSONTool() *tools.JSONTool {
	return tools.NewJSONTool()
}

// NewLinearTool creates a Linear integration tool.
func NewLinearTool(token string) *tools.LinearTool {
	return tools.NewLinearTool(token)
}

// NewMongoDBTool creates a MongoDB integration tool.
func NewMongoDBTool(endpoint, apiKey, dataSource, database string) *tools.MongoDBTool {
	return tools.NewMongoDBTool(endpoint, apiKey, dataSource, database)
}

// NewMySQLTool creates a MySQL integration tool.
func NewMySQLTool(dsn string) (*tools.MySQLTool, error) {
	return tools.NewMySQLTool(dsn)
}

// NewNotionTool creates a Notion integration tool.
func NewNotionTool(token string) *tools.NotionTool {
	return tools.NewNotionTool(token)
}

// NewPostgresTool creates a Postgres integration tool.
func NewPostgresTool(connStr string) (*tools.PostgresTool, error) {
	return tools.NewPostgresTool(connStr)
}

// NewRegexTool creates a regex utility tool.
func NewRegexTool() *tools.RegexTool {
	return tools.NewRegexTool()
}

// NewS3Tool creates an S3-compatible object storage tool.
func NewS3Tool(endpoint, accessKey, secretKey, region string) *tools.S3Tool {
	return tools.NewS3Tool(endpoint, accessKey, secretKey, region)
}

// NewScrapeWebsiteTool creates a simple website text scraping tool.
func NewScrapeWebsiteTool() *tools.ScrapeWebsiteTool {
	return tools.NewScrapeWebsiteTool()
}

// NewSendGridTool creates a SendGrid email tool.
func NewSendGridTool(apiKey string) *tools.SendGridTool {
	return tools.NewSendGridTool(apiKey)
}

// NewSQLiteTool creates a SQLite database tool.
func NewSQLiteTool(dbPath string) (*tools.SQLiteTool, error) {
	return tools.NewSQLiteTool(dbPath)
}

// NewSupabaseTool creates a Supabase integration tool.
func NewSupabaseTool(url, apiKey, table string) *tools.SupabaseTool {
	return tools.NewSupabaseTool(url, apiKey, table)
}

// NewTavilyTool creates a Tavily search tool.
func NewTavilyTool(apiKey string) *tools.TavilyTool {
	return tools.NewTavilyTool(apiKey)
}

// NewTwilioTool creates a Twilio SMS tool.
func NewTwilioTool(accountSID, authToken string) *tools.TwilioTool {
	return tools.NewTwilioTool(accountSID, authToken)
}

// NewWASMSandboxTool creates a WASM sandbox execution tool.
func NewWASMSandboxTool(ctx context.Context) *tools.WASMSandboxTool {
	return tools.NewWASMSandboxTool(ctx)
}

// NewBraveTool creates a Brave Search tool.
func NewBraveTool(apiKey string) *tools.BraveSearchTool {
	return tools.NewBraveSearchTool(apiKey)
}

// NewCSVTool creates a CSV file reading tool.
func NewCSVTool(chroot string) *tools.CSVReadTool {
	return tools.NewCSVReadTool(chroot)
}

// NewExcelTool creates an Excel file reading tool.
func NewExcelTool(chroot string) *tools.ExcelReadTool {
	return tools.NewExcelReadTool(chroot)
}

// NewHTMLTool creates an HTML file reading tool.
func NewHTMLTool(chroot string) *tools.HTMLReadTool {
	return tools.NewHTMLReadTool(chroot)
}

// NewXMLTool creates an XML file reading tool.
func NewXMLTool(chroot string) *tools.XMLReadTool {
	return tools.NewXMLReadTool(chroot)
}

// NewYamlTool creates a YAML file reading tool.
func NewYamlTool(chroot string) *tools.YAMLReadTool {
	return tools.NewYAMLReadTool(chroot)
}

// NewHTTPClientTool creates an HTTP client tool with SSRF protection.
func NewHTTPClientTool(opts ...func(*tools.HTTPTool)) *tools.HTTPTool {
	return tools.NewHTTPTool(opts...)
}

// NewRagTool creates a native RAG search tool.
func NewRagTool(mem *memory.LongTermMemory, dir string) *tools.NativeRAGTool {
	return tools.NewNativeRAGTool(mem, dir)
}

// NewPDFFile creates a PDF text extraction tool for the given scope.
func NewPDFFile(source string) *tools.PDFReadTool {
	return tools.NewPDFReadTool(source)
}

// NewDockerSandbox creates a Docker sandbox tool that executes code in an isolated container.
// Pass an image name (e.g. "python:3.11-slim") and safe=true for security-hardened execution.
func NewDockerSandbox(image string, safe bool) (Tool, error) {
	t, err := sandbox.NewDockerSandbox(image, safe)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// NewRedisCache creates a Redis-backed LLM cache.
func NewRedisCache(addr, password string, db int, ttl time.Duration) (*llm.RedisCache, error) {
	return llm.NewRedisCache(addr, password, db, ttl)
}

// NewFileCache creates a file-backed LLM cache in dir.
func NewFileCache(dir string) *llm.FileCache {
	return llm.NewFileCache(dir)
}

// SandboxConfig holds Docker sandbox configuration.
type SandboxConfig struct {
	Image   string
	Timeout time.Duration
	Safe    bool
}

// WithSafeMode returns a SandboxConfig with security-hardening enabled.
func WithSafeMode(cfg SandboxConfig, safe bool) SandboxConfig {
	cfg.Safe = safe
	return cfg
}

// NewCalculatorTool creates a new calculator tool.
func NewCalculatorTool() *tools.CalculatorTool {
	return tools.NewCalculatorTool()
}

// ============================================================
// Guardrail Constructors
// ============================================================

// NewHumanReviewGuardrail creates a human-in-the-loop guardrail.
func NewHumanReviewGuardrail(agentRole, toolName string) *guardrails.HumanReviewGuardrail {
	return guardrails.NewHumanReviewGuardrail(agentRole, toolName)
}

// NewMaxTokenGuardrail rejects outputs exceeding a specified word count (1 word ≈ 1 token).
func NewMaxTokenGuardrail(maxTokens int) *guardrails.MaxTokenGuardrail {
	return guardrails.NewMaxTokenGuardrail(maxTokens)
}

// NewContentFilterGuardrail blocks outputs matching any of the given regex patterns.
func NewContentFilterGuardrail(patterns ...string) (*guardrails.ContentFilterGuardrail, error) {
	return guardrails.NewContentFilterGuardrail(patterns)
}

// NewSchemaGuardrail validates that the output is valid JSON matching the provided schema struct.
func NewSchemaGuardrail(schema interface{}) *guardrails.SchemaGuardrail {
	return guardrails.NewSchemaGuardrail(schema)
}

// NewPIIRedactionGuardrail detects and rejects outputs containing PII (email addresses, SSNs).
func NewPIIRedactionGuardrail() *guardrails.PIIRedactionGuardrail {
	return guardrails.NewPIIRedactionGuardrail()
}

// NewToxicityGuardrail blocks outputs containing toxic words from the default list.
func NewToxicityGuardrail() *guardrails.ToxicityGuardrail {
	return guardrails.NewToxicityGuardrail()
}

// NewValidatorGuardrail creates a custom guardrail from a validation function.
func NewValidatorGuardrail(name string, fn guardrails.ValidatorFunc) *guardrails.ValidatorGuardrail {
	return guardrails.NewValidatorGuardrail(name, fn)
}

// NoEmptyString returns a validator that rejects empty or whitespace-only output.
func NoEmptyString() guardrails.ValidatorFunc {
	return guardrails.NoEmptyString()
}

// MinLength returns a validator that rejects output shorter than n characters.
func MinLength(n int) guardrails.ValidatorFunc {
	return guardrails.MinLength(n)
}

// MaxLength returns a validator that rejects output longer than n characters.
func MaxLength(n int) guardrails.ValidatorFunc {
	return guardrails.MaxLength(n)
}

// RunAll executes all guardrails against the given output. Returns the first error encountered.
func RunAll(gs []Guardrail, output string) error {
	return guardrails.RunAll(gs, output)
}

// AllViolations returns every guardrail violation as a slice of strings.
// Use this when you want to collect all failures instead of stopping at the first one.
func AllViolations(gs []Guardrail, output string) []string {
	return guardrails.AllViolations(gs, output)
}

// ============================================================
// Vector Store Constructors
// ============================================================

// NewPineconeStore securely wraps a Pinecone Vector DB initialization.
func NewPineconeStore(host, apiKey, namespace string) (*memory.PineconeStore, error) {
	return memory.NewPineconeStore(host, apiKey, namespace)
}

// NewRedisStore creates a Redis-backed memory store.
func NewRedisStore(addrs []string, password string, db int, prefix string) (*memory.RedisStore, error) {
	return memory.NewRedisStore(addrs, password, db, prefix)
}

// NewInMemCosineStore creates an in-memory cosine similarity store.
func NewInMemCosineStore() *memory.InMemCosineStore {
	return memory.NewInMemCosineStore()
}

// SplitterConfig alias for knowledge module configuration.
type SplitterConfig = knowledge.SplitterConfig

// NewSemanticSplitter safely instantiates an advanced token/semantic text splitter.
func NewSemanticSplitter(cfg SplitterConfig) *knowledge.SemanticSplitter {
	return knowledge.NewSemanticSplitter(cfg)
}

// NewSQLiteCheckpointer configures a local filesystem persist store for Flow checkpoints.
func NewSQLiteCheckpointer(dir string) (*flows.CheckpointManager, error) {
	return flows.NewCheckpointManager(dir)
}

// ============================================================
// Testing Constructors
// ============================================================

// NewCrewTest creates a multi-run test harness for evaluating crew or flow performance.
// The judge LLM is required for scoring outputs.
func NewCrewTest(judgeLLM LLMClient) *testing.CrewTest {
	return testing.NewCrewTest(judgeLLM)
}

// ScoreThresholdLinter checks that all results meet the pass threshold.
func ScoreThresholdLinter(suite *PerformanceSuite, threshold int) string {
	return testing.ScoreThresholdLinter(suite, threshold)
}

// ScoreSummary returns a one-line summary of the suite.
func ScoreSummary(suite *PerformanceSuite) string {
	return testing.ScoreSummary(suite)
}

// TraceCompareAll runs the trace comparison function across all adjacent pairs of results.
func TraceCompareAll(suite *PerformanceSuite, cmp func(r1, r2 TestResult) bool) bool {
	return testing.TraceCompareAll(suite, cmp)
}

// NewTestCrewConfig creates a minimal CrewConfig suitable for testing.
// It uses the provided agent and task, with default sequential processing.
func NewTestCrewConfig(agent core.Agent, task *tasks.Task) CrewConfig {
	return CrewConfig{
		Agents:  []core.Agent{agent},
		Tasks:   []*tasks.Task{task},
		Process: Sequential,
	}
}

// ============================================================
// LLM Option Helpers
// ============================================================

// Float64 creates a *float64 for LLM options.
func Float64(v float64) *float64 {
	return llm.Float64(v)
}

// Int creates a *int for LLM options.
func Int(v int) *int {
	return llm.Int(v)
}

// ToToolDef converts a tools.Tool to an llm.ToolDefinition for function calling.
func ToToolDef(t tools.Tool) llm.ToolDefinition {
	schema, _ := json.Marshal(t.ArgsSchema())
	return llm.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  schema,
	}
}

// ToToolDefs converts a slice of tools to llm.ToolDefinition structs for function calling.
func ToToolDefs(toolsSlice []tools.Tool) []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, len(toolsSlice))
	for i, t := range toolsSlice {
		defs[i] = ToToolDef(t)
	}
	return defs
}
