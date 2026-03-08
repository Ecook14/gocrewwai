# Welcome to Gocrew! 🚀

Hey there! I am absolutely thrilled to introduce you to **Gocrew**, an open-source project I’ve been pouring my heart into. Inspired by the incredible work of **CrewAI, LangChain, and LangGraph**, I've built a high-performance, strictly-typed orchestrator that brings the best of agentic AI to the Go ecosystem.

Think of Crew-GO as a framework for assembling a digital "crew." You assign each member a specific persona—like a Lead Researcher, a Senior Data Analyst, or an Expert Coder. You hand them a goal, provide them with tools, and they figure out how to communicate and collaborate to achieve that goal!

While many incredible AI tools out there use Python (like LangChain or the original CrewAI), I decided to build this entirely from the ground up in **Go**. 

Why Go? Because Go gives us incredible superpowers: **insane speed, extremely low memory footprints, and the ability to run hundreds of things at the exact same time without breaking a sweat.** Whether you're a curious beginner dipping your toes into AI, or a seasoned Go architect looking to build enterprise-grade reasoning engines, I’ve designed this so we can build reliable, crash-free AI systems together.

---

## 🌟 What makes Crew-GO incredibly special?

When we move AI orchestration out of Python scripts and into a compiled, strongly-typed Go binary, we unlock a completely new tier of performance:

1. **True Teamwork (Massive Concurrency)**: Python often struggles to do many things at the exact same time because of something called the Global Interpreter Lock (GIL). Go's native "goroutines" let us easily orchestrate hundreds of AI agents working, fetching data, and thinking in true parallel execution without slowing down!
2. **Rock-Solid Reliability (Type Safety)**: We eliminate those random frustrating `KeyError` crashes in the middle of a long execution. When we expect a specific JSON format from an AI, Crew-GO guarantees it gets mapped directly into your Go structs securely.
3. **No Network Hiccups (Resilience)**: LLM APIs go down. It happens. The framework automatically catches HTTP 429 (Rate Limit) and 5xx errors, applying Exponential Backoff to politely wait and retry without breaking your app.
4. **Fast and Tiny**: We compile down to a single, tiny, lightning-fast binary that you can drop anywhere—into a tiny Alpine Docker container, or directly onto an edge device.

---

## 💎 The Awesome Features We’ve Built

I wanted to make sure our agents have all the built-in superpowers they need to be productive right out of the box:

### 🛡️ 1. Safe Code Execution (Elite Sandboxing)
We all want our AI to write and execute code (like running a Python script to analyze a CSV). But blindly trusting an AI to run code on our servers is terrifying! That's why I built native **Sandboxes**. You can easily tell the agent: 
> *"Hey, you can write code, but you must run it inside an isolated, memory-capped Docker container, or inside a lightning-fast WebAssembly (WASM) engine."*
If the AI breaks something, it only breaks the temporary sandbox!

### 🖥️ 2. A Beautiful Live Dashboard & Creator Studio
I love peeking into the "black box" to see what the AI is thinking. So, we built a stunning, real-time glassmorphic web dashboard. Just add `--ui` to your terminal command, and you can:
- **Watch Live**: Watch your agents chat, use tools, and complete tasks live via WebSockets.
- **Creator Studio**: Dynamically build new Agents and Tasks directly from the UI without touching code.
- **MCP Hub**: Connect to remote Model Context Protocol servers with one click.
- **A2A Bridges**: Establish cross-platform protocol bridges between disparate agents to create a unified mesh network.
- **Human-in-the-Loop**: Review and approve sensitive tool actions (like shell commands or database writes) directly from your browser.

### 🧠 3. Advanced Persistence (Infinite Memory)
Our agents don't have goldfish memory! We've given them:
- **Short-term memory**: "What tools did I just use to solve the current step?"
- **Long-term memory**: Powered by Vector Databases (like ChromaDB, Pinecone, Qdrant, Redis), so they remember facts from conversations you had weeks ago.
- **Entity Memory**: A persistent tracker that extracts and remembers specific structured facts about people, companies, or concepts across disparate tasks.

### 🧰 4. A Massive Tool Arsenal (24+ Native Tools)
We've given our agents massive toolbelts to interact with the real world natively:
- **Web Browsing**: Full headless browser automation (Chromedp) to navigate complex React/Vue Single Page Applications.
- **Search & Research**: Native integrations with Google Serper, Exa.ai semantic search, Wikipedia, and Arxiv.
- **Data & Files**: Read and natively parse text, CSVs, PDFs, and Microsoft Word Documents without needing clunky external OCR services.
- **WebMCP Integration**: Agents can now use the new Web Model Context Protocol to dynamically discover and consume tools hosted entirely on remote websites!

### 🛡️ 5. Safety First (Enterprise Guardrails)
Sometimes we need to guarantee the AI doesn't do something silly before we ship it to a user. We built "Guardrails", which are rules an agent MUST pass before its work is accepted.
- **PII Redaction**: We can automatically hide sensitive info (like masking emails or credit cards).
- **LLM Review (HitL)**: We can logically force a "Critic Editor Agent" to review another agent's output. If the Critic says it's bad, the first agent is forced to try again until it gets it right!

---

## 🏗️ How does a Crew work? (Orchestration Patterns)

