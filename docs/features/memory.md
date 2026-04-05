# Feature Deep Dive: Memory ⚓🧠

Memory is the persistent state of your agents and crews. Gocrewwai includes an advanced memory system that allows agents to learn from past experiences, store user preferences, and recall relevant facts across different tasks.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai memory systems support **Recency + Relevance** scoring and persistent vector storage via SQLite, Redis, and Chroma.

---

## 🏗️ Memory Types

Gocrewwai implements three distinct memory systems, each optimized for a specific role:

### 1. Short-Term Memory (Contextual)
Powered by the LLM's own context window, this memory allows agents to remember details during the execution of a single task or within a crew's current run.

### 2. Long-Term Memory (Persistent)
This system stores processed insights and general knowledge in a persistent database (SQLite, Redis, etc.), allowing agents to recall information from previous crew executions.

### 3. Entity Memory
A specialized system for tracking and recalling specific details about actors (users, agents, companies) across long periods.

---

## 🚀 Persistent Memory Stores (Elite Style)

Using the `gocrew` SDK, you can initialize and assign vector stores to your agents with simple declarative configuration.

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    // 1. Initialize Persistent Memory
    sqlite, _ := gocrew.NewSQLiteStore("agent_memory.db")
    defer sqlite.Close()

    // 2. Assign Memory to Agent
    researcher := gocrew.NewAgent(gocrew.AgentConfig{
        Role:      "Memory-Enabled Researcher",
        Memory:    sqlite, // That's it! 
    })
}
```

## 🧠 Memory Orchestration

When an agent executes a task, it automatically:
1.  **Searches**: Performs a vector search in its memory for relevant past context.
2.  **Appends**: Injects findings into its prompt.
3.  **Learns**: After the task is complete, it summarizes the experience and stores it as a new memory item for future recall.

## 🛡️ Recency & Relevance Scoring

Gocrewwai uses a sophisticated scoring engine to ensure that only the most relevant memories are injected into the agent's prompt. This prevents context clutter and ensures the agent always reflects on its most pertinent past experiences.

---

[Back to LLMs Guide](./llms.md) | [Next: Knowledge](./knowledge.md)
