package agents

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/tools"
	"github.com/Ecook14/gocrewwai/pkg/api/mesh" // This will be generated from proto, placeholders for now
)

// RemoteAgent represents an agent running on a different engine instance via gRPC.
type RemoteAgent struct {
	Role      string
	Goal      string
	Backstory string
	Address   string // gRPC target, e.g., "10.0.0.5:50051"
	AuthToken string
}

func NewRemoteAgent(role, goal, backstory, address, token string) *RemoteAgent {
	return &RemoteAgent{
		Role:      role,
		Goal:      goal,
		Backstory: backstory,
		Address:   address,
		AuthToken: token,
	}
}

func (r *RemoteAgent) GetRole() string      { return r.Role }
func (r *RemoteAgent) GetGoal() string      { return r.Goal }
func (r *RemoteAgent) GetBackstory() string { return r.Backstory }
func (r *RemoteAgent) GetMaxRPM() int { return 0 }
func (r *RemoteAgent) SetMaxRPM(int) {}
func (r *RemoteAgent) GetUsageMetrics() map[string]int { return nil }
func (r *RemoteAgent) GetToolCount() int { return 0 }
func (r *RemoteAgent) Equip(tools ...tools.Tool) {}

func (r *RemoteAgent) Execute(ctx context.Context, input string, options map[string]interface{}) (interface{}, error) {
	// 1. Dial remote instance
	conn, err := grpc.Dial(r.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote agent at %s: %w", r.Address, err)
	}
	defer conn.Close()

	// 2. Create client
	client := mesh.NewMeshServiceClient(conn)

	// 3. Prepare task request
	req := &mesh.TaskRequest{
		TaskDescription: input,
		AgentRole:       r.Role,
		AgentGoal:       r.Goal,
		SessionId:       fmt.Sprintf("%v", options["session_id"]),
	}

	// 4. Remote Call
	resp, err := client.DelegateTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("remote task delegation failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("remote agent error: %s", resp.ErrorMessage)
	}

	return resp.Output, nil
}
