package protocols

import (
	"context"
	"fmt"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

var _ core.Agent = (*RemoteAgentAdapter)(nil)

// RemoteAgentAdapter implements the core.Agent interface for remote A2A agents.
type RemoteAgentAdapter struct {
	Card   *AgentCard
	Client *A2AClient
	FromID string
}

func NewRemoteAgentAdapter(card *AgentCard, client *A2AClient, fromID string) *RemoteAgentAdapter {
	return &RemoteAgentAdapter{
		Card:   card,
		Client: client,
		FromID: fromID,
	}
}

func (a *RemoteAgentAdapter) GetRole() string {
	return a.Card.Role
}

func (a *RemoteAgentAdapter) GetGoal() string {
	return a.Card.Description
}

func (a *RemoteAgentAdapter) GetMaxRPM() int {
	return 0 // TODO: Fetch from remote capability card if needed
}

func (a *RemoteAgentAdapter) SetMaxRPM(rpm int) {
	// Remote agents handle their own RPM limits
}

func (a *RemoteAgentAdapter) GetUsageMetrics() map[string]int {
	return make(map[string]int) // Remote metrics unified separately
}

func (a *RemoteAgentAdapter) GetToolCount() int {
	return 0 // Remote tool counts shared via capability discovery phase if needed
}

// Equip satisfies core.Agent. Remote agents manage their own tools externally.
func (a *RemoteAgentAdapter) Equip(tools ...tools.Tool) {
	// Remote agents handle their own tools
}

func (a *RemoteAgentAdapter) Execute(ctx context.Context, input string, options map[string]interface{}) (interface{}, error) {
	msg := A2AMessage{
		ID:        fmt.Sprintf("del-%d", time.Now().UnixNano()),
		From:      a.FromID,
		To:        a.Card.ID,
		Type:      A2ARequest,
		Action:    "delegate_task",
		Payload:   MarshalTaskRequest(A2ATaskRequest{Description: input}),
		Timestamp: time.Now(),
	}

	resp, err := a.Client.Send(ctx, a.Card.Endpoint, msg)
	if err != nil {
		return nil, fmt.Errorf("a2a delegation failed: %w", err)
	}

	taskRes, err := UnmarshalTaskResponse(resp.Payload)
	if err != nil {
		return nil, fmt.Errorf("a2a invalid response payload: %w", err)
	}

	if !taskRes.Success {
		return nil, fmt.Errorf("remote task failed: %s", taskRes.Error)
	}

	return taskRes.Result, nil
}
