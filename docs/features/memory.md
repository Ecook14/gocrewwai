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

### 4. Unified Memory (The Elite Tier)
In Gocrewwai v1.0, we introduce **UnifiedMemory**, which orchestrates all three systems via a single `Remember/Recall/Forget` API. It automatically handles vector embedding, scoring, and context injection.

---

## 🚀 Persistent Memory Stores (Elite Style)

Using the `gocrew` SDK, you can initialize and assign vector stores to your agents with simple declarative configuration.

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    // Option 1: SQLite (Local & Fast)
    localStore, _ := gocrew.NewSQLiteStore("agent_memory.db")
    defer localStore.Close()

    // Option 2: Redis (Distributed & Fast)
    redisStore, _ := gocrew.NewRedisStore(gocrew.RedisConfig{
        Address:  "localhost:6379",
        Password: "secure_password",
        DB:       0,
        Index:    "crew_memory_idx",
    })
    defer redisStore.Close()

    // Option 3: Pinecone (Cloud Enterprise)
    pineconeStore := gocrew.NewPineconeStore(
        os.Getenv("PINECONE_API_KEY"), 
        "https://my-index-xxxx.svc.pinecone.io",
    )

    // Assign Memory to Agent
    researcher := gocrew.NewAgent(gocrew.AgentConfig{
        Role:      "Memory-Enabled Researcher",
        Memory:    pineconeStore, // Swappable!
    })
}
```

## 🧠 Memory Orchestration (Under the Hood)

When an agent executes a task, it automatically interacts with the `UnifiedMemory` core. You generally don't need to call these methods manually, but this is what happens during the ReAct loop:

1. **`Recall(ctx, query)`**: Before thought generation, the agent performs a similarity search.
   ```go
   memories, _ := agent.Memory.Recall(ctx, task.Description)
   for _, mem := range memories {
       fmt.Printf("Recalled: %s (Score: %.2f)\n", mem.Content, mem.Score)
   }
   ```
2. **Execution**: The agent runs the task with injected context.
3. **`Remember(ctx, item)`**: Upon task completion, the output and tool results are mapped.
   ```go
   agent.Memory.Remember(ctx, memory.Item{
       Content:   finalResult,
       Metadata:  map[string]string{"task_id": "123", "role": "researcher"},
   })
   ```

## 🛡️ Recency & Relevance Scoring

Gocrewwai uses a sophisticated scoring engine inside the `Recall` method. It combines the cosine similarity of the vector search (Relevance) with a time-decay factor (Recency) to ensure that only the most pertinent and timely memories are injected into the agent's prompt.

---

[Back to LLMs Guide](./llms.md) | [Next: Knowledge](./knowledge.md)
