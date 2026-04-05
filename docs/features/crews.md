# Feature Deep Dive: Crews ⚓⛴️

A Crew is the core orchestration unit in Gocrewwai. It represents a collaborative group of agents working together to solve a sequence of tasks using a specific execution strategy.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai crews support **Sequential**, **Hierarchical**, and **Graph-based** orchestration with native **OpenTelemetry** tracing.

---

## 🏗️ The Crew Config (Elite Style)

In Gocrewwai v1.0, crews are constructed using the `CrewConfig` struct, providing a clean, declarative interface.

```go
myCrew := gocrew.NewCrew(gocrew.CrewConfig{
    Agents:      []gocrew.CoreAgent{researcher, writer},
    Tasks:       []*gocrew.Task{researchTask, writingTask},
    Process:     gocrew.Sequential,
    ManagerLLM:  gpt4o, // Required only for Hierarchical orchestration
    Verbose:     true,
    Planning:    true,  // Enable pre-execution task planning
})

result, err := myCrew.Kickoff(ctx)
```

### Key Parameters

| Parameter | Type | Description |
| :--- | :--- | :--- |
| **Agents** | `[]CoreAgent` | The slice of agents participating in the crew. |
| **Tasks** | `[]*Task` | The sequence of tasks to be performed. |
| **Process** | `ProcessType` | The orchestration strategy (Sequential, Hierarchical, Graph). |
| **ManagerLLM** | `LLMClient` | The LLM used for hierarchical delegation and synthesis. |
| **Verbose** | `bool` | Enables detailed logging of the crew's orchestration steps. |
| **MaxRPM** | `int` | Global rate limit to prevent API throttling during execution. |
| **Planning** | `bool` | Enables a pre-execution planning phase to optimize task division. |

---

## 🧩 Orchestration Processes

### ⚖️ Sequential (Default)
Agents work in a pre-defined order, passing their results to the next task's assigned agent. This is best for linear workflows.

### 👑 Hierarchical
A **Manager Agent** is automatically created to orchestrate the team. The manager delegates tasks dynamically based on agent goals and reviews the output of each agent before moving on.

### 📈 Graph
Utilizes a dynamic state machine to determine the next step based on real-time task results. This enables non-linear workflows and cyclic loops.

---

## 📊 Crew Observability

Every crew execution in Gocrewwai is automatically instrumented with **OpenTelemetry**. This includes:
1. **Trace Spans**: Every agent thought, tool execution, and task handoff is captured as a span.
2. **Token Metrics**: Total token usage and cost are aggregated across the entire crew run.
3. **Execution Metadata**: Detailed logs of inputs, outputs, and internal reasoning.

---

[Back to index](../index.md) | [Next: Processes](./processes.md)
