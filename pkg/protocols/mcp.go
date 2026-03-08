package protocols

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

// ---------------------------------------------------------------------------
// MCP (Model Context Protocol) — Client & Server
// ---------------------------------------------------------------------------
//
// MCP enables agents to discover and invoke tools hosted on external servers,
// and expose Gocrew tools as MCP-compatible resources.
//
// Spec reference: https://modelcontextprotocol.io

// ---------------------------------------------------------------------------
// MCP Types
// ---------------------------------------------------------------------------

// MCPToolDefinition describes a tool exposed via MCP.
type MCPToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"` // JSON Schema
}

// MCPResourceDefinition describes a resource exposed via MCP.
type MCPResourceDefinition struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// MCPToolCall represents a request to invoke an MCP tool.
type MCPToolCall struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"arguments"`
}

// MCPToolResult is the response from an MCP tool invocation.
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent represents a content block in an MCP response.
type MCPContent struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
}

// ---------------------------------------------------------------------------
// MCP Client — Discover and Invoke Remote Tools
// ---------------------------------------------------------------------------

// MCPClient connects to an MCP server to list and invoke tools.
type MCPClient struct {
	mu         sync.RWMutex
	ServerURL  string                `json:"server_url"`
	httpClient *http.Client
	Tools      []MCPToolDefinition   `json:"tools"`
	Resources  []MCPResourceDefinition `json:"resources"`
}

// NewMCPClient creates a client for an MCP server.
func NewMCPClient(serverURL string) *MCPClient {
	return &MCPClient{
		ServerURL:  serverURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Initialize performs the MCP handshake and discovers available tools.
func (c *MCPClient) Initialize(ctx context.Context) error {
	// List tools
	tools, err := c.listTools(ctx)
	if err != nil {
		return fmt.Errorf("mcp: failed to list tools: %w", err)
	}

	c.mu.Lock()
	c.Tools = tools
	c.mu.Unlock()

	// List resources (optional, may not be supported)
	resources, _ := c.listResources(ctx)
	c.mu.Lock()
	c.Resources = resources
	c.mu.Unlock()

	return nil
}

func (c *MCPClient) URL() string {
	return c.ServerURL
}

// ListTools returns the discovered tool definitions.
func (c *MCPClient) ListTools() []MCPToolDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]MCPToolDefinition, len(c.Tools))
	copy(result, c.Tools)
	return result
}

// ListResources returns the discovered resources.
func (c *MCPClient) ListResources() []MCPResourceDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]MCPResourceDefinition, len(c.Resources))
	copy(result, c.Resources)
	return result
}

// CallTool invokes a tool on the MCP server.
func (c *MCPClient) CallTool(ctx context.Context, call MCPToolCall) (*MCPToolResult, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      call.Name,
			"arguments": call.Params,
		},
	}

	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ServerURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mcp tool call failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var rpcResp struct {
		Result *MCPToolResult `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("mcp: failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("mcp error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func (c *MCPClient) listTools(ctx context.Context) ([]MCPToolDefinition, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}

	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ServerURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var rpcResp struct {
		Result struct {
			Tools []MCPToolDefinition `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, err
	}
	return rpcResp.Result.Tools, nil
}

func (c *MCPClient) listResources(ctx context.Context) ([]MCPResourceDefinition, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/list",
	}

	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ServerURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var rpcResp struct {
		Result struct {
			Resources []MCPResourceDefinition `json:"resources"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, err
	}
	return rpcResp.Result.Resources, nil
}

// ---------------------------------------------------------------------------
// MCP Server — Expose Crew-GO Tools via MCP
// ---------------------------------------------------------------------------

// MCPToolHandler is a function that executes a tool call.
type MCPToolHandler func(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error)

// MCPServer exposes Crew-GO tools as MCP-compatible endpoints.
type MCPServer struct {
	mu        sync.RWMutex
	tools     map[string]mcpRegisteredTool
	resources map[string]MCPResourceDefinition
}

type mcpRegisteredTool struct {
	Definition MCPToolDefinition
	Handler    MCPToolHandler
}

// NewMCPServer creates an MCP server.
func NewMCPServer() *MCPServer {
	return &MCPServer{
		tools:     make(map[string]mcpRegisteredTool),
		resources: make(map[string]MCPResourceDefinition),
	}
}

// RegisterTool adds a tool to the MCP server.
func (s *MCPServer) RegisterTool(def MCPToolDefinition, handler MCPToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[def.Name] = mcpRegisteredTool{Definition: def, Handler: handler}
}

// RegisterResource adds a resource definition to the MCP server.
func (s *MCPServer) RegisterResource(res MCPResourceDefinition) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[res.URI] = res
}

// Handler returns an http.Handler that serves MCP JSON-RPC requests.
func (s *MCPServer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.writeError(w, -32700, "Parse error", 0)
			return
		}

		var req struct {
			JSONRPC string                 `json:"jsonrpc"`
			ID      interface{}            `json:"id"`
			Method  string                 `json:"method"`
			Params  map[string]interface{} `json:"params"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			s.writeError(w, -32700, "Parse error", 0)
			return
		}

		switch req.Method {
		case "tools/list":
			s.handleToolsList(w, req.ID)
		case "tools/call":
			s.handleToolCall(w, r.Context(), req.ID, req.Params)
		case "resources/list":
			s.handleResourcesList(w, req.ID)
		case "initialize":
			s.handleInitialize(w, req.ID)
		default:
			s.writeError(w, -32601, fmt.Sprintf("Method not found: %s", req.Method), req.ID)
		}
	})
}

func (s *MCPServer) handleInitialize(w http.ResponseWriter, id interface{}) {
	s.writeResult(w, id, map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]bool{"listChanged": true},
			"resources": map[string]bool{"listChanged": true},
		},
		"serverInfo": map[string]string{
			"name":    "gocrew-mcp",
			"version": "1.0.0",
		},
	})
}

func (s *MCPServer) handleToolsList(w http.ResponseWriter, id interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]MCPToolDefinition, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, t.Definition)
	}

	s.writeResult(w, id, map[string]interface{}{"tools": tools})
}

func (s *MCPServer) handleToolCall(w http.ResponseWriter, ctx context.Context, id interface{}, params map[string]interface{}) {
	name, _ := params["name"].(string)
	args, _ := params["arguments"].(map[string]interface{})

	s.mu.RLock()
	tool, ok := s.tools[name]
	s.mu.RUnlock()

	if !ok {
		s.writeError(w, -32602, fmt.Sprintf("Tool not found: %s", name), id)
		return
	}

	result, err := tool.Handler(ctx, args)
	if err != nil {
		s.writeResult(w, id, &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		})
		return
	}

	s.writeResult(w, id, result)
}

func (s *MCPServer) handleResourcesList(w http.ResponseWriter, id interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]MCPResourceDefinition, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, r)
	}

	s.writeResult(w, id, map[string]interface{}{"resources": resources})
}

func (s *MCPServer) writeResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func (s *MCPServer) writeError(w http.ResponseWriter, code int, message string, id interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}
