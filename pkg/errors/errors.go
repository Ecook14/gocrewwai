// Package errors provides sentinel errors and typed error wrappers for the Gocrew framework.
// Centralizing errors improves debuggability and allows callers to use errors.Is/As.
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for common failure modes across the framework.
var (
	ErrNoAgent        = errors.New("agent is required but was nil")
	ErrNoTasks        = errors.New("crew requires at least one task to kickoff")
	ErrNoAgents       = errors.New("crew requires at least one agent to kickoff")
	ErrLLMFailed      = errors.New("llm generation failed")
	ErrToolNotFound   = errors.New("requested tool does not exist")
	ErrMaxIterations  = errors.New("agent hit maximum iteration limit without providing a final answer")
	ErrGuardrailFailed = errors.New("output failed guardrail validation")
	ErrValidation     = errors.New("input validation failed")
	ErrDelegation     = errors.New("agent delegation failed")
	ErrUnsupportedProcess = errors.New("unsupported process type")
)

// AgentError wraps an error with the agent role for richer diagnostics.
type AgentError struct {
	Role string
	Iter int
	Err  error
}

func (e *AgentError) Error() string {
	role := strings.Clone(e.Role)
	if e.Iter > 0 {
		return "agent '" + role + "' (iter " + fmt.Sprintf("%d", e.Iter) + "): " + e.Err.Error()
	}
	return "agent '" + role + "': " + e.Err.Error()
}

func (e *AgentError) Unwrap() error { return e.Err }

// NewAgentError creates a new AgentError.
func NewAgentError(role string, iter int, err error) *AgentError {
	return &AgentError{Role: role, Iter: iter, Err: err}
}

// TaskError wraps an error with the task index for pipeline diagnostics.
type TaskError struct {
	Index int
	Desc  string
	Err   error
}

func (e *TaskError) Error() string {
	return fmt.Sprintf("task %d (%s): %v", e.Index, e.Desc, e.Err)
}

func (e *TaskError) Unwrap() error { return e.Err }

// NewTaskError creates a new TaskError.
func NewTaskError(index int, desc string, err error) *TaskError {
	return &TaskError{Index: index, Desc: desc, Err: err}
}

// GuardrailError wraps a guardrail validation failure with the guardrail name.
type GuardrailError struct {
	GuardrailName string
	Err           error
}

func (e *GuardrailError) Error() string {
	return fmt.Sprintf("guardrail '%s': %v", e.GuardrailName, e.Err)
}

func (e *GuardrailError) Unwrap() error { return e.Err }

// NewGuardrailError creates a new GuardrailError.
func NewGuardrailError(name string, err error) *GuardrailError {
	return &GuardrailError{GuardrailName: name, Err: err}
}

// Wrap is a convenience for fmt.Errorf with %w.
func Wrap(sentinel error, msg string) error {
	return fmt.Errorf("%s: %w", msg, sentinel)
}
