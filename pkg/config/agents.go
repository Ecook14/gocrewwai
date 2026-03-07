package config

import (
	"fmt"
	"os"

	"github.com/Ecook14/crewai-go/pkg/agents"
	"gopkg.in/yaml.v3"
)

// AgentConfig represents the schema of an agents.yaml file
type AgentConfig struct {
	Role      string `yaml:"role"`
	Goal      string `yaml:"goal"`
	Backstory string `yaml:"backstory"`
	Verbose   bool   `yaml:"verbose,omitempty"`
}

// LoadAgents takes a filepath to an agents.yaml and returns initialized pointers.
func LoadAgents(filepath string) (map[string]*agents.Agent, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("could not read agents config file: %w", err)
	}

	var rawConfig map[string]AgentConfig
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse agents.yaml: %w", err)
	}

	result := make(map[string]*agents.Agent)
	for key, conf := range rawConfig {
		result[key] = &agents.Agent{
			Role:      conf.Role,
			Goal:      conf.Goal,
			Backstory: conf.Backstory,
			Verbose:   conf.Verbose,
		}
	}

	return result, nil
}
