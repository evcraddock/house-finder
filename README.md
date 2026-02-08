# house-finder

A tool to find and track houses for sale

## Prerequisites

- Go 1.21+
- golangci-lint
- A [RapidAPI](https://rapidapi.com/) key for the us-real-estate-listings API

## Setup

```bash
go mod download
cp .env.example .env
# Edit .env and add your RAPIDAPI_KEY
```

## Build

```bash
make build
```

This produces an `hf` binary in the project root. Or `make install` to put it in `$GOPATH/bin`.

## Usage

```bash
# Add a property (hits the API once)
hf add "123 Main St, City, ST 12345"

# List all properties
hf list

# Show property details
hf show 1

# Rate a property (1-4, 4 is best)
hf rate 1 4

# Add a comment
hf comment 1 "Great backyard"

# List comments
hf comments 1

# Start the web UI
hf serve

# JSON output for AI agents
hf list --format json

# Print version
hf version
```

## Development

### Build

```bash
make build
```

### Run Tests and Linting

```bash
make check
```

### Start Dev Environment

```bash
make dev
```

### Before Opening a PR

```bash
make pre-pr
```

### Available Make Commands

```bash
make help
```

## License

MIT
