package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

type AIStats struct {
	Trend     string `json:"trend"`
	Impact    string `json:"impact"`
	Certainty int    `json:"certainty"`
}

func main() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is required")
	}

	client := llm.NewOpenRouterClient(apiKey, "google/gemini-2.0-flash-lite-preview-02-05:free")

	agent := agents.NewAgentBuilder().
		Role("Data Analyst").
		Goal("Extract AI stats in JSON.").
		LLM(client).
		Build()

	var stats AIStats
	task := &tasks.Task{
		Description: "Identify the top AI trend for 2026. Return your answer as a JSON object with 'trend', 'impact', and 'certainty' (0-100).",
		Agent:       agent,
		OutputJSON:   &stats,
	}

	fmt.Println("🚀 Kicking off Robust Structured Output Demo...")
	result, err := task.Execute(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n--- Final Structured Result ---\n%+v\n", result)
	fmt.Printf("Trend: %s\n", stats.Trend)
}
