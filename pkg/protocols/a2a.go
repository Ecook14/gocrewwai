// Package protocols implements interoperability protocols for Gocrew agents.
//
// This package provides two major protocol implementations:
//
//   - A2A (Agent-to-Agent): Enables agents to discover, communicate with,
//     and delegate tasks to other agents across process and network boundaries.
//
//   - MCP (Model Context Protocol): Enables agents to discover and invoke
//     tools hosted on MCP-compatible servers, and expose Gocrew tools as
//     MCP resources.
package protocols

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"go.opentelemetry.io/otel/trace"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, add origin validation
	},
}

// ---------------------------------------------------------------------------
// A2A Client — Sending Messages
// ---------------------------------------------------------------------------

// A2AClient handles sending messages to remote agents.
type A2AClient struct {
	httpClient *http.Client
	AuthToken  string
	
	// Simple Circuit Breaker per endpoint
	mu       sync.Mutex
	failures map[string]int
	lastFail map[string]time.Time
}

func NewA2AClient(authToken string) *A2AClient {
	return &A2AClient{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		AuthToken:  authToken,
		failures:   make(map[string]int),
		lastFail:   make(map[string]time.Time),
	}
}

// Send sends an A2AMessage to the specified endpoint and returns the response.
func (c *A2AClient) Send(ctx context.Context, endpoint string, msg A2AMessage) (*A2AMessage, error) {
	return c.SendWithRetry(ctx, endpoint, msg, 3)
}

// GetStatus checks the health and status of a remote agent.
func (c *A2AClient) GetStatus(ctx context.Context, endpoint string, fromAgentID string, toAgentID string) (map[string]interface{}, error) {
	msg := A2AMessage{
		ID:        fmt.Sprintf("stat-%d", time.Now().UnixNano()),
		From:      fromAgentID,
		To:        toAgentID,
		Type:      A2ARequest,
		Action:    "status",
		Timestamp: time.Now(),
	}

	resp, err := c.Send(ctx, endpoint, msg)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

// SendWithRetry sends a message with exponential backoff.
func (c *A2AClient) SendWithRetry(ctx context.Context, endpoint string, msg A2AMessage, maxRetries int) (*A2AMessage, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			backoff := time.Duration(1<<uint(i)) * time.Second
			slog.Debug("a2a: retrying message", slog.String("id", msg.ID), slog.Duration("backoff", backoff))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.doSend(ctx, endpoint, msg)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		
		// Don't retry on certain errors (e.g., Auth failure)
		if strings.Contains(err.Error(), "status 401") || strings.Contains(err.Error(), "status 403") {
			return nil, err
		}
	}
	return nil, fmt.Errorf("a2a: all retries failed: %w", lastErr)
}

// Stream initiates a streaming task delegation and returns a channel for tokens.
func (c *A2AClient) Stream(ctx context.Context, endpoint string, msg A2AMessage) (<-chan string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	u.Scheme = "ws"
	if strings.HasPrefix(endpoint, "https") {
		u.Scheme = "wss"
	}
	u.Path = "/ws"

	header := http.Header{}
	if c.AuthToken != "" {
		header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return nil, fmt.Errorf("a2a: ws dial error: %w", err)
	}

	// Send initial task message
	if err := conn.WriteJSON(msg); err != nil {
		conn.Close()
		return nil, err
	}

	tokenCh := make(chan string, 100)
	go func() {
		defer close(tokenCh)
		defer conn.Close()
		for {
			var streamMsg A2AMessage
			if err := conn.ReadJSON(&streamMsg); err != nil {
				return
			}
			if streamMsg.Type == A2AStream {
				token, _ := streamMsg.Payload["token"].(string)
				tokenCh <- token
			} else if streamMsg.Type == A2AResponse {
				// Final result can also come via WebSocket
				return
			}
		}
	}()

	return tokenCh, nil
}

func (c *A2AClient) doSend(ctx context.Context, endpoint string, msg A2AMessage) (*A2AMessage, error) {
	if err := c.checkCircuit(endpoint); err != nil {
		return nil, err
	}

	// Otel Trace Propagation
	if span := telemetry.GetSpan(ctx); span != nil {
		msg.TraceID = span.SpanContext().TraceID().String()
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("a2a: failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-A2A-Version", "1.0")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.markFailure(endpoint)
		return nil, fmt.Errorf("a2a: network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 500 {
			c.markFailure(endpoint)
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("a2a: remote error (status %d): %s", resp.StatusCode, string(body))
	}

	c.markSuccess(endpoint)
	var response A2AMessage
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("a2a: failed to decode response: %w", err)
	}

	return &response, nil
}

