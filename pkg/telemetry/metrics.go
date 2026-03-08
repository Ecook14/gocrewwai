package telemetry

import (
	"net/http"
	"sync"
	"time"
	"fmt"
)

// ---------------------------------------------------------------------------
// Prometheus-Compatible Metrics (Zero External Dependencies)
// ---------------------------------------------------------------------------
//
// This is a lightweight, self-contained metrics system that exposes a
// /metrics endpoint compatible with Prometheus scraping. No dependency
// on the Prometheus Go client library is required.
//
// Usage:
//
//	metrics := telemetry.NewMetrics()
//	metrics.RecordLLMCall("gpt-4o", 150*time.Millisecond, nil)
//	metrics.RecordTokens("gpt-4o", 100, 50)
//	metrics.RecordTaskExecution("research", 2*time.Second, nil)
//	http.Handle("/metrics", metrics.Handler())

// Metrics collects operational data for Crew-GO agents.
type Metrics struct {
	mu sync.RWMutex

	// LLM Metrics
	llmCallsTotal    map[string]int64         // model → count
	llmCallErrors    map[string]int64         // model → error count
	llmLatencySum    map[string]float64       // model → total seconds
	llmLatencyBucket map[string]map[float64]int64 // model → bucket → count

	// Token Metrics
	promptTokensTotal     map[string]int64 // model → total prompt tokens
	completionTokensTotal map[string]int64 // model → total completion tokens
	costTotal             map[string]float64 // model → total cost USD

	// Task Metrics
	taskExecutionsTotal map[string]int64   // task_type → count
	taskErrors          map[string]int64   // task_type → error count
	taskLatencySum      map[string]float64 // task_type → total seconds

	// Agent Metrics
	activeAgents   int64
	agentsCreated  int64

	// Tool Metrics
	toolCallsTotal map[string]int64   // tool_name → count
	toolErrors     map[string]int64   // tool_name → error count
	toolLatencySum map[string]float64 // tool_name → total seconds

	// System
	startTime time.Time
}

// Standard histogram buckets for latency (seconds)
var defaultBuckets = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}

// NewMetrics creates a new metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		llmCallsTotal:        make(map[string]int64),
		llmCallErrors:        make(map[string]int64),
		llmLatencySum:        make(map[string]float64),
		llmLatencyBucket:     make(map[string]map[float64]int64),
		promptTokensTotal:    make(map[string]int64),
		completionTokensTotal: make(map[string]int64),
		costTotal:            make(map[string]float64),
		taskExecutionsTotal:  make(map[string]int64),
		taskErrors:           make(map[string]int64),
		taskLatencySum:       make(map[string]float64),
		toolCallsTotal:       make(map[string]int64),
		toolErrors:           make(map[string]int64),
		toolLatencySum:       make(map[string]float64),
		startTime:            time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Recording Methods
// ---------------------------------------------------------------------------

// RecordLLMCall records an LLM API call with model, latency, and error.
func (m *Metrics) RecordLLMCall(model string, latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.llmCallsTotal[model]++
	secs := latency.Seconds()
	m.llmLatencySum[model] += secs

	if err != nil {
		m.llmCallErrors[model]++
	}

	// Histogram buckets
	if m.llmLatencyBucket[model] == nil {
		m.llmLatencyBucket[model] = make(map[float64]int64)
	}
	for _, b := range defaultBuckets {
		if secs <= b {
			m.llmLatencyBucket[model][b]++
		}
	}
}

// RecordTokens records token usage for a model.
func (m *Metrics) RecordTokens(model string, promptTokens, completionTokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.promptTokensTotal[model] += int64(promptTokens)
	m.completionTokensTotal[model] += int64(completionTokens)
}

// RecordCost records cost for a model call.
func (m *Metrics) RecordCost(model string, costUSD float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.costTotal[model] += costUSD
}

// RecordTaskExecution records a task execution with type, latency, and error.
func (m *Metrics) RecordTaskExecution(taskType string, latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.taskExecutionsTotal[taskType]++
	m.taskLatencySum[taskType] += latency.Seconds()
	if err != nil {
		m.taskErrors[taskType]++
	}
}

// RecordToolCall records a tool invocation.
func (m *Metrics) RecordToolCall(toolName string, latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolCallsTotal[toolName]++
	m.toolLatencySum[toolName] += latency.Seconds()
	if err != nil {
		m.toolErrors[toolName]++
	}
}

// AgentStarted increments the active agent count.
func (m *Metrics) AgentStarted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeAgents++
	m.agentsCreated++
}

// AgentStopped decrements the active agent count.
func (m *Metrics) AgentStopped() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeAgents > 0 {
		m.activeAgents--
	}
}

// ---------------------------------------------------------------------------
// Snapshot — Read-Only Copy for Dashboards
// ---------------------------------------------------------------------------

