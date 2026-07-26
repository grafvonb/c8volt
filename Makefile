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
C8VOLT_IT_GO_TEST_FLAGS ?=
C8VOLT_IT_VERBOSE ?= $(if $(findstring -v,$(C8VOLT_IT_GO_TEST_FLAGS)),1,0)
C8VOLT_IT_GO_TEST ?= GOCACHE=/tmp/c8volt-gocache C8VOLT_IT_VERBOSE=$(C8VOLT_IT_VERBOSE) go test -tags=integration ./integration/cli
C8VOLT_IT_TIMEOUT ?= 60m
C8VOLT_IT_VOLUME_TIMEOUT ?= 90m
C8VOLT_IT_REAL_STATE_TIMEOUT ?= 90m
C8VOLT_IT_AUTOMATION ?= 0
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
INTEGRATION_CLI_TARGETS := \
	integration-cli-get \
	integration-cli-walk \
	integration-cli-update \
	integration-cli-cancel \
	integration-cli-delete \
	integration-cli-expect-resolve \
	integration-cli-deploy-embed-run \
	integration-cli-ops-analyse \
	integration-cli-ops-execute \
	integration-cli-ops-purge \
	integration-cli-ops-repair
INTEGRATION_CLI_VOLUME_TARGETS := \
	integration-cli-get-volume \
	integration-cli-walk-volume \
	integration-cli-update-volume \
	integration-cli-cancel-volume \
	integration-cli-delete-volume \
	integration-cli-expect-resolve-volume \
	integration-cli-deploy-embed-run-volume \
	integration-cli-ops-analyse-volume \
	integration-cli-ops-execute-volume \
	integration-cli-ops-purge-volume \
	integration-cli-ops-repair-volume
INTEGRATION_CLI_REAL_STATE_TARGETS := \
	integration-cli-real-state-gaps \
	integration-cli-real-state-jobs \
	integration-cli-real-state-incidents \
	integration-cli-real-state-listeners \
	integration-cli-real-state-bpmn-error \
	integration-cli-real-state-retention \
	integration-cli-real-state-destructive \
	integration-cli-real-state-cascade
INTEGRATION_TEST_TARGETS := \
	integration-test \
	integration-test-volume \
	integration-test-real-state \
	integration-test-all

.PHONY: help all tidy generate generate-clients build test licenses lint fmt vet clean install run cover cover.html release docs docs-content docs-site-install docs-site-serve demo-vhs-check integration-test-confirm $(DEMO_VHS_TARGETS) $(DEMO_VHS_ALIASES) $(INTEGRATION_CLI_TARGETS) $(INTEGRATION_CLI_VOLUME_TARGETS) $(INTEGRATION_CLI_REAL_STATE_TARGETS) $(INTEGRATION_TEST_TARGETS)

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

integration-test-confirm:
	@if [ "$(C8VOLT_IT_AUTOMATION)" = "1" ]; then \
		echo "Skipping integration test confirmation."; \
	else \
		printf "Integration tests may mutate real Camunda cluster state. Continue? [y/N] "; \
		read answer; \
		case "$$answer" in y|Y|yes|YES) ;; *) echo "Aborted."; exit 1 ;; esac; \
	fi

integration-test: $(INTEGRATION_CLI_TARGETS) ## Run all baseline CLI integration test slices.

integration-test-volume: $(INTEGRATION_CLI_VOLUME_TARGETS) ## Run all volume CLI integration test slices.

integration-test-real-state: $(INTEGRATION_CLI_REAL_STATE_TARGETS) ## Run all C89 real-state integration test slices.

integration-test-all: integration-test integration-test-volume integration-test-real-state ## Run baseline, volume, and real-state integration slices.

