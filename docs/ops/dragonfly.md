# Dragonfly Operations

Single source of truth for the KV layer backing memory, checkpoints, and LLM cache.

## Decision

- **Local/dev/self-hosted:** Dragonfly single primary + async replica, AOF `everysec` (`deploy/dragonfly/`).
- **Managed online:** Dragonfly Cloud endpoint via `REDIS_ADDR` (TLS + ACL user).
- **No cluster mode** until upstream edge-case bugs close (FLUSHSLOTS, multi-stream reads, eviction/replication races — observed Aug 2026). Vertical scaling covers current QPS.
- **Dual-support:** all engine code uses `pkg/kv` over go-redis; `REDIS_BACKEND=redis|dragonfly|valkey` is logging-only. Valkey is the BSD-licensed fallback.

## Env contract

| Var | Default | Notes |
|---|---|---|
| `REDIS_BACKEND` | `redis` | Rejects anything else (fail closed); logged at server startup |
| `REDIS_ADDR` | `localhost:6379` | Comma-separated for multi-addr; secrets never in repo |
| `REDIS_PASSWORD` | — | Auth only, never logged |
| `REDIS_DB` | `0` | 0–15 |
| `REDIS_POOL_SIZE` | `10` | 1–500 |
| `KV_TEST_ADDR` | `localhost:6379` | Compat-matrix target for `go test ./pkg/kv/ -run TestCompat` |

`cmd/server` validates the contract at boot and fails fast on typos.

## Keyspaces & eviction

| Use | Prefix | TTL | Policy |
|---|---|---|---|
| Memory items | `crew_memory:` | none (explicit Reset) | `noeviction` + app TTLs |
| Checkpoints | `crew_checkpoint:` | 72h default | `volatile-ttl` |
| LLM cache | cache keys | per-entry | `allkeys-lru` |

## Compat matrix

```bash
# Against local Dragonfly (deploy/dragonfly/docker-compose.yaml up):
REDIS_BACKEND=dragonfly KV_TEST_ADDR=localhost:6379 go test ./pkg/kv/ -run TestCompat -v
# Against Redis:
REDIS_BACKEND=redis KV_TEST_ADDR=localhost:6379 go test ./pkg/kv/ -run TestCompat -v
```

Engine commands exercised: SET/GET/DEL/SCAN/TTL-expiry — the full surface the engine uses. CI runs this matrix with service containers.

## Failover

1. Replica lag check (`INFO replication`), promote replica (local) or Cloud failover (managed).
2. Clients retry via `memory.WithRetry`; checkpoint/cache dials fail fast with clear errors.
3. Verify: `go test ./pkg/kv/ -run TestCompat` green against the new primary, then rotate `REDIS_ADDR`.

## License note (BSL)

Dragonfly is Business-Source-Licensed, not OSS. Self-hosting it as our own
backend and using Dragonfly Cloud as a customer is within normal use; **do not
resell Dragonfly itself as a service** without legal review. If BSL posture
changes, switch `REDIS_BACKEND=valkey` — no code changes required.
