.PHONY: help check lint fmt test

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: lint test ## Run linters and unit tests

lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not found; run inside nix-shell"; exit 1; }
	golangci-lint run ./...

fmt: ## Format Go files
	golangci-lint fmt ./...

test: ## Run Go unit tests
	go test ./...
