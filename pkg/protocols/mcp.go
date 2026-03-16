package protocols

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"sync"
	"strings"
	"os"
	"log/slog"
	"time"
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

// MCPResourceContent describes the content of a resource.
type MCPResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // Base64 encoded binary data
}

// MCPPromptDefinition describes a prompt exposed via MCP.
type MCPPromptDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Arguments   []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Required    bool   `json:"required"`
	} `json:"arguments"`
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

// MCPTransport defines the interface for communicating with an MCP server.
type MCPTransport interface {
	Initialize(ctx context.Context) error
	SendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error)
	SetNotificationHandler(handler func(method string, params json.RawMessage))
	Close() error
}

// HTTPTransport implements MCP communication over HTTP.
type HTTPTransport struct {
	URL        string
	httpClient *http.Client
	Headers    map[string]string
}

func NewHTTPTransport(url string) *HTTPTransport {
	return &HTTPTransport{
		URL:        url,
		httpClient: &http.Client{Timeout: 10 * time.Minute},
		Headers:    make(map[string]string),
	}
}

func (t *HTTPTransport) Initialize(ctx context.Context) error { return nil }

func (t *HTTPTransport) SetNotificationHandler(handler func(method string, params json.RawMessage)) {
	// HTTP doesn't support server-initiated notifications in this direction
}

func (t *HTTPTransport) SendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  method,
		"params":  params,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.URL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
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

func (t *HTTPTransport) Close() error { return nil }

// StdioTransport implements MCP communication over local standard I/O with async support.
type StdioTransport struct {
	Command string
	Args    []string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	encoder *json.Encoder
	
	mu       sync.Mutex
	pending  map[int64]chan *rpcResponse
	onNotify func(string, json.RawMessage)
	closing  chan struct{}
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewStdioTransport(command string, args ...string) *StdioTransport {
	return &StdioTransport{
		Command: command,
		Args:    args,
		pending: make(map[int64]chan *rpcResponse),
		closing: make(chan struct{}),
	}
}

func (t *StdioTransport) SetNotificationHandler(handler func(method string, params json.RawMessage)) {
	t.onNotify = handler
}

func (t *StdioTransport) Initialize(ctx context.Context) error {
	t.cmd = exec.CommandContext(ctx, t.Command, t.Args...)
	
	// Better stderr handling: pipe to a logger instead of raw os.Stderr
	t.cmd.Stderr = os.Stderr

	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return err
	}
	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := t.cmd.Start(); err != nil {
		return err
	}
	t.encoder = json.NewEncoder(t.stdin)

	// Start background reader
	go t.readLoop()

	return nil
}

