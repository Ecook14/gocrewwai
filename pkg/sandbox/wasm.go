package sandbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/telemetry"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WASM execution bounds.
const (
	// DefaultWasmTimeout bounds a single Execute call (deadline enforced via context).
	DefaultWasmTimeout = 120 * time.Second
	// MaxWasmBinaryBytes caps the compiled binary size (DoS guard).
	MaxWasmBinaryBytes = 10 << 20 // 10MB
	// MaxWasmOutputBytes caps captured stdout+stderr (DoS guard).
	MaxWasmOutputBytes = 1 << 20 // 1MB
	// MaxWasmMemoryPages caps linear memory per instance: 256 pages x 64KB = 16MB.
	MaxWasmMemoryPages = 256
	// MaxWasmEnvVars / MaxWasmEnvValueBytes bound environment injection.
	MaxWasmEnvVars       = 64
	MaxWasmEnvValueBytes = 4 << 10 // 4KB
)

// WasmProvider uses wazero to execute WebAssembly code.
type WasmProvider struct {
	runtime wazero.Runtime
	config  wazero.ModuleConfig
	cache   wazero.CompilationCache

	Timeout time.Duration

	mu           sync.Mutex
	cachedHash   string
	cachedModule wazero.CompiledModule
}

func NewWasmProvider(ctx context.Context) (*WasmProvider, error) {
	cache := wazero.NewCompilationCache()
	rConfig := wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(MaxWasmMemoryPages).
		WithCompilationCache(cache)
	r := wazero.NewRuntimeWithConfig(ctx, rConfig)

	// Instantiate WASI
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	return &WasmProvider{
		runtime: r,
		cache:   cache,
		config:  wazero.NewModuleConfig().WithStdout(io.Discard).WithStderr(io.Discard),
		Timeout: DefaultWasmTimeout,
	}, nil
}

// resolveWasmBytes disambiguates the code-as-bytes-vs-path confusion: if code
// names an existing .wasm file within size limits, it is read; otherwise code
// is treated as raw binary content. Either way the size is bounded.
func resolveWasmBytes(code string) ([]byte, error) {
	if strings.HasSuffix(code, ".wasm") {
		if info, err := os.Stat(code); err == nil && !info.IsDir() {
			if info.Size() > MaxWasmBinaryBytes {
				return nil, fmt.Errorf("wasm: file exceeds %d byte limit", MaxWasmBinaryBytes)
			}
			bin, err := os.ReadFile(code)
			if err != nil {
				return nil, fmt.Errorf("wasm: failed to read file: %w", err)
			}
			return bin, nil
		}
	}
	bin := []byte(code)
	if len(bin) > MaxWasmBinaryBytes {
		return nil, fmt.Errorf("wasm: binary exceeds %d byte limit", MaxWasmBinaryBytes)
	}
	if len(bin) == 0 {
		return nil, fmt.Errorf("wasm: empty binary")
	}
	return bin, nil
}

// cappedWriter bounds captured output.
type cappedWriter struct {
	buf cappedBuffer
}

type cappedBuffer struct {
	b   bytes.Buffer
	max int
}

func (w *cappedWriter) Write(p []byte) (int, error) {
	room := w.buf.max - w.buf.b.Len()
	if room <= 0 {
		return len(p), nil // discard overflow, report success to keep guest running
	}
	if len(p) > room {
		p = p[:room]
	}
	return w.buf.b.Write(p)
}

func (w *cappedWriter) String() string { return w.buf.b.String() }

// Execute runs a pre-compiled WASM binary.
// In a real agentic scenario, the agent might generate C/Go/Rust,
// which is then compiled to WASM and run here.
// 'code' is either raw binary content or a path to a .wasm file.
// The caller context deadline is honored; DefaultWasmTimeout applies when unset.
func (p *WasmProvider) Execute(ctx context.Context, code string, env map[string]string) (string, error) {
	ctx, span := telemetry.StartSpan(ctx, "sandbox.wasm.Execute")
	if span != nil {
		defer span.End()
	}
	GlobalMonitor.RecordStart()
	defer GlobalMonitor.RecordEnd("wasm")

	timeout := p.Timeout
	if timeout <= 0 {
		timeout = DefaultWasmTimeout
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bin, err := resolveWasmBytes(code)
	if err != nil {
		return "", err
	}

	// 1. Prepare bounded buffers for output
	stdout := &cappedWriter{buf: cappedBuffer{max: MaxWasmOutputBytes}}
	stderr := &cappedWriter{buf: cappedBuffer{max: MaxWasmOutputBytes}}
	config := wazero.NewModuleConfig().
		WithStdout(stdout).
		WithStderr(stderr).
		WithArgs("agent-tool")

	// 2. Inject environment variables (allowlist-shaped: bounded count/size, no empty keys)
	count := 0
	for k, v := range env {
		if strings.TrimSpace(k) == "" {
			continue
		}
		if count >= MaxWasmEnvVars {
			return "", fmt.Errorf("wasm: too many env vars (max %d)", MaxWasmEnvVars)
		}
		if len(v) > MaxWasmEnvValueBytes {
			return "", fmt.Errorf("wasm: env value for %q exceeds %d bytes", k, MaxWasmEnvValueBytes)
		}
		config = config.WithEnv(k, v)
		count++
	}

	// 3. Compile (with single-entry cache) and Run
	sum := sha256.Sum256(bin)
	hash := hex.EncodeToString(sum[:])

	p.mu.Lock()
	compiled := p.cachedModule
	if compiled == nil || p.cachedHash != hash {
		if p.cachedModule != nil {
			_ = p.cachedModule.Close(timeoutCtx)
			p.cachedModule = nil
			p.cachedHash = ""
		}
		p.mu.Unlock()
		compiled, err = p.runtime.CompileModule(timeoutCtx, bin)
		if err != nil {
			return "", fmt.Errorf("wasm: compilation failed: %w", err)
		}
		p.mu.Lock()
		// Only cache if nobody else populated meanwhile.
		if p.cachedModule == nil {
			p.cachedModule = compiled
			p.cachedHash = hash
		} else {
			// Another goroutine cached first; use ours once and close it after run.
			defer compiled.Close(context.Background())
		}
	}
	compiledToRun := compiled
	p.mu.Unlock()

	_, err = p.runtime.InstantiateModule(timeoutCtx, compiledToRun, config)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return stdout.String(), fmt.Errorf("wasm: execution timed out after %s: %w", timeout, err)
		}
		return stdout.String(), fmt.Errorf("wasm: execution failed: %w", err)
	}

	return stdout.String(), nil
}

func (p *WasmProvider) Close() error {
	p.mu.Lock()
	if p.cachedModule != nil {
		_ = p.cachedModule.Close(context.Background())
		p.cachedModule = nil
		p.cachedHash = ""
	}
	p.mu.Unlock()
	if p.cache != nil {
		_ = p.cache.Close(context.Background())
	}
	return p.runtime.Close(context.Background())
}
