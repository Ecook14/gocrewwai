package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	// Configure CodeInterpreter with E2B support
	interpreter := tools.NewCodeInterpreterTool(
		tools.WithE2B("dummy_e2b_key"), // Actual E2B Cloud Sandbox integration pattern
	)

	developer := agents.NewAgent(
		"Senior Developer",
		"Analyze the performance of this Python snippet using a cloud sandbox.",
		"Code Optimizer",
		model,
	)
	developer.Tools = []tools.Tool{interpreter}

	task := &tasks.Task{
		Description: "Run this python code to calculate the 40th Fibonacci number: 'def fib(n): return n if n <= 1 else fib(n-1) + fib(n-2); print(fib(40))'",
		Agent:       developer,
	}

	myCrew := crew.NewCrew([]core.Agent{developer}, []*tasks.Task{task})

	fmt.Println("🚀 Starting Sandboxing Demo (E2B Cloud Integration)...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nSandbox Execution Result:\n%s\n", result)
}
