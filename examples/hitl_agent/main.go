package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/dashboard"
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/guardrails"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

// ReviewableFileWriteTool is a wrapper that requires human approval.
type ReviewableFileWriteTool struct {
	tools.FileWriteTool
}

func (t *ReviewableFileWriteTool) RequiresReview() bool { return true }

func main() {
	// 1. Initialise the Dashboard Server in the background
	dashboard.Start("8081") // Using 8081 to avoid conflict if 8080 is used
	slog.Info("🖥️  Dashboard active at http://localhost:8081/web-ui")
	slog.Info("Please open the dashboard in your browser to approve the file write!")
	time.Sleep(3 * time.Second)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY")
		return
	}

	model := llm.NewOpenAIClient(apiKey)

	// 1. Create a tool that requires review
	fileTool := &ReviewableFileWriteTool{
		FileWriteTool: tools.FileWriteTool{},
	}

	// 2. Create an agent with the UI-driven HITL Guardrail
	agent := &agents.Agent{
		Role:      "Legal Clerk",
		Goal:      "Draft and save a legal document.",
		Backstory: "You are a meticulous clerk who understands the importance of human oversight.",
		LLM:       model,
		Tools:     []tools.Tool{fileTool},
		Verbose:   true,
		Guardrails: []guardrails.Guardrail{
			guardrails.NewHumanReviewGuardrail("Legal Clerk", "ReviewableFileWriteTool"),
		},
	}

	// 3. Execute a task
	fmt.Println("## Starting HITL Agent Execution ##")
	ctx := context.Background()
	result, err := agent.Execute(ctx, "Write a short 'Hello World' disclaimer to a file named 'disclaimer.txt'", nil)
	
	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
		return
	}

	fmt.Printf("\n## Final Result ##\n%v\n", result)
	
	slog.Info("✅ Demo finished. Keep the dashboard open to review the logs!")
	select {} // Keep running so user can see dashboard logs
}
