package tasks

import (
	"fmt"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the schema of a tasks.yaml file
type YAMLConfig struct {
	Description string `yaml:"description"`
	AgentKey    string `yaml:"agent"` // Maps back to the agent config key natively
}

// LoadTasks takes a filepath and a loaded agent map to rebuild the Task structures.
func LoadTasks(filepath string, loadedAgents map[string]*agents.Agent) (map[string]*Task, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("could not read tasks config file: %w", err)
	}

	var rawConfig map[string]YAMLConfig
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse tasks.yaml: %w", err)
	}

	result := make(map[string]*Task)
	for key, conf := range rawConfig {
		// Map the Agent pointer dynamically based on the string key provided in the YAML
		var targetAgent *agents.Agent
		if conf.AgentKey != "" && loadedAgents != nil {
			targetAgent = loadedAgents[conf.AgentKey]
		}

		result[key] = &Task{
			Description: conf.Description,
			Agent:       targetAgent,
		}
	}

	return result, nil
}