We can organize our digital teams in a few different exciting ways depending on how complex the job is:
- **Sequential**: Standard assembly line. Agent A finishes their research, then passes the baton to Agent B to write the report.
- **Hierarchical (Manager-Led)**: An AI "Manager" actively oversees the team, delegating tasks to workers, reading their outputs, and dynamically adjusting the plan on the fly.
- **Consensual**: We put multiple agents in a room to debate a topic until they synthesize an agreed-upon answer!
- **Cyclic Graphs**: Need an endless loop? Or a complex state machine that branches conditionally? The underlying engine supports arbitrary graph topologies natively.

---

## 🚀 Let's Build Your First Crew!

Are you ready to see this magic in action? It takes exactly 5 minutes.

### 1. What You Need
- **Go**: Version `1.22` or higher installed on your computer.
- **API Keys**: You'll need a key from an AI provider (like OpenAI, Anthropic, or Gemini).

### 1. Installation

**📦 For Elite Architects (Library Usage)**
Import the core orchestration engine directly into your Go projects:
```bash
go get github.com/Ecook14/gocrewwai
```

**🛠️ For Dynamic Operators (Global CLI)**
Install the `gocrew` command-line tool globally to scaffold projects and launch the Dashboard from anywhere:
```bash
go install github.com/Ecook14/gocrewwai/cmd/gocrewwai@latest
```
*Note: Ensure your `$GOPATH/bin` (usually `~/go/bin`) is in your system's `PATH`!*

### 2. Basic Library Usage (Import & Build)
Building a crew in code is highly idiomatic and strictly typed:

```go
import (
    "github.com/Ecook14/gocrewwai/pkg/agents"
    "github.com/Ecook14/gocrewwai/pkg/crew"
    "github.com/Ecook14/gocrewwai/pkg/tasks"
)

func main() {
    // 1. Define your Team
    writer := agents.NewAgent("Writer", "Write a blog post", "...", llm)

    // 2. Define the Mission
    task := tasks.NewTask("Write about Go 1.24", writer)

    // 3. Kickoff the Crew!
    myCrew := crew.NewCrew([]*agents.Agent{writer}, []*tasks.Task{task})
    myCrew.Kickoff(context.Background())
}
```

### 3. Rapid Scaffolding (CLI)
If you prefer starting with a template, our CLI provides instant elite-tier scaffolding:
```bash
~/go/bin/crewai create my-first-crew
cd my-first-crew
```

### 4. Watch Them Run!
Set your API key, and launch the engine (with the live UI turned on!):
```bash
export OPENAI_API_KEY=your-api-key-here
go run . --ui
```
*Pop open your browser to `http://localhost:8080/web-ui` and watch your agents start collaborating!*

## 📦 Compiling a Single Binary (Zero Dependencies)

One of the biggest advantages of Crew-GO over Python alternatives is the ability to ship your entire agent orchestration engine as a **single, standalone executable file**.

Because we aggressively utilize Go 1.16+ `//go:embed` directives, the **entire Glassmorphic Web UI Dashboard** (HTML, CSS, JavaScript) is baked natively into the compiled Go binary! You do not need to distribute any `web-ui` folders alongside your app.

To compile a production-ready binary for your specific operating system:

```bash
# Build the CrewAI scaffolding CLI for Windows
GOOS=windows GOARCH=amd64 go build -o crewai-cli.exe ./cmd/crewai

# Build a standalone standalone Dashboard app (from our examples) for Mac 
GOOS=darwin GOARCH=arm64 go build -o my-app ./examples/dashboard_demo

# Build for Linux Containers
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o my-app ./examples/dashboard_demo
```
Just hand that single executable file to your client, and they can run their AI agents securely on any machine without installing Python, Conda, Node.js, or any other dependencies!

---

## 🤝 Let's Build Together! (Calling all Go Devs!)

I genuinely believe we are building something incredibly special here, but **I really need your help!** 

This project is 100% open-source, and I wholeheartedly invite you to join me in making it the absolute best, most scalable AI orchestration framework in the Go ecosystem. Whether you're a senior architect who wants to debate goroutine channel buffering, or just writing your first few lines of Go and want to fix a typo in the docs, **there is a place for you here.**

**How you can help right now:**
- **Add a Tool**: Got an idea for a cool new tool (like integrating a new SQL database, or hooking up X/Twitter)? Look at `pkg/tools`, copy a simple tool, and submit a Pull Request!
- **Fix Bugs**: Find a panic or unexpected behavior? Let me know by opening an issue, or better yet, track it down and open a PR!
- **Dashboard Upgrades**: If you love writing vanilla JS/CSS or WebSockets, the live UI dashboard in `web-ui` can always use polish and new visualizations!
- **Memory Integrations**: Help me write bindings for more Vector Databases!

I've worked really hard to keep the codebase clean, modular, and easy to understand. Dive in, break things, ask questions, and let's build the future of AI in Go together! 

## 📖 Want to dive deeper into the code?

I've written up a bunch of detailed guides if you want to understand exactly how the engine ticks:
1. 🚀 **[Quickstart Guide](docs/quickstart.md)**: Explore the generated scaffold project.
2. 📖 **[Detailed Usage Guide](USAGE.md)**: Let's talk about configuring tools, memory databases, and YAML!
3. 🏗️ **[Internal Architecture](docs/architecture.md)**: Want to see exactly how our Agent execution loops and OpenTelemetry buses work under the hood? Read this!

---
**Gocrew** - Let's build reliable, powerful AI systems together.