func (c *A2AClient) checkCircuit(endpoint string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.failures[endpoint] >= 5 {
		if time.Since(c.lastFail[endpoint]) < 30*time.Second {
			return fmt.Errorf("a2a: circuit breaker open for %s", endpoint)
		}
		// Half-open: allow one request
		c.failures[endpoint] = 4 
	}
	return nil
}

func (c *A2AClient) markFailure(endpoint string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures[endpoint]++
	c.lastFail[endpoint] = time.Now()
}

func (c *A2AClient) markSuccess(endpoint string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.failures, endpoint)
}

// ---------------------------------------------------------------------------
// A2A Server — Receiving Messages
// ---------------------------------------------------------------------------

// A2AServer hosts a local agent and listens for incoming messages.
type A2AServer struct {
	Port      int
	Router    *A2ARouter
	AuthToken string
	server    *http.Server
}

func NewA2AServer(port int, router *A2ARouter, authToken string) *A2AServer {
	return &A2AServer{
		Port:      port,
		Router:    router,
		AuthToken: authToken,
	}
}

// Start launches the A2A server in a background goroutine.
func (s *A2AServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleMessage)
	mux.HandleFunc("/ws", s.handleWebSocket)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.Port),
		Handler: mux,
	}

	go func() {
		slog.Info("🚀 A2A Server starting", slog.Int("port", s.Port))
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("a2a: server error", slog.Any("error", err))
		}
	}()

	return nil
}

func (s *A2AServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("a2a: ws upgrade failed", slog.Any("error", err))
		return
	}
	defer conn.Close()

	// Initial message must be the task delegation
	var msg A2AMessage
	if err := conn.ReadJSON(&msg); err != nil {
		return
	}

	// Auth Validation (same as handleMessage)
	if s.AuthToken != "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Subprotocols or query params could also be used for WS auth, 
			// but here we check the initial header.
			authHeader = r.URL.Query().Get("token")
		}
		if authHeader != "Bearer "+s.AuthToken && authHeader != s.AuthToken {
			slog.Warn("a2a: ws unauthorized access attempt")
			return
		}
	}

	// We need a way to pass the "stream back" capability to the handler.
	// We'll use the context for this.
	ctx := context.WithValue(r.Context(), "a2a_ws_conn", conn)
	
	response, err := s.Router.Route(ctx, msg)
	if err != nil {
		_ = conn.WriteJSON(A2AMessage{Type: A2AError, Payload: map[string]interface{}{"error": err.Error()}})
		return
	}

	if response != nil {
		_ = conn.WriteJSON(response)
	}
}

