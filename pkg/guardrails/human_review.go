package guardrails

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/Ecook14/crewai-go/pkg/telemetry"
)

// generateReviewID generates a random 16-byte hex string.
func generateReviewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HumanReviewGuardrail pauses execution and pushes a review request to the Global Event Bus.
// It waits synchronously for the user to approve or reject the payload via the Web UI Dashboard.
type HumanReviewGuardrail struct {
	AgentRole string
	ToolName  string
}

func NewHumanReviewGuardrail(agentRole string, toolName string) *HumanReviewGuardrail {
	return &HumanReviewGuardrail{
		AgentRole: agentRole,
		ToolName:  toolName,
	}
}

func (g *HumanReviewGuardrail) Name() string { return "HumanReviewGuardrail" }

func (g *HumanReviewGuardrail) Validate(output string) error {
	reviewID := generateReviewID()
	
	// Issue request to global review manager.
	// This generates a 'review_requested' event on the bus and fully blocks the 
	// calling Goroutine until GlobalReviewManager.SubmitReview is called by the ws.go API.
	approved := telemetry.GlobalReviewManager.RequestReview(reviewID, g.AgentRole, g.ToolName, output)
	
	if !approved {
		return fmt.Errorf("human reviewer explicitly rejected the execution")
	}
	
	return nil
}
