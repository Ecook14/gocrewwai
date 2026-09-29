package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
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
//
// kickoffSemMu serializes mutation of the channel variable so tests
// (which patch kickoffSem to induce saturation) cannot race with the
// in-flight kickoff goroutine reading the global. The mutex is the *only*
// lock protecting the variable identity; the channel itself is goroutine-safe.
var (
	kickoffSem   = make(chan struct{}, 10)
	kickoffSemMu sync.RWMutex
)

// acquireSem takes a slot, returning the exact channel it was taken from.
// The caller MUST pass that channel to releaseSem. Returning it is what makes
// the acquire/release pair correct when the global is reassigned in between:
// releasing via a re-read of the global would return the token to a different
// channel, permanently draining the one that was actually acquired from.
func acquireSem() (chan struct{}, bool) {
	kickoffSemMu.RLock()
	sem := kickoffSem
	kickoffSemMu.RUnlock()
	select {
	case sem <- struct{}{}:
		return sem, true
	default:
		return nil, false
	}
}

// releaseSem returns a slot to the same channel acquireSem took it from.
func releaseSem(sem chan struct{}) {
	<-sem
}

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

// newLLMClient builds the provider client for a kickoff agent. It is a
// package-level var so tests can substitute a stub and avoid real provider
// calls; production always uses the OpenAI client.
var newLLMClient = func(apiKey string) llm.Client { return llm.NewOpenAIClient(apiKey) }

// idemOp is the single internal mutation primitive for the idempotency
// ledger. Public methods (next 4) are thin wrappers so call sites stay
// readable; the lock contract lives in one place.
type idemOp int

const (
	idemOpReserve idemOp = iota
	idemOpRelease
	idemOpFinalize
	idemOpFinish
)

// doIdem applies one ledger mutation under the lock and reports whether the
// op was applied. For idemOpReserve that bool is the load-bearing result: it
// is an atomic test-and-set, and false means another request already owns the
// key, so the caller must abort rather than continue as if it had won.
func (s *Server) doIdem(op idemOp, key, owner, sessionID, status string) bool {
	if key == "" && op != idemOpFinish {
		return false
	}
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	switch op {
	case idemOpReserve:
		if _, exists := s.idemKeys[key]; exists {
			return false
		}
		s.idemKeys[key] = &idemEntry{sessionID: sessionID, owner: owner, createdAt: time.Now()}
		s.idemBySession[sessionID] = key
	case idemOpRelease:
		if e, ok := s.idemKeys[key]; ok && !e.done {
			delete(s.idemKeys, key)
			delete(s.idemBySession, e.sessionID)
		}
	case idemOpFinalize:
		// Preserve original createdAt so TTL eviction still bounds ledger age.
		e, ok := s.idemKeys[key]
		owner := ""
		createdAt := time.Now()
		if ok {
			owner = e.owner
			createdAt = e.createdAt
		}
		s.idemKeys[key] = &idemEntry{sessionID: sessionID, owner: owner, createdAt: createdAt}
		s.idemBySession[sessionID] = key
	case idemOpFinish:
		if key, ok := s.idemBySession[sessionID]; ok {
			if e, ok := s.idemKeys[key]; ok {
				e.done = true
				e.status = status
			}
			delete(s.idemBySession, sessionID)
		}
	}
	return true
}

// reserveIdem atomically claims an Idempotency-Key for a session. The pair
// must be reserved BEFORE the session is persisted so a concurrent request
// with the same key but a different session_id cannot pass both the
// checkIdem gate and the session-running map gate in the same window.
func (s *Server) reserveIdem(key, owner, sessionID string) bool {
	return s.doIdem(idemOpReserve, key, owner, sessionID, "")
}

// releaseIdem frees a reservation (used when the next gate rejects the request).
func (s *Server) releaseIdem(key string) { s.doIdem(idemOpRelease, key, "", "", "") }

// finalizeIdem marks the reservation complete (kickoff launched) without
// rolling back. Preserves the original createdAt so the TTL eviction sweep
// still bounds how long the ledger can hold a zombie entry.
func (s *Server) finalizeIdem(key, sessionID string) {
	s.doIdem(idemOpFinalize, key, "", sessionID, "")
}

// finishIdem marks the entry as completed/failed after the crew settles.
func (s *Server) finishIdem(sessionID, status string) {
	s.doIdem(idemOpFinish, "", "", sessionID, status)
}

