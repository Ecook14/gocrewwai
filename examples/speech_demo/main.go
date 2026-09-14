package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")

	agent := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "Voice Agent",
		Goal:      "Say something inspiring.",
		Backstory: "Inspirational speaker",
		LLM:       model,
	})

	task := &gocrew.Task{
		Description: "Write a 1-sentence inspirational quote.",
		Agent:       agent,
	}

	myCrew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []gocrew.CoreAgent{agent},
		Tasks:   []*gocrew.Task{task},
	})

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
