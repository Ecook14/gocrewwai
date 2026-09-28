# Sandbox (`pkg/sandbox`)

Execution isolation for untrusted code.

## Providers

- `DockerProvider(image)` (`docker.go:22`): `NetworkMode:none`, `ReadonlyRootfs:true`, `CapDrop:[ALL]`, `Tmpfs /tmp rw,noexec,nosuid`, `Memory 512M`, `CPUQuota 50k`, `PidsLimit 100`, `User 1000:1000`, `Timeout 300s` with `context.WithTimeout`. Supply-chain note: pin images by digest; arbitrary `image` strings are caller-controlled.
- `wasm.go`: WASM provider (wazero) with `DefaultWasmTimeout 120s` ctx deadline, in-memory `CompilationCache` + single-entry compiled-module cache, `MaxWasmMemoryPages 256` (16MB linear), 10MB binary cap (raw bytes or `.wasm` path), bounded env (64 vars, 4KB each, no empty keys), 1MB output cap, `WithCloseOnContextDone(true)`.
- `provider.go:8 Monitor{ActiveSessions,TotalExecutions}` — no otel export yet.

## Security defaults

Empty env/sandbox config must deny execution on host. `SafeMode` alone gating only `RequiresReview()` is insufficient — callers must refuse host exec when no Docker/WASM backend is configured.
