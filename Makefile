BINARY   := frostmux
BINDIR   := $(HOME)/go/bin
PKG      := ./cmd/frostmux
BUILD    := go build
GOFLAGS  := -trimpath
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help
.PHONY: build run test lint clean release help

build: ## Compile binary to ~/go/bin
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
	rm -rf dist

release: ## Cross-compile release binaries into dist/
	@mkdir -p dist
	@for pair in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do \
		os=$${pair%%/*}; arch=$${pair##*/}; \
		out=dist/$(BINARY)_$(VERSION)_$${os}_$${arch}; \
		echo "building $${out}"; \
		GOOS=$${os} GOARCH=$${arch} CGO_ENABLED=0 \
			$(BUILD) $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $${out}/$(BINARY) $(PKG); \
		tar -czf $${out}.tar.gz -C $${out} $(BINARY); \
		rm -rf $${out}; \
	done

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
