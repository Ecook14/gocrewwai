package main

import (
	"context"
	"fmt"
	"log"

	// Single unified import! 🦾
	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	// ============================================================
	// OPTION 1: Pure Go (Library Style)
	// ============================================================

	gpt4 := gocrew.NewOpenAI("your-api-key", "gpt-4o")

	researcher := gocrew.NewAgent(gocrew.AgentConfig{
		Role:       "Senior Researcher",
		Goal:       "Find cutting-edge AI papers",
		Backstory:  "PhD in ML with 10 years of experience",
		LLM:        gpt4,
		Verbose:    true,
		InjectDate: true,
		Reasoning:  true,
	})

	writer := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Technical Writer",
		Goal:      "Write clear, engaging blog posts",
		Backstory: "Award-winning tech journalist",
		LLM:       gpt4,
	})

	researchTask := gocrew.NewTask(gocrew.TaskConfig{
		Description:    "Research the latest breakthroughs in AI agents",
		ExpectedOutput: "A bullet-point summary of top 5 findings",
		Agent:          researcher,
	})

	writeTask := gocrew.NewTask(gocrew.TaskConfig{
		Description:     "Write a blog post based on the research",
		ExpectedOutput:  "A 1000-word blog post in Markdown",
		Agent:           writer,
		Markdown:        true,
		OutputFile:      "./output/blog.md",
		CreateDirectory: true,
		Context:         []*gocrew.Task{researchTask},
	})

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []*gocrew.Agent{researcher, writer},
		Tasks:   []*gocrew.Task{researchTask, writeTask},
		Process: gocrew.Sequential,
		Verbose: true,
		MaxRPM:  10,
	})

	result, err := myCrew.Kickoff(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)

	// ============================================================
	// OPTION 2: YAML Config (Zero Code Definition)
	// ============================================================

	// yamlCrew, err := gocrew.LoadFromYAML("./crew.yaml")
	// if err != nil {
	//     log.Fatal(err)
	// }
	// result, err := yamlCrew.Kickoff(context.Background())
}
