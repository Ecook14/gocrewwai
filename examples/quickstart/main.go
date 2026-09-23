package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Ecook14/gocrewwai/gocrew"
)

func main() {
	ctx := context.Background()

	// Example: create agents, tasks, and run a crew
	agent := gocrew.NewAgent(gocrew.AgentConfig{
		Role:  "assistant",
		Goal:  "A helpful assistant",
	})

	task := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Respond with a greeting",
	})

	crew := gocrew.NewCrew(gocrew.CrewConfig{
		Agents: []gocrew.CoreAgent{agent},
		Tasks:  []*gocrew.Task{task},
	})

	result, err := crew.Kickoff(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Result:", result)
}
