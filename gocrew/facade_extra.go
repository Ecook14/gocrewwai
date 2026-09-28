// Package gocrew — extended facade coverage.
//
// This file closes the facade gaps vs the pkg/ surface: telemetry,
// protocols, config, crew checkpointing/visualization/async, LLM middleware
// and usage, remaining memory backends, knowledge/guardrail stragglers, the
// flows DAG engine, and the delegation AskQuestion variant. Thin wrappers
// only — no new behavior.
package gocrew

import (
	"context"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/auth"
	"github.com/Ecook14/gocrewwai/pkg/config"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/delegation"
	"github.com/Ecook14/gocrewwai/pkg/flows"
	"github.com/Ecook14/gocrewwai/pkg/guardrails"
	"github.com/Ecook14/gocrewwai/pkg/i18n"
	"github.com/Ecook14/gocrewwai/pkg/knowledge"
	"github.com/Ecook14/gocrewwai/pkg/kv"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/protocols"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/Ecook14/gocrewwai/pkg/webhook"
)

// ============================================================
// Telemetry / Observability (pkg/telemetry)
// ============================================================

type TelemetryConfig = telemetry.TelemetryConfig
type Metrics = telemetry.Metrics

// InitTelemetry initializes OpenTelemetry tracing + optional Prometheus metrics.
func InitTelemetry(cfg telemetry.TelemetryConfig) error {
	_, err := telemetry.InitTelemetry(cfg)
	return err
}

// DefaultTelemetryConfig returns stderr-export defaults.
func DefaultTelemetryConfig() telemetry.TelemetryConfig {
	return telemetry.DefaultTelemetryConfig()
}

// StartSpan starts a named span (no-op unless telemetry is enabled).
func StartSpan(ctx context.Context, name string) (context.Context, func()) {
	ctx, span := telemetry.StartSpan(ctx, name)
	if span == nil {
		return ctx, func() {}
	}
	return ctx, func() { span.End() }
}

// NewMetrics creates a fresh metrics registry.
func NewMetrics() *telemetry.Metrics {
	return telemetry.NewMetrics()
}

// NewJWTValidator creates an HS256 JWT validator for Bearer auth.
// Wire JWT_SECRET env to enable it on pkg/server, pkg/api, pkg/dashboard.
func NewJWTValidator(secret string) *auth.Validator {
	return auth.NewValidator(secret)
}

// KVConfigFromEnv builds the shared Redis-protocol config
// (REDIS_BACKEND/ADDR/PASSWORD/DB/POOL_SIZE). See docs/ops/dragonfly.md.
func KVConfigFromEnv() (kv.Config, error) {
	return kv.ConfigFromEnv()
}

// NewWebhookNotifier creates a signed outbound webhook notifier.
// The URL is SSRF-gated; empty secret sends unsigned.
func NewWebhookNotifier(url, secret string) (*webhook.Notifier, error) {
	return webhook.NewNotifier(url, secret)
}

// VerifyWebhookSignature checks an inbound webhook signature header.
func VerifyWebhookSignature(secret string, body []byte, signature, timestamp string, skew time.Duration) error {
	return webhook.VerifySignature(secret, body, signature, timestamp, skew)
}

// ============================================================
// Protocols: MCP + A2A (pkg/protocols)
// ============================================================

// NewMCPClient connects to an MCP server over HTTP. The URL is used as a
// transport endpoint only — SSRF-gated by utils.ValidateURL at the server
// boundary; validate untrusted URLs before registering.
func NewMCPClient(serverURL string) *protocols.MCPClient {
	return protocols.NewMCPClient(serverURL)
}

// NewA2AClient creates an A2A client bound to an auth token.
func NewA2AClient(authToken string) *protocols.A2AClient {
	return protocols.NewA2AClient(authToken)
}

// NewAgentCardBuilder starts an AgentCard builder for A2A discovery.
func NewAgentCardBuilder() *protocols.AgentCardBuilder {
	return protocols.NewAgentCardBuilder()
}

// ============================================================
// Config (pkg/config)
// ============================================================

type CrewConfigFile = config.Config

// LoadConfigFile reads crew config (JSON) from path.
func LoadConfigFile(path string) (*config.Config, error) {
	return config.LoadConfigFile(path)
}

