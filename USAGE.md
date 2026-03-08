# How to Use Crew-GO (The Deep Dive) 📖

Hey again! I'm so glad you're here to dive deeper. This guide is designed to walk you through exactly how you and I can leverage the awesome, highly-technical tools and capabilities we’ve built into Crew-GO.

I've kept things conversational because we're in this together, but I haven't held back on the technical specifics. If you're a Go developer looking to build production systems or collaborate on the architecture, you'll feel right at home with the code snippets below. Let's get to work!

---

## 1. Giving Your Agents Superpowers (The Native Tool Arsenal)

In Crew-GO, your agents aren't just stuck generating text—they can interact with the physical world, databases, and APIs using `Tools`. Every tool implements the simple `tools.Tool` interface. While I've built over 24 native tools for us to use, building your own is incredibly simple!

### A. Writing and Running Code Safely (Sandboxing)

Sometimes, we want our agents to write a quick Python script or a Go program to solve a complex math problem or analyze a raw CSV file. But letting an AI execute arbitrary `os/exec` commands natively on your production server is an enormous security risk! That's why I built heavily isolated **Sandboxes**.

**Using Docker (Max Isolation):**
We can lock the agent inside an ephemeral Docker container. We can even restrict how much RAM or CPU it is allowed to use!
```go
// We give the agent a tiny, isolated Python environment!
tool := tools.NewCodeInterpreterTool(
    tools.WithDocker("python:3.11-slim"),
    tools.WithLimits(512, 1024), // Limit to 512MB memory!
)
```

**Using WebAssembly / WASM (Zero-Dependency Speed):**
If you don't want the overhead of Docker, we can run code locally using an extremely secure technology called WASM (`wazero`). We even restrict what directories the WASM memory can access!
```go
tool := tools.NewWASMSandboxTool(
    tools.WithMount("/data/input", myMemFS), // Explicitly mount virtual filesystems
)
```

### B. Browsing the Web & Extracting Data

Want your agent to do research or read local documents? 

**Full Browser Automation:**
We can give the AI a literal web browser using `Chromedp` to click around on complex websites dynamically, easily bypassing basic scraping blockers.
```go
tool := tools.NewBrowserTool() // Operates completely headlessly!
```

**Native Document Extraction (Zero OCR):**
I recently updated the engine so agents can read standard enterprise files effortlessly. Our `IngestionEngine` parses PDFs, CSVs, and even Microsoft Word `.docx` natively without external APIs!
```go
// Just point the knowledge base to a directory!
knowledgeStore.ExtractDir(ctx, "/path/to/my/corporate/documents")
```

---

## 2. Setting Up Your Crew Using YAML (Separating Logic & Prompts)

Instead of hardcoding huge strings of prompts directly in our Go files, I find it so much cleaner and easier to use simple `.yaml` configuration files. This means prompt engineers or product managers can tweak the AI's personality, while you focus on writing the strict Go orchestration engine!

### Create an `agents.yaml` File
Let's describe our team members and connect their tools dynamically:
```yaml
DataAnalyst:
  role: "Lead SQL Data Analyst"
  goal: "Extract insights from our database and explain them clearly."
  backstory: "You are a friendly but rigorous statistician who double-checks every JOIN statement."
  verbose: true
  sandbox: "docker" # Tells the engine exactly how this specific agent should run code
  tools:
    - name: "PostgresTool"
      params:
        dsn: "postgres://user:pass@localhost:5432/db"
```

### Create a `tasks.yaml` File
Let's tell them exactly what to accomplish:
```yaml
analyze_retention:
  description: "Calculate the 30-day user retention rate."
  expected_output: "A well-formatted Markdown summary with exact numbers."
  agent: "DataAnalyst" # Link the task right back to the agent
```

### Bringing it together in Go
Loading this up into our strict types is super easy:
```go
// 1. Load the files
agentsMap, err := config.LoadAgents("config/agents.yaml")
tasksMap, err := config.LoadTasks("config/tasks.yaml", agentsMap)

// 2. Build the Crew Process!
myCrew := crew.Crew{
    Agents: []*agents.Agent{agentsMap["DataAnalyst"]},
    Tasks:  []*tasks.Task{tasksMap["analyze_retention"]},
    Process: crew.Sequential, // Or try crew.Hierarchical for Manager-led routing!
}
```

