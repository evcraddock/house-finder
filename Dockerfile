# Build stage
FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=1 go build \
    -ldflags "-X github.com/evcraddock/house-finder/internal/cli.Version=${VERSION}" \
    -o /hf ./cmd/hf

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /hf /usr/local/bin/hf

# Default data directory
RUN mkdir -p /data
ENV HF_DB_PATH=/data/houses.db

EXPOSE 8080

ENTRYPOINT ["hf"]
CMD ["serve", "--port", "8080", "--db", "/data/houses.db"]
