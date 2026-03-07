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
	"github.com/Ecook14/crewai-go/internal/server"
)

// printHelp prints the usage instructions
func printHelp() {
	fmt.Println("Crew-GO CLI")
	fmt.Println("Usage:")
	fmt.Println("  crewai create [project_name]   - Scaffold a new standard Go AI project")
	fmt.Println("  crewai kickoff                 - Execute the crew pipeline (original demo)") // Kept kickoff for existing demo
}

// Run is the main entrypoint executing standard CLI behavior.
func Run(args []string) error {
	if len(args) < 2 {
		printHelp()
		return nil
	}

	command := args[1]
	switch command {
	case "create":
		if len(args) < 3 {
			fmt.Println("Usage: crewai create [project_name]")
			os.Exit(1)
		}
		projectName := args[2]
		slog.Info("Initializing Elite Project Scaffolding", slog.String("project_name", projectName))
		if err := GenerateScaffolding(projectName); err != nil {
			slog.Error("Scaffolding failed", slog.Any("error", err))
			os.Exit(1)
		}
		return nil
	case "kickoff":
		ui := false
		for _, arg := range args {
			if arg == "--ui" {
				ui = true
				break
			}
		}
		return handleKickoff(ui)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// handleKickoff initializes a basic sample crew to prove the architecture compiles
func handleKickoff(showUI bool) error {
	if showUI {
		go server.StartDashboardServer("8080")
		slog.Info("🖥️  Dashboard available at http://localhost:8080/web-ui")
	}

	slog.Info("🚀 Kicking off the CrewAI Go Demo...")

	agent := &agents.Agent{
		Role:      "Architect",
		Goal:      "Ensure system stability",
		Backstory: "A highly logical bot designed to confirm Go structures.",
		Verbose:   true,
		// Elite Architecture Verification: Unbound LLM used for structural validation.
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

	slog.Info("✨ Final Output", slog.Any("result", result))
	return nil
}
