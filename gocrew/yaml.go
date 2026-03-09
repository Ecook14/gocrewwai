package gocrew

import (
	"fmt"
	"os"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"gopkg.in/yaml.v3"
)

// ============================================================
// YAML Configuration Types
// ============================================================

// YAMLConfig is the top-level structure for YAML-based crew definitions.
type YAMLConfig struct {
	LLM    YAMLLLMConfig    `yaml:"llm"`
	Agents []YAMLAgent      `yaml:"agents"`
	Tasks  []YAMLTask       `yaml:"tasks"`
	Crew   YAMLCrewConfig   `yaml:"crew"`
}

// YAMLLLMConfig defines the default LLM for the crew.
type YAMLLLMConfig struct {
	Provider string `yaml:"provider"` // "openai", "anthropic", "gemini", "groq", "openrouter"
	Model    string `yaml:"model"`
	APIKey   string `yaml:"api_key"`   // Can be env var like ${OPENAI_API_KEY}
}

// YAMLAgent defines an agent in YAML.
type YAMLAgent struct {
	Role               string   `yaml:"role"`
	Goal               string   `yaml:"goal"`
	Backstory          string   `yaml:"backstory"`
	Verbose            bool     `yaml:"verbose"`
	AllowDelegation    bool     `yaml:"allow_delegation"`
	AllowCodeExecution bool     `yaml:"allow_code_execution"`
	MaxIter            int      `yaml:"max_iter"`
	MaxRPM             int      `yaml:"max_rpm"`
	InjectDate         bool     `yaml:"inject_date"`
	Reasoning          bool     `yaml:"reasoning"`
	Tools              []string `yaml:"tools"` // Tool names from registry
	LLM                *YAMLLLMConfig `yaml:"llm,omitempty"` // Override default LLM
}

// YAMLTask defines a task in YAML.
type YAMLTask struct {
	Name           string   `yaml:"name"`
	Description    string   `yaml:"description"`
	ExpectedOutput string   `yaml:"expected_output"`
	AgentRole      string   `yaml:"agent"`      // Match by role name
	Context        []string `yaml:"context"`     // List of task names for context
	OutputFile     string   `yaml:"output_file"`
	Markdown       bool     `yaml:"markdown"`
	AsyncExecution bool     `yaml:"async_execution"`
	HumanInput     bool     `yaml:"human_input"`
}

// YAMLCrewConfig defines crew-level settings in YAML.
type YAMLCrewConfig struct {
	Process  string `yaml:"process"`  // "sequential", "hierarchical", etc.
	Verbose  bool   `yaml:"verbose"`
	MaxRPM   int    `yaml:"max_rpm"`
	Planning bool   `yaml:"planning"`
}

// ============================================================
// YAML Loader
// ============================================================

// LoadFromYAML reads a YAML config file and returns a fully assembled Crew.
func LoadFromYAML(path string) (*Crew, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML config: %w", err)
	}

	var cfg YAMLConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return buildCrewFromYAML(&cfg)
}

// LoadFromYAMLString parses a YAML string and returns a fully assembled Crew.
func LoadFromYAMLString(yamlStr string) (*Crew, error) {
	var cfg YAMLConfig
	if err := yaml.Unmarshal([]byte(yamlStr), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return buildCrewFromYAML(&cfg)
}

func buildCrewFromYAML(cfg *YAMLConfig) (*Crew, error) {
	// 1. Build default LLM
	defaultLLM, err := buildLLM(&cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to build default LLM: %w", err)
	}

	// 2. Build Agents
	agentMap := make(map[string]*agents.Agent)
	agentList := make([]*agents.Agent, 0, len(cfg.Agents))

	for _, ya := range cfg.Agents {
		agentLLM := defaultLLM
		if ya.LLM != nil {
			agentLLM, err = buildLLM(ya.LLM)
			if err != nil {
				return nil, fmt.Errorf("failed to build LLM for agent %s: %w", ya.Role, err)
			}
		}

		agent := agents.New(agents.AgentConfig{
			Role:               ya.Role,
			Goal:               ya.Goal,
			Backstory:          ya.Backstory,
			LLM:                agentLLM,
			Verbose:            ya.Verbose,
			AllowDelegation:    ya.AllowDelegation,
			AllowCodeExecution: ya.AllowCodeExecution,
			MaxIterations:      ya.MaxIter,
			MaxRPM:             ya.MaxRPM,
			InjectDate:         ya.InjectDate,
			Reasoning:          ya.Reasoning,
		})

		agentMap[strings.ToLower(strings.TrimSpace(ya.Role))] = agent
		agentList = append(agentList, agent)
	}

	// 3. Build Tasks
	taskMap := make(map[string]*tasks.Task)
	taskList := make([]*tasks.Task, 0, len(cfg.Tasks))

	for _, yt := range cfg.Tasks {
		// Find agent by role
		var taskAgent *agents.Agent
		if yt.AgentRole != "" {
			key := strings.ToLower(strings.TrimSpace(yt.AgentRole))
			if a, ok := agentMap[key]; ok {
				taskAgent = a
			}
		}

		task := tasks.New(tasks.TaskConfig{
			Name:           yt.Name,
			Description:    yt.Description,
			ExpectedOutput: yt.ExpectedOutput,
			Agent:          taskAgent,
			AgentRole:      yt.AgentRole,
			OutputFile:     yt.OutputFile,
			Markdown:       yt.Markdown,
			AsyncExecution: yt.AsyncExecution,
			HumanInput:     yt.HumanInput,
			CreateDirectory: true,
		})

		if yt.Name != "" {
			taskMap[strings.ToLower(strings.TrimSpace(yt.Name))] = task
		}
		taskList = append(taskList, task)
	}

	// 4. Wire task contexts (second pass)
	for i, yt := range cfg.Tasks {
		if len(yt.Context) > 0 {
			for _, ctxName := range yt.Context {
				key := strings.ToLower(strings.TrimSpace(ctxName))
				if ctxTask, ok := taskMap[key]; ok {
					taskList[i].Context = append(taskList[i].Context, ctxTask)
				}
			}
		}
	}

	// 5. Build Crew
	processType := crew.Sequential
	switch strings.ToLower(cfg.Crew.Process) {
	case "hierarchical":
		processType = crew.Hierarchical
	case "consensual":
		processType = crew.Consensual
	case "graph":
		processType = crew.Graph
	case "reflective":
		processType = crew.Reflective
	case "state_machine":
		processType = crew.StateMachine
	}

	return crew.New(crew.CrewConfig{
		Agents:  agentList,
		Tasks:   taskList,
		Process: processType,
		Verbose: cfg.Crew.Verbose,
		MaxRPM:  cfg.Crew.MaxRPM,
		Planning: cfg.Crew.Planning,
	}), nil
}

func buildLLM(cfg *YAMLLLMConfig) (llm.Client, error) {
	apiKey := resolveEnvVar(cfg.APIKey)

	switch strings.ToLower(cfg.Provider) {
	case "openai":
		return NewOpenAI(apiKey, cfg.Model), nil
	case "anthropic", "claude":
		return NewAnthropic(apiKey, cfg.Model), nil
	case "gemini", "google":
		return NewGemini(apiKey, cfg.Model), nil
	case "groq":
		return NewGroq(apiKey, cfg.Model), nil
	case "openrouter":
		return NewOpenRouter(apiKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}

// resolveEnvVar replaces ${VAR_NAME} patterns with environment variable values.
func resolveEnvVar(s string) string {
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		envKey := s[2 : len(s)-1]
		if val := os.Getenv(envKey); val != "" {
			return val
		}
	}
	return s
}
