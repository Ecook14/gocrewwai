package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ecook14/gocrewwai/gocrew"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/dashboard"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/Ecook14/gocrewwai/pkg/version"
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
	fmt.Println("  gocrew deploy [--out DIR]     - Build release binaries into DIR")
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
		fmt.Println("gocrew " + version.Display() + " (Autonomous Interoperability)")
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
	case "deploy":
		return handleDeploy(args[2:])
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", runArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Do not leak the full environment (which may contain secrets like API keys)
	// to child processes. Only pass PATH and essential Go toolchain vars.
	var safeEnv []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") ||
			strings.HasPrefix(e, "GOPATH=") ||
			strings.HasPrefix(e, "GOROOT=") ||
			strings.HasPrefix(e, "GOPROXY=") ||
			strings.HasPrefix(e, "GOSUMDB=") ||
			strings.HasPrefix(e, "GOFLAGS=") ||
			strings.HasPrefix(e, "GOMODCACHE=") {
			safeEnv = append(safeEnv, e)
		}
	}
	cmd.Env = safeEnv

	return cmd.Run()
}

func handleTrain(args []string) error {
	iterations := 5
	for i, arg := range args {
		if (arg == "-n" || arg == "--n_iterations") && i+1 < len(args) {
			var n int
			if _, err := fmt.Sscanf(args[i+1], "%d", &n); err != nil || n <= 0 || n > 100 {
				return fmt.Errorf("invalid iterations %q: must be 1-100", args[i+1])
			}
			iterations = n
		}
	}
	slog.Info("Starting Training Session", slog.Int("iterations", iterations))
	// Execute against the project crew: the project owns its config/agents.
	return runProject([]string{"train", "-n", fmt.Sprintf("%d", iterations)})
}

func handleTest(args []string) error {
	iterations := 3
	model := "gpt-4o-mini"
	for i, arg := range args {
		if (arg == "-n" || arg == "--n_iterations") && i+1 < len(args) {
			var n int
			if _, err := fmt.Sscanf(args[i+1], "%d", &n); err != nil || n <= 0 || n > 100 {
				return fmt.Errorf("invalid iterations %q: must be 1-100", args[i+1])
			}
			iterations = n
		}
		if (arg == "-m" || arg == "--model") && i+1 < len(args) {
			if strings.TrimSpace(args[i+1]) == "" || len(args[i+1]) > 64 {
				return fmt.Errorf("invalid model %q", args[i+1])
			}
			model = args[i+1]
		}
	}
	slog.Info("Starting Performance Test", slog.Int("iterations", iterations), slog.String("model", model))
	return runProject([]string{"test", "-n", fmt.Sprintf("%d", iterations), "-m", model})
}

