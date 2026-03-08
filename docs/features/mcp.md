# Feature: Model Context Protocol (MCP)

## Overview
**Model Context Protocol (MCP)** is a standardized protocol allowing AI Models (Clients) to securely connect to diverse data sources and tools (Servers). 

Crew-GO natively implements **both** the MCP Client and MCP Server protocols (`pkg/protocols/mcp.go`), establishing it as an Elite-tier orchestration framework capable of functioning in distributed, polyglot microservice environments.

## MCP Client (Agent consumption of external tools)
Crew-GO agents can connect to external infrastructure (e.g., a Python or Rust MCP Server running inside your VPC) to discover and execute tools they don't natively possess.

```go
// 1. Connect to an external internal tools MCP server
mcpClient := protocols.NewMCPClient("http://internal-tools.vpc:8080/mcp")
mcpClient.Initialize(context.Background())

// 2. Wrap MCP tools into Crew-GO agent tools
agent := agents.NewAgent(...)
// The agent securely discovers external functions natively!
```

## MCP Server (Exposing Crew-GO tools)
You can deploy a Crew-GO binary purely as a headless tool-server for other agents (like Claude Desktop or LangChain) to utilize its incredibly fast Go-based tools (like WASM sandboxing or parallel website scraping).

```go
server := protocols.NewMCPServer()

// Expose a Go function to the world
server.RegisterTool(protocols.MCPToolDefinition{
    Name: "wasm_execute",
    Description: "Runs high-speed WASM code",
}, myWasmHandler)

// Starts listening for standard JSON-RPC HTTP requests
http.ListenAndServe(":8080", server.Handler())
```

## Why this matters
By adopting MCP natively, Crew-GO transitions from being an isolated "script" to a fully integrated **Service Mesh for AI**, capable of sharing its 34+ high-performance tools across organizational boundaries efficiently via JSON-RPC.
