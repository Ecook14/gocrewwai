package cli

import (
	"context"
	"fmt"
	"log/slog"
	//"os"
	"os" // Added os import

	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/crew"
	"github.com/Ecook14/crewai-go/pkg/tasks"
)

// printHelp prints the usage instructions
func printHelp() {
	fmt.Println("Crew-GO CLI")
	fmt.Println("Usage:")
	fmt.Println("  crew-go create [project_name]   - Scaffold a new standard Go AI project")
	fmt.Println("  crew-go kickoff                 - Execute the crew pipeline (original demo)") // Kept kickoff for existing demo
}

// Run is the main entrypoint executing standard CLI behavior.
func Run(args []string) error { // Kept original signature for now, will adapt body
	if len(args) < 2 {
		printHelp() // Use new help function
		return nil
	}

	command := args[1]
	switch command {
	case "create":
		if len(args) < 3 {
			fmt.Println("Usage: crew-go create [project_name]")
			os.Exit(1) // Use os.Exit for CLI errors
		}
		projectName := args[2]
		// Assuming GenerateScaffolding is defined elsewhere or will be added.
		// For now, we'll just print a message.
		slog.Info("Attempting to generate scaffolding", slog.String("project_name", projectName))
		// Placeholder for actual scaffolding logic
		if err := GenerateScaffolding(projectName); err != nil { // Assuming this function exists
			slog.Error("Scaffolding failed", slog.Any("error", err))
			os.Exit(1)
		}
		return nil
	case "kickoff": // Kept original kickoff command
		return handleKickoff()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// handleKickoff initializes a basic sample crew to prove the architecture compiles
// and connects to the terminal successfully just like Python's `crewai run`.
func handleKickoff() error {
	slog.Info("🚀 Kicking off the CrewAI Go Demo...")

	agent := &agents.Agent{
		Role:      "Architect",
		Goal:      "Ensure system stability",
		Backstory: "A highly logical bot designed to confirm Go structures.",
		Verbose:   true,
		// LLM left unbound. By default Agent execute falls back to mock logic string.
	}

	task := &tasks.Task{
		Description: "Verify the Go translation",
		Agent:       agent,
	}

	c := crew.Crew{
		Process: crew.Sequential,
		Agents:  []*agents.Agent{agent},
		Tasks:   []*tasks.Task{task},
		Verbose: true,
	}

	ctx := context.Background()
	result, err := c.Kickoff(ctx)
	if err != nil {
		slog.Error("Crew Execution Failed", slog.Any("error", err))
		return err
	}

	fmt.Printf("\n✨ Final Output:\n%v\n", result)
	return nil
}
