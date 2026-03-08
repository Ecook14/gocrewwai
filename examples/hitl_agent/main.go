package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/tools"
)

// ReviewableFileWriteTool is a wrapper that requires human approval.
type ReviewableFileWriteTool struct {
	tools.FileWriteTool
}

func (t *ReviewableFileWriteTool) RequiresReview() bool { return true }

func main() {
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

	// 2. Create an agent with a StepReview callback
	agent := &agents.Agent{
		Role:      "Legal Clerk",
		Goal:      "Draft and save a legal document. ALWAYS ask for approval before saving.",
		Backstory: "You are a meticulous clerk who understands the importance of human oversight.",
		LLM:       model,
		Tools:     []tools.Tool{fileTool},
		Verbose:   true,
		StepReview: func(toolName string, input interface{}) bool {
			fmt.Printf("\n[HUMAN-IN-THE-LOOP] Agent wants to use tool: %s\n", toolName)
			fmt.Printf("Input: %v\n", input)
			fmt.Print("Do you approve? (y/n): ")
			
			reader := bufio.NewReader(os.Stdin)
			resp, _ := reader.ReadString('\n')
			return strings.TrimSpace(strings.ToLower(resp)) == "y"
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
}
