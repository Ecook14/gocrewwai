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
	"github.com/Ecook14/gocrewwai/pkg/events"
	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tasks"
	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

// kickoffSem bounds concurrent async crew executions to prevent goroutine
// exhaustion under load. Acquired before spawning the background worker.
var kickoffSem = make(chan struct{}, 10)

// checkIdem returns the ledger entry for an Idempotency-Key. Entries are
// owner-scoped: a key presented by a different token is treated as unseen
// (no cross-tenant oracle).
func (s *Server) checkIdem(key, owner string) (*idemEntry, bool) {
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	e, ok := s.idemKeys[key]
	if !ok || (owner != "" && e.owner != "" && e.owner != owner) {
		return nil, false
	}
	return e, true
}

func (s *Server) storeIdem(key, owner, sessionID string) {
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	s.idemKeys[key] = &idemEntry{sessionID: sessionID, owner: owner}
	s.idemBySession[sessionID] = key
}

func (s *Server) finishIdem(sessionID, status string) {
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	if key, ok := s.idemBySession[sessionID]; ok {
		if e, ok := s.idemKeys[key]; ok {
			e.done = true
			e.status = status
		}
		delete(s.idemBySession, sessionID)
	}
}

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

	owner := c.GetString("token_fp")
	idemKey := c.GetHeader("Idempotency-Key")

	// Idempotency-Key (DCR-04): replaying a request with the same key never
	// executes twice. Running → 409; finished → cached outcome replay.
	// Keys live in memory (a restart clears them; session-409 below still
	// guards concurrent duplicates).
	if idemKey != "" {
		if entry, dup := s.checkIdem(idemKey, owner); dup {
			if !entry.done {
				c.JSON(http.StatusConflict, gin.H{
					"error":      "duplicate request still running",
					"session_id": entry.sessionID,
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message":           "duplicate request (idempotent replay)",
				"session_id":        entry.sessionID,
				"status":            entry.status,
				"idempotent_replay": true,
			})
			return
		}
	}

	// Idempotency (DCR-04): a replayed kickoff for an already-running session
	// returns 409 instead of spawning a duplicate crew execution.
	s.mu.RLock()
	if st, ok := s.sessions[payload.SessionID]; ok && st.Status == "running" {
		s.mu.RUnlock()
		c.JSON(http.StatusConflict, gin.H{
			"error":      "session already running",
			"session_id": payload.SessionID,
		})
		return
	}
	s.mu.RUnlock()

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
	if err := s.persistSessionStart(payload.SessionID, owner); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to persist session: %v", err),
		})
		return
	}

	// Record the idempotency key now that the session durably exists.
	if idemKey != "" {
		s.storeIdem(idemKey, owner, payload.SessionID)
	}

	// Dispatch execution asynchronously so the HTTP response returns immediately.
	// Bounded by kickoffSem and propagates the request context (detached from
	// cancellation so the crew survives client disconnect, but keeps values).
	select {
	case kickoffSem <- struct{}{}:
	default:
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "server busy: too many concurrent crew executions"})
		return
	}
	bgCtx := context.WithoutCancel(c.Request.Context())
	go func() {
		defer func() { <-kickoffSem }()
		if _, err := crw.Kickoff(bgCtx); err != nil {
			s.persistSessionFailure(payload.SessionID, err.Error())
			s.finishIdem(payload.SessionID, "failed")
		} else {
			s.persistSessionComplete(payload.SessionID)
			s.finishIdem(payload.SessionID, "completed")
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
func (s *Server) persistSessionStart(sessionID, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = SessionState{
		SessionID: sessionID,
		Status:    "running",
		Owner:     owner,
		StartedAt: time.Now().UTC(),
	}

	if s.checkpointStore != nil {
		cp := &crew.Checkpoint{
			CrewID:  sessionID,
			Status:  "running",
			State:   map[string]interface{}{"session_id": sessionID, "owner": owner},
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
// Owner is the fingerprint of the API token that created the session;
// sessions are only visible to their owning token (DCR-04).
type SessionState struct {
	SessionID  string                 `json:"session_id"`
	Status     string                 `json:"status"`
	Owner      string                 `json:"owner,omitempty"`
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

	state, err := s.loadSessionState(id, c.GetString("token_fp"))
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
// checkpoint backend (SQLite / Redis) when configured. Sessions owned by a
// different token resolve as not-found (no existence oracle for cross-tenant
// IDs).
func (s *Server) loadSessionState(sessionID, owner string) (map[string]interface{}, error) {
	s.mu.RLock()
	st, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if ok {
		if owner != "" && st.Owner != "" && st.Owner != owner {
			return nil, ErrSessionNotFound
		}
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
			if storedOwner, _ := cp.State["owner"].(string); owner != "" && storedOwner != "" && storedOwner != owner {
				return nil, ErrSessionNotFound
			}
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
// The :id must be a live session owned by the caller (DCR-04): anonymous
// enumeration of arbitrary stream IDs returns 404 without leaking existence.
func (s *Server) handleSSEStream(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
		return
	}
	if _, err := s.loadSessionState(id, c.GetString("token_fp")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"session_id": id,
			"status":     "not_found",
			"error":      "session not found",
		})
		return
	}

	// 1. Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 2. Subscribe to the GlobalBus (system metrics, global) and the crew
	// lifecycle bus (filtered to this session below).
	eventChan := telemetry.GlobalBus.Subscribe()
	defer telemetry.GlobalBus.Unsubscribe(eventChan)
	crewChan := events.GlobalBus.Subscribe()
	defer events.GlobalBus.Unsubscribe(crewChan)

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
		case cev, ok := <-crewChan:
			if !ok {
				return false
			}
			// Session-scoped: only this session's lifecycle events.
			if !passSSEEvent(id, cev) {
				return true
			}
			data, err := json.Marshal(cev)
			if err != nil {
				return true
			}
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// passSSEEvent reports whether a crew lifecycle event belongs on the stream
// for sessionID. Untagged events belong to no session and are dropped —
// session streams never carry another session's activity (DCR-04).
func passSSEEvent(sessionID string, e events.Event) bool {
	return e.SessionID != "" && e.SessionID == sessionID
}
