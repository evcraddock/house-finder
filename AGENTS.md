# AI Agent Guidelines for house-finder

## Before Starting ANY Task

**ALWAYS use the `task-start-preflight` skill** when you hear:
- "start task", "work on task", "get started", "pick up task"
- "let's do task", "begin task", "tackle task"
- Or any variation of starting work

The preflight ensures you understand the task, check dependencies, and follow project guidelines.

## Required Reading

Before working, read and follow:
- [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) - workflow and PR process
- [docs/CODE_STANDARDS.md](docs/CODE_STANDARDS.md) - code style and patterns

You MUST follow these guidelines throughout your work.

## Project Overview

A tool to find and track houses for sale

## Tech Stack

- Language: Go
- Framework: None

## Development

**ALWAYS start the dev server using `make dev`** - this runs all services (app, database, etc.) via the Makefile.

Key Makefile targets:
- `make dev` - Start development server (REQUIRED)
- `make test` - Run tests
- `make lint` - Run linter
- `make fmt` - Format code

Read the Makefile to understand available commands before starting work.

## Dependencies

When adding packages:
- Use latest **STABLE** versions only
- Reject canary/beta/alpha/rc versions unless user explicitly approves
- Check for stable releases before adding dependencies

Non-stable versions (canary, beta, alpha, rc) can have bugs or incomplete features. Always ask before using them.

## Task Lifecycle

- **Starting**: ALWAYS run `task-start-preflight` skill first
- **Closing**: Run `task-close-preflight` skill

## PR Workflow

1. Create feature branch: `feat/<task-id>-<description>`
2. Run `./scripts/pre-pr.sh` before opening PR
3. After PR is created, use the `request-review` skill to spawn a separate agent to review the PR

## Conventions

- Follow standard Go project layout
- Use gofmt for formatting
- Handle errors explicitly, don't ignore them
- Use table-driven tests
- Document exported functions
