package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// WasmProvider uses wazero to execute WebAssembly code.
type WasmProvider struct {
	runtime wazero.Runtime
	config  wazero.ModuleConfig
}

func NewWasmProvider(ctx context.Context) (*WasmProvider, error) {
	r := wazero.NewRuntime(ctx)
	
	// Instantiate WASI
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	return &WasmProvider{
		runtime: r,
		config:  wazero.NewModuleConfig().WithStdout(io.Discard).WithStderr(io.Discard),
	}, nil
}

// Execute runs a pre-compiled WASM binary.
// In a real agentic scenario, the agent might generate C/Go/Rust, 
// which is then compiled to WASM and run here. 
// For this implementation, we assume 'code' is the path to a .wasm file or the binary data itself.
func (p *WasmProvider) Execute(ctx context.Context, code string, env map[string]string) (string, error) {
	// 1. Prepare buffers for output
	var stdout, stderr bytes.Buffer
	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stderr).
		WithArgs("agent-tool")

	// 2. Inject environment variables
	for k, v := range env {
		config = config.WithEnv(k, v)
	}

	// 3. Compile and Run
	// Note: In high-performance scenarios, we would cache the compiled module.
	compiled, err := p.runtime.CompileModule(ctx, []byte(code))
	if err != nil {
		return "", fmt.Errorf("wasm: compilation failed: %w", err)
	}

	_, err = p.runtime.InstantiateModule(ctx, compiled, config)
	if err != nil {
		return fmt.Sprintf("STDOUT: %s\nSTDERR: %s", stdout.String(), stderr.String()), fmt.Errorf("wasm: execution failed: %w", err)
	}

	return stdout.String(), nil
}

func (p *WasmProvider) Close() error {
	return p.runtime.Close(context.Background())
}
