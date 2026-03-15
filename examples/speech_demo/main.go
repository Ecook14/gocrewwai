package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := llm.NewOpenAIClient(apiKey)

	agent := agents.NewAgent("Voice Agent", "Say something inspiring.", "Inspirational speaker", model)

	task := &tasks.Task{
		Description: "Write a 1-sentence inspirational quote.",
		Agent:       agent,
	}

	myCrew := crew.NewCrew([]core.Agent{agent}, []*tasks.Task{task})

	fmt.Println("🚀 Executing Task and generating Speech (Elite Multimodal)...")
	result, _ := myCrew.Kickoff(context.Background())

	fmt.Printf("Agent Result: %s\n", result)

	// Call Elite Speech Method directly for demonstration
	speech, err := model.GenerateSpeech(context.Background(), fmt.Sprintf("%v", result), nil)
	if err != nil {
		fmt.Printf("Speech Generation Error: %v\n", err)
	} else {
		filename := "quote_audio.mp3"
		os.WriteFile(filename, speech, 0644)
		fmt.Printf("✅ Speech generated and saved to %s\n", filename)
	}
}
