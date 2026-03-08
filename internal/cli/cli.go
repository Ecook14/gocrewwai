package cli

import (
	"context"
	"fmt"
	"log/slog"
	//"os"
	"os" // Added os import

	"github.com/Ecook14/gocrewwai/pkg/dashboard"
	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	//"time"
)

// printHelp prints the usage instructions
func printHelp() {
	fmt.Println("Gocrew CLI")
	fmt.Println("Usage:")
	fmt.Println("  gocrewwai create [project_name]   - Scaffold a new standard Go AI project")
	fmt.Println("  gocrewwai kickoff                 - Execute the crew pipeline (original demo)") // Kept kickoff for existing demo
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
			fmt.Println("Usage: gocrewwai create [project_name]")
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
		dashboard.Start("8080")
		slog.Info("🖥️  Dashboard available at http://localhost:8080/web-ui")
		slog.Info("⏸️  Execution paused. Please open the dashboard and click 'START' to begin!")
		telemetry.GlobalExecutionController.Pause()
	}

	slog.Info("🚀 Kicking off the Gocrew Go Demo...")

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
		if !showUI {
			return err
		}
		// In UI mode, we log the error but keep the daemon alive
		slog.Warn("⚠️ Initial execution failed, but Creator Mode will remain active.")
	} else {
		slog.Info("✨ Final Output", slog.Any("result", result))
	}
	
	if showUI {
		return c.RunCreatorMode(ctx)
	}
	
	return nil
}
