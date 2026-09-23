package telemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Structured Logging — Agent-Aware, Level-Controlled
// ---------------------------------------------------------------------------

// LogConfig configures the structured logging system.
type LogConfig struct {
	Level     slog.Level // Minimum log level
	Format    string     // "json" or "text"
	Output    io.Writer  // Output destination (default: os.Stdout)
	AddSource bool       // Include file:line in logs
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
	Input       string                 `json:"input,omitempty"`  // Truncated and redacted for security
	Output      string                 `json:"output,omitempty"` // Truncated and redacted for security
	Duration    time.Duration          `json:"duration_ms,omitempty"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"` // Redacted for security
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogger writes structured, append-only audit entries for compliance.
// All entries are truncated and redacted for secrets/PII before writing.
type AuditLogger struct {
	mu      sync.Mutex
	writer  io.Writer
	maxLen  int // Max chars for input/output fields
	enabled bool
}

// NewAuditLogger creates an audit logger writing to the given destination.
// maxInputLen limits the size of logged input/output fields (0 = 2000 chars).
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

// DefaultAuditLogMaxLen returns the default maximum length for audit log entries.
func DefaultAuditLogMaxLen() int {
	return 2000
}

// Log writes an audit entry as a JSON line.
// It redacts known secret patterns (API keys, tokens, passwords) and PII
// patterns (email addresses, phone numbers) before writing to prevent
// accidental exposure of sensitive data in log files.
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

	// Redact secrets and PII from input/output fields.
	entry.Input = redactSensitiveData(entry.Input)
	entry.Output = redactSensitiveData(entry.Output)
	entry.Error = redactSensitiveData(entry.Error)

	// Redact secret-like values from metadata.
	if entry.Metadata != nil {
		entry.Metadata = redactMetadata(entry.Metadata)
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
// PII and Secret Redaction
// ---------------------------------------------------------------------------

// redactSensitiveData replaces known secret and PII patterns with [REDACTED].
func redactSensitiveData(s string) string {
	if s == "" {
		return s
	}

	s = apiKeyPattern.ReplaceAllString(s, "${1}[API_KEY:REDACTED]")
	s = bearerTokenPattern.ReplaceAllString(s, "${1}[TOKEN:REDACTED]")
	s = passwordPattern.ReplaceAllString(s, "${1}[PASSWORD:REDACTED]")
	s = secretPattern.ReplaceAllString(s, "${1}[SECRET:REDACTED]")
	s = emailPattern.ReplaceAllString(s, "[EMAIL:REDACTED]")
	s = phonePattern.ReplaceAllString(s, "[PHONE:REDACTED]")

	return s
}

// redactMetadata walks a metadata map and redacts any string values that
// look like secrets or PII.
func redactMetadata(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		key := strings.ToLower(k)
		if strings.Contains(key, "key") || strings.Contains(key, "secret") ||
			strings.Contains(key, "token") || strings.Contains(key, "password") ||
			strings.Contains(key, "api_key") || strings.Contains(key, "auth") {
			out[k] = "[REDACTED]"
			continue
		}
		switch val := v.(type) {
		case string:
			out[k] = redactSensitiveData(val)
		case map[string]interface{}:
			out[k] = redactMetadata(val)
		case []interface{}:
			out[k] = redactSlice(val)
		default:
			out[k] = val
		}
	}
	return out
}

// redactSlice redacts string elements in a slice that look like secrets or PII.
func redactSlice(s []interface{}) []interface{} {
	out := make([]interface{}, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case string:
			out[i] = redactSensitiveData(val)
		case map[string]interface{}:
			out[i] = redactMetadata(val)
		case []interface{}:
			out[i] = redactSlice(val)
		default:
			out[i] = v
		}
	}
	return out
}

// Precompiled regex patterns for sensitive data detection.
var (
	apiKeyPattern       = regexp.MustCompile(`(api[_-]?key["']?[:=]\s*["']?)[A-Za-z0-9_\-]{8,}(["']?)`)
	bearerTokenPattern = regexp.MustCompile(`(Bearer\s+)[A-Za-z0-9\-._~+/=]{10,}`)
	passwordPattern     = regexp.MustCompile(`(password["']?[:=]\s*["']?)[^\s"']{4,}(["']?)`)
	secretPattern       = regexp.MustCompile(`(secret["']?[:=]\s*["']?)[^\s"']{8,}(["']?)`)
	emailPattern        = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phonePattern        = regexp.MustCompile(`(?:\+?\d{1,3}[-.\s]?)?\(?\d{2,4}\)?[-.\s]?\d{3,4}[-.\s]?\d{4}`)
)

// ---------------------------------------------------------------------------
// EventBus -> Metrics Bridge
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
					metrics.RecordToolCall(name, 0, nil)
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
