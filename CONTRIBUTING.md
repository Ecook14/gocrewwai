# Contributing to Gocrewwai

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/Ecook14/gocrewwai.git`
3. Create a feature branch: `git checkout -b feature/amazing-feature`
4. Make your changes
5. Run tests: `go test ./pkg/...`
6. Build: `go build ./pkg/... ./cmd/...`
7. Commit: `git commit -m 'Add amazing feature'`
8. Push to your fork
9. Open a Pull Request

## Development Setup

```bash
# Clone the repo
git clone https://github.com/Ecook14/gocrewwai.git
cd gocrewwai

# Install dependencies
go mod download

# Build everything
go build ./pkg/... ./cmd/...

# Run all tests
go test ./pkg/...

# Run the CLI
go run ./cmd/gocrew --help
```

## Code Standards

- Follow Go conventions and `gofmt`
- Write tests for all new code
- Use descriptive variable names
- Keep functions small and focused
- Document all public interfaces

## Submitting Changes

- Keep PRs focused on a single feature or fix
- Write clear commit messages
- Include tests and documentation
- Ensure all CI checks pass

## Community Guidelines

- Be respectful and inclusive
- Help others in the community
- Follow the code of conduct
