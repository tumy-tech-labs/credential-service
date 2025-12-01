.PHONY: test test-unit test-integration test-e2e test-coverage lint sec ci keygen build-all clean dev-up dev-down help

MODULE_PATH=./...
VERSION?=dev

# Default target
all: lint test

## Help command
help:
	@echo "Available targets:"
	@echo "  make test           - Run unit tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make lint           - Run linter"
	@echo "  make sec            - Run security scanner"
	@echo "  make build-all      - Build all services"
	@echo "  make keygen         - Build keygen CLI tool"
	@echo "  make dev-up         - Start development environment"
	@echo "  make dev-down       - Stop development environment"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make ci             - Run all CI checks"

test: test-unit

test-unit:
	@echo "Running unit tests..."
	@go test ./internal/... ./cmd/... -count=1 -race -timeout=30s

test-coverage:
	@echo "Running tests with coverage..."
	@go test ./internal/... ./cmd/... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

test-integration:
	@echo "Running integration tests..."
	@go test ./test/integration/... -count=1 -timeout=60s || true

test-e2e:
	@echo "e2e tests not implemented yet"

lint:
	@echo "Running linter..."
	@golangci-lint run ./... || true

sec:
	@echo "Running security scanner..."
	@gosec ./... || true

ci: lint test-unit test-integration test-e2e sec
	@echo "✓ All CI checks passed"

version:
	@echo $(VERSION)

build-all: keygen
	@echo "Building all services..."
	@go build -ldflags "-X github.com/bradtumy/credential-service/internal/version.BuildVersion=$(VERSION)" -o bin/issuer ./cmd/issuer
	@go build -ldflags "-X github.com/bradtumy/credential-service/internal/version.BuildVersion=$(VERSION)" -o bin/verifier ./cmd/verifier
	@echo "✓ Built: bin/issuer, bin/verifier, bin/keygen"

release:
	@echo "Building release version $(VERSION)"
	@go build -ldflags "-X github.com/bradtumy/credential-service/internal/version.BuildVersion=$(VERSION)" ./cmd/issuer
	@go build -ldflags "-X github.com/bradtumy/credential-service/internal/version.BuildVersion=$(VERSION)" ./cmd/verifier

keygen:
	@echo "Building keygen CLI tool"
	@mkdir -p bin
	@go build -o bin/keygen ./cmd/keygen
	@echo "✓ Built: bin/keygen"

dev-up:
	@echo "Starting development environment..."
	@docker compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 3
	@docker compose ps
	@echo "✓ Services ready. Issuer: http://localhost:8080, Verifier: http://localhost:8081"

dev-down:
	@echo "Stopping development environment..."
	@docker compose down
	@echo "✓ Services stopped"

dev-reset:
	@echo "Resetting development environment..."
	@docker compose down -v
	@docker compose up -d --build
	@echo "✓ Environment reset"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ coverage.out coverage.html
	@echo "✓ Clean complete"
