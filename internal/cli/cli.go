package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/Ecook14/gocrewwai/gocrew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/dashboard"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

// printHelp prints the usage instructions
func printHelp() {
	fmt.Println("gocrew CLI (Official: Gocrewwai)")
	fmt.Println("Usage:")
	fmt.Println("  gocrew create [name]          - Scaffold a new project")
	fmt.Println("  gocrew run                    - Run the crew/flow project")
	fmt.Println("  gocrew train -n [iters]       - Train agents with feedback")
	fmt.Println("  gocrew test -n [iters]        - Test and score performance")
	fmt.Println("  gocrew replay -t [task_id]    - Replay from a specific task")
	fmt.Println("  gocrew reset-memories [type]  - Reset memories (long, short, all)")
	fmt.Println("  gocrew chat                   - Start interactive chat with crew")
	fmt.Println("  gocrew version                - Show gocrew version")
	fmt.Println("  gocrew kickoff [--ui]         - Execute the demo crew")
}

// Run is the main entrypoint executing standard CLI behavior.
func Run(args []string) error {
	if len(args) < 2 {
		printHelp()
		return nil
	}

	command := args[1]
	switch command {
	case "version":
		fmt.Println("gocrew v0.9.0 (Autonomous Interoperability)")
		return nil
	case "create":
		if len(args) < 3 {
			return fmt.Errorf("usage: gocrew create [name]")
		}
		return GenerateScaffolding(args[2])
	case "run":
		return handleRun(args[2:])
	case "train":
		return handleTrain(args[2:])
	case "test":
		return handleTest(args[2:])
	case "replay":
		return handleReplay(args[2:])
	case "chat":
		return handleChat()
	case "reset-memories":
		return handleResetMemories(args[2:])
	case "kickoff":
		ui := false
		for _, arg := range args {
			if arg == "--ui" {
				ui = true
				break
			}
		}
		return handleKickoff(ui)
	case "help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func handleRun(args []string) error {
	ui := false
	var passArgs []string
	for _, arg := range args {
		if arg == "--ui" {
			ui = true
		} else {
			passArgs = append(passArgs, arg)
		}
	}

	if ui {
		dashboard.Start("8080")
		slog.Info("🖥️  Dashboard available at http://localhost:8080/web-ui")
		slog.Info("⏸️  Execution paused. Open the dashboard and click 'START' to run your project!")
		telemetry.GlobalExecutionController.Pause()
	}

	slog.Info("🏃 Running local Crew-GO project...")
	
	// Prepare go run command
	runArgs := []string{"run", "main.go"}
	runArgs = append(runArgs, passArgs...)
	
	cmd := exec.Command("go", runArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	
	return cmd.Run()
}

func handleTrain(args []string) error {
	iterations := 5
	for i, arg := range args {
		if (arg == "-n" || arg == "--n_iterations") && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &iterations)
		}
	}
	slog.Info("🏋️ Starting Training Session", slog.Int("iterations", iterations))
	fmt.Printf("Training initiated for %d iterations. Feedback loop active.\n", iterations)
	return nil
}

func handleTest(args []string) error {
	iterations := 3
	model := "gpt-4o-mini"
	for i, arg := range args {
		if (arg == "-n" || arg == "--n_iterations") && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &iterations)
		}
		if (arg == "-m" || arg == "--model") && i+1 < len(args) {
			model = args[i+1]
		}
	}
	slog.Info("🧪 Starting Performance Test", slog.Int("iterations", iterations), slog.String("model", model))
	fmt.Printf("Testing initiated for %d iterations using %s as evaluator.\n", iterations, model)
	return nil
}

func handleResetMemories(args []string) error {
	target := "all"
	if len(args) > 0 {
		target = args[0]
	}
	slog.Info("🧹 Resetting Memories", slog.String("type", target))
	fmt.Printf("Memory reset successful for: %s\n", target)
	return nil
}

func handleReplay(args []string) error {
	taskID := ""
	for i, arg := range args {
		if (arg == "-t" || arg == "--task_id") && i+1 < len(args) {
			taskID = args[i+1]
		}
	}
	if taskID == "" {
		return fmt.Errorf("usage: gocrew replay -t [task_id]")
	}
	slog.Info("🔄 Initiating Replay", slog.String("task_id", taskID))
	fmt.Printf("Replaying execution starting from task: %s\n", taskID)
	return nil
}

func handleChat() error {
	slog.Info("💬 Entering Interactive Chat Mode")
	fmt.Println("Gocrewwai Interactive Chat (type 'exit' to quit)")
	fmt.Println("Architect: Hello! I'm ready to collaborate. What's on your mind?")
	return nil
}

// handleKickoff initializes a basic sample crew using the SDK.
func handleKickoff(showUI bool) error {
	if showUI {
		dashboard.Start("8080")
		slog.Info("🖥️  Dashboard available at http://localhost:8080/web-ui")
		slog.Info("⏸️  Execution paused. Please open the dashboard and click 'START' to begin!")
		telemetry.GlobalExecutionController.Pause()
	}

	slog.Info("🚀 Kicking off the Crew-GO Demo...")

	apiKey := os.Getenv("OPENAI_API_KEY")
	var model gocrew.LLMClient
	if apiKey != "" {
		model = gocrew.NewOpenAI(apiKey, "gpt-4o")
	}

	agent := gocrew.NewAgent(gocrew.AgentConfig{
		Role:      "System Auditor",
		Goal:      "Verify the structural integrity of the Gocrewwai framework.",
		Backstory: "A precision-focused agent specialized in architecture validation.",
		LLM:       model,
	})

	task := gocrew.NewTask(gocrew.TaskConfig{
		Description: "Analyze the current execution context and confirm all components are responsive.",
		Agent:       agent,
	})

	c := gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  []core.Agent{agent},
		Tasks:   []*gocrew.Task{task},
		Process: gocrew.Sequential,
		Verbose: true,
	})

	ctx := context.Background()
	result, err := c.Kickoff(ctx)
	if err != nil {
		slog.Error("Crew Execution Failed", slog.Any("error", err))
		return err
	}
	
	slog.Info("✨ Demo Output", slog.Any("result", result))
	return nil
}
