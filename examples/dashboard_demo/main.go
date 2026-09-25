package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Ecook14/gocrewwai/gocrew"
	"github.com/Ecook14/gocrewwai/pkg/dashboard"
)

func main() {
	dashboard.Start("8080")
	fmt.Println("🖥️  Dashboard active at http://localhost:8080/web-ui")
	fmt.Println("Please open the dashboard in your browser before the crew starts!")

	time.Sleep(5 * time.Second)

	apiKey := os.Getenv("OPENAI_API_KEY")
	client := gocrew.NewOpenAI(apiKey, "gpt-4o")

	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Researcher",
		Goal:      "Find the latest news about Go 1.24",
		Backstory: "You are a tech journalist looking for cutting-edge updates.",
		LLM:       client,
		Tools:     []gocrew.Tool{gocrew.NewBrowserTool()},
		Verbose:   true,
	})

	writer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Writer",
		Goal:      "Write a blog post based on the research",
		Backstory: "You are a professional tech blogger.",
		LLM:       client,
		Verbose:   true,
	})

	writer.Guardrails = []gocrew.Guardrail{gocrew.NewHumanReviewGuardrail("Writer", "Final Draft Publisher")}

	task1 := &gocrew.Task{
		Description: "Search for Go 1.24 release notes and key features.",
		Agent:       researcher,
	}

	task2 := &gocrew.Task{
		Description: "Summarize the findings into a 200-word blog post.",
		Agent:       writer,
		Context:     []*gocrew.Task{task1},
	}

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{researcher, writer},
		Tasks:   []*gocrew.Task{task1, task2},
		Verbose: true,
	})

	fmt.Println("🚀 Starting Live Demo...")
	_, err := myCrew.Kickoff(context.Background())
	if err != nil {
		fmt.Printf("Demo failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Demo finished. Keep the dashboard open to review the logs!")
	select {}
}
