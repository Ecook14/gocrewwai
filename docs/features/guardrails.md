# Guardrails (`pkg/guardrails`)

Input/output safety: `guardrails.go`, `human_review.go`, `sanitizer.go:29` (6 injection regex + 50KB `MaxInputLength` + HTML strip), `json.go`.

PII redaction covers email/SSN (`guardrails.go:140-191`); API-key/secret redaction is regex-based via `pkg/telemetry/logging.go:redactSensitiveData` — bypassable, LLM-review optional. `Validate(output string)` has no `ctx` except `LLMReviewGuardrail.Validate(ctx)` (`guardrails.go:298`); no timeout/retry/audit-log persistence/otel yet.

Use with `pkg/tasks` HITL (`HUMAN_IN_THE_LOOP.md`) and `pkg/tools RequiresReview()` gating.
