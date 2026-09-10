package aamarva

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("Expected non-nil client")
	}
	if c.BaseURL != "https://aamarva.com/api" {
		t.Errorf("Expected BaseURL 'https://aamarva.com/api', got '%s'", c.BaseURL)
	}
}

func TestClient_GetADK(t *testing.T) {
	c := NewClient()
	result, err := c.GetADK(context.Background())
	if err != nil {
		t.Skipf("ADK fetch skipped: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected ADK fetch to succeed")
	}
}

func TestClient_GetStats(t *testing.T) {
	c := NewClient()
	result, err := c.GetStats(context.Background())
	if err != nil {
		t.Skipf("Stats fetch skipped: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected stats fetch to succeed")
	}
}

func TestClient_SearchPosts(t *testing.T) {
	c := NewClient()
	result, err := c.SearchPosts(context.Background(), "AI", 5)
	if err != nil {
		t.Skipf("Post search skipped: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected post search to succeed")
	}
}

func TestClient_SearchAgents(t *testing.T) {
	c := NewClient()
	result, err := c.SearchAgents(context.Background(), "agent", 5)
	if err != nil {
		t.Skipf("Agent search skipped: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected agent search to succeed")
	}
}

func TestClient_Authenticated(t *testing.T) {
	c := NewClient()
	if c.Authenticated() {
		t.Error("Fresh client should not be authenticated")
	}
}
