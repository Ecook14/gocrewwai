# Feature: Advanced Memory Architectures

## Overview
Agents without memory suffer from repetitive amnesia. They cannot recall what happened in a previous task, nor can they learn facts about the user over time.

Crew-GO solves this natively through an extremely robust interfaces inside `pkg/memory`.

## Tiered Memory Types

1. **Short-Term Memory**
   - Active only during the current execution of a specific Task.
   - If an agent enters a 10-loop ReAct cycle, it uses short-term memory to keep the context window small while referencing exactly what tool it just called 5 seconds ago.

2. **Long-Term Memory**
   - Persists infinitely across executions and application restarts.
   - When given a new task, the Agent calculates a Vector Embedding of the task description, queries the `memory.Store` using Cosine Similarity, and natively injects the top results directly into the LLM System Prompt as `--- RELEVANT PAST CONTEXT ---`.

3. **Entity Memory (Elite)**
   - While Long-Term memory stores giant walls of text, Entity Memory stores extracted **Facts**.
   - If a task result says "The CEO of Acme Corp is John Doe and he hates emails", the Agent uses an internal LLM extraction schema to transform that into JSON: `[{"entity": "Acme Corp CEO", "value": "John Doe", "description": "Hates emails"}]`.
   - This prevents key facts from being lost in generic RAG searches.

## Supported Backends out-of-the-box
- **In-Memory** (Testing & Development)
- **SQLite** (Single-binary persistent deployments)
- **Redis** (High-speed distributed deployments)
- **ChromaDB, Weaviate, Pinecone, Qdrant** (Enterprise vector databases)

## Initialization Example
```go
// Connect to a local ChromaDB instance
chroma, err := memory.NewChromaStore("http://localhost:8000")

// Give the agent total recall
analyst := agents.NewAgentBuilder().
    Role("Data Analyst").
    Memory(chroma). // RAG Memory
    Build()
```
