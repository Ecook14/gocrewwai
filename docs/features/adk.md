# ADK + Delegation Bridge (`pkg/adk`, `pkg/delegation`, `pkg/protocols/a2a_adapter.go`)

## ADK (`pkg/adk/adapter.go`)

- `NewADKAgent(agent).Run(ctx,input)` → `Event`; `RunWithTools` snapshots `GetToolCount`, equips delta, records `toolCache`, bounds input at 20000 chars.
- `GetTools()` returns cached tools (nil until `RunWithTools`/`ToolsToADK` populates).
- `adkToolAdapter.Execute` delegates to inner `Run(ctx,map)` — returns error if tool has no `Run` (no fake output).
- `StoreToADKState(store)` → `memoryStateAdapter`: `Get` checks cache then `Remember/Recall` on `gocrew.MemoryStore` with 5s timeout; `Set` writes cache + best-effort `Remember`. Nil store = cache-only.
- `ContentToGocInput`/`GocResultToContent` capped at 20000 chars. `sessionAwareAgent` tracks `tools/execCount` with mutex; `Execute` prefers session last-user-message, validates non-empty.

## Delegation (`pkg/delegation`)

- `DelegateWorkTool` / `AskQuestion` for in-crew delegation.
- `a2a_bridge.go: DelegateViaA2A(ctx,role,task,ctx)` — registry `RegisterA2AAdapter`, `MaxDelegationDepth=5` cycle guard via context depth, 20000-char task cap, ctx-cancel respected. Enforces `collaboration.md` max-depth contract.
- `pkg/protocols/a2a_adapter.go`: `GetMaxRPM` from `Card.Metadata["max_rpm"]`, `GetToolCount=len(Capabilities)`, `GetUsageMetrics={delegations:1}`.

## Stale-doc note

`docs/features/collaboration.md:65` previously implied unbounded delegation; depth guard is now authoritative.
