# Config / Errors / Compat / Utils (`pkg/config`, `pkg/errors`, `pkg/compat`, `pkg/utils`)

## `pkg/utils/security.go`

- `ValidatePath(path,chroot)`: `Clean` → depth cap 256 → `Abs` → `HasPrefix` + exact-or-descendant check (sibling `/dataX` rejected). Lexical only — does NOT follow symlinks.
- `ValidatePathResolved`: `ValidatePath` + `EvalSymlinks` on longest existing prefix, re-check containment. Use for file opens.
- `ValidateURL(raw)`: allow `http/https` only, reject credentials, `localhost`/`metadata.google.internal`, loopback/link-local/private/unspecified IPs inline or via `LookupIP` DNS. Used by `pkg/files`, `pkg/knowledge IngestURL`, `pkg/protocols WebMCP`.
- `MaxURLBytes=10MB` bound for `LimitReader`.

## `pkg/compat/compat.go`

`UnwrapError` → `errors.Unwrap`; `IsError` → `errors.Is`; `AsError` → `errors.As`; `IsCanceled/Timeout` → `errors.Is(ctx)`; `IsNotFound/Conflict/Unauthorized/RateLimited` → case-insensitive substring (DB/HTTP message tolerant).

## `pkg/errors/errors.go`

10 sentinels; `Wrap(msg: %w)` preserves chain; no codes/HTTP mapping yet — pair with `compat` helpers.

## `pkg/config`

`LoadConfigFile` reads `CREW_CONFIG_PATH`; secrets via env expansion — do not commit `config.json` with keys. Singleton `once`, no hot reload. `GetClient` supports openai/anthropic/google/openrouter.
