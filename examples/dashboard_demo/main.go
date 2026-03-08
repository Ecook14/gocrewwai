package main

import (
	"context"
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

func main() {
	// 1. Initialise the Dashboard Server in the background
	dashboard.Start("8080")
	slog.Info("🖥️  Dashboard active at http://localhost:8080/web-ui")
	slog.Info("Please open the dashboard in your browser before the crew starts!")
	
	time.Sleep(5 * time.Second) // Give user time to open the page

	// 2. Setup a demo Crew
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := llm.NewOpenAIClient(apiKey)

	researcher := agents.NewAgent(
		"Researcher",
		"Find the latest news about Go 1.24",
		"You are a tech journalist looking for cutting-edge updates.",
		client,
	)
	researcher.Tools = []tools.Tool{tools.NewSearchWebTool()}
	researcher.Verbose = true

	writer := agents.NewAgent(
		"Writer",
		"Write a blog post based on the research",
		"You are a professional tech blogger.",
		client,
	)
	
	// Inject the new HITL Guardrail
	// The Go thread will synchronously Pause until the user clicks "Approve" in the Dashboard
	writer.Guardrails = []guardrails.Guardrail{
		guardrails.NewHumanReviewGuardrail("Writer", "Final Draft Publisher"),
	}

	task1 := &tasks.Task{
		Description: "Search for Go 1.24 release notes and key features.",
		Agent:       researcher,
	}

	task2 := &tasks.Task{
		Description: "Summarize the findings into a 200-word blog post.",
		Agent:       writer,
		Context:     []*tasks.Task{task1},
	}

	myCrew := crew.Crew{
		Agents:  []*agents.Agent{researcher, writer},
		Tasks:   []*tasks.Task{task1, task2},
		Process: crew.Sequential,
		Verbose: true,
	}

	slog.Info("🚀 Starting Live Demo...")
	_, err := myCrew.Kickoff(context.Background())
	if err != nil {
		slog.Error("Demo failed", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("✅ Demo finished. Keep the dashboard open to review the logs!")
	select {} // Keep running so user can see logs
}