// storeIdem (legacy public path used by tests) — TTL-evicts to make room
// before refusing, matching the original TestIdempotencyLedger contract.
func (s *Server) storeIdem(key, owner, sessionID string) {
	s.idemMu.Lock()
	defer s.idemMu.Unlock()
	if _, exists := s.idemKeys[key]; !exists && len(s.idemKeys) >= maxIdemKeys {
		now := time.Now()
		for k, e := range s.idemKeys {
			if now.Sub(e.createdAt) > idemEntryTTL {
				delete(s.idemKeys, k)
			}
			if len(s.idemKeys) < maxIdemKeys {
				break
			}
		}
		if len(s.idemKeys) >= maxIdemKeys {
			return
		}
	}
	s.idemKeys[key] = &idemEntry{sessionID: sessionID, owner: owner, createdAt: time.Now()}
	s.idemBySession[sessionID] = key
}

// Bounds for the multi-agent kickoff arrays. Sized like the existing flat
// limits (max=32 tools, max=1000 iterations): large enough for real crews,
// small enough that one request cannot exhaust the worker pool or the
// idempotency ledger with fan-out.
const (
	maxKickoffAgents = 10
	maxKickoffTasks  = 32
)

// kickoffAgent is one entry of the optional agents[] array. Only fields the
// handler actually honors are accepted here — notably there is no tools or
// system_prompt field, because the flat path already demonstrates that
// accepting fields and silently dropping them is worse than rejecting them.
// (Flat agent_system_prompt/task_tools remain accepted-but-ignored for
// backward compatibility and are documented as such.)
type kickoffAgent struct {
	Role          string `json:"role" binding:"required,max=256"`
	Goal          string `json:"goal" binding:"max=2000"`
	Backstory     string `json:"backstory" binding:"max=4000"`
	Model         string `json:"model" binding:"max=128"`
	APIKey        string `json:"api_key" binding:"max=512"`
	MaxIterations int    `json:"max_iterations" binding:"min=0,max=1000"`
}

// kickoffTask is one entry of the optional tasks[] array. AgentRole wires
// the task to an agent by role; it may be omitted only when the request
// defines exactly one agent.
type kickoffTask struct {
	Description    string `json:"description" binding:"required,max=20000"`
	ExpectedOutput string `json:"expected_output" binding:"max=20000"`
	AgentRole      string `json:"agent_role" binding:"max=256"`
}

// kickoffRequest is the POST /api/v1/crews/kickoff body. Two shapes are
// accepted and never mixed: the legacy flat single-agent fields, or the
// agents[]/tasks[] arrays for multi-agent crews. Flat agent_role and
// task_description carry no `required` tag (unlike before) because the
// multi-agent shape omits them; the flat path enforces presence in code so
// its 400 behavior is unchanged.
type kickoffRequest struct {
	SessionID          string         `json:"session_id" binding:"omitempty,max=128"`
	AgentRole          string         `json:"agent_role" binding:"max=256"`
	AgentGoal          string         `json:"agent_goal" binding:"max=2000"`
	AgentBackstory     string         `json:"agent_backstory" binding:"max=4000"`
	AgentModel         string         `json:"agent_model" binding:"max=128"`
	AgentSystemPrompt  string         `json:"agent_system_prompt" binding:"max=4000"`
	TaskDescription    string         `json:"task_description" binding:"max=20000"`
	TaskExpectedOutput string         `json:"task_expected_output" binding:"max=20000"`
	TaskTools          []string       `json:"task_tools" binding:"max=32,dive,max=128"`
	CrewProcess        string         `json:"crew_process" binding:"max=32"`
	MaxIterations      int            `json:"max_iterations" binding:"min=0,max=1000"`
	Agents             []kickoffAgent `json:"agents" binding:"max=10,dive"`
	Tasks              []kickoffTask  `json:"tasks" binding:"max=32,dive"`
}

// apiError is a rejection the handler renders as-is: status plus a message
// safe to expose (never includes secrets or internals).
type apiError struct {
	status  int
	message string
}