func (t *StdioTransport) readLoop() {
	decoder := json.NewDecoder(t.stdout)
	for {
		var msg struct {
			ID     *int64          `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		if err := decoder.Decode(&msg); err != nil {
			if err != io.EOF {
				slog.Error("mcp: stdio read error", slog.Any("error", err))
			}
			return
		}

		if msg.ID != nil {
			t.mu.Lock()
			ch, ok := t.pending[*msg.ID]
			delete(t.pending, *msg.ID)
			t.mu.Unlock()

			if ok {
				ch <- &rpcResponse{Result: msg.Result, Error: msg.Error}
			}
		} else if msg.Method != "" && t.onNotify != nil {
			t.onNotify(msg.Method, msg.Params)
		}
	}
}

func (t *StdioTransport) SendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := time.Now().UnixNano()
	resCh := make(chan *rpcResponse, 1)

	t.mu.Lock()
	t.pending[id] = resCh
	t.mu.Unlock()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	if err := t.encoder.Encode(req); err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, err
	}

	select {
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, ctx.Err()
	case res := <-resCh:
		if res.Error != nil {
			return nil, fmt.Errorf("mcp error %d: %s", res.Error.Code, res.Error.Message)
		}
		return res.Result, nil
	}
}

func (t *StdioTransport) Close() error {
	close(t.closing)
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.cmd != nil {
		_ = t.cmd.Process.Kill()
		return t.cmd.Wait()
	}
	return nil
}

// SSETransport implements MCP communication over Server-Sent Events with stream consumption.
type SSETransport struct {
	URL        string
	httpClient *http.Client
	Headers    map[string]string
	
	mu         sync.RWMutex
	postURL    string
	onNotify   func(string, json.RawMessage)
	ready      chan struct{}
	closing    chan struct{}
}

func NewSSETransport(url string) *SSETransport {
	return &SSETransport{
		URL:        url,
		httpClient: &http.Client{Timeout: 0}, // Stream should not timeout
		Headers:    make(map[string]string),
		ready:      make(chan struct{}),
		closing:    make(chan struct{}),
	}
}

func (t *SSETransport) SetNotificationHandler(handler func(method string, params json.RawMessage)) {
	t.onNotify = handler
}

func (t *SSETransport) Initialize(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", t.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("mcp: sse connection failed with status %d", resp.StatusCode)
	}

	go t.consumeStream(resp.Body)

	// Wait for the 'endpoint' event to signal readiness
	select {
	case <-t.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(15 * time.Second):
		return fmt.Errorf("mcp: timed out waiting for sse endpoint event")
	}
}

func (t *SSETransport) consumeStream(body io.ReadCloser) {
	defer body.Close()
	
	// Simple SSE line parser
	var currentEvent string
	buf := make([]byte, 4096)
	for {
		select {
		case <-t.closing:
			return
		default:
		}

		n, err := body.Read(buf)
		if err != nil {
			if err != io.EOF {
				slog.Error("mcp: sse stream read error", slog.Any("error", err))
			}
			return
		}

		lines := strings.Split(string(buf[:n]), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				currentEvent = ""
				continue
			}

			if strings.HasPrefix(line, "event:") {
				currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if currentEvent == "endpoint" {
					// Use url.Parse to handle relative/absolute endpoint URIs
					t.mu.Lock()
					if strings.HasPrefix(data, "http") {
						t.postURL = data
					} else {
						base, _ := url.Parse(t.URL)
						rel, _ := url.Parse(data)
						t.postURL = base.ResolveReference(rel).String()
					}
					t.mu.Unlock()
					
					// Signal readiness
					select {
					case <-t.ready:
					default:
						close(t.ready)
					}
				} else if currentEvent == "message" && t.onNotify != nil {
					// SSE can push JSON-RPC notifications as "message" events
					t.onNotify("message", json.RawMessage(data))
				} else {
					slog.Debug("mcp: sse received data", slog.String("event", currentEvent), slog.String("data", data))
				}
			}
		}
	}
}

func (t *SSETransport) SendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	t.mu.RLock()
	postURL := t.postURL
	t.mu.RUnlock()

	if postURL == "" {
		return nil, fmt.Errorf("mcp: sse transport not ready (no endpoint discovered)")
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  method,
		"params":  params,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	// Responses for requests sent via POST come back in the POST response
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
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

func (t *SSETransport) Close() error {
	close(t.closing)
	return nil
}

// MCPClient connects to an MCP server to list and invoke tools, resources, and prompts.
type MCPClient struct {
	mu        sync.RWMutex
	Transport MCPTransport          `json:"-"`
	Tools     []MCPToolDefinition     `json:"tools"`
	Resources []MCPResourceDefinition `json:"resources"`
	Prompts   []MCPPromptDefinition   `json:"prompts"`
	
	// Sampling callback (Server asking Client for LLM completion)
	OnSample func(ctx context.Context, prompt string) (string, error)
}

// NewMCPClient creates a client for an MCP server using HTTP by default.
func NewMCPClient(serverURL string) *MCPClient {
	return NewMCPClientWithTransport(NewHTTPTransport(serverURL))
}

// NewMCPClientWithTransport creates a client with a custom transport (Stdio, SSE, etc.).
func NewMCPClientWithTransport(transport MCPTransport) *MCPClient {
	c := &MCPClient{
		Transport: transport,
	}
	transport.SetNotificationHandler(c.handleNotification)
	return c
}

func (c *MCPClient) handleNotification(method string, params json.RawMessage) {
	if method == "sampling/createMessage" && c.OnSample != nil {
		var req struct {
			Messages []struct {
				Content struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(params, &req); err == nil && len(req.Messages) > 0 {
			// In a real implementation we'd return a JSON-RPC response, 
			// but for notifications/sampling we need to bridge this back.
			// Currently Gocrew MCP handles sampling as an async callback.
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				prompt := req.Messages[len(req.Messages)-1].Content.Text
				_, _ = c.OnSample(ctx, prompt)
			}()
		}
	}
}

// Initialize performs the MCP handshake and discovers available tools, resources, and prompts.
func (c *MCPClient) Initialize(ctx context.Context) error {
	if err := c.Transport.Initialize(ctx); err != nil {
		return err
	}

	// List tools
	tools, err := c.listTools(ctx)
	if err != nil {
		return fmt.Errorf("mcp: failed to list tools: %w", err)
	}
	c.mu.Lock()
	c.Tools = tools
	c.mu.Unlock()

	// List resources
	resources, _ := c.listResources(ctx)
	c.mu.Lock()
	c.Resources = resources
	c.mu.Unlock()

	// List prompts
	prompts, _ := c.listPrompts(ctx)
	c.mu.Lock()
	c.Prompts = prompts
	c.mu.Unlock()

	return nil
}

// CallTool invokes a tool on the MCP server.
func (c *MCPClient) CallTool(ctx context.Context, call MCPToolCall) (*MCPToolResult, error) {
	raw, err := c.Transport.SendRequest(ctx, "tools/call", call)
	if err != nil {
		return nil, err
	}

	var res MCPToolResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("mcp: failed to parse tool result: %w", err)
	}
	return &res, nil
}

// ReadResource retrieves a resource from the MCP server.
func (c *MCPClient) ReadResource(ctx context.Context, uri string) ([]MCPResourceContent, error) {
	raw, err := c.Transport.SendRequest(ctx, "resources/read", map[string]interface{}{
		"uri": uri,
	})
	if err != nil {
		return nil, err
	}

	var res struct {
		Contents []MCPResourceContent `json:"contents"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	return res.Contents, nil
}

// GetPrompt retrieves a prompt template from the MCP server.
func (c *MCPClient) GetPrompt(ctx context.Context, name string, args map[string]string) (string, error) {
	raw, err := c.Transport.SendRequest(ctx, "prompts/get", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}

	var res struct {
		Messages []struct {
			Content struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", err
	}
	if len(res.Messages) > 0 {
		return res.Messages[0].Content.Text, nil
	}
	return "", nil
}

func (c *MCPClient) listTools(ctx context.Context) ([]MCPToolDefinition, error) {
	params := map[string]interface{}{}
	raw, err := c.Transport.SendRequest(ctx, "tools/list", params)
	if err != nil {
		return nil, err
	}

	var res struct {
		Tools []MCPToolDefinition `json:"tools"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	return res.Tools, nil
}

func (c *MCPClient) listResources(ctx context.Context) ([]MCPResourceDefinition, error) {
	params := map[string]interface{}{}
	raw, err := c.Transport.SendRequest(ctx, "resources/list", params)
	if err != nil {
		return nil, err
	}

	var res struct {
		Resources []MCPResourceDefinition `json:"resources"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	return res.Resources, nil
}

func (c *MCPClient) listPrompts(ctx context.Context) ([]MCPPromptDefinition, error) {
	params := map[string]interface{}{}
	raw, err := c.Transport.SendRequest(ctx, "prompts/list", params)
	if err != nil {
		return nil, nil
	}

	var res struct {
		Prompts []MCPPromptDefinition `json:"prompts"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, err
	}
	return res.Prompts, nil
}


// ---------------------------------------------------------------------------
// MCP Server — Expose Crew-GO Tools via MCP
// ---------------------------------------------------------------------------

// MCPToolHandler is a function that executes a tool call.
type MCPToolHandler func(ctx context.Context, params map[string]interface{}) (*MCPToolResult, error)

// MCPServer exposes Crew-GO tools as MCP-compatible endpoints.
type MCPServer struct {
	mu         sync.RWMutex
	tools      map[string]mcpRegisteredTool
	resources  map[string]MCPResourceDefinition
	sseClients []chan string // For SSE notifications
}

type mcpRegisteredTool struct {
	Definition MCPToolDefinition
	Handler    MCPToolHandler
}

// NewMCPServer creates an MCP server.
func NewMCPServer() *MCPServer {
	return &MCPServer{
		tools:      make(map[string]mcpRegisteredTool),
		resources:  make(map[string]MCPResourceDefinition),
		sseClients: make([]chan string, 0),
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

// Handler returns an http.Handler that serves MCP requests (SSE and POST).
func (s *MCPServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)
	
	// Legacy support for single POST endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.handleMessage(w, r)
		} else {
			http.Error(w, "Use /sse for connection", http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func (s *MCPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// 1. Send the endpoint event so the client knows where to POST
	// For simplicity, we assume the POST endpoint is at /message on the same host
	fmt.Fprintf(w, "event: endpoint\ndata: /message\n\n")
	flusher.Flush()

	// 2. Register this client for notifications
	ch := make(chan string, 10)
	s.mu.Lock()
	s.sseClients = append(s.sseClients, ch)
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		for i, c := range s.sseClients {
			if c == ch {
				s.sseClients = append(s.sseClients[:i], s.sseClients[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}()

	// 3. Keep connection alive
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-time.After(30 * time.Second):
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (s *MCPServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	slog.Info("[📥 MCP REQUEST]", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.String("remote", r.RemoteAddr))
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
