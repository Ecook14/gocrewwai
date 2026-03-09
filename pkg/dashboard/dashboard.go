package dashboard

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"strings"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/memory"
	"github.com/Ecook14/gocrewwai/pkg/protocols"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/Ecook14/gocrewwai/pkg/tools"
	"github.com/gorilla/websocket"
	"github.com/Ecook14/gocrewwai/web-ui"
	"runtime"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for the dashboard
	},
}

// WSServer manages WebSocket connections and broadcasts telemetry events.
type WSServer struct {
	clients   map[*websocket.Conn]bool
	broadcast chan telemetry.Event
	mu        sync.Mutex
}

func NewWSServer() *WSServer {
	return &WSServer{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan telemetry.Event),
	}
}

// Start launches the WebSocket server and begins broadcasting events.
func (s *WSServer) Start(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleConnections)
	mux.HandleFunc("/api/review", s.handleReview)
	mux.HandleFunc("/api/start", s.handleStart)
	mux.HandleFunc("/api/stop", s.handleStop)

	// Entity Creation Endpoints
	mux.HandleFunc("/api/create/agent", s.handleCreateAgent)
	mux.HandleFunc("/api/create/task", s.handleCreateTask)
	mux.HandleFunc("/api/create/mcp", s.handleCreateMCP)
	mux.HandleFunc("/api/create/a2a", s.handleCreateA2A)
	mux.HandleFunc("/api/config", s.handleUpdateConfig)

	// Metadata Endpoints
	mux.HandleFunc("/api/tools", s.handleListTools)
	mux.HandleFunc("/api/memory", s.handleListMemoryProviders)
	mux.HandleFunc("/api/providers", s.handleListProviders)
	mux.HandleFunc("/api/list", s.handleListAll)
	mux.HandleFunc("/api/delete", s.handleDeleteEntity)

	// Serve static files for the dashboard from the embedded binary
	fs := http.FileServer(http.FS(webui.Files))
	mux.Handle("/web-ui/", http.StripPrefix("/web-ui/", fs))

	// Initialize default tools for UI
	tools.InitDefaultRegistry()

	go s.handleMessages()
	go s.publishMetrics()

	slog.Info("🚀 Crew-GO Dashboard Server started", slog.String("port", port))
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("Dashboard server failed", slog.Any("error", err))
	}
}

func (s *WSServer) handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", slog.Any("error", err))
		return
	}
	defer ws.Close()

	s.mu.Lock()
	s.clients[ws] = true
	s.mu.Unlock()

	slog.Info("New Dashboard client connected")

	// Keep connection alive
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			s.mu.Lock()
			delete(s.clients, ws)
			s.mu.Unlock()
			slog.Info("Dashboard client disconnected")
			break
		}
	}
}

func (s *WSServer) handleReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ReviewID string `json:"review_id"`
		Approved bool   `json:"approved"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	telemetry.GlobalReviewManager.SubmitReview(req.ReviewID, req.Approved)
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *WSServer) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	telemetry.GlobalExecutionController.Resume()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"started"}`))
}

