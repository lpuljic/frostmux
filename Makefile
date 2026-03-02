BINARY   := frostmux
BINDIR   := $(HOME)/go/bin
PKG      := ./cmd/frostmux
BUILD    := go build
GOFLAGS  := -trimpath
LDFLAGS  := -s -w -X lpuljic/frostmux/internal/cli.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.DEFAULT_GOAL := help
.PHONY: build run test lint clean help

build: ## Compile binary to ~/go/bin/core
	@mkdir -p $(BINDIR)
	$(BUILD) $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINDIR)/$(BINARY) $(PKG)

run: build ## Build and run
	$(BINDIR)/$(BINARY)

test: ## Run tests with race detector
	go test -v -race -count=1 ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

clean: ## Remove build artifacts
	rm -f $(BINDIR)/$(BINARY)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

