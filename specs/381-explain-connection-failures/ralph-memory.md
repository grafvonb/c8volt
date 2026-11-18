# Ralph Memory

Feature: 381-explain-connection-failures
Started: 2026-11-18T17:36:22Z

## Codebase Patterns

- `recognizedConnectionFailureReason(err, topologyEndpoint)` in `cmd/cmd_views_config_test_connection.go` owns typed cause/status wording. `boundedCommandFailureReason` prioritizes request-rate evidence, delegates with generic endpoint context, then retains its class fallback. Connection-test cancellation/setup/class-only advice stays in its wrapper.
- `httpc.FailureDetails` preserves malformed-response precedence and independently discovered joined annotations. Facade normalization preserves auth/status annotations but may discard bare transport types; never infer lost evidence from text.
- `TestGetProcessDefinitionLifecycleSearchDiagnostics/oauth-refusal` reuses the process-definition helper with a child-only fake default transport. It records one token refusal and zero unexpected/gateway requests, rejects all other targets, disables version checking only for this fixture, and keeps zero retries.
- `requireProcessDefinitionErrorDiagnostic` handles whole plain/JSON ERROR assertions. `bulkCompletionDiagnosticRecords` remains appropriate for plain-time/JSON, not plain. Command parent fixtures clear inherited configuration/helper variables with `isolateRequestRateEnvironment`.

## Decisions

- T003/US3 is complete with test-only changes. Refusal and ordinary error envelopes cover quiet+JSON and suppressed exits; connection failures retain diagnostic-only output even with JSON. Successful connection JSON preserves its base URL/config file and survives quiet mode; successful reads retain exactly one advisory and one operation request.
- The request-rate deletion fixture now includes a first HTTP 504 rejection followed by two unsent waits. It retains one dispatched request, three failed reports, original 504/body evidence in JSON, timeout class/exit, and the exact local-wait human aggregate with zero accepted. Existing partial-success, interruption, accepted-cancellation confirmation, and reporter cases passed. The terminal compatibility wrapper reuses existing acceptance/abort/activity tests under isolated environment configuration.

- T002/US2 is complete with tests only. Search adds WARN and quiet overriding each DEBUG source. Connection and deletion matrices now exercise both configured and inherited writers for every covered mode; WARN+verbose suppresses INFO while retaining ERROR. Batch partial success runs through normal/verbose/quiet, both DEBUG sources and JSON+DEBUG.
- Batch fixtures seed credentials, URL/method/body, cursor, resource key and forged-line markers; human sinks reject them, while original machine detail retains them. JSON keeps one envelope plus EOF, `invalid_input` class, `invalid` outcome and exit 2, with three reports and zero/one accepted item.

- T001/US1 is complete. Runtime changes stay in the two planned view files; root metadata/docs were updated and generated. No service, facade, normalization, logger, JSON, exit, retry, or command-handler changes.
- Formatter cases cover raw/wrapped/joined and normalized typed causes, status zero, all specified statuses, auth attribution, misleading text, class-only/lost-evidence fallback, cancellation and local-admission precedence. Config-specific exact fallbacks remain tested.
- Trace reviewed: OAuth editor adds `AuthenticationFailure`; versioned search and facade forward it; command supplies bounded human override; handler logs ERROR through its configured logger or renders the original JSON envelope before exiting. Batch rendering retains partial reports and exits once. No new raw-error interpolation or diagnostic print was introduced.

## Gotchas

- CI follow-up found stale human class-only expectations in 14 shared-formatter consumer test files. Inventory every `boundedCommandFailureReason` caller, including creation and lifecycle planning; preserve machine classes/exits and command-specific fallbacks. The corrected expectations passed the full isolated race/coverage suite (20,119 test/subtest events, no failures/skipped tests); see progress.md and the quickstart consumer-coverage section.

- `resource.DeleteReport.Ok` uses `omitempty`: failed report JSON omits `ok`, rather than encoding false. Successful connection JSON intentionally contains `base_url`; apply raw-URL exclusions to diagnostic sinks while preserving and asserting that result field. Failure stdout remains empty and seeded secrets are rejected across all captures.

- Existing deletion fixture performs 6 definition reads and 36 statistics searches (preview and execution each read 3 definitions, with 6 statistics queries per read), then 3 mutation requests. Partial success has 2 failures; all-failure has 3. Preflight failure stops after 1 definition read / 1 search / 0 mutations. HTTP 400 machine outcome is `invalid`, not `failed`.

- This machine's selected `/Users/adam.boczek/Development/Lang/go/go1.26.10` lacks `vet`. Use the complete cached same-version toolchain via `GOROOT=/Users/adam.boczek/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.darwin-arm64`; retain vet, `GOENV=off`, and empty `GOFLAGS`.
- Direct `httpc.New` test configs require `RequestRateReportInterval: "0s"`. A canceled transport admission creates real typed local-unsent evidence without network dispatch.
- Docs generation refreshes the homepage build provenance automatically. Generated output was reviewed, not hand-edited.

## Reusable Commands

- Final T003 passed: the T003 selector in tasks.md plus `TestGetProcessDefinitionLifecycleSearchDiagnostics`, `TestConfigConnectionFailureReasons`, and `TestConnectionFailureConfirmationCompatibility`, with `-count=1 -json` (49.751s; 625 passing test/subtest events, zero failures/skips). The latter wrapper runs real Darwin PTY acceptance and abort checks; no live service is involved.
- Combined-evidence negative check: `go test ./cmd -run '^TestRequestRateMutationLocalFailure$/^definition-rejected$/^plain-time$/^--verbose$' -count=1`. Moving connection recognition before local admission fails the unsent-guidance assertion; production was restored byte-for-byte before final validation.

- T002 passed: `go test ./cmd -run '^(TestGetProcessDefinitionLifecycleSearchDiagnostics|TestConfigConnectionDiagnosticLogging|TestDeleteProcessDefinitionUnfinishedDiagnosticLogging)$' -count=1` (11.659s).
- Batch mutation-detection selector: `go test ./cmd -run '^TestDeleteProcessDefinitionUnfinishedDiagnosticLogging$/^(plain-time|json)$/^configured$/^normal$' -count=1`. Both direct configured-stderr output and appended raw joined error fail; restored production passes (0.741s).

- Prefix Go checks and docs generation with `GOROOT=/Users/adam.boczek/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.darwin-arm64 GOENV=off GOFLAGS=` on this machine.
- T001 passed: `go test ./cmd -run '^(TestBoundedCommandFailureReasons|TestConfigConnectionFailureReasons|TestConfigConnectionDiagnosticLogging|TestCommandErrorEnvelopeProcessDefinition|TestGetProcessDefinitionLifecycleSearchDiagnostics|TestDeleteProcessDefinitionUnfinishedDiagnosticLogging)$' -count=1`.
- Original regression only: `go test ./cmd -run '^TestGetProcessDefinitionLifecycleSearchDiagnostics$/^oauth-refusal$' -count=1`.
- `make docs-content`; `git diff --check`; explicit touched-file `gofmt`.

## Do Not Repeat

- Do not run live-environment tests, import #379, broaden normalization, duplicate status switches, or change machine error evidence to satisfy human redaction assertions.
- The original refusal failed against the old formatter (`internal`). Temporary direct-write and raw-error mutations both failed the intended exact logger assertion; source was restored byte-for-byte. Do not retain those mutations.
- Do not rerun a passed full suite just for bookkeeping. The CI consumer regressions justified the broader race/coverage check recorded in progress.md; the original narrow selectors alone did not cover every shared-formatter caller. Real-engine behavior was not validated.

## Current Handoff

- Feature complete; no handoff required.
