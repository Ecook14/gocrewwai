package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/config"
)

func getE2BBaseURL() string {
	val := config.Get().GetToolParam("e2b", "base_url")
	if val != "" {
		return val
	}
	return "https://api.e2b.dev"
}

// CodeInterpreterTool allows agents to execute Python or Go code snippets in a sandboxed environment.
// When SafeMode is true or no sandbox is configured, code executes directly on the host — dangerous.
// Use WithDockerConfig() or WithE2BConfig() to enable container isolation.
// The tool's DockerHardened field controls whether security-hardened Docker flags are applied.
//
// Example:
//
//	opts := []CodeInterpreterOption{
//	    WithDockerConfig("python:3.11-slim"),
//	    WithDockerHardened(true),
//	}
//	tool := NewCodeInterpreterTool(opts...)
type CodeInterpreterOption func(*CodeInterpreterTool)

// CodeInterpreterTool allows agents to execute Python or Go code snippets.
type CodeInterpreterTool struct {
	BaseTool
	SafeMode    bool
	E2BKey      string
	DockerImage string
	MemoryMB    int64
	CPUShares   int64
}

func NewCodeInterpreterTool(opts ...CodeInterpreterOption) *CodeInterpreterTool {
	t := &CodeInterpreterTool{
		BaseTool: BaseTool{
			NameValue:        "CodeInterpreterTool",
			DescriptionValue: "Execute snippets in Python, Go, or Shell (bash/sh). Input: {'language': 'python'|'go'|'bash'|'sh', 'code': '...'}. Runs via Docker if configured.",
		},
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func WithSafeMode(safe bool) CodeInterpreterOption {
	return func(t *CodeInterpreterTool) {
		t.SafeMode = safe
	}
}

func WithE2B(apiKey string) CodeInterpreterOption {
	return func(t *CodeInterpreterTool) {
		t.E2BKey = apiKey
	}
}

func WithDocker(image string) CodeInterpreterOption {
	return func(t *CodeInterpreterTool) {
		t.DockerImage = image
	}
}

func WithLimits(memMB, cpu int64) CodeInterpreterOption {
	return func(t *CodeInterpreterTool) {
		t.MemoryMB = memMB
		t.CPUShares = cpu
	}
}

// E2B API Structures
type e2bCreateRequest struct {
	TemplateID string                 `json:"templateID"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type e2bInstance struct {
	InstanceID string `json:"instanceID"`
	TemplateID string `json:"templateID"`
}

type e2bCommandRequest struct {
	Cmd string `json:"cmd"`
}

type e2bCommandResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

func (t *CodeInterpreterTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	langRaw, langOK := input["language"].(string)
	codeRaw, codeOK := input["code"].(string)
	if !codeOK || codeRaw == "" {
		return "", fmt.Errorf("'code' is required and must be a string")
	}
	if _, hasLang := input["language"]; hasLang && !langOK {
		return "", fmt.Errorf("'language' must be a string")
	}
	lang, code := langRaw, codeRaw

	// Sandbox-first: if E2B or Docker is configured, use the sandbox.
	if t.E2BKey != "" {
		return t.runE2B(ctx, lang, code)
	}

	// When a Docker image is configured, use Docker.
	if t.DockerImage != "" {
		switch lang {
		case "python":
			return t.runDocker(ctx, "python3", "-c", code)
		case "go":
			return t.runDocker(ctx, "go", "run", "-", code)
		case "bash", "sh":
			return t.runDocker(ctx, "sh", "-c", code)
		default:
			return "", fmt.Errorf("unsupported language: %s", lang)
		}
	}

	// No sandbox is configured. Host execution is not permitted because
	// it provides no isolation boundary for generated code. Return a clear
	// error that directs the operator to configure E2B or a Docker image.
	return "", fmt.Errorf(
		"code interpreter sandbox not configured: set E2B_API_KEY or CodeInterpreterTool.WithDocker(image) to enable isolated execution. Host execution is disabled for security",
	)
}

func (t *CodeInterpreterTool) runCommand(ctx context.Context, name string, args ...string) (string, error) {
	if t.DockerImage != "" {
		return t.runDocker(ctx, name, args...)
	}

	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (t *CodeInterpreterTool) runDocker(ctx context.Context, name string, args ...string) (string, error) {
	// Run untrusted code in an isolated Docker container with full
	// security hardening: no network, read-only rootfs, dropped capabilities,
	// non-root user, memory/CPU/pids limits. This mirrors the hardening
	// in pkg/sandbox/docker.go.
	fullCmd := append([]string{name}, args...)
	cmdStr := strings.Join(fullCmd, " ")

	// Base64-encode the command to safely pass it through the docker CLI
	// without shell escaping issues.
	b64Cmd := base64.StdEncoding.EncodeToString([]byte(cmdStr))

	dockerArgs := []string{
		"run", "--rm",
		"--network", "none",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges:true",
		"--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=64m",
		"--user", "1000:1000",
		"--memory", fmt.Sprintf("%dm", t.MemoryMB),
		"--cpu-shares", fmt.Sprintf("%d", t.CPUShares),
		"--pids-limit", "100",
		t.DockerImage,
		"sh", "-c",
		fmt.Sprintf("echo %s | base64 -d | sh 2>&1 || true", b64Cmd),
	}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (t *CodeInterpreterTool) runPython(ctx context.Context, code string) (string, error) {
	if t.DockerImage != "" {
		return t.runDocker(ctx, "python3", "-c", code)
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("script_%d.py", os.Getpid()))
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return "", err
	}
	defer os.Remove(tmpFile)

	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}
	return t.runCommand(ctx, pythonCmd, tmpFile)
}

func (t *CodeInterpreterTool) runGo(ctx context.Context, code string) (string, error) {
	if t.DockerImage != "" {
		// Go is harder to run via simple docker -c because it needs compilation.
		// For now we assume the image has 'go' installed.
		return t.runDocker(ctx, "go", "run", "-e", code)
	}

	tmpDir, err := os.MkdirTemp("", "go-run-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return "", err
	}

	return t.runCommand(ctx, "go", "run", tmpFile)
}

func (t *CodeInterpreterTool) runE2B(ctx context.Context, lang, code string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}

	// 1. Create Sandbox Instance
	createReq := e2bCreateRequest{TemplateID: "base"}
	body, err := json.Marshal(createReq)
	if err != nil {
		return "", fmt.Errorf("e2b: failed to marshal create request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", getE2BBaseURL()+"/instances", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("e2b: failed to create request: %w", err)
	}
	req.Header.Set("X-API-Key", t.E2BKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("e2b: instance creation failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("e2b: instance creation failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var inst e2bInstance
	if err := json.NewDecoder(resp.Body).Decode(&inst); err != nil {
		return "", fmt.Errorf("e2b: failed to decode instance response: %w", err)
	}

	// Ensure cleanup
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()
	go func() {
		delReq, _ := http.NewRequestWithContext(cleanupCtx, "DELETE", getE2BBaseURL()+"/instances/"+inst.InstanceID, nil)
		delReq.Header.Set("X-API-Key", t.E2BKey)
		client.Do(delReq)
	}()

	// 2. Prepare Command
	var cmdStr string
	switch lang {
	case "python":
		cmdStr = fmt.Sprintf("python3 -c '%s'", strings.ReplaceAll(code, "'", "'\\''"))
	case "bash", "sh":
		cmdStr = code
	default:
		cmdStr = code
	}

	cmdReq := e2bCommandRequest{Cmd: cmdStr}
	cmdBody, err := json.Marshal(cmdReq)
	if err != nil {
		return "", fmt.Errorf("e2b: failed to marshal command: %w", err)
	}
	execReq, err := http.NewRequestWithContext(ctx, "POST", getE2BBaseURL()+"/instances/"+inst.InstanceID+"/commands", bytes.NewBuffer(cmdBody))
	if err != nil {
		return "", fmt.Errorf("e2b: failed to create command request: %w", err)
	}
	execReq.Header.Set("X-API-Key", t.E2BKey)
	execReq.Header.Set("Content-Type", "application/json")

	execResp, err := client.Do(execReq)
	if err != nil {
		return "", fmt.Errorf("E2B execution failed: %w", err)
	}
	defer execResp.Body.Close()

	var cmdResp e2bCommandResponse
	if err := json.NewDecoder(execResp.Body).Decode(&cmdResp); err != nil {
		return "", err
	}

	output := cmdResp.Stdout
	if cmdResp.Stderr != "" {
		output += "\nErrors:\n" + cmdResp.Stderr
	}

	return output, nil
}

func (t *CodeInterpreterTool) runBash(ctx context.Context, code string) (string, error) {
	if t.DockerImage != "" {
		return t.runDocker(ctx, "sh", "-c", code)
	}
	return t.runCommand(ctx, "bash", "-c", code)
}

func (t *CodeInterpreterTool) RequiresReview() bool {
	return t.SafeMode
}

var _ Tool = (*CodeInterpreterTool)(nil)
