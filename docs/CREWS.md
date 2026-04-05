# Crews in Gocrewwai ⛴️

A Crew represents a collaborative group of agents working together to solve a sequence of tasks. It is the core orchestration unit that defines how work is shared and reviewed.

## 🏗️ Crew Anatomy

In Gocrewwai, a Crew is defined by its agents, tasks, and its orchestration process.

| Field | Type | Description |
| :--- | :--- | :--- |
| **Agents** | `[]CoreAgent` | The slice of agents participating in the crew. |
| **Tasks** | `[]*Task` | The sequence of tasks to be performed. |
| **Process** | `ProcessType` | The orchestration strategy (Sequential, Hierarchical, etc.). |
| **ManagerLLM** | `LLMClient` | The LLM used for hierarchical delegation and synthesis. |
| **Verbose** | `bool` | Enables detailed logging of the crew's orchestration steps. |
| **MaxRPM** | `int` | Global rate limit to prevent API throttling during execution. |
| **Planning** | `bool` | Enables a pre-execution planning phase to optimize task division. |

## 🚀 Creating a Crew (Elite Style)

Using the `gocrew` SDK, you can create crews using a declarative configuration:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    myCrew := gocrew.NewCrew(gocrew.CrewConfig{
        Agents:  []gocrew.CoreAgent{researcher, writer},
        Tasks:   []*gocrew.Task{task1, task2},
        Process: gocrew.Sequential,
        Verbose: true,
    })

    result, err := myCrew.Kickoff(ctx)
}
```

## 🧠 Orchestration Processes

### ⚖️ Sequential (Default)
Agents work in a pre-defined order, passing their results to the next task's assigned agent. This is best for linear workflows.

### 👑 Hierarchical
A **Manager Agent** is automatically created (or assigned via `ManagerAgent`) to orchestrate the team. The manager:
1.  **Delegates**: Assigns tasks dynamically based on agent goals.
2.  **Reviews**: Validates the output of each agent before moving on.
3.  **Synthesizes**: Aggregates all results into a final unified answer.

### 📈 Graph
Utilizes a dynamic state machine to determine the next step based on real-time task results. This enables non-linear workflows and cyclic loops (e.g., self-correction).

---

[Back to Tasks Guide](./TASKS.md) | [Next: LLM Providers Guide](./LLM_PROVIDERS.md)
