package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// maxSandboxOutputBytes caps combined captured stdout+stderr retained in
	// host memory. Guest memory limits cannot constrain the host buffer that
	// collects the guest's output, so a small program emitting a large stream
	// would otherwise exhaust host memory (CWE-400/770).
	maxSandboxOutputBytes = 2 << 20 // 2MiB
	maxSandboxCodeBytes   = 256 * 1024
)

// combinedOutputBudget caps combined writes across stdout and stderr.
type combinedOutputBudget struct {
	mu        sync.Mutex
	remaining int64
	overflow  bool
}

func (b *combinedOutputBudget) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.remaining <= 0 {
		b.overflow = true
		return 0, fmt.Errorf("sandbox output budget exceeded")
	}
	if int64(len(p)) > b.remaining {
		b.overflow = true
		n := int(b.remaining)
		b.remaining = 0
		return n, fmt.Errorf("sandbox output budget exceeded")
	}
	b.remaining -= int64(len(p))
	return len(p), nil
}

// sharedCappedBuffer pairs a bytes.Buffer with a shared budget.
type sharedCappedBuffer struct {
	buf    bytes.Buffer
	budget *combinedOutputBudget
}

func (c *sharedCappedBuffer) Write(p []byte) (int, error) {
	// Reserve from the shared budget first so combined output is bounded.
	if _, err := c.budget.Write(p); err != nil {
		return 0, err
	}
	return c.buf.Write(p)
}

// DockerProvider executes code within a Docker container.
type DockerProvider struct {
	cli     *client.Client
	image   string
	Timeout time.Duration
}

func NewDockerProvider(image string) (*DockerProvider, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker: failed to create client: %w", err)
	}
	return &DockerProvider{cli: cli, image: image, Timeout: 300 * time.Second}, nil
}

// Execute runs the code using the 'sh -c' command inside the container.
// The container is run with security-hardening defaults:
//   - No network access (--network none)
//   - Read-only root filesystem (--read-only)
//   - A writable /tmp tmpfs (--tmpfs /tmp)
//   - Dropping all capabilities (--cap-drop ALL)
//   - Non-root user (--user 1000:1000)
//   - Memory limit (--memory)
//   - CPU quota (--cpu-quota)
//   - Pid limit (--pids-limit)
func (p *DockerProvider) Execute(ctx context.Context, code string, env map[string]string) (string, error) {
	ctx, span := telemetry.StartSpan(ctx, "sandbox.docker.Execute")
	if span != nil {
		span.SetAttributes(attribute.String("sandbox.image", p.image))
		defer span.End()
	}
	if len(code) > maxSandboxCodeBytes {
		return "", fmt.Errorf("docker: code exceeds %d byte budget", maxSandboxCodeBytes)
	}
	// 1. Pull image if needed (simplified: assuming it exists or let container create fail)
	// In production, we'd check if image exists or Pull it.

	// 2. Prepare environment variables
	var envList []string
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	// 3. Create container with security hardening
	hostConfig := &container.HostConfig{
		NetworkMode:    "none",
		ReadonlyRootfs: true,
		CapDrop:        []string{"ALL"},
		Tmpfs: map[string]string{
			"/tmp": "rw,noexec,nosuid,size=65536k",
		},
		Resources: container.Resources{
			Memory:    512 * 1024 * 1024,
			CPUQuota:  50000,
			PidsLimit: func() *int64 { v := int64(100); return &v }(),
		},
	}

	resp, err := p.cli.ContainerCreate(timeoutCtx, &container.Config{
		Image: p.image,
		Cmd:   []string{"sh", "-c", code},
		Env:   envList,
		User:  "1000:1000",
	}, hostConfig, nil, nil, "")

	if err != nil {
		return "", fmt.Errorf("docker: failed to create container: %w", err)
	}
	// Cleanup uses a fresh bounded context independent of execution
	// cancellation: reusing the expired timeoutCtx would leave timed-out
	// workloads running indefinitely. Failures are reported, not ignored.
	cleanup := func() {
		cctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = p.cli.ContainerStop(cctx, resp.ID, container.StopOptions{})
		if err := p.cli.ContainerRemove(cctx, resp.ID, container.RemoveOptions{Force: true}); err != nil {
			slog.Warn("docker: container cleanup failed", "id", resp.ID, "error", err)
		}
	}
	defer cleanup()

	// 4. Start container
	if err := p.cli.ContainerStart(timeoutCtx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("docker: failed to start container: %w", err)
	}

	// 5. Wait for completion
	statusCh, errCh := p.cli.ContainerWait(timeoutCtx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("docker: error waiting for container: %w", err)
		}
	case <-statusCh:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	// 6. Capture logs
	out, err := p.cli.ContainerLogs(timeoutCtx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", fmt.Errorf("docker: failed to get logs: %w", err)
	}
	defer out.Close()

	var stdout, stderr sharedCappedBuffer
	budget := &combinedOutputBudget{remaining: maxSandboxOutputBytes}
	stdout.budget = budget
	stderr.budget = budget
	_, copyErr := stdcopy.StdCopy(&stdout, &stderr, out)
	if budget.overflow {
		return "", fmt.Errorf("docker: output exceeded %d byte budget (truncated)", maxSandboxOutputBytes)
	}
	if copyErr != nil && copyErr != io.EOF {
		return "", fmt.Errorf("docker: failed to copy logs: %w", copyErr)
	}

	if stderr.buf.Len() > 0 {
		return stdout.buf.String(), fmt.Errorf("docker: execution error: %s", stderr.buf.String())
	}

	return stdout.buf.String(), nil
}

func (p *DockerProvider) Close() error {
	return p.cli.Close()
}
