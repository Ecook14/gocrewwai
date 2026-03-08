package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/tools"
	"gopkg.in/yaml.v3"
)

// ToolConfig represents a tool and its key-value settings in YAML.
type ToolConfig struct {
	Name   string                 `yaml:"name"`
	Params map[string]interface{} `yaml:"params"`
}

// AgentConfig represents the schema of an agents.yaml file
type AgentConfig struct {
	Role      string       `yaml:"role"`
	Goal      string       `yaml:"goal"`
	Backstory string       `yaml:"backstory"`
	Verbose   bool         `yaml:"verbose,omitempty"`
	Sandbox   string       `yaml:"sandbox,omitempty"` // "local", "docker", "e2b", "wasm"
	Tools     []ToolConfig `yaml:"tools"`
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
		var agentTools []tools.Tool
		for _, tc := range conf.Tools {
			t, err := tools.CreateTool(tc.Name, tc.Params)
			if err != nil {
				slog.Warn("Failed to load tool for agent", slog.String("agent", key), slog.String("tool", tc.Name), slog.Any("error", err))
				continue
			}
			agentTools = append(agentTools, t)
		}

		result[key] = &agents.Agent{
			Role:      conf.Role,
			Goal:      conf.Goal,
			Backstory: conf.Backstory,
			Verbose:   conf.Verbose,
			Sandbox:   conf.Sandbox,
			Tools:     agentTools,
		}
	}

	return result, nil
}
