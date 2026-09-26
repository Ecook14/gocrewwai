package testutil

import (
	"context"
	"errors"
	"testing"

	"github.com/Ecook14/gocrewwai/pkg/llm"
	"github.com/Ecook14/gocrewwai/pkg/tools"
)

func TestMockClientImplementsInterfaces(t *testing.T) {
	var _ llm.Client = NewSimpleMock("hi")
	var _ llm.Embedder = NewSimpleMock("hi")
	var _ llm.AudioGenerator = NewSimpleMock("hi")
	var _ tools.Tool = &MockTool{NameValue: "m"}
}

func TestSimpleMockGenerate(t *testing.T) {
	m := NewSimpleMock("hello")
	out, err := m.Generate(context.Background(), []llm.Message{{Role: "user", Content: "hi"}}, llm.GenerateOptions{})
	if err != nil || out != "hello" {
		t.Fatalf("got %q, %v", out, err)
	}
	if m.CallCount() != 1 || len(m.CallsForMethod("Generate")) != 1 {
		t.Fatal("call not recorded")
	}
	m.Reset()
	if m.CallCount() != 0 {
		t.Fatal("reset failed")
	}
}

func TestSequenceAndErrorMocks(t *testing.T) {
	s := NewSequenceMock("a", "b")
	for i, want := range []string{"a", "b", "b"} {
		got, err := s.Generate(context.Background(), nil, llm.GenerateOptions{})
		if err != nil || got != want {
			t.Fatalf("call %d: got %q, %v", i, got, err)
		}
	}
	e := NewErrorMock(errors.New("boom"))
	if _, err := e.Generate(context.Background(), nil, llm.GenerateOptions{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestMockEmbeddingDeterministic(t *testing.T) {
	m := NewSimpleMock("x")
	a, _ := m.GenerateEmbedding(context.Background(), "hello")
	b, _ := m.GenerateEmbedding(context.Background(), "hello")
	if len(a) != 8 || len(b) != 8 {
		t.Fatalf("bad dims %d %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("non-deterministic embedding")
		}
	}
}

func TestMockToolExecute(t *testing.T) {
	mt := &MockTool{NameValue: "t", ExecuteFunc: func(ctx context.Context, in map[string]interface{}) (string, error) {
		return "ran", nil
	}}
	out, err := mt.Execute(context.Background(), nil)
	if err != nil || out != "ran" {
		t.Fatalf("got %q, %v", out, err)
	}
	if mt.ArgsSchema() != nil || mt.CacheFunction(nil) != "" {
		t.Fatal("unexpected schema/cache defaults")
	}
}
