package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	serperKey := os.Getenv("SERPER_API_KEY")
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	search := gocrew.NewSerperTool(serperKey)
	scraper := gocrew.NewScraperTool()

	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Researcher",
		Goal:      "Find the latest news about Go 1.25 release.",
		Backstory: "Curious technology scout",
		LLM:       model,
		Tools:     []gocrew.Tool{search, scraper},
	})

	task := &gocrew.Task{
		Description: "Search for 'Go 1.25 release date and features' and summarize the top 3 points.",
		Agent:       researcher,
	}

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher},
		Tasks:   []*gocrew.Task{task},
		Verbose: true,
	})

	fmt.Println("🚀 Starting Web Search & Scrape Demo...")
	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nFinal Report:\n%s\n", result)
}
