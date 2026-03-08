package telemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Structured Logging — Agent-Aware, Level-Controlled
// ---------------------------------------------------------------------------

// LogConfig configures the structured logging system.
type LogConfig struct {
	Level      slog.Level // Minimum log level
	Format     string     // "json" or "text"
	Output     io.Writer  // Output destination (default: os.Stdout)
	AddSource  bool       // Include file:line in logs
}

// DefaultLogConfig returns sensible logging defaults.
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: os.Stdout,
	}
}

// NewLogger creates a structured logger from config.
func NewLogger(cfg LogConfig) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(cfg.Output, opts)
	default:
		handler = slog.NewJSONHandler(cfg.Output, opts)
	}

	return slog.New(handler)
}

// AgentLogger creates a child logger with agent context pre-set.
func AgentLogger(parent *slog.Logger, agentRole, taskDesc string) *slog.Logger {
	return parent.With(
		slog.String("agent_role", agentRole),
		slog.String("task", taskDesc),
	)
}

// ---------------------------------------------------------------------------
// Audit Logger — Compliance & Security Event Tracking
// ---------------------------------------------------------------------------

// AuditEntry represents a single auditable event.
type AuditEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"` // "llm_call", "tool_exec", "task_complete", etc
	AgentRole   string                 `json:"agent_role,omitempty"`
	Action      string                 `json:"action"`
	Input       string                 `json:"input,omitempty"`  // Truncated for security
	Output      string                 `json:"output,omitempty"` // Truncated for security
	Duration    time.Duration          `json:"duration_ms,omitempty"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogger writes structured, append-only audit entries for compliance.
type AuditLogger struct {
	mu      sync.Mutex
	writer  io.Writer
	maxLen  int // Max chars for input/output fields
	enabled bool
}

// NewAuditLogger creates an audit logger writing to the given destination.
// maxInputLen limits the size of logged input/output fields (0 = unlimited).
func NewAuditLogger(writer io.Writer, maxInputLen int) *AuditLogger {
	if maxInputLen <= 0 {
		maxInputLen = 2000
	}
	return &AuditLogger{
		writer:  writer,
		maxLen:  maxInputLen,
		enabled: true,
	}
}

// Log writes an audit entry as a JSON line.
func (a *AuditLogger) Log(entry AuditEntry) error {
	if !a.enabled {
		return nil
	}

	entry.Timestamp = time.Now()

	// Truncate for security/size
	if len(entry.Input) > a.maxLen {
		entry.Input = entry.Input[:a.maxLen] + "...[truncated]"
	}
	if len(entry.Output) > a.maxLen {
		entry.Output = entry.Output[:a.maxLen] + "...[truncated]"
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	_, err = fmt.Fprintf(a.writer, "%s\n", data)
	return err
}

// SetEnabled toggles audit logging on/off.
func (a *AuditLogger) SetEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = enabled
}

// ---------------------------------------------------------------------------
// EventBus → Metrics Bridge
// ---------------------------------------------------------------------------

// BridgeEventsToMetrics subscribes to the global EventBus and auto-records metrics.
func BridgeEventsToMetrics(metrics *Metrics) chan Event {
	ch := GlobalBus.Subscribe()
	go func() {
		for event := range ch {
			switch event.Type {
			case EventAgentStarted:
				metrics.AgentStarted()
			case EventAgentFinished:
				metrics.AgentStopped()
			case EventToolStarted:
				if name, ok := event.Payload["tool"].(string); ok {
					metrics.RecordToolCall(name, 0, nil) // Start count only
				}
			case EventTaskFinished:
				taskType := "unknown"
				if desc, ok := event.Payload["description"].(string); ok && len(desc) > 50 {
					taskType = desc[:50]
				} else if desc, ok := event.Payload["description"].(string); ok {
					taskType = desc
				}
				metrics.RecordTaskExecution(taskType, 0, nil)
			}
		}
	}()
	return ch
}
