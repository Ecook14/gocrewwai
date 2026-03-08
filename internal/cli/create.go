package cli

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

// GenerateScaffolding creates a standard boilerplate Gocrew project in the current directory.
// Mirrors `crewai create crew [name]`
func GenerateScaffolding(projectName string) error {
	baseDir := filepath.Join(".", projectName)

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

	// 3. Write .env
	envFile := `OPENAI_API_KEY=sk-your-key-here
`
	if err := os.WriteFile(filepath.Join(baseDir, ".env"), []byte(envFile), 0644); err != nil {
		return err
	}

	// 4. Write main.go
	mainGo := `package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/config"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/dashboard"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")

	// 1. Load Configurations
	agentsMap, err := config.LoadAgents("config/agents.yaml")
	if err != nil {
		panic(err)
	}

	tasksMap, err := config.LoadTasks("config/tasks.yaml", agentsMap)
	if err != nil {
		panic(err)
	}

	// 2. Bind LLM
	for _, a := range agentsMap {
		a.LLM = llm.NewOpenAIClient(apiKey)
	}

	// 3. Assemble tasks and agents
	var taskList []*tasks.Task
	for _, t := range tasksMap {
		taskList = append(taskList, t)
	}

	var agentList []*agents.Agent
	for _, a := range agentsMap {
		agentList = append(agentList, a)
	}

	myCrew := crew.Crew{
		Agents:  agentList,
		Tasks:   taskList,
		Process: crew.Sequential,
		Verbose: true,
	}

	// 4. Start the Dashboard (Background)
	dashboard.Start("8080")

	slog.Info("Starting Boilerplate Gocrew...")
	res, err := myCrew.Kickoff(context.Background())
	if err != nil {
		slog.Error("Crew execution failed", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Finished!", slog.Any("result", res))
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