func (s *WSServer) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	telemetry.GlobalExecutionController.Pause()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"stopped"}`))
}

func (s *WSServer) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Role         string `json:"role"`
		Goal         string `json:"goal"`
		Backstory    string `json:"backstory"`
		Provider     string `json:"provider"`
		LLMModel     string `json:"llm_model"`
		APIKey       string `json:"api_key"`
		MaxRPM       int    `json:"max_rpm"`
		Memory       string `json:"memory"`
		MemoryConfig struct {
			ConnectionString string `json:"connection_string"`
		} `json:"memory_config"`
		Tools           []string `json:"tools"`
		AllowDelegation bool     `json:"allow_delegation"`
		SelfHealing     bool     `json:"self_healing"`
		Index           *int     `json:"index"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Dynamic Creation Logic
	var client llm.Client
	apiKey := req.APIKey
	
	switch strings.ToLower(req.Provider) {
	case "openai":
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		client = llm.NewOpenAIClient(apiKey)
	case "anthropic":
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		client = llm.NewAnthropicClient(apiKey, req.LLMModel)
	case "google_gemini", "google gemini", "gemini":
		if apiKey == "" {
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
		client = llm.NewGeminiClient(apiKey, req.LLMModel)
	case "openrouter":
		if apiKey == "" {
			apiKey = os.Getenv("OPENROUTER_API_KEY")
		}
		client = llm.NewOpenRouterClient(apiKey, req.LLMModel)
	case "groq":
		if apiKey == "" {
			apiKey = os.Getenv("GROQ_API_KEY")
		}
		client = llm.NewGroqClient(apiKey, req.LLMModel)
	default:
		// Default to OpenAI for backward compatibility or if not specified
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		client = llm.NewOpenAIClient(apiKey)
	}

	agent := &agents.Agent{
		Role:      req.Role,
		Goal:      req.Goal,
		Backstory: req.Backstory,
		Provider:  req.Provider,
		LLMModel:  req.LLMModel,
		LLM:       client,
		MaxRPM:    req.MaxRPM,
		AllowDelegation: req.AllowDelegation,
		SelfHealing:     req.SelfHealing,
		Verbose:   true,
	}

	// Dynamic Memory Initialization
	if req.Memory != "" && req.Memory != "None" {
		connStr := req.MemoryConfig.ConnectionString
		switch req.Memory {
		case "SQLite (Local)":
			dbPath := "memory.db"
			if connStr != "" {
				dbPath = connStr
			}
			if store, err := memory.NewSQLiteStore(dbPath); err == nil {
				agent.Memory = store
			}
		case "ChromaDB":
			if store, err := memory.NewChromaStore(connStr, "crew_memory", 0); err == nil {
				agent.Memory = store
			}
		case "Pinecone (Vector)":
			// Assuming connStr is "host|apiKey|namespace" for now or just host
			if store, err := memory.NewPineconeStore(connStr, req.APIKey, "crew_memory"); err == nil {
				agent.Memory = store
			}
		case "Qdrant (Vector)":
			if store, err := memory.NewQdrantStore(connStr, "crew_memory", 1536); err == nil {
				agent.Memory = store
			}
		case "Remote (REST/gRPC)":
			// Default to Redis if "Remote" is selected
			if store, err := memory.NewRedisStore([]string{connStr}, "", 0, "crew_memory:"); err == nil {
				agent.Memory = store
			}
		}
	}

	// Tool Assignment (Fixing the reported issue)
	if len(req.Tools) > 0 {
		for _, toolName := range req.Tools {
			if t, err := tools.GlobalRegistry.Get(toolName); err == nil {
				agent.Tools = append(agent.Tools, t)
			} else {
				// Try dynamic creation if not in registry
				if dt, err := tools.CreateTool(toolName, nil); err == nil {
					agent.Tools = append(agent.Tools, dt)
				} else {
					slog.Warn("⚠️ Tool not found or failed to create", slog.String("tool", toolName))
				}
			}
		}
	}

	if req.Index != nil {
		telemetry.GlobalDynamicRegistry.UpdateAgent(*req.Index, agent)
		slog.Info("Updated Agent via UI", slog.String("role", req.Role), slog.Int("index", *req.Index))
	} else {
		telemetry.GlobalDynamicRegistry.AddAgent(agent, true)
		slog.Info("Staged New Agent via UI", slog.String("role", req.Role))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "created",
		"agent": req,
	})
}

func (s *WSServer) handleCreateMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Index   *int     `json:"index"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := protocols.NewMCPClient(req.Command)
	if req.Index != nil {
		telemetry.GlobalDynamicRegistry.UpdateMCP(*req.Index, client)
	} else {
		telemetry.GlobalDynamicRegistry.AddMCPClient(client, true)
	}

	slog.Info("Staged New MCP Server via UI", slog.String("mcp_name", req.Name), slog.String("command", req.Command))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "created",
		"mcp": req,
	})
}

func (s *WSServer) handleCreateA2A(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
		Model    string `json:"model"`
		Index    *int   `json:"index"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bridge := map[string]string{
		"sender":   req.Sender,
		"receiver": req.Receiver,
		"model":    req.Model,
	}

	if req.Index != nil {
		telemetry.GlobalDynamicRegistry.UpdateA2A(*req.Index, bridge)
	} else {
		telemetry.GlobalDynamicRegistry.AddA2ABridge(bridge, true)
	}

	slog.Info("Staged New A2A Bridge via UI", slog.String("sender", req.Sender), slog.String("receiver", req.Receiver))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "created",
		"a2a": req,
	})
}

func (s *WSServer) handleListTools(w http.ResponseWriter, r *http.Request) {
	names := tools.GlobalRegistry.ListNames()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(names)
}

