# Distributed KV Store — Makefile
# Run `make` or `make help` to see available targets.

.DEFAULT_GOAL := help
SHELL := /bin/bash

.PHONY: help fmt vet test test-race clean proto build up down logs run-local

# True iff the module currently contains at least one .go file.
HAS_GO := $(shell find . -name '*.go' -not -path './.git/*' -print -quit 2>/dev/null)

help: ## Show this help
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "} {printf "  %-12s %s\n", $$1, $$2}'

# --- Working targets ---
# Each guards on HAS_GO so an empty module (M0) does not error.

fmt: ## Format all Go source
	@if [ -n "$(HAS_GO)" ]; then go fmt ./...; else echo "fmt: no Go packages yet"; fi

vet: ## Static analysis
	@if [ -n "$(HAS_GO)" ]; then go vet ./...; else echo "vet: no Go packages yet"; fi

test: ## Run all tests
	@if [ -n "$(HAS_GO)" ]; then go test ./...; else echo "test: no Go packages yet"; fi

test-race: ## Run all tests with the race detector
	@if [ -n "$(HAS_GO)" ]; then go test -race ./...; else echo "test-race: no Go packages yet"; fi

clean: ## Remove build artifacts and local data dirs
	@rm -rf bin/ data/

# --- Placeholders filled in by later milestones ---

proto: ## (M3) Regenerate Go from proto/*.proto
	@command -v protoc >/dev/null 2>&1 || { echo "protoc is required for make proto"; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "protoc-gen-go is required for make proto"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "protoc-gen-go-grpc is required for make proto"; exit 1; }
	protoc \
		--go_out=paths=source_relative:. \
		--go-grpc_out=paths=source_relative:. \
		proto/kv/kv.proto proto/raft/raft.proto

build: ## (M1) Compile the node binary into ./bin
	@echo "make build: not yet implemented (lands in Milestone 1)"; exit 1

up: ## (M3) docker compose up --build
	@echo "make up: not yet implemented (lands in Milestone 3)"; exit 1

down: ## (M3) docker compose down -v
	@echo "make down: not yet implemented (lands in Milestone 3)"; exit 1

logs: ## (M3) docker compose logs -f
	@echo "make logs: not yet implemented (lands in Milestone 3)"; exit 1

run-local: ## (M3) Run 3 nodes as local processes (no Docker)
	@echo "make run-local: not yet implemented (lands in Milestone 3)"; exit 1
