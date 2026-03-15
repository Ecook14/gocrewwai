package sandbox

import (
	"context"
	"sync"
)

// Monitor tracks the state and health of sandbox providers.
type Monitor struct {
	ActiveSessions int
	TotalExecutions int
	LastStatus     string
	mu             sync.Mutex
}

func (m *Monitor) RecordStart() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ActiveSessions++
	m.TotalExecutions++
}

func (m *Monitor) RecordEnd(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ActiveSessions--
	m.LastStatus = status
}

var GlobalMonitor = &Monitor{}

// Provider defines the interface for isolated code execution.
type Provider interface {
	// Execute runs the provided code snippet in a sandboxed environment.
	// code: The script or commands to run.
	// env: Optional environment variables to inject.
	Execute(ctx context.Context, code string, env map[string]string) (string, error)
	
	// Close cleans up any resources used by the provider.
	Close() error
}

// Result represents the output of a sandboxed execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}
