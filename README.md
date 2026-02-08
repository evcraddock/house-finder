# house-finder

A tool to find and track houses for sale

## Prerequisites

- Go 1.21+

## Installation

```bash
go mod download
```

## How to Work on This Project

### Start the Dev Environment

```bash
make dev
```

This starts all services defined in `Procfile.dev`. The command returns immediately (daemonized).

### View Logs

```bash
# Stream all logs (Ctrl+C to stop)
make dev-logs

# Quick peek at recent logs
make dev-tail
```

### Check Status

```bash
make dev-status
```

### Stop the Dev Environment

```bash
make dev-stop
```

### Run Tests and Linting

```bash
make check
```

### Before Opening a PR

```bash
make pre-pr
```

### Available Make Commands

```bash
make help
```

## Dev Environment Setup

If `make dev` fails, the dev environment needs configuration. See the "Set up dev environment" task (#1635) for details on configuring `Procfile.dev` and any required services.

## License

MIT
