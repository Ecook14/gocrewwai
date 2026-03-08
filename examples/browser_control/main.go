package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/crew"
	"github.com/Ecook14/crewai-go/pkg/llm"
	"github.com/Ecook14/crewai-go/pkg/tasks"
	"github.com/Ecook14/crewai-go/pkg/tools"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	browser := tools.NewBrowserTool()

	browserAgent := agents.NewAgent(
		"Web Automator",
		"Browse the internet purposefully.",
		"An agent that can use a real browser.",
		model,
		agents.WithTools([]agents.Tool{browser}),
	)

	task := &tasks.Task{
		Description: "Navigate to 'https://news.ycombinator.com', find the top story, and return its title.",
		Agent:       browserAgent,
	}

	myCrew := crew.NewCrew(
		[]*agents.Agent{browserAgent},
		[]*tasks.Task{task},
		crew.WithVerbose(true),
	)

	fmt.Println("🚀 Starting Browser Control Demo (Requires Chrome/Chromium installed)...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nBrowser result: %s\n", result)
}