func handleResetMemories(args []string) error {
	target := "all"
	store := ""
	conn := ""
	for i, arg := range args {
		switch arg {
		case "--store":
			if i+1 < len(args) {
				store = args[i+1]
			}
		case "--conn":
			if i+1 < len(args) {
				conn = args[i+1]
			}
		default:
			if !strings.HasPrefix(arg, "-") && target == "all" && store == "" {
				target = arg
			}
		}
	}
	if target != "all" && target != "long" && target != "short" {
		return fmt.Errorf("invalid memory type %q: must be all|long|short", target)
	}
	_ = target // reserved for scoped reset once stores expose scopes
	if store == "" {
		return fmt.Errorf("usage: gocrew reset-memories [all|long|short] --store sqlite --conn <path>")
	}
	if store != "sqlite" {
		return fmt.Errorf("unsupported store %q: only sqlite reset is supported", store)
	}
	if conn == "" || len(conn) > 256 || filepath.IsAbs(conn) || strings.Contains(conn, "..") || strings.ContainsAny(conn, `/\`) {
		return fmt.Errorf("invalid --conn %q: sqlite basename only", conn)
	}
	slog.Info("Resetting Memories", slog.String("store", store), slog.String("conn", conn))
	st, err := memory.NewSQLiteStore(conn)
	if err != nil {
		return fmt.Errorf("failed to open sqlite store: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := st.Reset(ctx); err != nil {
		return fmt.Errorf("memory reset failed: %w", err)
	}
	fmt.Printf("Memory reset successful for sqlite store: %s\n", conn)
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
	if len(taskID) > 128 || strings.ContainsAny(taskID, "/\\..") {
		return fmt.Errorf("invalid task_id %q", taskID)
	}
	slog.Info("Initiating Replay", slog.String("task_id", taskID))
	// Execute against the project crew, which owns checkpoints/state.
	return runProject([]string{"replay", "-t", taskID})
}

func handleChat() error {
	slog.Info("Entering project chat mode")
	// The project owns its agents/config; delegate the interactive loop.
	return runProject([]string{"chat"})
}

// handleDeploy builds release binaries (gocrew CLI + server) into --out DIR
// (default ./dist). It validates the tree looks like gocrewwai (go.mod +
// cmd/) and fails closed otherwise. Webhook triggers for managed hosting
// remain roadmap; binary + Dockerfile artifacts are the deploy unit.
func handleDeploy(args []string) error {
	out := "./dist"
	for i, arg := range args {
		if arg == "--out" && i+1 < len(args) {
			out = args[i+1]
		}
		if strings.HasPrefix(arg, "--out=") {
			out = strings.TrimPrefix(arg, "--out=")
		}
	}
	if out == "" || len(out) > 256 || strings.Contains(out, "..") {
		return fmt.Errorf("invalid --out %q", out)
	}
	for _, need := range []string{"go.mod", "cmd/gocrew", "cmd/server", "Dockerfile"} {
		if _, err := os.Stat(need); err != nil {
			return fmt.Errorf("deploy must run from the gocrewwai repo root: missing %s", need)
		}
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return fmt.Errorf("failed to create out dir: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	targets := map[string]string{
		"./cmd/gocrew": "gocrew",
		"./cmd/server": "gocrewwai-server",
	}
	for pkg, bin := range targets {
		dest := filepath.Join(out, bin)
		cmd := exec.CommandContext(ctx, "go", "build", "-o", dest, pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		slog.Info("building deploy artifact", slog.String("pkg", pkg), slog.String("out", dest))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("build %s failed: %w", pkg, err)
		}
	}
	fmt.Printf("deploy artifacts ready in %s/ (gocrew, gocrewwai-server) + Dockerfile\n", out)
	return nil
}

// runProject executes `go run main.go <args>` in the current directory with a
// bounded context and a secrets-stripped environment. Fails closed when there
// is no gocrew project here.
func runProject(args []string) error {
	if _, err := os.Stat("main.go"); err != nil {
		return fmt.Errorf("not a gocrew project directory: main.go not found")
	}
	for _, a := range args {
		if a == "" || len(a) > 512 {
			return fmt.Errorf("invalid project arg %q", a)
		}
	}
	runArgs := append([]string{"run", "main.go"}, args...)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", runArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	var safeEnv []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PATH=") ||
			strings.HasPrefix(e, "GOPATH=") ||
			strings.HasPrefix(e, "GOROOT=") ||
			strings.HasPrefix(e, "GOPROXY=") ||
			strings.HasPrefix(e, "GOSUMDB=") ||
			strings.HasPrefix(e, "GOFLAGS=") ||
			strings.HasPrefix(e, "GOMODCACHE=") ||
			strings.HasPrefix(e, "OPENAI_API_KEY=") ||
			strings.HasPrefix(e, "ANTHROPIC_API_KEY=") ||
			strings.HasPrefix(e, "GEMINI_API_KEY=") ||
			strings.HasPrefix(e, "GROQ_API_KEY=") ||
			strings.HasPrefix(e, "OPENROUTER_API_KEY=") ||
			strings.HasPrefix(e, "CREW_CONFIG_PATH=") {
			safeEnv = append(safeEnv, e)
		}
	}
	cmd.Env = safeEnv
	return cmd.Run()
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
