package tools

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestAskHumanInjectedIO(t *testing.T) {
	var out bytes.Buffer
	tool := NewAskHumanTool(true,
		WithHumanInput(strings.NewReader("yes, proceed\n")),
		WithHumanOutput(&out),
	)
	resp, err := tool.Execute(context.Background(), map[string]interface{}{"question": "Ship it?"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp != "yes, proceed" {
		t.Fatalf("response = %q", resp)
	}
	if !strings.Contains(out.String(), "Ship it?") {
		t.Fatalf("prompt missing from output: %q", out.String())
	}
}

func TestAskHumanDisabled(t *testing.T) {
	tool := NewAskHumanTool(false)
	if _, err := tool.Execute(context.Background(), map[string]interface{}{"question": "x"}); err == nil {
		t.Fatal("expected disabled error")
	}
}
