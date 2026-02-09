# house-finder

A tool to find and track houses for sale. Add properties by address, rate them, leave comments, and browse via CLI or web UI.

## Prerequisites

- Go 1.21+
- golangci-lint
- A [RapidAPI](https://rapidapi.com/) key for the us-real-estate-listings API

## Setup

```bash
go mod download
cp .env.example .env
# Edit .env and add your RAPIDAPI_KEY, HF_ADMIN_EMAIL, etc.
```

## Build

```bash
make build
```

This produces an `hf` binary in the project root. Or `make install` to put it in `$GOPATH/bin`.

## Getting Started

The CLI requires the web server to be running. The server is the single source of truth — all CLI commands talk to it via REST API.

### 1. Start the server

```bash
hf serve
```

Or for development with live-reload:

```bash
make dev
```

### 2. Log in

```bash
hf login
```

This opens a browser to authenticate and generate an API key. Paste the key when prompted. The key is saved to `~/.config/hf/config.yaml`.

### 3. Use the CLI

```bash
# Add a property (server does API lookup)
hf add "123 Main St, City, ST 12345"

# List all properties
hf list

# Filter by minimum rating
hf list --rating 3

# Show property details + comments
hf show 1

# Rate a property (1-4, 4 is best)
hf rate 1 4

# Add a comment
hf comment 1 "Great backyard"

# List comments
hf comments 1

# Remove a property
hf remove 1

# JSON output
hf list --format json

# Check connection and auth status
hf status

# Remove stored API key
hf logout

# Print version
hf version
```

## Configuration

The CLI stores config at `~/.config/hf/config.yaml`:

```yaml
server_url: http://localhost:8080
api_key: hf_...
```

Environment variable overrides:

- `HF_SERVER_URL` — server URL (default: `http://localhost:8080`)
- `HF_API_KEY` — API key (overrides config file)

## Web UI

The web UI is available at `http://localhost:8080` when the server is running. It provides:

- Property list with ratings
- Property detail view with comments
- Inline rating and commenting via HTMX
- Dark mode toggle
- Settings page for passkey and API key management

## Authentication

- **Magic link email** — passwordless login via email link
- **Passkeys** — WebAuthn/FIDO2 for fast login after initial setup
- **API keys** — Bearer token auth for CLI and API access

Set `HF_ADMIN_EMAIL` in `.env` to enable authentication. In dev mode (`HF_DEV_MODE=true`), magic links are logged to the console instead of emailed.

## REST API

All endpoints require `Authorization: Bearer <api_key>` header.

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/properties | List all (optional ?min_rating=N) |
| POST | /api/properties | Add by address (JSON: `{"address": "..."}`) |
| GET | /api/properties/{id} | Show property + comments |
| DELETE | /api/properties/{id} | Remove property |
| POST | /api/properties/{id}/rate | Set rating (JSON: `{"rating": 3}`) |
| GET | /api/properties/{id}/comments | List comments |
| POST | /api/properties/{id}/comments | Add comment (JSON: `{"text": "..."}`) |

## Development

```bash
make build      # Build binary
make check      # Run linter + tests
make dev        # Start dev environment with live-reload
make pre-pr     # Run before opening a PR
make help       # Show all commands
```

## License

MIT
