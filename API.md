# Crew-GO API Reference

Gocrewwai provides two primary ways to interact with the engine: a strictly-typed **Go SDK** for library-first development, and a **Remote API** for HTTP/WebSocket-based orchestration.

---

## 🏗️ Go SDK (`gocrew` Facade)

The `gocrew` package provides a unified, ergonomic entry point for the entire framework. It uses type aliases and convenient constructors to minimize imports.

### Core Handlers

#### `NewAgent(cfg AgentConfig) *Agent`
Creates a new autonomous agent.
- **AgentConfig**: See [Agents Feature Guide](./docs/features/agents.md) for full field reference.

#### `NewTask(cfg TaskConfig) *Task`
Defines a specific unit of work.
- **TaskConfig**: See [Tasks Feature Guide](./docs/features/tasks.md).

#### `NewCrew(cfg CrewConfig) *Crew`
Assembles agents and tasks into an orchestrated team.
- **CrewConfig**: See [Crews Feature Guide](./docs/features/crews.md).

#### `Kickoff(ctx context.Context, cfg CrewConfig) (interface{}, error)`
Convenience function to create and execute a crew in one call. If the final task specifies an `OutputJSON` struct, the returned `interface{}` contains that populated struct.

To extract the struct safely, use the generic `GetOutput[T]` helper:
```go
result, _ := myCrew.Kickoff(ctx)
parsedStruct := gocrew.GetOutput[MyCustomStruct](result)
fmt.Println(parsedStruct.Field)
```

### Memory & Persistence

#### `NewMemory(store MemoryStore, llm LLMClient, cfg *UnifiedMemoryConfig) *UnifiedMemory`
Initializes the reactive memory system.

#### `NewSQLiteStore(path string) (*SQLiteStore, error)`
Creates a persistent local memory store.

### LLM Providers

| Constructor | Description |
| :--- | :--- |
| `NewOpenAI(key, model)` | OpenAI / GPT service. |
| `NewAnthropic(key, model)` | Anthropic / Claude service. |
| `NewGemini(key, model)` | Google Gemini service. |
| `NewGroq(key, model)` | Groq high-speed inference. |
| `NewOpenRouter(key, model)` | Multi-model routing via OpenRouter. |

---

## 🔌 Remote API (HTTP/WebSocket)

The `Crew-GO` engine exposes a lightweight HTTP and WebSocket server (typically running on `localhost:8080`) when the `--ui` flag is provided or when `server.StartDashboardServer()` is invoked manually.

This server provides real-time telemetry streaming and bi-directional control over the executing Go orchestrator.

---

## 🔌 WebSocket Endpoints

### `WS /ws`
The core telemetry delivery stream. The web dashboard connects to this endpoint to receive ultra-low latency, real-time events emitted by the `GlobalBus` in the Go backend.

**Expected Message Flow:**
The server pushes JSON-encoded `telemetry.Event` objects.
```json
{
  "timestamp": "2024-03-08T15:04:05Z",
  "type": "agent_started",
  "agent_role": "Senior Data Scientist",
  "payload": {
    "task": "Analyze data trends"
  }
}
```

---

## 🛑 Execution Control Endpoints

These endpoints interact directly with `telemetry.GlobalExecutionController` to pause or resume the underlying Go goroutines orchestrating the Crew.

### `POST /api/start`
Resumes the paused Crew execution engine, unblocking the `sync.Cond` condition variables.

**Response:**
```json
{ "status": "started" }
```

### `POST /api/stop`
Suspends the Crew execution. Agents currently thinking or executing block on their next ReAct iteration loop until `/api/start` is called.

**Response:**
```json
{ "status": "stopped" }
```

---

## 🧑‍⚖️ Human-in-the-Loop (HITL) Endpoints

These endpoints manage the asynchronous human approval flow for sensitive tool usage.

### `POST /api/review`
Submits a human approval or rejection for a pending `HumanReviewGuardrail` block. When an agent attempts an action requiring review, it pauses and emits a `review_requested` event over WebSocket. This endpoint resolves that pause.

**Request Body:**
```json
{
  "review_id": "uuid-string-provided-in-ws-event",
  "approved": true
}
```

**Response:**
```json
{ "status": "ok" }
```

---

## 🏗️ Entity Creation Endpoints (Authoring APIs)

These new endpoints allow external clients to dynamically stage and instantiate architectural configuration objects (Agents, Tasks) in the Go engine's memory.

### `POST /api/create/agent`
Stages a new AI Agent entity dynamically.

**Request Body:**
```json
{
  "role": "Lead Researcher",
  "goal": "Uncover hidden patterns in documentation",
  "backstory": "A meticulous librarian turned AI data scientist...",
  "provider": "openrouter",
  "llm_model": "anthropic/claude-3-opus",
  "api_key": "sk-or-...",
  "memory": "remote_(rest/grpc)",
  "memory_config": {
    "connection_string": "https://api.pinecone.io"
  },
  "tools": ["Google Search", "Arxiv Search"]
}
```

**Parameters:**
- **role**: The agent's functional title.
- **goal**: The primary objective.
- **backstory**: Contextual grounding.
- **provider**: `openai`, `anthropic`, `gemini`, `openrouter`, `ollama`, or `custom`.
- **llm_model**: Name of the model.
- **api_key**: (Optional) Service-specific API key.
- **memory**: `sqlite_(local)`, `pinecone_(vector)`, `remote_(rest/grpc)`, etc.
- **memory_config**: Object containing `connection_string` for remote stores.
- **tools**: Array of tool names to assign.

**Response:**
```json
{
  "status": "created",
  "agent": { ... }
}
```

### `POST /api/create/task`
Stages a new assigned Task dynamically.

**Request Body:**
```json
{
  "description": "Synthesize the parsed PDFs into a summary document.",
  "agent_role": "Lead Researcher"
}
```

**Response:**
```json
{ "status": "created", "task": { ... } }
```

### `POST /api/create/mcp`
Connects a new Model Context Protocol (MCP) server.

**Request Body:**
```json
{
  "name": "Local Filesystem",
  "command": "npx",
  "args": ["-y", "@modelcontextprotocol/server-filesystem", "/Users"]
}
```

**Response:**
```json
{ "status": "created", "mcp": { ... } }
```

### `POST /api/create/a2a`
Establishes a cross-platform Agent-to-Agent bridge.

**Request Body:**
```json
{
  "sender": "Research Agent",
  "receiver": "Writer Agent",
  "model": "gpt-4o"
}
```

**Response:**
```json
{ "status": "created", "a2a": { ... } }
```

---

## 🔍 Metadata Endpoints

These endpoints provide available configuration options to populate UI dropdowns.

### `GET /api/tools`
Returns a list of all available built-in and sandboxed tools.

**Response:**
```json
["Google Search", "Arxiv Search", "Wikipedia Search", ...]
```

### `GET /api/memory`
Returns a list of supported memory store backends.

**Response:**
```json
["SQLite (Local)", "Pinecone (Vector)", "Qdrant (Vector)", ...]
```
