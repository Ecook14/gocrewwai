package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// DockerProvider executes code within a Docker container.
type DockerProvider struct {
	cli   *client.Client
	image string
}

func NewDockerProvider(image string) (*DockerProvider, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker: failed to create client: %w", err)
	}
	return &DockerProvider{cli: cli, image: image}, nil
}

// Execute runs the code using the 'sh -c' command inside the container.
func (p *DockerProvider) Execute(ctx context.Context, code string, env map[string]string) (string, error) {
	// 1. Pull image if needed (simplified: assuming it exists or let container create fail)
	// In production, we'd check if image exists or Pull it.

	// 2. Prepare environment variables
	var envList []string
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	// 3. Create container
	resp, err := p.cli.ContainerCreate(ctx, &container.Config{
		Image: p.image,
		Cmd:   []string{"sh", "-c", code},
		Env:   envList,
	}, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("docker: failed to create container: %w", err)
	}
	defer p.cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})

	// 4. Start container
	if err := p.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("docker: failed to start container: %w", err)
	}

	// 5. Wait for completion
	statusCh, errCh := p.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
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
	out, err := p.cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", fmt.Errorf("docker: failed to get logs: %w", err)
	}
	defer out.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, out)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("docker: failed to copy logs: %w", err)
	}

	if stderr.Len() > 0 {
		return stdout.String(), fmt.Errorf("docker: execution error: %s", stderr.String())
	}

	return stdout.String(), nil
}

func (p *DockerProvider) Close() error {
	return p.cli.Close()
}
