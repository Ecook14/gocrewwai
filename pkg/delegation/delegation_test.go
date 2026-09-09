package delegation

import (
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/agents"
	"github.com/Ecook14/gocrewwai/pkg/core"
)

func TestDelegationManager_Basic(t *testing.T) {
	dm := NewDelegateWorkTool(nil)
	if dm == nil {
		t.Fatal("Expected non-nil delegation tool")
	}
}

func TestDelegationManager_AddAgent(t *testing.T) {
	agent := &agents.Agent{Role: "TestAgent"}
	dm := NewDelegateWorkTool([]core.Agent{agent})
	if dm == nil {
		t.Fatal("Expected non-nil delegation tool")
	}
}
