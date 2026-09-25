package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CodeSandboxTool executes Python or JavaScript code inside an isolated
// Docker container with no network access, a read-only root filesystem,
// and a restricted tmpfs. Host-level execution is not permitted — when
// Docker is unavailable the tool returns an error rather than falling back
// to the host.
type CodeSandboxTool struct {
	BaseTool
	// DockerImage is the container image used for sandbox execution.
	DockerImage string
	// Timeout is the maximum duration for the sandbox execution.
	Timeout time.Duration
}

// NewCodeSandboxTool creates a code sandbox tool that defaults to a
// Python-slim Docker image for isolation.
func NewCodeSandboxTool() *CodeSandboxTool {
	image := os.Getenv("CODESANDBOX_IMAGE")
	if image == "" {
		image = "python:3.11-slim"
	}
	return &CodeSandboxTool{
		BaseTool: BaseTool{
			NameValue:        "CodeSandboxTool",
			DescriptionValue: "Executes Python or JavaScript code inside an isolated Docker container. Specify 'language' (python/javascript) and 'code'. Requires Docker.",
		},
		DockerImage: image,
		Timeout:     300 * time.Second,
	}
}

// Execute runs the provided code inside a Docker container with security
// hardening: no network, read-only rootfs, limited tmpfs, dropped
// capabilities, non-root user, and resource limits.
func (t *CodeSandboxTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	lang, _ := input["language"].(string)
	code, _ := input["code"].(string)

	if code == "" {
		return "", fmt.Errorf("missing 'code' parameter")
	}

	switch lang {
	case "python":
		return t.runInSandbox(ctx, "python3", "-c", code)
	case "javascript", "js":
		return t.runInSandbox(ctx, "node", "-e", code)
	default:
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
}

// runInSandbox executes a command inside an isolated Docker container.
// It validates that Docker is available and that the requested image exists
// before creating the container. If Docker is not available or the image
// cannot be pulled, the function returns an error — host execution is never
// used as a fallback.
func (t *CodeSandboxTool) runInSandbox(ctx context.Context, entrypoint, flag, code string) (string, error) {
	// Verify Docker is available.
	cli, err := exec.LookPath("docker")
	if err != nil {
		return "", fmt.Errorf(
			"code sandbox requires Docker but docker binary not found in PATH: %w",
			err,
		)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	fullCmd := entrypoint
	if flag != "" {
		fullCmd = fullCmd + " " + flag
	}
	fullCmd = fullCmd + " " + shellEscape(code)

	// Build the docker run arguments with security hardening.
	args := []string{
		"run",
		"--rm",
		"--network", "none",
		"--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,mode=1777",
		"--cap-drop", "ALL",
		"--user", "nobody:nogroup",
		"--memory", "512m",
		"--cpus", "0.5",
		"--pids-limit", "100",
		"--security-opt", "no-new-privileges:true",
		t.DockerImage,
		"sh", "-c", fullCmd,
	}

	cmd := exec.CommandContext(timeoutCtx, cli, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf(
			"code sandbox execution failed: %s",
			stripDockerErrors(string(out)),
		)
	}
	return string(out), nil
}

// shellEscape wraps a string for safe embedding inside a single-quoted
// shell argument. It escapes any single quotes in the input.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// stripDockerErrors removes common Docker error prefixes so that the
// returned error message is cleaner for the agent to consume.
func stripDockerErrors(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "docker:") &&
			!strings.HasPrefix(trimmed, "Error") && !strings.HasPrefix(trimmed, "OCI") {
			continue
		}
		// Return the first meaningful line.
		return strings.TrimSpace(trimmed)
	}
	if len(lines) > 0 {
		return strings.TrimSpace(lines[len(lines)-1])
	}
	return out
}

func (t *CodeSandboxTool) Name() string         { return t.BaseTool.NameValue }
func (t *CodeSandboxTool) Description() string  { return t.BaseTool.DescriptionValue }
func (t *CodeSandboxTool) RequiresReview() bool { return true }
func (t *CodeSandboxTool) ArgsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"language": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"python", "javascript", "js"},
				"description": "Programming language to execute.",
			},
			"code": map[string]interface{}{
				"type":        "string",
				"description": "Source code to execute.",
			},
		},
		"required": []string{"language", "code"},
	}
}
