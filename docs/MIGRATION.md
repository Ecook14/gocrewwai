# Migration Guide: Moving to Gocrewwai 🔄

Welcome to the performant future of agentic orchestration. This guide will help you migrate your existing applications from CrewAI, LangChain, or LangGraph to the Gocrewwai framework.

## 🏗️ Why Migrate to Gocrewwai?

| Feature | Python Frameworks (CrewAI, LangChain) | Gocrewwai (Go) |
| :--- | :--- | :--- |
| **Performance** | Significant overhead from interpreted Python. | Native machine code with superior execution speed. |
| **Concurrency** | Limited by Global Interpreter Lock (GIL). | Native Goroutines for effortless, high-scale parallelization. |
| **Deployment** | Large Docker images, complex dependencies. | Single, small, static binary with zero dependencies. |
| **Type Safety** | Primarily dynamic typing (Python). | Strong compile-time safety and typed orchestration. |

## 🚀 Migrating from CrewAI

Gocrewwai shares many core concepts with CrewAI, making the transition straightforward.

| CrewAI Concept | Gocrewwai Equivalent |
| :--- | :--- |
| `Agent` | `gocrew.Agent` |
| `Task` | `gocrew.Task` |
| `Crew` | `gocrew.Crew` |
| `Process.sequential` | `gocrew.Sequential` |
| `Process.hierarchical` | `gocrew.Hierarchical` |
| `Tool` | `gocrew.Tool` (Go interface) |

### 🛠️ Code Example: CrewAI vs. Gocrewwai

**Before (CrewAI - Python):**
```python
researcher = Agent(role='Researcher', goal='Find info', backstory='Expert...')
task = Task(description='Research AI', agent=researcher)
crew = Crew(agents=[researcher], tasks=[task], process=Process.sequential)
crew.kickoff()
```

**After (Gocrewwai - Go):**
```go
researcher := gocrew.NewAgent(gocrew.AgentConfig{Role: "Researcher", Goal: "Find info", Backstory: "Expert..."})
task := gocrew.NewTask(gocrew.TaskConfig{Description: "Research AI", Agent: researcher})
crew := gocrew.NewCrew(gocrew.CrewConfig{Agents: []gocrew.CoreAgent{researcher}, Tasks: []*gocrew.Task{task}, Process: gocrew.Sequential})
crew.Kickoff(ctx)
```

## 🧠 Migrating from LangGraph

If you're coming from LangGraph, you'll find Gocrewwai's **Typed Flows** to be a more intuitive and type-safe experience.

| LangGraph Concept | Gocrewwai Equivalent |
| :--- | :--- |
| `StateGraph` | `gocrew.TypedFlow[T]` |
| `Nodes` | `flow.Node` functions |
| `Checkpointer` | `persistence.Store` (SQLite/Redis) |
| `interrupt_before` | `flow.AddInterrupt()` |

---

[Back to tool error handling guide](./TOOL_ERROR_HANDLING.md) | [Back to index](./index.md)
