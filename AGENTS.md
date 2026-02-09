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

## NEVER Push Directly to Main

**No exceptions. No "quick fixes". No "obvious bugs".**

Always:
1. Create a feature branch (`feat/<task-id>-<description>`)
2. Create a PR
3. Wait for explicit human approval ("merge", "approved", "LGTM")
4. Only then merge

This applies even when:
- You're confident the fix is correct
- It's a one-line change
- You're in the middle of debugging
- The user seems to want it done quickly

**The process exists because the human needs to review and approve changes before they ship.** Pushing directly to main takes that decision away from them. It's irreversible.

## Project Overview

A tool to find and track houses for sale. Add properties by address (fetches data from RapidAPI), rate them 1-4, leave comments, and browse via CLI or web UI.

## Tech Stack

- **Language**: Go
- **Database**: SQLite (mattn/go-sqlite3 via CGO)
- **CLI**: Cobra
- **Web UI**: HTMX + Go templates
- **Auth**: Magic link email + passkeys (WebAuthn) + API keys

## Architecture

The web server is the single source of truth. The CLI is a thin HTTP client that talks to the server's REST API.

- `cmd/hf/` — CLI entry point
- `internal/cli/` — CLI commands (thin HTTP client)
- `internal/client/` — HTTP client for the REST API
- `internal/web/` — Web server, handlers, templates
- `internal/auth/` — Auth (tokens, sessions, passkeys, API keys, users)
- `internal/property/` — Property model and repository
- `internal/comment/` — Comment model and repository
- `internal/mls/` — RapidAPI MLS client
- `internal/db/` — SQLite setup and migrations

## Development Approach: Vertical Slices

Build features as **vertical slices**, not horizontal layers. Each task should deliver visible, working progress.

**❌ Don't do this (layers):**
1. Build all database schema
2. Build all backend routes
3. Build all API endpoints
4. Then finally UI you can see

**✅ Do this (slices):**
1. API endpoint + handler + test → working
2. CLI command wired up → usable
3. Web UI displays the data → visible

Each task should result in something a human can see or interact with.

**Keep PRs small.** Humans review every PR. Break work into small units:
- Aim for PRs under 300 lines changed
- One logical change per PR
- If a feature is big, split it into multiple PRs that build on each other
- It's better to merge 3 small PRs than 1 large one

## Development

**ALWAYS start the dev server using `make dev`** — this runs all services via overmind with live-reload (air).

Key Makefile targets:
- `make dev` — Start development server (REQUIRED)
- `make dev-stop` — Stop dev environment
- `make dev-status` — Check if running
- `make dev-tail` — Show last 100 lines of logs (non-blocking)
- `make dev-connect s=app` — Attach to app terminal
- `make check` — Run linter + tests
- `make build` — Build the `hf` binary
- `make install` — Install to $GOPATH/bin
- `make pre-pr` — Run pre-PR checks
- `make release` — Build release binary
- `make docker` — Build Docker image

Read the Makefile to understand available commands before starting work.

## Visual Verification

When testing UI changes, **open a browser** (use Playwright) instead of using curl. Take screenshots to verify:
- Page layouts and styling
- Form interactions
- Any user-facing changes

## Dependencies

When adding packages:
- Use latest **STABLE** versions only
- Reject canary/beta/alpha/rc versions unless user explicitly approves
- Check for stable releases before adding dependencies

**NEVER remove or replace mattn/go-sqlite3.** It requires CGO but is battle-tested and standard. Do not suggest modernc.org/sqlite or other alternatives.

## Task Lifecycle

- **Starting**: ALWAYS run `task-start-preflight` skill first
- **Closing**: Run `task-close-preflight` skill

## PR Workflow

1. Create feature branch: `feat/<task-id>-<description>`
2. Run `make check` before every commit
3. Run `./scripts/pre-pr.sh` before opening PR
4. Create PR with `gh pr create`
5. **Wait for CI to pass** before requesting review
6. Use the `request-review` skill to spawn a separate agent to review the PR
7. **Wait for human approval before merging**

### Wait for CI Before Requesting Review

After creating a PR, CI runs automatically. **Do not request a review until CI passes.**

```bash
gh pr checks <number>
```

If CI fails:
- Check the failure: `gh run view <run-id> --log-failed`
- Fix the issue, commit, and push
- Wait for CI to pass before requesting review

### NEVER Merge Without Human Approval

**Agent reviews do not replace human approval.** The agent review catches issues early — it is NOT permission to merge.

After an agent review completes:
1. Show the review results to the user
2. **Stop and wait for explicit human approval**
3. Only merge when the user says "merge", "approved", "LGTM", or similar

**DO NOT:**
- Auto-merge after agent review
- Assume approval because the review passed
- Merge and then tell the user about it

**The human decides when to merge. Always.**

## Testing Requirements

Write tests for code with logic. Don't write tests just to have tests.

**Do test:**
- Business logic and data transformations
- API endpoint handlers
- Functions with conditionals or branching
- Edge cases and error handling
- Auth flows

**Don't test:**
- Schema/type definitions
- Simple pass-through functions with no logic
- Third-party library integration in isolation

## Database Changes

Migrations are in `internal/db/migrations.go` as an ordered list of SQL statements. Each new table or schema change gets appended to the list. Migrations run automatically on `db.Open()`.

- Append new migrations to the end of the list — never modify existing ones
- Use `CREATE TABLE IF NOT EXISTS` and `ALTER TABLE` patterns
- Test that `db.Open()` works with the new migration

## Conventions

- Follow standard Go project layout
- Use gofmt for formatting
- Handle errors explicitly, don't ignore them (errcheck has `check-blank: true`)
- Use table-driven tests
- Document exported functions
- Commit format: `<type>: <short description>\n\nTask: #<task-id>`
- Binary name is `hf` (not house-finder)