// handleKickoff accepts a full crew execution request and starts execution.
// It validates the payload, constructs the crew from the definition, persists
// the session, and dispatches execution to the Crew engine.
func (s *Server) handleKickoff(c *gin.Context) {
	var payload kickoffRequest

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// CrewAI-compatible process names only; anything else is rejected rather
	// than silently defaulting.
	switch payload.CrewProcess {
	case "", "sequential", "hierarchical", "consensual", "graph", "reflective", "state_machine", "state-machine":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported crew_process"})
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
		// Reserve the idempotency key BEFORE persisting the session so a
		// concurrent request with the same key can't slip past both
		// checkIdem (miss) and the session-running map (miss).
		//
		// reserveIdem is an atomic test-and-set: if it reports false, another
		// in-flight request won the race between our checkIdem miss and this
		// call, so we must return 409 rather than continue. Without this the
		// loser proceeded and both sessions were persisted, which is exactly
		// the duplicate-execution bug the gate exists to prevent.
		if !s.reserveIdem(idemKey, owner, payload.SessionID) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "duplicate request still running",
			})
			return
		}
	}

	// Idempotency (DCR-04): a replayed kickoff for an already-running session
	// returns 409 instead of spawning a duplicate crew execution.
	s.mu.RLock()
	if st, ok := s.sessions[payload.SessionID]; ok && st.Status == "running" {
		s.mu.RUnlock()
		s.releaseIdem(idemKey)
		c.JSON(http.StatusConflict, gin.H{
			"error":      "session already running",
			"session_id": payload.SessionID,
		})
		return
	}
	s.mu.RUnlock()

	if len(payload.Agents) > 0 {
		crw, apiErr := buildMultiCrew(payload)
		if apiErr != nil {
			c.JSON(apiErr.status, gin.H{"error": apiErr.message})
			return
		}
		s.dispatchCrew(c, crw, payload, owner, idemKey)
		return
	}

	if len(payload.Tasks) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'tasks[]' requires 'agents[]'; tasks cannot run without agents"})
		return
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
	if payload.AgentModel != "" && newLLMClient != nil {
		// Reject empty / unconfigured API key. Constructing an OpenAI client
		// with "" lets unauthenticated kickoffs execute against an empty
		// bearer (noisy DoS surface, log-leak surface).
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "api_provider_unconfigured",
				"message": "agent_model requires a configured OPENAI_API_KEY",
			})
			return
		}
		llmClient = newLLMClient(apiKey)
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

	s.dispatchCrew(c, crw, payload, owner, idemKey)
}

// buildMultiCrew constructs a multi-agent crew from agents[]/tasks[].
// On any rejection it returns a non-nil *apiError and the caller renders it;
// apiError messages are safe to expose (no secrets, no internals).
//
// Contract:
//   - agents[] and the flat agent_role/task_description must never be mixed.
//   - Every task wires to an agent by role; duplicate agent roles are
//     rejected because wiring would be ambiguous.
//   - A role-less task is allowed only with exactly one agent.
//   - Each agent's LLM key resolves per-agent api_key first, then the
//     OPENAI_API_KEY environment. A requested model with no key anywhere is
//     a 503, matching the flat path.
//   - Per-agent api_key values are used only to construct the client and are
//     never logged or persisted (persistSession* stores session_id/owner
//     only).
func buildMultiCrew(payload kickoffRequest) (*crew.Crew, *apiError) {
	if payload.AgentRole != "" || payload.TaskDescription != "" {
		return nil, &apiError{http.StatusBadRequest,
			"cannot combine flat 'agent_role'/'task_description' with 'agents[]'/'tasks[]'"}
	}
	if len(payload.Agents) > maxKickoffAgents {
		return nil, &apiError{http.StatusBadRequest,
			fmt.Sprintf("too many agents: got %d, max %d", len(payload.Agents), maxKickoffAgents)}
	}
	if len(payload.Tasks) > maxKickoffTasks {
		return nil, &apiError{http.StatusBadRequest,
			fmt.Sprintf("too many tasks: got %d, max %d", len(payload.Tasks), maxKickoffTasks)}
	}
	if len(payload.Tasks) == 0 {
		return nil, &apiError{http.StatusBadRequest,
			"'tasks[]' is required with 'agents[]'"}
	}

	envKey := os.Getenv("OPENAI_API_KEY")
	byRole := make(map[string]*agents.Agent, len(payload.Agents))
	coreAgents := make([]core.Agent, 0, len(payload.Agents))
	for i := range payload.Agents {
		a := &payload.Agents[i]
		if _, dup := byRole[a.Role]; dup {
			return nil, &apiError{http.StatusBadRequest,
				fmt.Sprintf("duplicate agent role %q: roles must be unique so tasks wire unambiguously", a.Role)}
		}
		var client llm.Client
		if a.Model != "" && newLLMClient != nil {
			key := a.APIKey
			if key == "" {
				key = envKey
			}
			if key == "" {
				return nil, &apiError{http.StatusServiceUnavailable,
					"api_provider_unconfigured: agent '" + a.Role + "' requests a model but no api_key was provided and OPENAI_API_KEY is unset"}
			}
			client = newLLMClient(key)
		}
		iters := a.MaxIterations
		if iters == 0 {
			iters = payload.MaxIterations
		}
		agent := agents.NewAgentLegacy(a.Role, a.Goal, a.Backstory, client,
			[]agents.AgentOption{agents.WithMaxIterations(iters)}...,
		)
		if agent == nil {
			return nil, &apiError{http.StatusInternalServerError,
				"failed to construct agent '" + a.Role + "'"}
		}
		byRole[a.Role] = agent
		coreAgents = append(coreAgents, agent)
	}

	crewTasks := make([]*tasks.Task, 0, len(payload.Tasks))
	for i := range payload.Tasks {
		t := &payload.Tasks[i]
		role := t.AgentRole
		if role == "" {
			if len(byRole) != 1 {
				return nil, &apiError{http.StatusBadRequest,
					fmt.Sprintf("task %d has no 'agent_role' and %d agents are defined; role is required", i, len(byRole))}
			}
			for only := range byRole {
				role = only
			}
		}
		agent, ok := byRole[role]
		if !ok {
			return nil, &apiError{http.StatusBadRequest,
				fmt.Sprintf("task %d references unknown agent role %q", i, role)}
		}
		task := tasks.NewTask(t.Description, agent)
		task.ExpectedOutput = t.ExpectedOutput
		crewTasks = append(crewTasks, task)
	}

	opts := []crew.CrewOption{}
	if payload.CrewProcess != "" {
		// CrewProcess was already validated against the supported names.
		opts = append(opts, crew.WithProcess(crew.ProcessType(payload.CrewProcess)))
	}
	return crew.NewCrew(coreAgents, crewTasks, opts...), nil
}

