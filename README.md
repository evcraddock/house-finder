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

This produces a `house-finder` binary in the project root.

## Usage

```bash
# Add a property (hits the API once)
./house-finder add "123 Main St, City, ST 12345"

# List all properties
./house-finder list

# Show property details
./house-finder show 1

# Rate a property (1-4, 4 is best)
./house-finder rate 1 4

# Add a comment
./house-finder comment 1 "Great backyard"

# List comments
./house-finder comments 1

# Start the web UI
./house-finder serve

# JSON output for AI agents
./house-finder list --format json
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
