package gocrew

import (
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/tools"
)

// Facade smoke test: guards against the fabricated-API class of errors by
// asserting the primary SDK surface exists and wires to real implementations.
func TestFacadeConstructors(t *testing.T) {
	if NewFileCache("./x") == nil {
		t.Fatal("NewFileCache nil")
	}
	if NewCodeInterpreterTool() == nil {
		t.Fatal("NewCodeInterpreterTool nil")
	}
	if NewCalculatorTool() == nil {
		t.Fatal("NewCalculatorTool nil")
	}
}

func TestToToolDefsRoundTrip(t *testing.T) {
	defs := ToToolDefs([]tools.Tool{NewCalculatorTool()})
	if len(defs) != 1 || defs[0].Name == "" {
		t.Fatalf("defs = %+v", defs)
	}
	if len(ToToolDefs(nil)) != 0 {
		t.Fatal("nil slice must yield empty defs")
	}
}
