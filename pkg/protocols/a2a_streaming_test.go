package protocols

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type mockStreamingAgent struct {
	tokens []string
}

func (m *mockStreamingAgent) Execute(ctx context.Context, input string, options map[string]interface{}) (interface{}, error) {
	if cb, ok := options["stream_callback"].(func(string)); ok {
		for _, t := range m.tokens {
			cb(t)
			time.Sleep(10 * time.Millisecond) // Simulate network/thinking latency
		}
	}
	return "Final Result", nil
}

func (m *mockStreamingAgent) GetRole() string { return "Mock" }

func TestA2AWebSocketStreaming(t *testing.T) {
	// 1. Setup Server
	router := NewA2ARouter()
	agent := &mockStreamingAgent{tokens: []string{"Hello", " ", "World", "!"}}
	
	router.Handle("delegate_task", func(ctx context.Context, msg A2AMessage) (*A2AMessage, error) {
		req, _ := UnmarshalTaskRequest(msg.Payload)
		options := make(map[string]interface{})
		
		if conn, ok := ctx.Value("a2a_ws_conn").(*websocket.Conn); ok {
			options["stream_callback"] = func(token string) {
				_ = conn.WriteJSON(A2AMessage{
					Type:    A2AStream,
					Payload: map[string]interface{}{"token": token},
				})
			}
		}

		res, err := agent.Execute(ctx, req.Description, options)
		return &A2AMessage{
			Type:    A2AResponse,
			Payload: map[string]interface{}{"result": res},
		}, err
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			// Extract handleWebSocket logic or call it
			conn, _ := upgrader.Upgrade(w, r, nil)
			defer conn.Close()
			var msg A2AMessage
			conn.ReadJSON(&msg)
			ctx := context.WithValue(r.Context(), "a2a_ws_conn", conn)
			resp, _ := router.Route(ctx, msg)
			conn.WriteJSON(resp)
		}
	}))
	defer server.Close()

	// 2. Setup Client
	client := NewA2AClient("")
	msg := A2AMessage{
		Action:  "delegate_task",
		Payload: MarshalTaskRequest(A2ATaskRequest{Description: "Test Task"}),
	}

	// 3. Start Streaming
	tokenCh, err := client.Stream(context.Background(), server.URL, msg)
	if err != nil {
		t.Fatalf("Failed to start stream: %v", err)
	}

	var received []string
	for token := range tokenCh {
		received = append(received, token)
	}

	expected := "Hello World!"
	actual := strings.Join(received, "")
	if actual != expected {
		t.Errorf("Expected tokens %q, got %q", expected, actual)
	}
}
