package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	// Configure CodeInterpreter with E2B support
	interpreter := gocrew.NewCodeInterpreter(true)

	developer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Senior Developer",
		Goal:      "Analyze the performance of this Python snippet using a cloud sandbox.",
		Backstory: "Code Optimizer",
		LLM:       model,
	})
	developer.Tools = []gocrew.Tool{interpreter}

	task := &gocrew.Task{
		Description: "Run this python code to calculate the 40th Fibonacci number: 'def fib(n): return n if n <= 1 else fib(n-1) + fib(n-2); print(fib(40))'",
		Agent:       developer,
	}

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{developer},
		Tasks:   []*gocrew.Task{task},
	})

	fmt.Println("🚀 Starting Sandboxing Demo (E2B Cloud Integration)...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nSandbox Execution Result:\n%s\n", result)
}
