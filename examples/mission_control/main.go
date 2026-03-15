package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/protocols"
)

func main() {
	ctx := context.Background()
	model := llm.NewOpenAIClient("") // Uses env key

	slog.Info("🚀 Starting v0.9 Mission Control Demo")

	// 1. Setup a Remote Worker (Simulated on another port)
	worker := agents.New(agents.AgentConfig{
		Role:      "Researcher",
		Goal:      "Find cutting-edge news about AI sandboxing.",
		Backstory: "You are a cyber-security researcher focused on LLM safety.",
		A2APort:   5001,
	})
	
	// Register worker in the Global Registry for discovery
	protocols.GlobalA2ARegistry.Register(&protocols.AgentCard{
		ID:           worker.A2AID,
		Name:         worker.Role,
		Role:         worker.Role,
		Description:  worker.Goal,
		Endpoint:     "http://localhost:5001",
		Capabilities: []string{"research", "security"},
	})

	// 2. Setup the Manager
	manager := agents.NewManagerAgent(model, nil)
	manager.A2AID = "mission-control-manager"
	manager.Verbose = true

	slog.Info("🧠 Manager is scanning the network for agents...")
	time.Sleep(1 * time.Second) // Wait for discovery simulation

	// 3. Execute Parallel Distributed Tasks
	tasks := []string{
		"Research the latest wazero updates for Go sandboxing.",
		"Research Docker-in-Docker security best practices for CI/CD.",
	}

	results, err := manager.DelegateParallelTasks(ctx, tasks)
	if err != nil {
		slog.Error("❌ Mission failed", slog.Any("error", err))
		return
	}

	slog.Info("🏁 Mission Accomplished! Results aggregated.")
	for id, res := range results {
		fmt.Printf("\n--- Result from %s ---\n%s\n", id, res)
	}
}
