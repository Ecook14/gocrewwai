package sandbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/tools"
)

type DockerConfig struct {
	Image   string
	Timeout int
}

func DefaultDockerConfig() DockerConfig {
	return DockerConfig{Image: "python:3.11-slim", Timeout: 300}
}

func WithSafeMode(safe bool) DockerConfig {
	cfg := DefaultDockerConfig()
	if !safe {
		cfg.Timeout = 600
	}
	return cfg
}

func NewDockerSandbox(image string, safe bool) (tools.Tool, error) {
	if image == "" {
		image = "python:3.11-slim"
	}
	provider, err := NewDockerProvider(image)
	if err != nil {
		return nil, err
	}
	return &dockerSandboxTool{provider: provider, safe: safe}, nil
}

type dockerSandboxTool struct {
	provider *DockerProvider
	safe     bool
}

func (d *dockerSandboxTool) Name() string {
	return "DockerSandbox"
}

func (d *dockerSandboxTool) Description() string {
	if d.safe {
		return "Executes code in an isolated, security-hardened Docker container."
	}
	return "Executes code in a Docker container with relaxed security."
}

func (d *dockerSandboxTool) RequiresReview() bool {
	return false
}

func (d *dockerSandboxTool) ArgsSchema() []tools.ArgSchema {
	return []tools.ArgSchema{{
		Name: "code", Type: "string", Description: "Code to execute in Docker sandbox.", Required: true,
	}}
}

func (d *dockerSandboxTool) CacheFunction(input map[string]interface{}) string {
	return ""
}

func (d *dockerSandboxTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	code, ok := input["code"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'code' parameter")
	}
	if code == "" {
		return "", fmt.Errorf("code parameter is empty")
	}
	if len(code) > 10000 {
		return "", fmt.Errorf("code exceeds maximum length of 10000 characters")
	}
	if d.safe {
		if strings.ContainsAny(code, "`$()|;&") {
			return "", fmt.Errorf("safe mode: shell metacharacters not allowed")
		}
	}
	env := map[string]string{"SAFE_MODE": fmt.Sprintf("%v", d.safe)}
	stdout, err := d.provider.Execute(ctx, code, env)
	if err != nil {
		return stdout, err
	}
	return stdout, nil
}

func ShowDockerImage(cfg DockerConfig) string { return cfg.Image }

func IsSandboxAvailable() bool {
	_, err := NewDockerProvider("python:3.11-slim")
	return err == nil
}

func RequireSandbox() {
	if !IsSandboxAvailable() {
		panic("Docker sandbox not available")
	}
}