// dispatchCrew persists the session, finalizes the idempotency reservation,
// and launches the crew in the background, returning 202. It is shared by
// the flat and multi-agent paths so persistence, idempotency, and semaphore
// semantics cannot drift between them.
func (s *Server) dispatchCrew(c *gin.Context, crw *crew.Crew, payload kickoffRequest, owner, idemKey string) {
	// Persist the session as "running" via the checkpoint backend.
	if err := s.persistSessionStart(payload.SessionID, owner); err != nil {
		// Release the idempotency reservation so a retry (with the same
		// key, different session_id) isn't blocked by a zombie entry.
		s.releaseIdem(idemKey)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to persist session: %v", err),
		})
		return
	}

	if idemKey != "" {
		// Reservation was made before persistSessionStart; release on failure paths.
		s.finalizeIdem(idemKey, payload.SessionID)
	}

	// Dispatch execution asynchronously so the HTTP response returns immediately.
	// Bounded by kickoffSem and propagates the request context (detached from
	// cancellation so the crew survives client disconnect, but keeps values).
	sem, acquired := acquireSem()
	if !acquired {
		// Semaphore saturated: release the idempotency reservation so a
		// retry with the same key (after the queue drains) isn't blocked.
		s.releaseIdem(idemKey)
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "server busy: too many concurrent crew executions"})
		return
	}
	bgCtx := context.WithoutCancel(c.Request.Context())
	// Track the goroutine on the server's WaitGroup so Shutdown waits for
	// in-flight kickoffs (prevents leaked goroutines racing the test cleanup
	// chain or causing post-shutdown writes).
	s.kickoffWG.Add(1)
	go func() {
		defer releaseSem(sem)
		defer s.kickoffWG.Done()
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
// It writes a checkpoint via the SQLite/Redis backend when available,
// otherwise falls back to the in-memory session tracker. The checkpoint
// write is mandatory when a store is configured: a failure here is
// returned to the caller so the kickoff rejects rather than running
// against an unrecoverable session.
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
			// Surface the error: a session whose checkpoint didn't durably
			// persist is not recoverable (no record of agent/task state).
			// Rolling back the in-memory insert keeps the maps consistent.
			delete(s.sessions, sessionID)
			return fmt.Errorf("checkpoint save failed: %w", err)
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
