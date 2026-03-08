package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/Ecook14/crewai-go/pkg/telemetry"
	"github.com/gorilla/websocket"
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
	http.HandleFunc("/ws", s.handleConnections)
	http.HandleFunc("/api/review", s.handleReview)
	http.HandleFunc("/api/start", s.handleStart)
	http.HandleFunc("/api/stop", s.handleStop)

	// Serve static files for the dashboard
	fs := http.FileServer(http.Dir("web-ui"))
	http.Handle("/web-ui/", http.StripPrefix("/web-ui/", fs))

	go s.handleMessages()

	slog.Info("🚀 WebSocket Server started", slog.String("port", port))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("WebSocket server failed", slog.Any("error", err))
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

func (s *WSServer) handleMessages() {
	// Subscribe to the Global Event Bus
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

// StartDashboardServer is a helper to launch the UI backend.
func StartDashboardServer(port string) {
	server := NewWSServer()
	server.Start(port)
}
