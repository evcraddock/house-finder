#!/bin/bash
set -e

echo "Running pre-PR checks..."

echo "→ Formatting..."
go fmt ./...

echo "→ Linting..."
golangci-lint run

echo "→ Running tests..."
go test ./...

echo "✓ All checks passed!"
