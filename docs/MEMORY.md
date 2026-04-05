# Memory in Gocrewwai 🧠

Memory is the persistent state of your agents and crews. Gocrewwai includes an advanced memory system that allows agents to learn from past experiences, store user preferences, and recall relevant facts across different tasks.

## 🏗️ Memory Types

Gocrewwai implements three distinct memory systems, each optimized for a specific role:

### 1. Short-Term Memory (Contextual)
Powered by the LLM's own context window, this memory allows agents to remember details during the execution of a single task or within a crew's current run.

### 2. Long-Term Memory (Persistent)
This system stores processed insights and general knowledge in a persistent database (SQLite, Redis, etc.), allowing agents to recall information from previous crew executions.

### 3. Entity Memory
A specialized system for tracking and recalling specific details about actors (users, agents, companies) across long periods.

## 🚀 Persistent Memory Stores

Gocrewwai supports various high-performance vector-indexed stores through the `memory.Store` interface.

| Store | SDK Constructor | Key Features |
| :--- | :--- | :--- |
| **SQLite** | `gocrew.NewSQLiteStore` | Perfect for local desktop and lightweight apps. |
| **Redis** | `gocrew.NewRedisStore` | Distributed, low-latency state sharing. |
| **Chroma** | `gocrew.NewChromaStore` | Dedicated vector database for advanced RAG. |
| **Qdrant** | `gocrew.NewQdrantStore` | High-scale, cloud-native vector storage. |

## 🛠️ Using Memory in an Agent

Simply provide a store instance to the `AgentConfig`:

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

---

[Back to Tools Guide](./TOOLS.md) | [Next: Flows Guide](./FLOWS.md)
