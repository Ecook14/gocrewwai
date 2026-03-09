package tools

import (
	"bytes"
	"context"
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
)

const e2bBaseURL = "https://api.e2b.dev"

// CodeInterpreterOption defines a functional option for CodeInterpreterTool.
type CodeInterpreterOption func(*CodeInterpreterTool)

// CodeInterpreterTool allows agents to execute Python or Go code snippets.
type CodeInterpreterTool struct {
	BaseTool
	SafeMode   bool
	E2BKey     string
	DockerImage string
	MemoryMB    int64
	CPUShares   int64
}

func NewCodeInterpreterTool(opts ...CodeInterpreterOption) *CodeInterpreterTool {
	t := &CodeInterpreterTool{
		BaseTool: BaseTool{
			NameValue:        "CodeInterpreter",
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
	lang, _ := input["language"].(string)
	code, _ := input["code"].(string)

	if code == "" {
		return "", fmt.Errorf("'code' is required")
	}

	if t.E2BKey != "" {
		return t.runE2B(ctx, lang, code)
	}

	switch lang {
	case "python":
		return t.runPython(ctx, code)
	case "go":
		return t.runGo(ctx, code)
	case "bash", "sh":
		return t.runBash(ctx, code)
	default:
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
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
	// Simple docker run --rm Image sh -c "command args..."
	// For production, we would handle file mounting.
	fullCmd := append([]string{name}, args...)
	managedCmd := fmt.Sprintf("'%s'", strings.Join(fullCmd, "' '")) // Rough escaping

	dockerArgs := []string{"run", "--rm"}
	if t.MemoryMB > 0 {
		dockerArgs = append(dockerArgs, "-m", fmt.Sprintf("%dm", t.MemoryMB))
	}
	if t.CPUShares > 0 {
		dockerArgs = append(dockerArgs, "--cpu-shares", fmt.Sprintf("%d", t.CPUShares))
	}
	dockerArgs = append(dockerArgs, t.DockerImage, "sh", "-c", managedCmd)

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
	body, _ := json.Marshal(createReq)
	req, err := http.NewRequestWithContext(ctx, "POST", e2bBaseURL+"/instances", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-API-Key", t.E2BKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("E2B instance creation failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("E2B instance creation failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var inst e2bInstance
	if err := json.NewDecoder(resp.Body).Decode(&inst); err != nil {
		return "", err
	}

	// Ensure cleanup
	defer func() {
		delReq, _ := http.NewRequestWithContext(context.Background(), "DELETE", e2bBaseURL+"/instances/"+inst.InstanceID, nil)
		delReq.Header.Set("X-API-Key", t.E2BKey)
		_, _ = client.Do(delReq)
	}()

	// 2. Prepare Command
	var cmdStr string
	switch lang {
	case "python":
		cmdStr = fmt.Sprintf("python3 -c '%s'", strings.ReplaceAll(code, "'", "'\\''"))
	case "bash", "sh":
		cmdStr = code
	default:
		cmdStr = code // Just try to run it as a command
	}

	cmdReq := e2bCommandRequest{Cmd: cmdStr}
	cmdBody, _ := json.Marshal(cmdReq)
	execReq, err := http.NewRequestWithContext(ctx, "POST", e2bBaseURL+"/instances/"+inst.InstanceID+"/commands", bytes.NewBuffer(cmdBody))
	if err != nil {
		return "", err
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
