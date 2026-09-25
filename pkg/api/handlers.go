package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/core"
	"github.com/Ecook14/gocrewwai/pkg/crew"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

// handleKickoff accepts a full crew execution request and starts execution.
// It validates the payload, constructs the crew from the definition, persists
// the session, and dispatches execution to the Crew engine.
func (s *Server) handleKickoff(c *gin.Context) {
	var payload struct {
		SessionID          string   `json:"session_id"`
		AgentRole          string   `json:"agent_role"`
		AgentGoal          string   `json:"agent_goal"`
		AgentBackstory     string   `json:"agent_backstory"`
		AgentModel         string   `json:"agent_model"`
		AgentSystemPrompt  string   `json:"agent_system_prompt"`
		TaskDescription    string   `json:"task_description"`
		TaskExpectedOutput string   `json:"task_expected_output"`
		TaskTools          []string `json:"task_tools"`
		CrewProcess        string   `json:"crew_process"`
		MaxIterations      int      `json:"max_iterations"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if payload.SessionID == "" {
		payload.SessionID = fmt.Sprintf("sess_%d", time.Now().UnixMilli())
	}

	if payload.AgentRole == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'agent_role' is required"})
		return
	}

	if payload.TaskDescription == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'task_description' is required"})
		return
	}

	// Build the agent from the request payload.
	// When an LLM client is not supplied in the request, the agent is created
	// without one — the caller is responsible for injecting a valid llm.Client
	// before execution. The handler rejects the request if neither a model name
	// nor an existing LLM client is available, because an agent without an LLM
	// cannot execute.
	var llmClient llm.Client
	if payload.AgentModel != "" {
		if tc := llm.NewOpenAIClient(""); tc != nil {
			llmClient = tc
		}
	}

	agentOpts := []agents.AgentOption{
		agents.WithMaxIterations(payload.MaxIterations),
	}

	agent := agents.NewAgentLegacy(
		payload.AgentRole,
		payload.AgentGoal,
		payload.AgentBackstory,
		llmClient,
		agentOpts...,
	)

	if agent == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to construct agent",
		})
		return
	}

	// Build the task from the request payload. The agent is attached to the task
	// so the crew can dispatch work to it during execution.
	task := tasks.NewTask(payload.TaskDescription, agent)

	// Build the crew. NewCrew accepts agents first, then tasks.
	crw := crew.NewCrew([]core.Agent{agent}, []*tasks.Task{task})

	// Persist the session as "running" via the checkpoint backend.
	if err := s.persistSessionStart(payload.SessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to persist session: %v", err),
		})
		return
	}

	// Dispatch execution asynchronously so the HTTP response returns immediately.
	go func() {
		if _, err := crw.Kickoff(context.Background()); err != nil {
			s.persistSessionFailure(payload.SessionID, err.Error())
		} else {
			s.persistSessionComplete(payload.SessionID)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Crew execution started",
		"session_id": payload.SessionID,
	})
}

// persistSessionStart records that a session has been created and is running.
// It writes a checkpoint via the SQLite backend when available, otherwise
// falls back to the in-memory session tracker.
func (s *Server) persistSessionStart(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = SessionState{
		SessionID: sessionID,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}

	if s.checkpointStore != nil {
		cp := &crew.Checkpoint{
			CrewID:  sessionID,
			Status:  "running",
			State:   map[string]interface{}{"session_id": sessionID},
			Version: 1,
		}
		if err := s.checkpointStore.Save(context.Background(), cp); err != nil {
			s.logger.Log(telemetry.AuditEntry{
				EventType: "session_checkpoint",
				Action:    "persist_session_start",
				Success:   false,
				Error:     err.Error(),
			})
		}
	}

	return nil
}

// persistSessionFailure records that a session failed.
func (s *Server) persistSessionFailure(sessionID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if st, ok := s.sessions[sessionID]; ok {
		st.Status = "failed"
		st.Error = reason
		st.FinishedAt = time.Now().UTC()
		s.sessions[sessionID] = st
	}

	if s.checkpointStore != nil {
		cp := &crew.Checkpoint{
			CrewID:  sessionID,
			Status:  "failed",
			Error:   reason,
			State:   map[string]interface{}{"session_id": sessionID},
			Version: 1,
		}
		if err := s.checkpointStore.Save(context.Background(), cp); err != nil {
			s.logger.Log(telemetry.AuditEntry{
				EventType: "session_checkpoint",
				Action:    "persist_session_failure",
				Success:   false,
				Error:     err.Error(),
			})
		}
	}

	return nil
}

// persistSessionComplete records that a session completed successfully.
func (s *Server) persistSessionComplete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if st, ok := s.sessions[sessionID]; ok {
		st.Status = "completed"
		st.FinishedAt = time.Now().UTC()
		s.sessions[sessionID] = st
	}

	if s.checkpointStore != nil {
		cp := &crew.Checkpoint{
			CrewID:  sessionID,
			Status:  "completed",
			State:   map[string]interface{}{"session_id": sessionID},
			Version: 1,
		}
		if err := s.checkpointStore.Save(context.Background(), cp); err != nil {
			s.logger.Log(telemetry.AuditEntry{
				EventType: "session_checkpoint",
				Action:    "persist_session_complete",
				Success:   false,
				Error:     err.Error(),
			})
		}
	}

	return nil
}

// SessionState holds the observable state of a running session.
type SessionState struct {
	SessionID  string                 `json:"session_id"`
	Status     string                 `json:"status"`
	StartedAt  time.Time              `json:"started_at,omitempty"`
	FinishedAt time.Time              `json:"finished_at,omitempty"`
	Result     map[string]interface{} `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// handleGetSession returns the current state of a session from the persistence layer.
func (s *Server) handleGetSession(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
		return
	}

	state, err := s.loadSessionState(id)
	if err != nil {
		if err == ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"session_id": id,
				"status":     "not_found",
				"error":      "session not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"session_id": id,
			"status":     "error",
			"error":      err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, state)
}

// loadSessionState resolves a session's current state from the persistence layer.
// It checks the in-memory session tracker first, then falls back to the
// checkpoint backend (SQLite / Redis) when configured.
func (s *Server) loadSessionState(sessionID string) (map[string]interface{}, error) {
	s.mu.RLock()
	st, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if ok {
		return map[string]interface{}{
			"session_id": st.SessionID,
			"status":     st.Status,
			"started_at": st.StartedAt.Format(time.RFC3339),
		}, nil
	}

	if s.checkpointStore != nil {
		cp, err := s.checkpointStore.LoadLatest(context.Background(), sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to load session checkpoint: %w", err)
		}
		if cp != nil {
			result := map[string]interface{}{
				"session_id": cp.CrewID,
				"status":     cp.Status,
			}
			if cp.Status == "failed" && cp.Error != "" {
				result["error"] = cp.Error
			}
			return result, nil
		}
	}

	return nil, ErrSessionNotFound
}

// ErrSessionNotFound is returned when a session ID has no recorded state.
var ErrSessionNotFound = fmt.Errorf("session not found")

// handleSSEStream streams events from the GlobalBus to the client.
func (s *Server) handleSSEStream(c *gin.Context) {
	// 1. Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 2. Subscribe to the GlobalBus
	eventChan := telemetry.GlobalBus.Subscribe()
	defer telemetry.GlobalBus.Unsubscribe(eventChan)

	// 3. Stream loop
	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return false
			}
			// Format as SSE data
			data, err := json.Marshal(event)
			if err != nil {
				return true // Skip bad events
			}
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
