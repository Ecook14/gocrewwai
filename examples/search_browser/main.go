package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	serperKey := os.Getenv("SERPER_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	// Tools
	search := tools.NewSerperTool(serperKey)
	scraper := tools.NewScraperTool()

	researcher := agents.NewAgent(
		"Researcher",
		"Find the latest news about Go 1.25 release.",
		"Curious technology scout",
		model,
		agents.WithTools([]agents.Tool{search, scraper}),
	)

	task := &tasks.Task{
		Description: "Search for 'Go 1.25 release date and features' and summarize the top 3 points.",
		Agent:       researcher,
	}

	myCrew := crew.NewCrew(
		[]*agents.Agent{researcher},
		[]*tasks.Task{task},
		crew.WithVerbose(true),
	)

	fmt.Println("🚀 Starting Web Search & Scrape Demo...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFinal Report:\n%s\n", result)
}
