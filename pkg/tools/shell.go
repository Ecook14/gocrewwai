package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ShellTool allows agents to execute shell commands on the host system.
// This tool is inherently dangerous and always requires human review.
//
// Input examples:
//
//	{"command": "ls -la /tmp"}
//	{"command": "echo hello", "timeout": 10}
//	{"command": "df -h"}
type ShellTool struct {
	BaseTool
	AllowedCommands []string      // Whitelist (empty = allow all)
	BlockedCommands []string      // Blacklist
	DefaultTimeout  time.Duration // Default command timeout
	WorkingDir      string        // Default working directory
}

// NewShellTool creates a shell execution tool.
func NewShellTool(opts ...func(*ShellTool)) *ShellTool {
	t := &ShellTool{
		BaseTool: BaseTool{
			NameValue:        "ShellTool",
			DescriptionValue: "Execute shell commands on the host system. Input: {'command': 'shell command string', 'timeout': seconds}. Returns stdout and stderr. DANGEROUS: always requires approval.",
		},
		DefaultTimeout: 30 * time.Second,
		BlockedCommands: []string{
			"rm -rf /", "mkfs", "dd if=/dev/zero", ":(){ :|:& };:",
			"shutdown", "reboot", "halt", "poweroff",
		},
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// WithAllowedCommands restricts execution to only these command prefixes.
func WithAllowedCommands(cmds []string) func(*ShellTool) {
	return func(t *ShellTool) {
		t.AllowedCommands = cmds
	}
}

// WithShellTimeout sets the default timeout.
func WithShellTimeout(d time.Duration) func(*ShellTool) {
	return func(t *ShellTool) {
		t.DefaultTimeout = d
	}
}

// WithWorkingDir sets the default working directory.
func WithWorkingDir(dir string) func(*ShellTool) {
	return func(t *ShellTool) {
		t.WorkingDir = dir
	}
}

func (t *ShellTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	command, _ := input["command"].(string)
	if command == "" {
		return "", fmt.Errorf("'command' is required")
	}

	// Security: Check blocked commands
	cmdLower := strings.ToLower(strings.TrimSpace(command))
	for _, blocked := range t.BlockedCommands {
		if strings.Contains(cmdLower, strings.ToLower(blocked)) {
			return "", fmt.Errorf("command blocked for safety: contains '%s'", blocked)
		}
	}

	// Security: Check allowed commands whitelist
	if len(t.AllowedCommands) > 0 {
		allowed := false
		for _, prefix := range t.AllowedCommands {
			if strings.HasPrefix(cmdLower, strings.ToLower(prefix)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("command not in allowed list: %s", command)
		}
	}

	// Parse timeout
	timeout := t.DefaultTimeout
	if ts, ok := input["timeout"].(float64); ok && ts > 0 {
		timeout = time.Duration(ts) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	if t.WorkingDir != "" {
		cmd.Dir = t.WorkingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var result strings.Builder
	if stdout.Len() > 0 {
		result.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if result.Len() > 0 {
			result.WriteString("\n--- STDERR ---\n")
		}
		result.WriteString(stderr.String())
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return result.String(), fmt.Errorf("command timed out after %v", timeout)
		}
		return result.String(), fmt.Errorf("command failed: %w\n%s", err, result.String())
	}

	// Truncate very long output
	output := result.String()
	if len(output) > 50000 {
		output = output[:50000] + "\n... [output truncated at 50KB]"
	}

	return output, nil
}

func (t *ShellTool) RequiresReview() bool { return true } // Always requires review
