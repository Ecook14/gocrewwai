# Feature Deep Dive: MCP Hub ⚓🧰🌐

Model Context Protocol (MCP) is the industry standard for connecting AI agents to external tools and data sources. Gocrewwai provides first-class, high-performance support for MCP, acting as a powerful **MCP Hub**.

---

> [!IMPORTANT]
> **Status: v1.0.0 (Stable).** Gocrewwai supports **HTTP**, **Stdio**, and **SSE (Server-Sent Events)** MCP transports with native tool filtering and validation.

---

## 🏗️ The Gocrewwai MCP Architecture

Gocrewwai allows agents to consume tools from any remote MCP server with zero code changes.

| Transport | Gocrewwai Implementation | Best Use Case |
| :--- | :--- | :--- |
| **Stdio** | Native local command execution | Running CLI tools or local Python scripts. |
| **HTTP** | Standard JSON-RPC over HTTP | Connecting to remote SaaS tool APIs. |
| **SSE** | Real-time streaming transport | Long-running connections and enterprise tool hubs. |

---

## 🚀 Connecting to an MCP Server (Elite Style)

In Gocrewwai v1.0, MCP servers can be connected directly via the `AgentConfig` using a DSL-like string array:

```go
agent := gocrew.NewAgent(gocrew.AgentConfig{
    Role: "Universal Librarian",
    // 1. Declare MCP Servers
    MCPS: []string{
        "http://localhost:8080/mcp",                      // Remote HTTP/SSE
        "stdio:npx -y @modelcontextprotocol/server-everything", // Local Stdio
    },
    // 2. Optional: Filter precisely which tools this agent can use
    MCPAllowList: []string{"echo", "calculate"},
    MCPBlockList: []string{"danger_tool"},
})
```

The engine automatically handles:
1. **Initialization**: Connecting to each server and performing the handshake.
2. **Discovery**: Mapping remote tools, resources, and prompts into the agent's toolbelt.
3. **Sampling**: If a remote MCP server requests an LLM completion (Sampling), the agent's own LLM is used to provide the response, governed by the `MCPSamplingPolicy`.

## 🛡️ MCP Tool Filtering & Security

Gocrewwai gives you granular control over which remote tools an agent is allowed to execute. This is critical when connecting to powerful servers like the `docker` or `filesystem` MCPs.

### 1. Allow/Block Lists

You can use the declarative `AgentConfig` to explicitly define routing.

```go
agent := gocrew.NewAgent(gocrew.AgentConfig{
    Role: "Read-Only Analyst",
    MCPS: []string{"stdio:npx -y @modelcontextprotocol/server-filesystem /app/data"},
    // Only allow safe reading tools:
    MCPAllowList: []string{"read_file", "list_directory", "search_files"},
    // Explicitly block mutating tools:
    MCPBlockList: []string{"write_file", "delete_file", "execute_command"},
})
```

If the agent's LLM attempts to call `write_file`, the engine will immediately reject the JSON RPC call without communicating with the remote server, triggering the agent's self-healing loop.

### 2. Native Validation

All MCP tool results are automatically validated against the agent's internal reasoning loop. This prevents "Tool Hallucination" and ensures that the agent correctly understands the remote output before proceeding.

### 3. Human-in-the-Loop Hooks

For hyper-sensitive tools that you still want the agent to use, you can attach an interceptor:

```go
agent.Guardrails = append(agent.Guardrails, guardrails.NewHumanReviewGuardrail(
    "execute_query", // The MCP Tool Name
    "SQL execution requires manual review.",
))
```
When triggered, this pauses the orchestration thread and pushes an alert to the Dashboard.

---

[Back to Delegation Guide](./agent_delegation.md) | [Next: Events](./events.md)
