package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/llm"
)

func main() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is required")
	}

	// 1. Setup Cache
	cache := llm.NewFileCache("./demo_cache")
	defer os.RemoveAll("./demo_cache") // Cleanup for demo

	// 2. Setup Client
	client := llm.NewOpenRouterClient(apiKey, "")

	// 3. Create Agent with Cache
	agent := agents.NewAgentBuilder().
		Role("Fast Researcher").
		Goal("Provide quick answers.").
		LLM(client).
		Cache(cache).
		Verbose(true).
		Build()

	ctx := context.Background()
	prompt := "What is the capital of France?"

	fmt.Println("🚀 First run (Cold Cache)...")
	start := time.Now()
	res1, err := agent.Execute(ctx, prompt, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %v\nTime: %v\n\n", res1, time.Since(start))

	fmt.Println("🚀 Second run (Hot Cache)...")
	start = time.Now()
	res2, err := agent.Execute(ctx, prompt, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %v\nTime: %v\n\n", res2, time.Since(start))

	fmt.Println("✅ Caching Demo Complete")
}
