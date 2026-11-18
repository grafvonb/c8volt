# Ralph Memory

Feature: 381-explain-connection-failures
Started: 2026-11-18T17:36:22Z

## Codebase Patterns

- `recognizedConnectionFailureReason(err, topologyEndpoint)` in `cmd/cmd_views_config_test_connection.go` owns typed cause/status wording. `boundedCommandFailureReason` prioritizes request-rate evidence, delegates with generic endpoint context, then retains its class fallback. Connection-test cancellation/setup/class-only advice stays in its wrapper.
- `httpc.FailureDetails` preserves malformed-response precedence and independently discovered joined annotations. Facade normalization preserves auth/status annotations but may discard bare transport types; never infer lost evidence from text.
- `TestGetProcessDefinitionLifecycleSearchDiagnostics/oauth-refusal` reuses the process-definition helper with a child-only fake default transport. It records one token refusal and zero unexpected/gateway requests, rejects all other targets, disables version checking only for this fixture, and keeps zero retries.
- `requireProcessDefinitionErrorDiagnostic` handles whole plain/JSON ERROR assertions. `bulkCompletionDiagnosticRecords` remains appropriate for plain-time/JSON, not plain. Command parent fixtures clear inherited configuration/helper variables with `isolateRequestRateEnvironment`.

## Decisions

- T001/US1 is complete. Runtime changes stay in the two planned view files; root metadata/docs were updated and generated. No service, facade, normalization, logger, JSON, exit, retry, or command-handler changes.
- Formatter cases cover raw/wrapped/joined and normalized typed causes, status zero, all specified statuses, auth attribution, misleading text, class-only/lost-evidence fallback, cancellation and local-admission precedence. Config-specific exact fallbacks remain tested.
- Trace reviewed: OAuth editor adds `AuthenticationFailure`; versioned search and facade forward it; command supplies bounded human override; handler logs ERROR through its configured logger or renders the original JSON envelope before exiting. Batch rendering retains partial reports and exits once. No new raw-error interpolation or diagnostic print was introduced.

## Gotchas

- This machine's selected `/Users/adam.boczek/Development/Lang/go/go1.26.10` lacks `vet`. Use the complete cached same-version toolchain via `GOROOT=/Users/adam.boczek/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.darwin-arm64`; retain vet, `GOENV=off`, and empty `GOFLAGS`.
- Direct `httpc.New` test configs require `RequestRateReportInterval: "0s"`. A canceled transport admission creates real typed local-unsent evidence without network dispatch.
- Docs generation refreshes the homepage build provenance automatically. Generated output was reviewed, not hand-edited.

## Reusable Commands

- Prefix Go checks and docs generation with `GOROOT=/Users/adam.boczek/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.darwin-arm64 GOENV=off GOFLAGS=` on this machine.
- T001 passed: `go test ./cmd -run '^(TestBoundedCommandFailureReasons|TestConfigConnectionFailureReasons|TestConfigConnectionDiagnosticLogging|TestCommandErrorEnvelopeProcessDefinition|TestGetProcessDefinitionLifecycleSearchDiagnostics|TestDeleteProcessDefinitionUnfinishedDiagnosticLogging)$' -count=1`.
- Original regression only: `go test ./cmd -run '^TestGetProcessDefinitionLifecycleSearchDiagnostics$/^oauth-refusal$' -count=1`.
- `make docs-content`; `git diff --check`; explicit touched-file `gofmt`.

## Do Not Repeat

- Do not run live-environment tests, import #379, broaden normalization, duplicate status switches, or change machine error evidence to satisfy human redaction assertions.
- The original refusal failed against the old formatter (`internal`). Temporary direct-write and raw-error mutations both failed the intended exact logger assertion; source was restored byte-for-byte. Do not retain those mutations.
- Do not rerun the full suite just for bookkeeping: focused checks cover this rendering change. Real-engine behavior was not validated.

## Current Handoff

- Next dependency-ready work is T002 / US2: identify and fill missing adversarial logger/destination combinations in the existing search, connection and repeated-deletion fixtures. T001 baseline passes; retain its shared implementation and exact wording. Recheck selection against tasks.md. T003 depends on T002 and remains untouched. No blocking dependencies.