// GetConfiguredClient resolves an LLM client by model name via the registry.
func GetConfiguredClient(modelName string) (llm.Client, error) {
	return config.GetClient(modelName)
}

// ============================================================
// Crew checkpointing / visualization / async (pkg/crew)
// ============================================================

type CrewCheckpointConfig = crew.SQLiteCheckpointConfig
type CrewRedisCheckpointConfig = crew.RedisCheckpointConfig

// NewCrewCheckpointManager creates the file-based crew checkpoint manager.
func NewCrewCheckpointManager(baseDir string) *crew.CheckpointManager {
	return crew.NewCheckpointManager(baseDir)
}

// NewCrewSQLiteCheckpointStore creates the SQLite crew checkpoint store.
func NewCrewSQLiteCheckpointStore(cfg crew.SQLiteCheckpointConfig) (*crew.SQLiteCheckpointStore, error) {
	return crew.NewSQLiteCheckpointStore(cfg)
}

// NewCrewRedisCheckpointStore creates the Redis crew checkpoint store.
func NewCrewRedisCheckpointStore(cfg crew.RedisCheckpointConfig) (*crew.RedisCheckpointStore, error) {
	return crew.NewRedisCheckpointStore(cfg)
}

// CrewToDOT renders the crew graph in Graphviz DOT format.
func CrewToDOT(c *Crew) string {
	return crew.GenerateDOT(c)
}

// CrewToMermaid renders the crew as a Mermaid flowchart.
func CrewToMermaid(c *Crew) string {
	return crew.GenerateMermaidFlow(c)
}

// CrewTaskDependencyDOT renders task dependencies in DOT format.
func CrewTaskDependencyDOT(c *Crew) string {
	return crew.GenerateTaskDependencyDOT(c)
}

// RecordCrewFeedback captures human feedback for a crew run (training input).
func RecordCrewFeedback(ctx context.Context, crewID string, feedback string) error {
	return crew.RecordFeedback(ctx, crewID, feedback)
}

// ============================================================
// LLM middleware, cache helpers, usage, tracing (pkg/llm)
// ============================================================

type LLMMiddlewareOption = llm.MiddlewareOption

// WrapLLMClient decorates a client with middleware (rate limit, timeout,
// logging, retries, cache, circuit breaker).
func WrapLLMClient(inner llm.Client, opts ...llm.MiddlewareOption) *llm.MiddlewareClient {
	return llm.WrapClient(inner, opts...)
}

func LLMWithRateLimit(maxRequests int, window time.Duration) llm.MiddlewareOption {
	return llm.WithRateLimit(maxRequests, window)
}

func LLMWithTimeout(timeout time.Duration) llm.MiddlewareOption {
	return llm.WithTimeout(timeout)
}

func LLMWithMaxRetries(n int) llm.MiddlewareOption {
	return llm.WithMaxRetries(n)
}

func LLMWithCircuitBreaker(threshold int, cooldown time.Duration) llm.MiddlewareOption {
	return llm.WithCircuitBreaker(threshold, cooldown)
}

func LLMWithCache(cache llm.Cache) llm.MiddlewareOption {
	return llm.WithCache(cache)
}

// NewLLMUsageTracker creates a token-usage tracker.
func NewLLMUsageTracker() *llm.UsageTracker {
	return llm.NewUsageTracker()
}

// NewTracedLLMClient wraps a client with OpenTelemetry spans (no prompt leak).
func NewTracedLLMClient(name string, client llm.Client) *llm.TracedClient {
	return llm.NewTracedClient(name, client)
}

// WithDefaultLLMOptions injects default temperature/max-tokens/stop/model.
func WithDefaultLLMOptions(client llm.Client, defaults llm.GenerateOptions) *llm.DefaultsClient {
	return llm.WithDefaults(client, defaults)
}

// ============================================================
// Memory backends (pkg/memory)
// ============================================================

// NewChromaStore connects to a ChromaDB collection.
func NewChromaStore(baseURL, collectionName string, timeout time.Duration) (*memory.ChromaStore, error) {
	return memory.NewChromaStore(baseURL, collectionName, timeout)
}

// NewQdrantStore connects to a Qdrant collection.
func NewQdrantStore(baseURL, collectionName string, vectorSize int) (*memory.QdrantStore, error) {
	return memory.NewQdrantStore(baseURL, collectionName, vectorSize)
}

