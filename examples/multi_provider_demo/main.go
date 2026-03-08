package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

func main() {
	ctx := context.Background()

	// 1. Initialize different providers
	// You can set these in your environment, e.g., export ANTHROPIC_API_KEY=sk-...
	claude := llm.NewAnthropicClient(os.Getenv("ANTHROPIC_API_KEY"), "claude-3-5-sonnet-20240620")
	gpt4o := llm.NewOpenAIClient(os.Getenv("OPENAI_API_KEY"))
	gemini := llm.NewGeminiClient(os.Getenv("GOOGLE_API_KEY"), "gemini-1.5-pro")
	groq := llm.NewGroqClient(os.Getenv("GROQ_API_KEY"), "llama3-70b-8192")

	fmt.Println("🤖 Multi-Provider Crew Initializing...")

	// 2. Assign different providers to different agents
	researcher := agents.NewAgentBuilder().
		Role("Deep Researcher").
		Goal("Uncover latest breakthrough in room-temperature superconductors.").
		LLM(claude). // Using Claude for deep reasoning
		Verbose(true).
		Build()

	analyst := agents.NewAgentBuilder().
		Role("Technical Analyst").
		Goal("Evaluate the commercial viability of research findings.").
		LLM(gpt4o). // Using OpenAI for standard analysis
		Verbose(true).
		Build()

	writer := agents.NewAgentBuilder().
		Role("Creative Writer").
		Goal("Write an engaging summary for a tech newsletter.").
		LLM(gemini). // Using Gemini for creative long-form synthesis
		Verbose(true).
		Build()

	fastChecker := agents.NewAgentBuilder().
		Role("Fact Checker").
		Goal("Verify specific dates and names in the summary.").
		LLM(groq). // Using Groq for lightning-fast verification
		Verbose(true).
		Build()

	// 3. Define Tasks
	task1 := tasks.NewTaskBuilder().
		Description("Research recent LK-99 developments or similar 2026 breakthroughs.").
		Agent(researcher).
		Build()

	task2 := tasks.NewTaskBuilder().
		Description("Analyze the market impact based on the findings.").
		Agent(analyst).
		Context(task1).
		Build()

	task3 := tasks.NewTaskBuilder().
		Description("Synthesize research and analysis into a 200-word newsletter.").
		Agent(writer).
		Context(task1, task2).
		Build()

	task4 := tasks.NewTaskBuilder().
		Description("Perform a final ultra-fast fact check on names and numbers.").
		Agent(fastChecker).
		Context(task3).
		Build()

	// 4. Create and Kickoff the Crew
	multiCrew := crew.NewCrewBuilder().
		Agents(researcher, analyst, writer, fastChecker).
		Tasks(task1, task2, task3, task4).
		Process(crew.Sequential).
		Verbose(true).
		Build()

	fmt.Println("🚀 Kicking off Multi-Provider Crew...")
	result, err := multiCrew.Kickoff(ctx)
	if err != nil {
		log.Fatalf("Crew failed: %v", err)
	}

	fmt.Printf("\n--- FINAL MULTI-PROVIDER OUTPUT ---\n%v\n", result)
}
