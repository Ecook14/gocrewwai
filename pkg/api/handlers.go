package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Ecook14/gocrewwai/pkg/telemetry"
)

// handleKickoff parses the visual crew definition and starts execution.
func (s *Server) handleKickoff(c *gin.Context) {
	var payload struct {
		SessionID string `json:"session_id"`
		// More fields for Agents/Tasks definition will go here
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if payload.SessionID == "" {
		payload.SessionID = fmt.Sprintf("sess_%d", interface{}(nil).(interface {
			time() int64
		}).time()) // Placeholder for timestamp-based ID
	}

	// Forward to Crew engine (To be integrated with actual Crew.Kickoff)
	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Crew execution started",
		"session_id": payload.SessionID,
	})
}

// handleGetSession returns the current state of a session.
func (s *Server) handleGetSession(c *gin.Context) {
	id := c.Param("id")
	// Integration with pkg/persistence will go here
	c.JSON(http.StatusOK, gin.H{
		"session_id": id,
		"status":     "running",
	})
}

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
