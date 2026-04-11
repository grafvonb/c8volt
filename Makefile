BINARY := c8volt
BIN_DIR := bin
PKG := ./...
COVER_DIR := .coverage
COVER_OUT := $(COVER_DIR)/coverage.out
COVER_HTML := $(COVER_DIR)/coverage.html
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo unknown)
LDFLAGS ?= -X github.com/grafvonb/c8volt/cmd.version=$(VERSION) -X github.com/grafvonb/c8volt/cmd.commit=$(COMMIT) -X github.com/grafvonb/c8volt/cmd.date=$(DATE)

.PHONY: help all tidy generate generate-clients build test lint fmt vet clean install run cover cover.html release docs docs-content docs-site-install docs-site-build docs-site-build-root docs-site-serve

help: ## Show all available Make targets with a short description.
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: tidy fmt vet lint test build docs ## Run the full local quality pipeline: tidy, format, vet, lint, test, build, and all docs.

tidy: ## Synchronize Go module dependencies with the current imports.
	go mod tidy

generate: ## Run all Go generate directives in the repository.
	go generate $(PKG)

generate-clients: ## Regenerate API clients using the repository refresh script.
	bash api/refresh-clients.sh

docs: docs-content docs-site-build ## Build all documentation outputs, including generated markdown and the local docs site.

docs-content: ## Regenerate the markdown CLI reference and sync the docs homepage from README.md.
	go run ./docsgen -out ./docs/cli -format markdown

docs-site-install: ## Install the Ruby dependencies needed to build and serve the docs site locally.
	./scripts/docs-site.sh install

docs-site-build: docs-content ## Build the local Jekyll docs site after refreshing the generated docs content.
	./scripts/docs-site.sh build

docs-site-build-root: docs-content ## Build docs for hosts that serve the site from / (no subpath baseurl).
	./scripts/docs-site.sh build-root

docs-site-serve: docs-content ## Serve the local Jekyll docs site with live reload after refreshing generated docs content.
	./scripts/docs-site.sh serve

build: ## Compile the c8volt binary into the local bin directory.
	mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

install: ## Install the c8volt binary into the current Go install location.
	go install -ldflags "$(LDFLAGS)" .

run: build ## Build the binary and print the CLI help output.
	./$(BIN_DIR)/$(BINARY) --help

test: ## Run the full Go test suite with the race detector enabled.
	go test $(PKG) -race -count=1

lint: ## Run golangci-lint across the repository.
	golangci-lint run

fmt: ## Format all Go packages in the repository.
	go fmt $(PKG)

vet: ## Run go vet across all Go packages.
	go vet $(PKG)

clean: ## Remove local build artifacts and coverage output.
	rm -rf $(BIN_DIR) $(COVER_DIR)

cover: ## Generate a text coverage report and print the total coverage summary.
	mkdir -p $(COVER_DIR)
	go test $(PKG) -race -covermode=atomic -coverprofile=$(COVER_OUT)
	go tool cover -func=$(COVER_OUT) | tail -n 1

cover.html: cover ## Generate the HTML coverage report after collecting coverage data.
	go tool cover -html=$(COVER_OUT) -o $(COVER_HTML)
	@echo "Open $(COVER_HTML)"

release: ## Build release artifacts locally with GoReleaser without publishing them.
	goreleaser release --clean --skip=publish
