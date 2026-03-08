# Feature Deep Dive: Model Context Protocol (MCP) 🌐

Hey there! Let's talk about one of the most bleeding-edge, enterprise-tier features we've built into Gocrew: native support for the **Model Context Protocol (MCP)**.

If you haven't heard of it yet, MCP is a massive new open-source standard created by Anthropic. Think of it like a "USB-C port for AI applications." 

Instead of writing custom Go code to wrap every single API in the world (like writing a custom Jira tool, a custom Salesforce tool, etc.), MCP allows external servers to broadcast a standard definition of what tools they offer. Our agents can then discover those tools and seamlessly execute them over HTTP or JSON-RPC!

---

## 🔌 Why did we build this into Crew-GO?

By adopting WebMCP and native MCP natively in `pkg/protocols/mcp.go`, **Gocrew transitions from being an isolated "script" to a fully integrated Service Mesh for AI.**

We've implemented this protocol in BOTH directions!

### 1. The MCP Client (Agent Consumption)
Crew-GO agents can connect to external infrastructure (e.g., a Python or Rust MCP Server running securely inside your company's VPC) to dynamically discover and execute tools they don't natively possess.

```go
// 1. Point the client at an existing internal tools MCP server.
mcpClient := protocols.NewMCPClient("http://internal-tools.vpc:8080/mcp")
mcpClient.Initialize(context.Background())

// 2. Use our built-in bridge to convert the remote tool into a Crew-GO Native Tool!
remoteTools := mcpClient.ListTools()
wrappedTool := tools.WrapMCPToolForCrewGo(mcpClient, remoteTools[0])

// 3. Hand the dynamically discovered tool to the Agent!
agent := agents.NewAgentBuilder().
    Role("IT Admin").
    Tools(wrappedTool).
    Build()
```

### 2. The MCP Server (Exposing Crew-GO Tools)
What if you already have agents running in another framework (like LangChain, or Claude Desktop), but you want them to use Crew-GO's lightning-fast, ultra-secure WASM sandboxes or native Web Browsers?

You can deploy your Crew-GO binary purely as a headless tool-server!

```go
server := protocols.NewMCPServer()

// Expose our amazing Go tools to the world!
tools.RegisterAllToolsOnMCPServer(server, myToolRegistry)

// Start listening for standard JSON-RPC HTTP requests
log.Println("Gocrew MCP Server listening on :8080")
http.ListenAndServe(":8080", server.Handler())
```

---

## 🌟 The new WebMCP Standard

We didn't stop at standard JSON-RPC servers. We also integrated the brand new **WebMCP** standard. 

If a website (like an internal corporate dashboard) includes a tiny HTML `<script type="application/mcp+json">` tag describing its APIs, you can simply hand an agent our `WebMCPTool`. The agent will autonomously visit the URL, parse the HTML, discover what buttons/APIs the website allows it to push, and natively format the HTTP requests to execute them! 

Zero fragile HTML scraping required.

---

## 🤝 Let's Push the Protocol Forward!

This protocol is incredibly new and evolving rapidly. If you are passionate about distributed systems, IPC, or AI interoperability, this is the perfect place to jump into the codebase.

Check out `pkg/protocols/mcp.go` and `pkg/protocols/webmcp.go`. If you find ways to optimize the JSON-RPC marshaling, or if you want to add support for raw STDIO transports (for running local MCP binaries), **I am explicitly asking for your help!** Submit a Pull Request and let's build the ultimate AI mesh network.