// NewWeaviateStore connects to a Weaviate class.
func NewWeaviateStore(host, className, apiKey string) (*memory.WeaviateStore, error) {
	return memory.NewWeaviateStore(host, className, apiKey)
}

// NewConversationStore creates an in-memory conversation memory.
func NewConversationStore() *memory.InMemConversationStore {
	return memory.NewInMemConversationStore()
}

// NewEntityStore creates an in-memory entity memory.
func NewEntityStore() *memory.InMemEntityStore {
	return memory.NewInMemEntityStore()
}

// NewRemoteKnowledgeSource queries a remote mesh knowledge collection.
func NewRemoteKnowledgeSource(address, collection string, k int) *memory.RemoteKnowledgeSource {
	return memory.NewRemoteKnowledgeSource(address, collection, k)
}

// RememberMany stores multiple memories with intra-batch dedup.
func RememberMany(ctx context.Context, mem *memory.UnifiedMemory, contents []string) {
	mem.RememberMany(ctx, contents)
}

// ============================================================
// Knowledge + guardrail stragglers
// ============================================================

// NewKnowledgeSearchTool builds retrieval over a memory store.
func NewKnowledgeSearchTool(store memory.Store, client llm.Client) *knowledge.KnowledgeSearchTool {
	return knowledge.NewKnowledgeSearchTool(store, client)
}

// NewTokenSplitter splits text into token-bounded chunks.
func NewTokenSplitter(chunkSize, chunkOverlap int) *knowledge.TokenSplitter {
	return knowledge.NewTokenSplitter(chunkSize, chunkOverlap)
}

// NewJSONGuardrail validates outputs against a JSON schema.
func NewJSONGuardrail() *guardrails.JSONValidGuardrail {
	return guardrails.NewJSONValidator()
}

// NewXMLGuardrail validates well-formed XML, optionally enforcing the root.
func NewXMLGuardrail(root ...string) *guardrails.XMLValidGuardrail {
	return guardrails.NewXMLValidator(root...)
}

// NewCSVGuardrail validates CSV shape, optionally enforcing column count.
func NewCSVGuardrail(columns ...int) *guardrails.CSVValidGuardrail {
	return guardrails.NewCSVValidator(columns...)
}

// NewRegexGuardrail validates output against a compiled pattern.
func NewRegexGuardrail(pattern string) (*guardrails.RegexGuardrail, error) {
	return guardrails.NewRegexValidator(pattern)
}

// DefaultInputSanitizer returns the default prompt-injection sanitizer.
func DefaultInputSanitizer() *guardrails.Sanitizer {
	return guardrails.DefaultSanitizer()
}

// NewLLMReviewGuardrail validates outputs with a secondary LLM.
// The reviewer must implement Review(ctx, prompt) (string, error); pass a
// *llm-backed adapter or test double.
func NewLLMReviewGuardrail(reviewer guardrails.Reviewer, criteria string) *guardrails.LLMReviewGuardrail {
	return &guardrails.LLMReviewGuardrail{Reviewer: reviewer, Criteria: criteria}
}

// ============================================================
// Flows DAG engine (pkg/flows)
// ============================================================

type GraphFlow = flows.Flow
type GraphFlowNode = flows.FlowNode
type GraphFlowState = flows.State
type GraphNodeType = flows.NodeType

// NewGraphFlow creates a DAG-engine flow (router/parallel/map/reduce).
func NewGraphFlow(id string, initial flows.State) *flows.Flow {
	return flows.NewFlow(id, "", initial)
}

// NewFlowEngine creates the DAG execution engine.
func NewFlowEngine(checkpointer *flows.CheckpointManager) *flows.Engine {
	return &flows.Engine{Checkpointer: checkpointer}
}

// NewDelegationAskQuestionTool creates the crew-delegation question tool for
// the given coworkers. (NewAskQuestionTool is the human-pause variant from
// pkg/tools; this is the pkg/delegation coworker variant.)
func NewDelegationAskQuestionTool(coworkers []CoreAgent) *delegation.AskQuestionTool {
	return delegation.NewAskQuestionTool(coworkers)
}

// SupportedLanguages lists embedded i18n languages.
func SupportedLanguages() []string {
	return i18n.SupportedLanguages()
}
