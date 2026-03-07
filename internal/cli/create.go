package cli

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// GenerateScaffolding creates a standard boilerplate Crew-GO project in the current directory.
// Mirrors `crewai create crew [name]`
func GenerateScaffolding(projectName string) error {
	baseDir := filepath.Join(".", projectName)

	slog.Info("Scaffolding new Crew-GO project...", slog.String("name", projectName))

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

	"github.com/Ecook14/crewai-go/pkg/config"
	"github.com/Ecook14/crewai-go/pkg/crew"
	"github.com/Ecook14/crewai-go/pkg/llm"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")

	// 1. Load Configurations
	agents, err := config.LoadAgents("config/agents.yaml")
	if err != nil {
		panic(err)
	}

	tasksMap, err := config.LoadTasks("config/tasks.yaml", agents)
	if err != nil {
		panic(err)
	}

	// 2. Bind LLM
	for _, a := range agents {
		a.LLM = llm.NewOpenAIClient(apiKey)
	}

	// 3. Assemble tasks into slice
	var taskList []*tasks.Task
	for _, t := range tasksMap {
		taskList = append(taskList, t)
	}

	var agentList []*agents.Agent
	for _, a := range agents {
		agentList = append(agentList, a)
	}

	myCrew := crew.Crew{
		Agents:  agentList,
		Tasks:   taskList,
		Process: crew.Sequential,
		Verbose: true,
	}

	slog.Info("Starting Boilerplate Crew...")
	res, _ := myCrew.Kickoff(context.Background())
	slog.Info("Finished!", slog.Any("result", res))
}
`
	if err := os.WriteFile(filepath.Join(baseDir, "main.go"), []byte(mainGo), 0644); err != nil {
		return err
	}

	slog.Info("✅ Project successfully scaffolded!", slog.String("path", baseDir))
	return nil
}
