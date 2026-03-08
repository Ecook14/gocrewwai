package config

import (
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"gopkg.in/yaml.v3"
)

// TaskConfig represents the schema of a tasks.yaml file
type TaskConfig struct {
	Description string `yaml:"description"`
	AgentKey    string `yaml:"agent"` // Maps back to the agent config key natively
}

// LoadTasks takes a filepath and a loaded agent map to rebuild the Task structures.
func LoadTasks(filepath string, loadedAgents map[string]*agents.Agent) (map[string]*tasks.Task, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("could not read tasks config file: %w", err)
	}

	var rawConfig map[string]TaskConfig
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse tasks.yaml: %w", err)
	}

	result := make(map[string]*tasks.Task)
	for key, conf := range rawConfig {
		// Map the Agent pointer dynamically based on the string key provided in the YAML
		var targetAgent *agents.Agent
		if conf.AgentKey != "" && loadedAgents != nil {
			targetAgent = loadedAgents[conf.AgentKey]
		}

		result[key] = &tasks.Task{
			Description: conf.Description,
			Agent:       targetAgent,
		}
	}

	return result, nil
}
