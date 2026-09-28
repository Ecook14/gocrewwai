# Flow vs Flows — Role Split

Two orchestration packages exist; they are **not duplicates**:

- `pkg/flow` (singular) — **canonical event-driven sequential flow** (CrewAI Flow-like):
  `Flow{Nodes, Emit/On, Start, Kickoff}` with `RWMutex` state merging, `Router(ctx,state)`,
  `Or_/And_` listener combinators, `HumanFeedback` pause points, `PersistentFlow`
  auto-persist. `JSONFilePersistence` uses ID sanitization, `0700/0600` perms,
  atomic tmp+rename writes. Use for linear/event pipelines.

- `pkg/flows` (plural) — **graph/DAG engine** for branching workflows:
  `NodeStep/NodeRouter/NodeParallel/NodeMap/NodeReduce` with `Engine.Run/Resume`,
  `MaxFlowDepth=256` cycle guard, ctx-cancel at every node, branch/map errors
  surfaced (all-failed returns first error; partial logs warn), checkpoint
  failures warn-logged (never silently dropped), `Flow.State` under `RWMutex`
  (`getState/setState/getNode`), `CheckpointManager` with ID sanitization,
  `0700/0600` perms, atomic writes, 10MB size-bounded load. Use for complex
  conditional/parallel graphs.

Do not mutate `flows.Flow.Nodes/Initial` while `Run/Resume` is in flight.
`OnStepStream` tokens may contain sensitive output — callers must redact.
