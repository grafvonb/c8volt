# Ralph Memory

Feature: 360-business-id-creation-views
Started: 2026-10-31T11:43:38Z

## Codebase Patterns

- Base process creation and observation models now carry `BusinessId *string`; public/native-to-domain/domain-to-public/public-to-domain mappings copy it with `toolx.CopyPtr`. Nil remains absent; observed empty strings remain valid facts.
- v810 maps both generated request variants and native creation/read responses. Successful ordinary waiting replaces acceptance identity with the observed pointer, including nil; no-wait performs no read and retains native acceptance identity.
- Creation collection facades map results even when returning `ferrors.FromDomain(err)`. Sequential creation returns the successful prefix. Counted creation uses pointer-valued pool slots: nil means failed/unattempted, and only successful returns are compacted in pool-index order. Shared pool behavior is unchanged.
- Counted creation snapshots the business-ID pointer before scheduling and copies it for each attempt. Cancellation with skipped work cannot return a success merely because the pool produced no worker error.
- CLI creation uses scalar `--bid` plus `Flags().Changed("bid")` and copies the pointer into each selector input. Adapter validation keeps version precedence authoritative.
- `cmd/cmd_views_processinstance_run.go` owns creation rows, stable counted sorting, one-envelope success/failure rendering and compact aggregate errors. Human rows use `renderHumanLine`; only JSON and keys go to stdout. `strconv.Quote` preserves opaque IDs safely on one human line.
- Creation renders partial results before `exitAfterRenderedResult`. An operation error wins over a writer error; no second envelope is attempted. Empty `ProcessInstances` serializes as `{}` because both total and items use `omitempty`.
- Creation always suppresses workflow-detail service logs. Its existing semantic progress reporter retains completion/count facts but disables per-attempt failure warnings and clears raw `FailureDetail`; verbose facts remain INFO and transport diagnostics remain independently redacted DEBUG.

## Decisions

- T001 and the T002 CLI implementation form the US1 creation MVP. Resource inspection/report coverage remains scheduled as T003–T006 (US2); do not infer that creation views complete ordinary PI views.
- `domain.ValidateBusinessID` is creation-only: nil allowed, empty/malformed UTF-8 rejected, maximum 256 UTF-16 units. v88/v89 reject any non-nil identity as unsupported at adapter entry before validation; bulk code must not preempt that precedence.
- Creation reuses existing generated clients, service signatures and the shared pool; the default version is unchanged. README documents public compatibility and CLI streams; CLI docs are generated from source metadata.

## Gotchas

- Creation aggregate errors report requested, successful, and without-successful-result counts; missing results may represent unattempted or unconfirmed work, not backend rejections. The bounded reason is the preserved facade class; unsupported business-ID input also names Camunda 8.10.
- Unsupported capability uses the existing exit code 1; invalid input uses 2 and conflicts use 6. `--no-err-codes` still returns 0 without rewriting the failure envelope.
- `decodeBulkCompletionEnvelope` is job-specific (asserts command path). Creation tests use `decodeRunBusinessIDEnvelope`; `bulkCompletionDiagnosticRecords` is reusable for strict configured-log format checks.
- Command help prose is checked by `TestAllCommandHelpFocusesOnActions`: keep literal stdout/stderr descriptions in README rather than help prose. The configured INFO logger remains discoverable in creation help.
- v88 service test construction requires both strict Camunda and strict Operate mocks. v89/v810 need only the Camunda mock.
- BusinessID test inventory is nonempty in domain, core process-instance service, all three adapters, and process facade. Traversal/waiter/walker have no new BusinessID tests; their full existing suites passed separately.

## Reusable Commands

- Use the `isolated` environment-sanitizing helper in `quickstart.md` before all validation. This iteration used an equivalent temporary Python wrapper outside the repository; do not rely on that temporary file surviving.
- Targeted: `go test ./internal/domain ./internal/services/processinstance/... ./c8volt/process -run 'Test.*BusinessID|Test.*Create.*ProcessInstance' -count=1`.
- Targeted race: `go test -race ./internal/services/processinstance/... ./c8volt/process -run 'Test.*BusinessID|Test.*Create.*ProcessInstance' -count=1`.
- Broader affected packages: `go test ./internal/domain ./internal/services/processinstance/... ./c8volt/process -count=1`; repository compilation: `go build ./...`.
- Broader CLI regression: `go test ./cmd -count=1` passed after updating quiet-output and help-focus assertions.
- CLI: `go test ./cmd -run 'TestRunProcessInstance|TestRunHelp|Test.*BusinessID' -count=1`, and the same selection with `-race`; inventory with `go test ./cmd -list 'Test.*BusinessID'`.
- Documentation: `make docs-content`; formatting: targeted `gofmt`; whitespace: `git diff --check`.

## Do Not Repeat

- Do not restore nil-on-error collection facade returns or return the fixed-length pool slots directly: regression tests demonstrated lost successful predecessors and fabricated zero-key slots.
- Do not infer business IDs from creation inputs or related resources; do not normalize opaque values or validate observed empty strings as input.
- Do not bypass configured logging or emit raw joined errors. Negative mutation checks on creation tests detected human direct stdout, direct stderr error printing, and `log.Error(runErr.Error())`; production sources were restored.
- Do not run live validation, discovery, cluster inspection or cleanup. Creation checks use mocks/owned HTTP fixtures and real-terminal subprocesses. Actual engine behavior is unverified; the full repository race suite remains assigned to T006.

## Current Handoff

- Next dependency-ready work is T003 [US2]: complete process-instance inspection and observed mutation-impact views, including the duplicated operations mapper. Recheck tasks.md before selection; T001 and T002 are complete, so no dependency blocks T003. T004–T006 remain in scheduled order. The full repository race suite remains T006's gate.
