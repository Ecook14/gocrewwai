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

Using the `gocrew` SDK, you can connect an agent to a remote MCP server in seconds:

```go
package main

import "github.com/Ecook14/gocrewwai/gocrew"

func main() {
    // 1. Initialize MCP Client (Stdio transport)
    mcpClient, _ := gocrew.NewMCPClient("python", "path/to/server.py")
    defer mcpClient.Close()

    // 2. Discover and Add Remote Tools to Agent
    agent := gocrew.NewAgent(gocrew.AgentConfig{
        Role:  "MCP-Enabled Researcher",
        Tools: mcpClient.GetTools(), // Fetch all remote tools!
    })
}
```

## 🛡️ MCP Tool Filtering & Security

Gocrewwai gives you granular control over which remote tools an agent is allowed to execute.

### 1. Allow/Block Lists
Explicitly define the specific MCP tools that an agent can see and use. This prevents "Tool Overload" and ensures that sensitive tools are only available to authorized agents.

### 2. Native Validation
All MCP tool results are automatically validated against the agent's internal reasoning loop. This prevents "Tool Hallucination" and ensures that the agent correctly understands the remote output.

### 3. Human-in-the-Loop
Configure specific MCP tools (e.g., `git.delete_branch`) to require manual approval before execution via the **Dashboard**.

---

[Back to Delegation Guide](./agent_delegation.md) | [Next: Events](./events.md)