func (s *WSServer) handleListMemoryProviders(w http.ResponseWriter, r *http.Request) {
	providers := []string{
		"SQLite (Local)",
		"Pinecone (Vector)",
		"Qdrant (Vector)",
		"ChromaDB",
		"Remote (REST/gRPC)",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(providers)
}

func (s *WSServer) handleListProviders(w http.ResponseWriter, r *http.Request) {
	providers := []string{
		"OpenAI",
		"Anthropic",
		"Google Gemini",
		"OpenRouter",
		"Groq",
		"Ollama",
		"Custom",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(providers)
}

func (s *WSServer) handleListAll(w http.ResponseWriter, r *http.Request) {
	data := telemetry.GlobalDynamicRegistry.ListAll()
	
	agentsCount := 0
	if agents, ok := data["agents"].([]interface{}); ok {
		agentsCount = len(agents)
	}
	tasksCount := 0
	if tasks, ok := data["tasks"].([]interface{}); ok {
		tasksCount = len(tasks)
	}

	slog.Info("Syncing Dashboard Entities", slog.Int("agents", agentsCount), slog.Int("tasks", tasksCount))
	
	// Marshal first to ensure we don't send a partial 200 OK response
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to marshal entity list", slog.Any("error", err))
		http.Error(w, "Internal Server Error: Failed to generate entity list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func (s *WSServer) handleDeleteEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Type {
	case "agent":
		telemetry.GlobalDynamicRegistry.DeleteAgent(req.Index)
	case "task":
		telemetry.GlobalDynamicRegistry.DeleteTask(req.Index)
	case "mcp":
		telemetry.GlobalDynamicRegistry.DeleteMCP(req.Index)
	case "a2a":
		telemetry.GlobalDynamicRegistry.DeleteA2A(req.Index)
	default:
		http.Error(w, "Invalid entity type", http.StatusBadRequest)
		return
	}

	slog.Info("Deleted entity via UI", slog.String("type", req.Type), slog.Int("index", req.Index))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"deleted"}`))
}

func (s *WSServer) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Description    string `json:"description"`
		AgentRole      string `json:"agent_role"`
		ExpectedOutput string `json:"expected_output"`
		AsyncExecution bool   `json:"async_execution"`
		Timeout        int    `json:"timeout"`
		Index          *int   `json:"index"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task := &tasks.Task{
		Description:    req.Description,
		AgentRole:      req.AgentRole,
		ExpectedOutput: req.ExpectedOutput,
		AsyncExecution: req.AsyncExecution,
		Timeout:        time.Duration(req.Timeout) * time.Second,
	}

	if req.Index != nil {
		telemetry.GlobalDynamicRegistry.UpdateTask(*req.Index, task)
	} else {
		telemetry.GlobalDynamicRegistry.AddTask(task, true)
	}
	slog.Info("Staged New Task via UI", slog.String("description", req.Description))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "created",
		"task": req,
	})
}

func (s *WSServer) handleMessages() {
	eventCh := telemetry.GlobalBus.Subscribe()
	defer telemetry.GlobalBus.Unsubscribe(eventCh)

	for event := range eventCh {
		s.mu.Lock()
		for client := range s.clients {
			err := client.WriteJSON(event)
			if err != nil {
				slog.Error("WebSocket write error", slog.Any("error", err))
				client.Close()
				delete(s.clients, client)
			}
		}
		s.mu.Unlock()
	}
}

func (s *WSServer) publishMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	startTime := time.Now()

	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		event := telemetry.Event{
			Type:      "system_metrics",
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"memory_mb":     m.Alloc / 1024 / 1024,
				"goroutines":    runtime.NumGoroutine(),
				"uptime_secs":   time.Since(startTime).Seconds(),
			},
		}

		s.mu.Lock()
		for client := range s.clients {
			client.WriteJSON(event)
		}
		s.mu.Unlock()
	}
}

// Start launches the Dashboard Server in a non-blocking goroutine.
func Start(port string) {
	server := NewWSServer()
	go server.Start(port)
}

func (s *WSServer) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProcessType string `json:"process_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	telemetry.GlobalDynamicRegistry.SetProcessType(req.ProcessType)
	slog.Info("Updated Global Process Type via UI", slog.String("process_type", req.ProcessType))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"updated"}`))
}
