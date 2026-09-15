#!/bin/bash
set -e
export PATH="/content/openclaw-ws/.hermes/go/bin:$PATH"
export GOTOOLCHAIN=local
cd /content/openclaw-ws/projects/gocrewwai

# Strip any illegal build constraints from go.mod
sed -i '/\/\/go:build/d; /\/\/ \+build/d' go.mod

echo "=== go mod tidy ==="
go mod tidy 2>&1
grep '^go ' go.mod

echo "=== vet ==="
go vet ./pkg/adk/... ./pkg/compat/... ./pkg/server/... ./pkg/protocols/... 2>&1

echo "=== build ==="
go build -o /dev/null ./cmd/gocrew ./cmd/server 2>&1

echo "=== test ==="
go test ./pkg/adk/... ./pkg/compat/... ./pkg/server/... ./pkg/protocols/... -count=1 -timeout 5m 2>&1 | tail -10

echo "=== ALL DONE ==="
