# Feature Deep Dive: Advanced Memory Architectures 🧠

Hey! Let's talk about memory. 

If you've played with basic AI chatbots, you know their biggest flaw: they suffer from total amnesia. The second you start a new prompt, they forget everything you told them yesterday.

For enterprise-grade autonomous agents, that's unacceptable. If our agents are doing research or writing code, they need to recall past lessons, remember user preferences, and reference previous task outputs. 

Gocrew solves this natively through a highly robust set of interfaces inside `pkg/memory`.

---

## 📚 The Three Tiers of Memory

We've built a three-tiered memory architecture to give our Go agents total recall without blowing up the LLM token context window.

### 1. Short-Term Memory (Contextual)
- **What it is:** The immediate scratchpad.
- **How it works:** When an agent enters a 10-loop ReAct reasoning cycle, it uses short-term memory to remember exactly what tool it just called 5 seconds ago. This prevents the agent from getting stuck in an infinite loop of calling the exact same failing Google search over and over.

### 2. Long-Term Memory (RAG / Vector Stores)
- **What it is:** Persistent, infinite storage across executions and application restarts.
- **How it works:** When you hand the agent a new Task, the Engine calculates a mathematical Vector Embedding of the task description in the background. It then natively queries your attached `memory.Store` using Cosine Similarity, grabs the most relevant past experiences, and seamlessly injects them directly into the LLM System Prompt.

### 3. Entity Memory (High Precision)
- **What it is:** Structured database facts. Long-Term memory stores giant vague walls of text. Entity Memory stores exact *Key-Value concepts*.
- **How it works:** If an agent reads a 100-page document and notices a fact like "The CEO of Acme Corp is John Doe and he hates emails", the Agent uses an internal JSON extraction schema to transform that into clean structured data: `[{"entity": "Acme Corp CEO", "value": "John Doe", "description": "Hates emails"}]`.
- This ensures absolute precision. When the agent later sees the word "Acme Corp", it pulls that exact JSON object into context natively.

---

## 💾 Supported Memory Databases Out-Of-The-Box

Gocrew provides native Go adapters for the most popular vector and KV stores in the world.

- **In-Memory** (`NewInMemCosineStore`): Perfect for fast local testing and development.
- **SQLite** (`NewSQLiteStore`): Amazing for single-binary persistent deployments that you don't want to spin up Docker for!
- **Redis** (`NewRedisStore`): Crucial for high-speed distributed deployments if you are running Crews across multiple Kubernetes pods.
- **ChromaDB, Pinecone, Qdrant**: Best-in-class, enterprise vector databases designed for massive RAG scale.

---

## 💻 Code Example

Giving an agent infinite memory is as easy as passing it a database connection!

```go
// 1. Connect to a local ChromaDB instance
chromaStore, err := memory.NewChromaStore("http://localhost:8000")
if err != nil {
    log.Fatal("Failed to connect to memory:", err)
}

// 2. Give the agent total recall!
analyst := agents.NewAgentBuilder().
    Role("Data Analyst").
    Goal("Analyze trends").
    Memory(chromaStore). // They are now hooked up to the DB!
    Build()
```

---

## 🤝 Help Me Build New Integrations!

Vector database technology moves incredibly fast. Do you have a favorite database that we don't support yet, like **Milvus**, **Weaviate**, or even a standard **Postgres pgvector** adapter?

Because we used clean interfaces throughout `pkg/memory`, adding a new database takes less than 100 lines of Go code. 

**Please consider mapping your favorite database and submitting a Pull Request!** Let's make sure our agents can remember anything, anywhere.
