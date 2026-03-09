# Feature Deep Dive: Tools 🧰

Tools are the "hands" of your agents. They allow agents to interact with the real world—searching the web, running code, reading files, or calling APIs.

---

## 🏗️ The Tool Interface

In Gocrew, a tool is any Go struct that implements the `tools.Tool` interface:

```go
type Tool interface {
    Name() string
    Description() string
    ArgsSchema() []ArgSchema
    Run(ctx context.Context, input string) (string, error)
    CacheFunction() func(input string) string // Optional
}
```

### High-Precision Arguments

We use `ArgsSchema` to tell the LLM exactly what JSON structure to provide. This ensures that tool calls are valid and easy to parse.

---

## 🔌 Native Tool Arsenal

Gocrew ships with 24+ native tools, including:
- **SearchWeb**: Powered by Chromedp for full browser automation (SPA support).
- **CodeInterpreter**: Securely execute Python/Go/Shell scripts in Docker or E2B.
- **FileTools**: Read, write, and list files securely.
- **GitHub/Slack/Notion**: First-class SaaS integrations.
- **MCP Bridge**: Connect to any Model Context Protocol (MCP) server.

---

## 🛡️ Secure Execution (Sandboxing)

For code-based tools, Gocrew provides several isolation layers:
- **Docker**: Run execution in ephemeral containers.
- **E2B**: Offload execution to remote, secure cloud sandboxes.
- **WASM**: Use WebAssembly for zero-dependency local isolation.

---

## 🧠 Smart Caching

Tools can optionally provide a `CacheFunction`. If enabled, the results of tool calls are indexed by the agent's `Cache` (e.g., Redis or SQLite), saving costs and reducing latency for repetitive tasks.

---

## 🛠️ Creating Custom Tools

Creating a custom tool is as easy as implementing the interface. You can then pass it to any agent:

```go
agent := gocrew.NewAgentBuilder().
    Tools(myCustomTool).
    Build()
```

---
**Gocrew** - Empowering agents with real-world capabilities.
