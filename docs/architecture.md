# Gocrewwai Architecture (Under the Hood) 🏗️

Welcome to the engine room! If you're a Go developer who wants to understand exactly how Gocrew differs from other frameworks, or if you're looking to contribute to the codebase, this is the document for you. 

I've engineered Gocrew to take massive advantage of Go's built-in concurrency model, strict typing, and high-performance routing. Let's break down exactly how it works.

---

## The Agent Reasoning Protocol (ReAct Loop)

Gocrew Agents do not just predictably "call an LLM and return the text." They utilize the **ReAct (Reason + Act)** pattern to create infinite, autonomous reasoning loops.

If you look at `pkg/agents/agent.go`, you'll see the core execution loop. When a Task is given to an Agent:

1.  **Context Construction**: The Engine grabs the Agent's Role, Goal, the current Task, AND the historical context of previous tasks, and compiles them into a massive, immutable System Prompt.
2.  **Memory Recall (RAG)**: If a `MemoryStore` is attached (like ChromaDB or Redis), the Agent calculates the vector embedding of the current string payload and queries the database for relevant past experiences, silently injecting them into the context window.
3.  **The LLM Evaluation**: The Agent pings the LLM API via the `MiddlewareClient`.
4.  **Action Routing**:
    *   If the LLM responds with a final answer, the Go loop `break`s and the text is returned.
    *   If the LLM responds with a `ToolCall` (e.g., `{"name": "SearchWeb", "args": {"query": "golang updates"}}`), the Go context loop pauses.
5.  **Native Execution & Observation**: The Go engine natively executes the requested Tool payload. The output (or error) is captured and injected back into the LLM context message array as a new "Observation".
6.  **Self-Healing**: If a Go `panic` occurred or an API timed out inside the Tool, the LLM literally "reads" the Go error output. Because it's a reasoning engine, it will attempt to adjust its JSON parameters and try the tool again on the next iteration.
7.  **Loop Returns to Step 3**. This continues until the agent believes it has satisfied the Task, or until the `MaxIterations` limit is hit.

---

## The Go-Native Orchestration Engine (`pkg/crew/crew.go`)

Most Python frameworks rely on fake asynchronous loops (`asyncio`) tightly bound by the Global Interpreter Lock (GIL). Gocrewwai utilizes true native hardware threads (`goroutines`). 

This gets incredibly exciting when we look at **Hierarchical Processing**.

### Hierarchical Fast Fan-Out Deep-Dive
When `Process: crew.Hierarchical` is invoked in your Crew builder:
1.  The `Crew` generates an invisible, super-smart `ManagerAgent`.
2.  A Go `sync.WaitGroup` is initialized.
3.  A loop over all Tasks triggers a massive, instantaneous Fan-Out. Every single Task drops into its own `goroutine` via `go func(t *tasks.Task) {...}`.
4.  Inside each goroutine, the `ManagerAgent` is pinged with the Task payload. The Manager evaluates the task and routes it to the worker Agent best suited for the job!
5.  The worker Agents run their ReAct loops simultaneously across all hardware cores.
6.  As tasks finish, an `errCh := make(chan error, len(tasks))` captures the results safely without causing race conditions.
7.  The `WaitGroup.Wait()` blocks the main thread until all parallel streams conclude.
8.  Finally, the `ManagerAgent` receives a fan-in of all results and synthesizes the finalized coherent buffer.

This allows us to effortlessly perform tasks like scraping 50 websites at the exact same time without locking the thread!

---

## Global Telemetry & Observability Bus (`pkg/telemetry`)

We hate it when AI frameworks act like black boxes where you can't see why it took 45 seconds to answer a simple question. Gocrewwai fixes this entirely with a Global Go Channel Event Bus.

### Event Propagation
The `telemetry.EventBus` is an internal Pub/Sub broker wrapped in a strict `sync.RWMutex`.

As an Agent performs operations deep within the call stack, it fires non-blocking events natively:
```go
telemetry.GlobalBus.Publish(telemetry.Event{
    Type:      telemetry.EventToolStarted,
    AgentRole: "Researcher",
    Payload:   map[string]interface{}{"tool": "SearchWeb"},
})
```

Because it uses isolated Go channels, this publishing has negligible impact on execution latency or frame rates.

### The Dashboard WebSocket Bridge (`pkg/dashboard/dashboard.go`)
When you launch the UI Dashboard (using `--ui` or `dashboard.Start`), the system initializes:
1.  A lightweight Go HTTP handler for standard `html/css/js` delivery.
2.  A WebSocket `/ws` upgrade handler.
3.  It calls `telemetry.GlobalBus.Subscribe()`, grabbing the live firehose of ReAct events.
4.  It asynchronously marshals those events into JSON chunks over the TCP socket, giving your browser real-time frame rates of the AI reasoning cycle without polling!

## Bi-Directional Execution Control (`pkg/telemetry/events.go`)

Telemetry in Gocrewwai is not just for watching—it's for **control**. Using Go's `sync.Cond` and the `GlobalExecutionController`, the Dashboard sends signals back to the engine:
1. **Pause/Resume**: Blocks worker goroutines until a signal is received, prevents CPU spinning while execution is "Idle".
2. **HITL Reviews**: The engine identifies sensitive tool calls, fires a `review_requested` event, and parks the goroutine until the dashboard sends an approval via the `/api/review` endpoint.

---

## Help Me Expand the Architecture!

If you are an experienced Go systems engineer reading this, you might notice areas we can optimize—perhaps we could add a Kafka sink to the Telemetry Bus? Or use a worker-pool pattern instead of raw WaitGroups to save memory during massive fan-outs?

**I am explicitly asking you to come help me build it.** Submit a PR, open an issue, and let's craft the most beautiful, fault-tolerant orchestration architecture the open-source community has ever seen!
