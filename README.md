# CrewAI Go (Crew-GO) 🚀

The official Go implementation of the CrewAI framework. Designed for engineers who want to build multi-agent AI applications natively in Go, bringing the elegance of CrewAI's Python orchestrator to high-performance statically-typed architectures.

## Features

* **Agents**: Independent actors powered by Go Context-aware execution, ReAct Autonomous Tool-Calling, and LLM prompting.
* **Tasks**: Declarative assignments mapped strictly to Agents with explicit output routing context pipes.
* **Crews**: Orchestration engines with synchronous (Sequential) and asynchronous WaitGroup routing.
* **Tools**: Extensible interfaces allowing agents to scrape, read, and autonomously act on external data.
* **Memory & Knowledge**: Built-in SQLite local vector stores, automatic document chunking for RAG, and MD5 LLM Response caching.
* **Flows**: Event-driven state machine pipelines utilizing native Go Channels (`chan`).
* **YAML Configs**: Effortlessly unmarshal `agents.yaml` and `tasks.yaml` using `gopkg.in/yaml.v3` directly into execution structs.

## Documentation

* ⚡️ **[Quickstart Guide](./docs/quickstart.md)**: Setup your first agent and run simple Sequential pipelines.
* 🔥 **[Advanced Usage & Parsing](./docs/advanced_usage.md)**: Learn how to leverage native Go Concurrency pools, Task Context Timeouts, and JSON Schema extraction models.

## Installation

1. **Install the package directly to your module**:

```bash
go get github.com/Ecook14/crewai-go
```

2. **(Optional) Install the CLI Tool for boilerplate scaffolding**:

```bash
go install github.com/Ecook14/crewai-go/cmd/crew-go@latest
```

## Quickstart

Building an AI application in Go is just as simple as in Python. Check out the `examples/researcher_app` folder for a real demo!

```go
package main

import (
	"context"
	"fmt"
	"github.com/Ecook14/crewai-go/pkg/agents"
	"github.com/Ecook14/crewai-go/pkg/crew"
	"github.com/Ecook14/crewai-go/pkg/tasks"
)

func main() {
	// 1. Create an Agent
	researcher := &agents.Agent{
		Role:      "Researcher",
		Goal:      "Find the answer",
		Backstory: "Expert investigator.",
	}

	// 2. Assign a Task
	task := &tasks.Task{
		Description: "Investigate Go integrations",
		Agent:       researcher,
	}

	// 3. Assemble and Kickoff the Crew
	c := crew.Crew{
		Process: crew.Sequential,
		Agents:  []*agents.Agent{researcher},
		Tasks:   []*tasks.Task{task},
	}

	result, _ := c.Kickoff(context.Background())
	fmt.Println(result)
}
```

## Running the Interactive Examples

You can run the built-in examples straight from the terminal. The `researcher_app` features dynamic JSON output mapping and live OpenAI network bindings.

```shell
```shell
cd Crew-GO
OPENAI_API_KEY="your-sk-key" go run examples/advanced_app/main.go
```

## Acknowledgements & Thanks 🙏

`Crew-GO` is thoroughly and deeply inspired by the pioneering work of **João Moura** and the entire core team behind the original [CrewAI](https://github.com/crewAIInc/crewAI) Python framework. 

Their visionary orchestration patterns, the ReAct loop models, and overall framework design completely reshaped the way developers build multi-agent LLM systems. This Go SDK exists to compliment their ecosystem by offering a statically typed, high-performance, and natively concurrent alternative for systems engineers who require strictly compiled backend deployments while maintaining the elegance of the original CrewAI interface logic. 

Thank you to the Python community for leading the charge in Agentic workflows!
