package agents

import (
	"bytes"
	"os"
	"testing"
)

func TestAgentOutDefault(t *testing.T) {
	a := &Agent{}
	if a.agentOut() != os.Stdout {
		t.Fatal("default must be os.Stdout")
	}
}

func TestAgentOutInjected(t *testing.T) {
	var buf bytes.Buffer
	a := &Agent{Stdout: &buf}
	if a.agentOut() != &buf {
		t.Fatal("injected writer not returned")
	}
}