integration-cli-get: integration-test-confirm ## Run destructive CLI integration tests for get commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestGetFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-get-volume: integration-test-confirm ## Run destructive volume CLI integration tests for get commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeGetFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-walk-volume: integration-test-confirm ## Run destructive volume CLI integration tests for walk commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeWalkFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-update-volume: integration-test-confirm ## Run destructive volume CLI integration tests for update commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeUpdateFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-cancel-volume: integration-test-confirm ## Run destructive volume CLI integration tests for cancel commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeCancelFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-delete-volume: integration-test-confirm ## Run destructive volume CLI integration tests for delete commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeDeleteFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-expect-resolve-volume: integration-test-confirm ## Run destructive volume CLI integration tests for expect and resolve commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeExpectResolveFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-deploy-embed-run-volume: integration-test-confirm ## Run destructive volume CLI integration tests for deploy, embed, and run commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeDeployEmbedRunFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-ops-analyse-volume: integration-test-confirm ## Run destructive volume CLI integration tests for ops analyse commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeOpsAnalyseFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-ops-execute-volume: integration-test-confirm ## Run destructive volume CLI integration tests for ops execute commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeOpsExecuteFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-ops-purge-volume: integration-test-confirm ## Run destructive volume CLI integration tests for ops purge commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeOpsPurgeFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-ops-repair-volume: integration-test-confirm ## Run destructive volume CLI integration tests for ops repair commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestVolumeOpsRepairFamily -count=1 -timeout=$(C8VOLT_IT_VOLUME_TIMEOUT)

integration-cli-real-state-gaps: integration-test-confirm ## Run non-destructive C89 real-state gap and matrix validation.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestRealStateGapFamily -count=1 -timeout=$(C8VOLT_IT_REAL_STATE_TIMEOUT)

integration-cli-real-state-jobs: integration-test-confirm ## Run destructive C89 real-state integration tests for jobs.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestRealStateJobsFamily -count=1 -timeout=$(C8VOLT_IT_REAL_STATE_TIMEOUT)

integration-cli-real-state-incidents: integration-test-confirm ## Run destructive C89 real-state integration tests for incidents.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestRealStateIncidentsFamily -count=1 -timeout=$(C8VOLT_IT_REAL_STATE_TIMEOUT)

integration-cli-real-state-listeners: integration-test-confirm ## Run destructive C89 real-state integration tests for listener state.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestRealStateListenersFamily -count=1 -timeout=$(C8VOLT_IT_REAL_STATE_TIMEOUT)

integration-cli-real-state-bpmn-error: integration-test-confirm ## Run destructive C89 real-state integration tests for BPMN error job state.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestRealStateBPMNErrorFamily -count=1 -timeout=$(C8VOLT_IT_REAL_STATE_TIMEOUT)

integration-cli-real-state-retention: integration-test-confirm ## Run destructive C89 real-state integration tests for retention semantics.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestRealStateRetentionFamily -count=1 -timeout=$(C8VOLT_IT_REAL_STATE_TIMEOUT)

integration-cli-real-state-destructive: integration-test-confirm ## Run destructive C89 real-state integration tests for destructive post-state semantics.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestRealStateDestructiveFamily -count=1 -timeout=$(C8VOLT_IT_REAL_STATE_TIMEOUT)

integration-cli-real-state-cascade: integration-test-confirm ## Run destructive C89 real-state integration tests for parent/child cascade semantics.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestRealStateCascadeFamily -count=1 -timeout=$(C8VOLT_IT_REAL_STATE_TIMEOUT)

integration-cli-walk: integration-test-confirm ## Run destructive CLI integration tests for walk commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestWalkFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-update: integration-test-confirm ## Run destructive CLI integration tests for update commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestUpdateFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-cancel: integration-test-confirm ## Run destructive CLI integration tests for cancel commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestCancelFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-delete: integration-test-confirm ## Run destructive CLI integration tests for delete commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestDeleteFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-expect-resolve: integration-test-confirm ## Run destructive CLI integration tests for expect and resolve commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestExpectResolveFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-deploy-embed-run: integration-test-confirm ## Run destructive CLI integration tests for deploy, embed, and run commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestDeployEmbedRunFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-ops-analyse: integration-test-confirm ## Run destructive CLI integration tests for ops analyse commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestOpsAnalyseFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-ops-execute: integration-test-confirm ## Run destructive CLI integration tests for ops execute commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestOpsExecuteFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-ops-purge: integration-test-confirm ## Run destructive CLI integration tests for ops purge commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestOpsPurgeFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

integration-cli-ops-repair: integration-test-confirm ## Run destructive CLI integration tests for ops repair commands.
	$(C8VOLT_IT_GO_TEST) $(C8VOLT_IT_GO_TEST_FLAGS) -run TestOpsRepairFamily -count=1 -timeout=$(C8VOLT_IT_TIMEOUT)

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
