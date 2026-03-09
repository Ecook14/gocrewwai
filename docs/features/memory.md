# Feature Deep Dive: Unified Memory 🧠

Memory allows Gocrew agents to learn over time, recall past interactions, and maintain context across complex, multi-stage workflows.

---

## 🏗️ The Multi-Tiered Architecture

Gocrew uses a three-tier memory system inspired by human cognition:

1. **Short-Term Memory**: The agent's current "Thought Stream" and recent task history.
2. **Long-Term Memory**: Persistent storage (Vector DB) for recalling facts across different sessions and crews.
3. **Entity Memory**: A specialized store for remembering specific details about objects, people, or concepts (e.g., "User prefers Python over Go").

---

## 🛠️ Composite Scoring Logic

Unlike simple vector search, Gocrew uses a **Composite Scoring** algorithm to decide *which* memory to recall:

| Metric | Description |
| :--- | :--- |
| **Recency** | How recently the memory was recorded. |
| **Relevance** | Semantic similarity to the current task. |
| **Importance** | How "critical" the agent flagged the memory during storage. |

**Final Score = (w1 * Recency) + (w2 * Relevance) + (w3 * Importance)**

---

## 💾 Supported Backends

Gocrew is database-agnostic. You can configure your memory store during crew setup:
- **Local**: SQLite (default), In-Memory.
- **Cloud**: Pinecone, Qdrant, Weaviate, Redis.

```go
agent := gocrew.NewAgentBuilder().
    Memory(true).
    MemoryStore(gocrew.NewSQLiteStore("memory.db")).
    Build()
```

---

## 🔄 Memory Lifecycle

1. **Observe**: The agent performs a task.
2. **Flag**: The agent (or engine) identifies "memorable" facts.
3. **Store**: Facts are vectorized and assigned an importance score.
4. **Recall**: During the next task, relevant memories are retrieved and injected into the "Context" window.

---
**Gocrew** - Building agents that truly learn.
