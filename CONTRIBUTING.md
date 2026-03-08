# Contributing to Crew-GO

Thank you for your interest in contributing! This guide will help you get started.

## Development Setup

1. **Prerequisites**: Go 1.22+ installed
2. **Clone and build**:
   ```bash
   git clone https://github.com/Ecook14/gocrew.git
   cd Crew-GO
   go build ./...
   ```
3. **Run tests**:
   ```bash
   go test ./... -v
   ```

## Project Structure

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full layout and dependency graph.

## Coding Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- All exported types and functions must have doc comments
- Use `context.Context` as the first parameter for any function involving I/O or concurrency
- Return errors using types from `pkg/errors` (wrap with sentinels, use typed errors)
- Interfaces belong in the package that *uses* them, not the package that implements them

## Adding a New Tool

1. Create a new file in `pkg/tools/` (e.g., `my_tool.go`)
2. Implement the `Tool` interface:
   ```go
   type MyTool struct{}
   func (t *MyTool) Name() string { return "MyTool" }
   func (t *MyTool) Description() string { return "Does something useful." }
   func (t *MyTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) { ... }
   ```
3. Add tests in `pkg/tools/tools_test.go`

## Adding a New Guardrail

1. Create a struct implementing `guardrails.Guardrail` in `pkg/guardrails/`
2. Implement `Name() string` and `Validate(output string) error`
3. Add tests in `pkg/guardrails/guardrails_test.go`

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Write tests for new functionality
4. Ensure `go test ./...` passes
5. Ensure `go vet ./...` reports no issues
6. Submit a pull request with a clear description

## Testing

- Unit tests live alongside the code (`*_test.go`)
- Tests should not require API keys or network access (use mocks)
- Use `t.TempDir()` for file-based tests
- Aim for table-driven tests where applicable
