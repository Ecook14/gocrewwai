package sandbox

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	_ Provider = (*DockerProvider)(nil)
	_ Provider = (*WasmProvider)(nil)
)

func TestMonitorStartEnd(t *testing.T) {
	m := &Monitor{}
	m.RecordStart()
	m.RecordStart()
	m.RecordEnd("ok")
	if m.ActiveSessions != 1 {
		t.Fatalf("ActiveSessions = %d, want 1", m.ActiveSessions)
	}
	if m.TotalExecutions != 2 {
		t.Fatalf("TotalExecutions = %d, want 2", m.TotalExecutions)
	}
	if m.LastStatus != "ok" {
		t.Fatalf("LastStatus = %q, want ok", m.LastStatus)
	}
}

func TestMonitorConcurrent(t *testing.T) {
	m := &Monitor{}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.RecordStart()
			m.RecordEnd("ok")
		}()
	}
	wg.Wait()
	if m.ActiveSessions != 0 {
		t.Fatalf("ActiveSessions = %d, want 0", m.ActiveSessions)
	}
	if m.TotalExecutions != 50 {
		t.Fatalf("TotalExecutions = %d, want 50", m.TotalExecutions)
	}
}

func TestNewDockerProviderDefaults(t *testing.T) {
	p, err := NewDockerProvider("python:3.11-slim")
	if err != nil {
		t.Skipf("docker client unavailable: %v", err)
	}
	defer p.Close()
	if p.Timeout != 300*time.Second {
		t.Fatalf("Timeout = %v, want 300s", p.Timeout)
	}
}

// TestDockerExecute exercises the failure path without a daemon (default)
// or the success path when GOCREW_TEST_DOCKER=1 (daemon + image ready).
func TestDockerExecute(t *testing.T) {
	p, err := NewDockerProvider("python:3.11-slim")
	if err != nil {
		t.Skipf("docker client unavailable: %v", err)
	}
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := p.Execute(ctx, "echo hi", nil)
	if os.Getenv("GOCREW_TEST_DOCKER") == "1" {
		if err != nil {
			t.Fatalf("with daemon: %v", err)
		}
		if !strings.Contains(out, "hi") {
			t.Fatalf("stdout = %q, want hi", out)
		}
		return
	}
	if err == nil {
		t.Skip("daemon unexpectedly present; failure-path needs absent daemon")
	}
}

// TestRequireSandboxNoPanic guards the DCR-01 fix: the missing-sandbox
// path must return an error, never panic, whatever the environment.
func TestRequireSandboxNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RequireSandbox panicked: %v", r)
		}
	}()
	_ = RequireSandbox()
}

func TestWasmInvalidModule(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := NewWasmProvider(ctx)
	if err != nil {
		t.Fatalf("NewWasmProvider: %v", err)
	}
	defer p.Close()
	if _, err := p.Execute(ctx, "not a wasm module", nil); err == nil {
		t.Fatal("expected compilation error, got nil")
	}
}

func TestWasmEmptyModule(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := NewWasmProvider(ctx)
	if err != nil {
		t.Fatalf("NewWasmProvider: %v", err)
	}
	defer p.Close()
	// Minimal valid module: magic + version, no sections.
	out, err := p.Execute(ctx, "\x00asm\x01\x00\x00\x00", map[string]string{"SAFE_MODE": "true"})
	if err != nil {
		t.Fatalf("empty module: %v", err)
	}
	if out != "" {
		t.Fatalf("empty module stdout = %q, want empty", out)
	}
}

func TestShowDockerImage(t *testing.T) {
	if got := ShowDockerImage(DockerConfig{Image: "img:1"}); got != "img:1" {
		t.Fatalf("ShowDockerImage = %q", got)
	}
}