func (s *A2AServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Auth Validation
	if s.AuthToken != "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+s.AuthToken {
			slog.Warn("a2a: unauthorized access attempt", slog.String("remote", r.RemoteAddr))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var msg A2AMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid message format", http.StatusBadRequest)
		return
	}

	// Route the message
	ctx := r.Context()
	if msg.TraceID != "" {
		// In a real Otel setup, we'd use a propagator, but here we simulate 
		// by starting a new span with the context and logging the trace ID.
		var span trace.Span
		ctx, span = telemetry.StartSpan(ctx, "A2A."+msg.Action)
		if span != nil {
			defer span.End()
			slog.Debug("📊 Joined distributed trace", slog.String("trace_id", msg.TraceID))
		}
	}

	response, err := s.Router.Route(ctx, msg)
	if err != nil {
		slog.Error("a2a: routing error", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if response == nil {
		// Async acknowledgement if no immediate response
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *A2AServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// ---------------------------------------------------------------------------
// A2A Protocol — Agent-to-Agent Communication
// ---------------------------------------------------------------------------

// Register is a global registry for Agent-to-Agent discovery.
var GlobalA2ARegistry = NewAgentRegistry()

// AgentCard declares an agent's identity, capabilities, and endpoint.
// This is the core discovery object in the A2A protocol.
type AgentCard struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Role         string            `json:"role"`
	Description  string            `json:"description,omitempty"`
	Capabilities []string          `json:"capabilities"` // e.g., ["research", "code_review", "writing"]
	Endpoint     string            `json:"endpoint"`     // HTTP endpoint for receiving messages
	Version      string            `json:"version,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

// A2AMessage is the standard envelope for inter-agent communication.
type A2AMessage struct {
	ID          string                 `json:"id"`
	From        string                 `json:"from"`         // Sender agent ID
	To          string                 `json:"to"`           // Recipient agent ID
	Type        A2AMessageType         `json:"type"`         // request, response, event, error
	Action      string                 `json:"action"`       // What the sender wants (e.g., "delegate_task", "ask_question")
	Payload     map[string]interface{} `json:"payload"`
	CorrelationID string              `json:"correlation_id,omitempty"` // Links request ↔ response
	TraceID       string              `json:"trace_id,omitempty"`       // OpenTelemetry Trace Context
	Timestamp     time.Time              `json:"timestamp"`
}

// A2AMessageType categorizes inter-agent messages.
type A2AMessageType string

const (
	A2ARequest  A2AMessageType = "request"
	A2AResponse A2AMessageType = "response"
	A2AEvent    A2AMessageType = "event"
	A2AError    A2AMessageType = "error"
	A2AStream   A2AMessageType = "stream" // For incremental token streaming
)

// A2ATaskRequest is the payload for delegating a task to another agent.
type A2ATaskRequest struct {
	Description    string                 `json:"description"`
	ExpectedOutput string                 `json:"expected_output,omitempty"`
	Context        string                 `json:"context,omitempty"`
	Priority       int                    `json:"priority,omitempty"` // 1=highest, 5=lowest
	Deadline       time.Time              `json:"deadline,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// A2ATaskResponse is the payload for returning a task result.
type A2ATaskResponse struct {
	Result  string `json:"result"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ---------------------------------------------------------------------------
// Agent Registry — Discovery Service
// ---------------------------------------------------------------------------

// AgentRegistry provides agent discovery for the A2A protocol.
type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]*AgentCard
}

// NewAgentRegistry creates an empty registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*AgentCard),
	}
}

// Register adds an agent card to the registry.
func (r *AgentRegistry) Register(card *AgentCard) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if card.CreatedAt.IsZero() {
		card.CreatedAt = time.Now()
	}
	r.agents[card.ID] = card
}

// Unregister removes an agent from the registry.
func (r *AgentRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, id)
}

// Lookup finds an agent by ID.
func (r *AgentRegistry) Lookup(id string) (*AgentCard, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	card, ok := r.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return card, nil
}

// FindByCapability returns all agents that declare the given capability.
func (r *AgentRegistry) FindByCapability(capability string) []*AgentCard {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*AgentCard
	for _, card := range r.agents {
		for _, cap := range card.Capabilities {
			if cap == capability {
				matches = append(matches, card)
				break
			}
		}
	}
	return matches
}

// ListAll returns all registered agent cards.
func (r *AgentRegistry) ListAll() []*AgentCard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*AgentCard, 0, len(r.agents))
	for _, card := range r.agents {
		result = append(result, card)
	}
	return result
}

// ---------------------------------------------------------------------------
// A2A Message Handler
// ---------------------------------------------------------------------------

// A2AHandler processes incoming A2A messages.
type A2AHandler func(ctx context.Context, msg A2AMessage) (*A2AMessage, error)

// A2ARouter routes incoming messages to registered handlers by action.
type A2ARouter struct {
	mu       sync.RWMutex
	handlers map[string]A2AHandler
	fallback A2AHandler
}

// NewA2ARouter creates a message router.
func NewA2ARouter() *A2ARouter {
	return &A2ARouter{
		handlers: make(map[string]A2AHandler),
	}
}

// Handle registers a handler for a specific action.
func (r *A2ARouter) Handle(action string, handler A2AHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[action] = handler
}

// SetFallback sets a handler for unrecognized actions.
func (r *A2ARouter) SetFallback(handler A2AHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = handler
}

// Route dispatches a message to the appropriate handler.
func (r *A2ARouter) Route(ctx context.Context, msg A2AMessage) (*A2AMessage, error) {
	r.mu.RLock()
	handler, ok := r.handlers[msg.Action]
	fallback := r.fallback
	r.mu.RUnlock()

	if ok {
		return handler(ctx, msg)
	}
	if fallback != nil {
		return fallback(ctx, msg)
	}
	return nil, fmt.Errorf("no handler for action: %s", msg.Action)
}

// ---------------------------------------------------------------------------
// Serialization Helpers
// ---------------------------------------------------------------------------

// MarshalTaskRequest converts a task request into a generic payload map.
func MarshalTaskRequest(req A2ATaskRequest) map[string]interface{} {
	data, _ := json.Marshal(req)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}

// UnmarshalTaskRequest extracts a task request from a message payload.
func UnmarshalTaskRequest(payload map[string]interface{}) (*A2ATaskRequest, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var req A2ATaskRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// MarshalTaskResponse converts a task response into a generic payload map.
func MarshalTaskResponse(res A2ATaskResponse) map[string]interface{} {
	data, _ := json.Marshal(res)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}

// UnmarshalTaskResponse extracts a task response from a message payload.
func UnmarshalTaskResponse(payload map[string]interface{}) (*A2ATaskResponse, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var res A2ATaskResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