---

## 3. Acting Like a Manager: Security & Guardrails

Sometimes, we need to guarantee that an AI doesn't do something silly, dangerous, or malformatted. I built "Guardrails" in `pkg/guardrails` to act as absolute blocking rules. If an AI fails a guardrail, the system returns an error directly to the AI and forces it to fix its mistake autonomously!

### Hiding Private Information (PII Redaction)
If you're dealing with customer data, we securely scrub out names, emails, and credit cards *before* the data ever hits your database.
```go
redactor := guardrails.NewPIIRedactor()
agent.Guardrails = append(agent.Guardrails, redactor)
// Output: "Contact the user at [EMAIL REDACTED]"
```

### Having an Agent Review Another Agent (LLM-in-the-Loop)
We can actually set up a "Critic Supervisor" agent whose ONLY job is to aggressively grade the work of the first agent!
```go
critic := agents.NewAgent("Reviewer", "Be extremely harsh on code quality", "...", llmClient)
reviewGuardrail := guardrails.NewLLMReview(critic)

workerCoder.Guardrails = append(workerCoder.Guardrails, reviewGuardrail)
```

---

## 4. Elite Engine Features (For the Go Architects)

If you are a backend engineer wondering how we handle scale and safety, here are the real powerhouse features.

### A. Strongly-Typed Output Extraction (Go Generics)
Stop manually parsing JSON strings or dealing with messy `interface{}` maps! Ask the AI to return JSON, and Crew-GO uses Go Generics to safely and strictly unmarshal the result for you.

```go
type AnalysisResult struct {
    Trends []string `json:"trends"`
    Score  int      `json:"score"`
}

// ... wait for crew to finish the task ...

// Securely and strictly extract the result!
result, err := tasks.GetOutput[AnalysisResult](marketTask)
if err != nil {
    log.Fatal(err) // We know immediately if the AI messed up the format!
}
fmt.Printf("Team Quality Score: %d\n", result.Score)
```

### B. Giving Your Agents Infinite Memory (Vector Stores)
In Crew-GO, I built a system where agents can store and verify information permanently natively using Vector math (Embeddings).

**Databases we support out-of-the-box in `pkg/memory`:**
*   **In-Memory**: Great for testing.
*   **SQLite**: Perfect for local apps that don't need heavy infrastructure.
*   **Redis**, **ChromaDB**, **Pinecone**, **Qdrant**: What I highly recommend if we are deploying thousands of agents to production!

```go
// Connect to Redis for distributed infinite memory!
store, _ := memory.NewRedisStore("localhost:6379", "password")
agent.Memory = store 
```

### C. Connect to Everything: WebMCP (Model Context Protocol)
This is cutting-edge engineering. We implemented the emerging **WebMCP** standard. If you give an agent the `WebMCPTool`, you can literally point it at an external web URL that supports MCP headers. The agent will read the website's exposed functional schema, and autonomously figure out how to fire HTTP POST/GET requests to perform actions on that website natively! No custom API wrappers required!

### D. The Real-Time Telemetry Event Bus
At the core of the engine is `pkg/telemetry`. Everything the engine does (Agents thinking, tools completing, memory searches) emits an `Event` to a global Go channel bus. You can tap into this Event pipeline natively to pipe monitoring to Datadog, Prometheus, or build custom WebSocket interfaces!

```go
// Subscribe to the global brain impulses!
subID, eventChannel := telemetry.GlobalBus.Subscribe()
defer telemetry.GlobalBus.Unsubscribe(subID)

go func() {
    for event := range eventChannel {
        if event.Type == telemetry.EventToolFinished {
            fmt.Printf("Tool %s finished in %s\n", event.Payload["tool"], event.Payload["duration"])
        }
    }
}()
```

---

## Let's Build This Together!

If you are reading through this technical deep dive and thinking, *"Wow, it would be cool if it could also use Kafka for the event bus..."* or *"I want to add a Postgres memory store..."*, **I am asking you to come help me build it!** 

Check out the open issues on GitHub, send me a pull request (even if it's just fixing a comment!), talk to me on the discussions page, and let's make this the most incredibly stable, collaborative, and feature-rich AI framework the Go community has ever touched!
