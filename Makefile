# SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
# SPDX-License-Identifier: GPL-3.0-or-later

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
DEMO_VHS_TARGETS := \
	demo-vhs-fast-start \
	demo-vhs-ops-execute-retention-policy \
	demo-vhs-ops-execute-smoke-test \
	demo-vhs-ops-purge-all-process-definitions \
	demo-vhs-ops-purge-orphan-process-instances \
	demo-vhs-ops-purge-process-instances-with-incidents \
	demo-vhs-ops-repair-incident \
	demo-vhs-ops-repair-process-instance
DEMO_VHS_ALIASES := \
	demo-vhs-rp \
	demo-vhs-st \
	demo-vhs-apd \
	demo-vhs-opi \
	demo-vhs-piwi \
	demo-vhs-inc \
	demo-vhs-pi

.PHONY: help all tidy generate generate-clients build test licenses lint fmt vet clean install run cover cover.html release docs docs-content docs-site-install docs-site-build docs-site-build-root docs-site-serve demo-vhs-check $(DEMO_VHS_TARGETS) $(DEMO_VHS_ALIASES)

help: ## Show all available Make targets with a short description.
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-55s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: tidy fmt vet lint licenses test build docs ## Run the full local quality pipeline: tidy, format, vet, lint, licenses, test, build, and all docs.

tidy: ## Synchronize Go module dependencies with the current imports.
	go mod tidy

generate: ## Run all Go generate directives in the repository.
	go generate $(PKG)

generate-clients: ## Regenerate API clients using the repository refresh script.
	bash api/refresh-clients.sh

docs: docs-content docs-site-build ## Build all documentation outputs, including generated markdown and the local docs site.

docs-content: ## Regenerate the markdown CLI reference and sync the docs homepage from README.md.
	go run -ldflags "$(LDFLAGS)" ./docsgen -out ./docs/cli -format markdown

docs-site-install: ## Install the Ruby dependencies needed to build and serve the docs site locally.
	./scripts/docs-site.sh install

docs-site-build: docs-content ## Build the local Jekyll docs site after refreshing the generated docs content.
	./scripts/docs-site.sh build

docs-site-build-root: docs-content ## Build docs for hosts that serve the site from / (no subpath baseurl).
	./scripts/docs-site.sh build-root

docs-site-serve: docs-content ## Serve the local Jekyll docs site with live reload after refreshing generated docs content.
	./scripts/docs-site.sh serve

demo-vhs-check: ## Verify local VHS recording prerequisites and required recording environment variables.
	./demos/vhs/scripts/check-vhs.sh

demo-vhs-fast-start: ## Render the live Camunda-backed Fast Start VHS screencast.
	./demos/vhs/scripts/render.sh fast-start

demo-vhs-ops-execute-retention-policy: ## Render the ops execute retention policy VHS screencast.
	./demos/vhs/scripts/render.sh ops-execute-retention-policy

demo-vhs-rp: demo-vhs-ops-execute-retention-policy ## Alias for demo-vhs-ops-execute-retention-policy.

demo-vhs-ops-execute-smoke-test: ## Render the ops execute smoke test VHS screencast.
	./demos/vhs/scripts/render.sh ops-execute-smoke-test

demo-vhs-st: demo-vhs-ops-execute-smoke-test ## Alias for demo-vhs-ops-execute-smoke-test.

demo-vhs-ops-purge-all-process-definitions: ## Render the ops purge all process definitions VHS screencast.
	./demos/vhs/scripts/render.sh ops-purge-all-process-definitions

demo-vhs-apd: demo-vhs-ops-purge-all-process-definitions ## Alias for demo-vhs-ops-purge-all-process-definitions.

demo-vhs-ops-purge-orphan-process-instances: ## Render the ops purge orphan process instances VHS screencast.
	./demos/vhs/scripts/render.sh ops-purge-orphan-process-instances

demo-vhs-opi: demo-vhs-ops-purge-orphan-process-instances ## Alias for demo-vhs-ops-purge-orphan-process-instances.

demo-vhs-ops-purge-process-instances-with-incidents: ## Render the ops purge process instances with incidents VHS screencast.
	./demos/vhs/scripts/render.sh ops-purge-process-instances-with-incidents

demo-vhs-piwi: demo-vhs-ops-purge-process-instances-with-incidents ## Alias for demo-vhs-ops-purge-process-instances-with-incidents.

demo-vhs-ops-repair-incident: ## Render the ops repair incident VHS screencast.
	./demos/vhs/scripts/render.sh ops-repair-incident

demo-vhs-inc: demo-vhs-ops-repair-incident ## Alias for demo-vhs-ops-repair-incident.

demo-vhs-ops-repair-process-instance: ## Render the ops repair process instance VHS screencast.
	./demos/vhs/scripts/render.sh ops-repair-process-instance

demo-vhs-pi: demo-vhs-ops-repair-process-instance ## Alias for demo-vhs-ops-repair-process-instance.

build: ## Compile the c8volt binary into the local bin directory.
	mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

install: ## Install the c8volt binary into the current Go install location.
	go install -ldflags "$(LDFLAGS)" .

run: build ## Build the binary and print the CLI help output.
	./$(BIN_DIR)/$(BINARY) --help

test: ## Run the full Go test suite with the race detector enabled.
	go test $(PKG) -race -count=1

licenses: ## Check Go dependency licenses.
	go tool go-licenses check $(PKG)

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
