.PHONY: build install dev dev-stop dev-status dev-logs dev-tail check pre-pr release help

BINARY := hf
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/evcraddock/house-finder/internal/cli.Version=$(VERSION)"
SOCKET := ./.overmind.sock

build: ## Build the binary
	go build $(LDFLAGS) -o $(BINARY) ./cmd/hf

install: ## Install to $GOPATH/bin
	go install $(LDFLAGS) ./cmd/hf

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

dev: ## Start the dev environment (daemonized)
	@if [ -S $(SOCKET) ] && overmind ps -s $(SOCKET) > /dev/null 2>&1; then \
		echo "Dev environment already running"; \
		overmind ps -s $(SOCKET); \
	else \
		$(MAKE) -s _kill-orphan-hf; \
		rm -f $(SOCKET); \
		overmind start -f Procfile.dev -s $(SOCKET) -D; \
		sleep 2; \
		overmind ps -s $(SOCKET); \
	fi

dev-stop: ## Stop the dev environment
	@if [ -S $(SOCKET) ]; then overmind quit -s $(SOCKET) || true; fi
	@rm -f $(SOCKET)
	@tmux list-sessions 2>/dev/null | grep overmind | cut -d: -f1 | xargs -r -n1 tmux kill-session -t 2>/dev/null || true
	@$(MAKE) -s _kill-orphan-hf

_kill-orphan-hf: ## Kill orphaned hf processes on port 8080
	@PID=$$(fuser 8080/tcp 2>/dev/null | awk '{print $$1}'); \
	if [ -n "$$PID" ]; then \
		CMD=$$(ps -p $$PID -o comm= 2>/dev/null); \
		if [ "$$CMD" = "hf" ]; then \
			echo "Killing orphaned hf process (PID $$PID) on port 8080"; \
			kill $$PID 2>/dev/null || true; \
			sleep 1; \
		fi; \
	fi

dev-status: ## Check if dev environment is running
	@if [ -S $(SOCKET) ] && overmind ps -s $(SOCKET) > /dev/null 2>&1; then \
		echo "running"; \
	else \
		echo "stopped"; \
	fi

dev-logs: ## Stream all logs (Ctrl+C to stop)
	overmind echo -s $(SOCKET)

dev-tail: ## Show last 100 lines of logs (non-blocking)
	@if [ -S $(SOCKET) ]; then \
		TMUX_SOCK=$$(ls -t /tmp/tmux-$$(id -u)/overmind-house-finder-* 2>/dev/null | head -1); \
		if [ -n "$$TMUX_SOCK" ]; then \
			for pane in $$(tmux -S "$$TMUX_SOCK" list-panes -a -F '#{pane_id}' 2>/dev/null); do \
				tmux -S "$$TMUX_SOCK" capture-pane -p -t "$$pane" -S -100 2>/dev/null; \
			done; \
		else \
			echo "Tmux socket not found"; \
		fi; \
	else \
		echo "Dev environment not running"; \
	fi

dev-connect: ## Connect to a dev process (usage: make dev-connect s=app)
	@if [ -S $(SOCKET) ]; then \
		overmind connect -s $(SOCKET) $(or $(s),app); \
	else \
		echo "Dev environment not running"; \
	fi

release: ## Build release binary for current platform
	CGO_ENABLED=1 go build $(LDFLAGS) -o dist/$(BINARY)-$$(go env GOOS)-$$(go env GOARCH) ./cmd/hf
	@echo "Built: dist/$(BINARY)-$$(go env GOOS)-$$(go env GOARCH)"

docker: ## Build Docker image
	docker build --build-arg VERSION=$(VERSION) -t house-finder:$(VERSION) .

check: ## Run linting and tests
	golangci-lint run && go test ./...

pre-pr: ## Run pre-PR checks
	./scripts/pre-pr.sh

# Connect to specific service terminal (replace 'app' with service name from Procfile.dev)
# connect-app: ## Connect to app terminal
# 	overmind connect -s $(SOCKET) app
