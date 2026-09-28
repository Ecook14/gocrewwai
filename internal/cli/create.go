package cli

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GenerateScaffolding creates a standard boilerplate Gocrew project in the current directory.
// Mirrors `gocrew create [name]`. projectName is validated to a single path
// segment (no traversal, no absolute paths).
func GenerateScaffolding(projectName string) error {
	clean := filepath.Clean(projectName)
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.Contains(clean, "..") || strings.ContainsRune(clean, '/') || strings.ContainsRune(clean, '\\') {
		return fmt.Errorf("invalid project name %q: must be a single directory name", projectName)
	}
	if len(clean) == 0 || len(clean) > 64 {
		return fmt.Errorf("invalid project name %q: length must be 1-64 chars", projectName)
	}
	baseDir := filepath.Join(".", clean)

	slog.Info("Scaffolding new Gocrew project...", slog.String("name", projectName))

	dirs := []string{
		filepath.Join(baseDir, "src"),
		filepath.Join(baseDir, "config"),
		filepath.Join(baseDir, "tools"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed creating directory %s: %w", d, err)
		}
	}

	// 1. Write agents.yaml
	agentsYaml := `designer:
  role: "Lead Software Designer"
  goal: "Architect scalable Go solutions based on requirements."
  backstory: "You are a senior engineer who favors interfaces and clean architecture."
  verbose: true
  sandbox: "docker"
  tools:
    - name: "BrowserTool"
      params:
        timeout: 60
`
	if err := os.WriteFile(filepath.Join(baseDir, "config", "agents.yaml"), []byte(agentsYaml), 0644); err != nil {
		return err
	}

	// 2. Write tasks.yaml
	tasksYaml := `design_task:
  description: "Review the initial user requirements and output a system architecture document."
  agent: "designer"
`
	if err := os.WriteFile(filepath.Join(baseDir, "config", "tasks.yaml"), []byte(tasksYaml), 0644); err != nil {
		return err
	}

	// 3. Write .env template (key placeholder — user must supply their own)
	envFile := `#OPENAI_API_KEY=sk-...  # Replace with your actual key
`
	if err := os.WriteFile(filepath.Join(baseDir, ".env"), []byte(envFile), 0644); err != nil {
		return err
	}

	// 4. Write main.go (subcommand-aware: kickoff/train/test/replay/chat)
	mainGo := `package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/Ecook14/gocrewwai/gocrew"
	"github.com/Ecook14/gocrewwai/pkg/config"
)

func mustLoadCrew(apiKey string) *gocrew.Crew {
	// 1. Load Configurations (using the specialized loaders)
	agentsMap, err := config.LoadAgents("config/agents.yaml")
	if err != nil {
		slog.Error("failed to load agents", slog.Any("error", err))
		os.Exit(1)
	}

	tasksMap, err := config.LoadTasks("config/tasks.yaml", agentsMap)
	if err != nil {
		slog.Error("failed to load tasks", slog.Any("error", err))
		os.Exit(1)
	}

	// 2. Bind LLM to Agents
	model := gocrew.NewOpenAI(apiKey, "gpt-4o")
	for _, a := range agentsMap {
		a.LLM = model
	}

	// 3. Assemble
	var agentList []gocrew.Agent
	for _, a := range agentsMap {
		agentList = append(agentList, a)
	}

	var taskList []*gocrew.Task
	for _, t := range tasksMap {
		taskList = append(taskList, t)
	}

	return gocrew.NewCrew(gocrew.CrewConfig{
		Agents:  agentList,
		Tasks:   taskList,
		Process: gocrew.Sequential,
		Verbose: true,
	})
}

func main() {
	ctx := context.Background()
	apiKey := os.Getenv("OPENAI_API_KEY")

	cmd := "kickoff"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "train":
		fs := flag.NewFlagSet("train", flag.ExitOnError)
		n := fs.Int("n", 5, "training iterations")
		_ = fs.Parse(os.Args[2:])
		if *n <= 0 || *n > 100 {
			slog.Error("invalid iterations: must be 1-100")
			os.Exit(1)
		}
		myCrew := mustLoadCrew(apiKey)
		slog.Info("🏋️ Training crew", slog.Int("iterations", *n))
		if err := myCrew.Train(ctx, *n, nil); err != nil {
			slog.Error("training failed", slog.Any("error", err))
			os.Exit(1)
		}
	case "test":
		fs := flag.NewFlagSet("test", flag.ExitOnError)
		n := fs.Int("n", 3, "test iterations")
		m := fs.String("m", "", "judge model for LLM-scored evaluation (default: pass-count smoke test)")
		_ = fs.Parse(os.Args[2:])
		if *n <= 0 || *n > 100 {
			slog.Error("invalid iterations: must be 1-100")
			os.Exit(1)
		}
		if *m != "" {
			// LLM-judged evaluation against the first task's expected output.
			myCrew := mustLoadCrew(apiKey)
			rubric := "general quality"
			if len(myCrew.Tasks) > 0 && myCrew.Tasks[0].ExpectedOutput != "" {
				rubric = myCrew.Tasks[0].ExpectedOutput
			}
			judge := gocrew.NewOpenAI(apiKey, *m)
			suite, err := myCrew.Test(ctx, judge, *n, rubric)
			if err != nil {
				slog.Error("evaluation failed", slog.Any("error", err))
				os.Exit(1)
			}
			fmt.Printf("test: avg score %.2f, pass rate %.2f over %d runs\n", suite.AverageScore, suite.PassRate, *n)
			break
		}
		passed := 0
		for i := 0; i < *n; i++ {
			myCrew := mustLoadCrew(apiKey)
			if _, err := myCrew.Kickoff(ctx); err != nil {
				slog.Warn("iteration failed", slog.Int("i", i+1), slog.Any("error", err))
				continue
			}
			passed++
		}
		fmt.Printf("test: %d/%d iterations passed\n", passed, *n)
		if passed == 0 {
			os.Exit(1)
		}
	case "replay":
		fs := flag.NewFlagSet("replay", flag.ExitOnError)
		taskID := fs.String("t", "", "task ID to replay from")
		_ = fs.Parse(os.Args[2:])
		if *taskID == "" {
			slog.Error("usage: main.go replay -t [task_id]")
			os.Exit(1)
		}
		myCrew := mustLoadCrew(apiKey)
		res, err := myCrew.Replay(ctx, *taskID)
		if err != nil {
			slog.Error("replay failed", slog.Any("error", err))
			os.Exit(1)
		}
		slog.Info("🏁 Replay finished!", slog.Any("result", res))
	case "chat":
		myCrew := mustLoadCrew(apiKey)
		if len(myCrew.Agents) == 0 {
			slog.Error("no agents in crew")
			os.Exit(1)
		}
		fmt.Println("Gocrewwai Chat (type 'exit' to quit)")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 1024), 64*1024)
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if line == "exit" {
				break
			}
			res, err := myCrew.Agents[0].Execute(ctx, line, nil)
			if err != nil {
				slog.Warn("agent error", slog.Any("error", err))
				continue
			}
			fmt.Printf("%v\n", res)
		}
	default: // kickoff
		myCrew := mustLoadCrew(apiKey)
		slog.Info("🚀 Starting Crew-GO Scaffolded Project...")
		res, err := myCrew.Kickoff(ctx)
		if err != nil {
			slog.Error("Crew execution failed", slog.Any("error", err))
			os.Exit(1)
		}
		slog.Info("🏁 Finished!", slog.Any("result", res))
	}
}
`
	if err := os.WriteFile(filepath.Join(baseDir, "main.go"), []byte(mainGo), 0644); err != nil {
		return err
	}

	// 5. Elite Hardening: Automatic module initialization
	slog.Info("Running 'go mod init'...", slog.String("project", projectName))
	initCmd := exec.Command("go", "mod", "init", projectName)
	initCmd.Dir = baseDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run go mod init: %v (output: %s)", err, string(out))
	}

	slog.Info("Running 'go mod tidy' to fetch dependencies...")
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = baseDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run go mod tidy: %v (output: %s)", err, string(out))
	}

	slog.Info("✅ Project successfully scaffolded and hardened!", slog.String("path", baseDir))
	return nil
}