// Snapshot returns a point-in-time copy of all metrics.
type MetricsSnapshot struct {
	UptimeSeconds         float64
	LLMCallsTotal         map[string]int64
	LLMCallErrors         map[string]int64
	PromptTokensTotal     map[string]int64
	CompletionTokensTotal map[string]int64
	CostTotalUSD          map[string]float64
	TaskExecutionsTotal   map[string]int64
	ToolCallsTotal        map[string]int64
	ActiveAgents          int64
	AgentsCreated         int64
}

// Snapshot returns a thread-safe copy of current metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snap := MetricsSnapshot{
		UptimeSeconds:         time.Since(m.startTime).Seconds(),
		LLMCallsTotal:         copyMapInt64(m.llmCallsTotal),
		LLMCallErrors:         copyMapInt64(m.llmCallErrors),
		PromptTokensTotal:     copyMapInt64(m.promptTokensTotal),
		CompletionTokensTotal: copyMapInt64(m.completionTokensTotal),
		CostTotalUSD:          copyMapFloat64(m.costTotal),
		TaskExecutionsTotal:   copyMapInt64(m.taskExecutionsTotal),
		ToolCallsTotal:        copyMapInt64(m.toolCallsTotal),
		ActiveAgents:          m.activeAgents,
		AgentsCreated:         m.agentsCreated,
	}
	return snap
}

// ---------------------------------------------------------------------------
// Prometheus Text Format Handler
// ---------------------------------------------------------------------------

// Handler returns an http.Handler that serves metrics in Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// Uptime
		writeGauge(w, "crew_go_uptime_seconds", "Uptime in seconds", time.Since(m.startTime).Seconds())

		// Agent gauges
		writeGaugeInt(w, "crew_go_active_agents", "Currently active agents", m.activeAgents)
		writeCounterInt(w, "crew_go_agents_created_total", "Total agents created", m.agentsCreated)

		// LLM calls
		for model, count := range m.llmCallsTotal {
			writeCounterIntLabel(w, "crew_go_llm_calls_total", "Total LLM API calls", "model", model, count)
		}
		for model, count := range m.llmCallErrors {
			writeCounterIntLabel(w, "crew_go_llm_call_errors_total", "Total LLM API errors", "model", model, count)
		}
		for model, sum := range m.llmLatencySum {
			writeCounterFloatLabel(w, "crew_go_llm_latency_seconds_sum", "Sum of LLM call latencies", "model", model, sum)
		}

		// Tokens
		for model, count := range m.promptTokensTotal {
			writeCounterIntLabel(w, "crew_go_prompt_tokens_total", "Total prompt tokens", "model", model, count)
		}
		for model, count := range m.completionTokensTotal {
			writeCounterIntLabel(w, "crew_go_completion_tokens_total", "Total completion tokens", "model", model, count)
		}

		// Cost
		for model, cost := range m.costTotal {
			writeCounterFloatLabel(w, "crew_go_cost_usd_total", "Total LLM cost in USD", "model", model, cost)
		}

		// Tasks
		for taskType, count := range m.taskExecutionsTotal {
			writeCounterIntLabel(w, "crew_go_task_executions_total", "Total task executions", "type", taskType, count)
		}
		for taskType, count := range m.taskErrors {
			writeCounterIntLabel(w, "crew_go_task_errors_total", "Total task errors", "type", taskType, count)
		}

		// Tools
		for tool, count := range m.toolCallsTotal {
			writeCounterIntLabel(w, "crew_go_tool_calls_total", "Total tool invocations", "tool", tool, count)
		}
		for tool, count := range m.toolErrors {
			writeCounterIntLabel(w, "crew_go_tool_errors_total", "Total tool errors", "tool", tool, count)
		}
	})
}

// ---------------------------------------------------------------------------
// Global Instance
// ---------------------------------------------------------------------------

var (
	globalMetrics     *Metrics
	globalMetricsOnce sync.Once
)

// GlobalMetrics returns the singleton metrics collector.
func GlobalMetrics() *Metrics {
	globalMetricsOnce.Do(func() {
		globalMetrics = NewMetrics()
	})
	return globalMetrics
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeGauge(w http.ResponseWriter, name, help string, val float64) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n%s %g\n", name, help, name, name, val)
}

func writeGaugeInt(w http.ResponseWriter, name, help string, val int64) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n%s %d\n", name, help, name, name, val)
}

func writeCounterInt(w http.ResponseWriter, name, help string, val int64) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, help, name, name, val)
}

func writeCounterIntLabel(w http.ResponseWriter, name, help, labelKey, labelVal string, val int64) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s{%s=\"%s\"} %d\n", name, help, name, name, labelKey, labelVal, val)
}

func writeCounterFloatLabel(w http.ResponseWriter, name, help, labelKey, labelVal string, val float64) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s{%s=\"%s\"} %g\n", name, help, name, name, labelKey, labelVal, val)
}

func copyMapInt64(m map[string]int64) map[string]int64 {
	c := make(map[string]int64, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func copyMapFloat64(m map[string]float64) map[string]float64 {
	c := make(map[string]float64, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}
