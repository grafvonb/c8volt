# Ralph Memory

Feature: 360-business-id-creation-views
Started: 2026-10-31T11:43:38Z

## Codebase Patterns

- Base process creation and observation models now carry `BusinessId *string`; public/native-to-domain/domain-to-public/public-to-domain mappings copy it with `toolx.CopyPtr`. Nil remains absent; observed empty strings remain valid facts.
- v810 maps both generated request variants and native creation/read responses. Successful ordinary waiting replaces acceptance identity with the observed pointer, including nil; no-wait performs no read and retains native acceptance identity.
- Creation collection facades map results even when returning `ferrors.FromDomain(err)`. Sequential creation returns the successful prefix. Counted creation uses pointer-valued pool slots: nil means failed/unattempted, and only successful returns are compacted in pool-index order. Shared pool behavior is unchanged.
- Counted creation snapshots the business-ID pointer before scheduling and copies it for each attempt. Cancellation with skipped work cannot return a success merely because the pool produced no worker error.

## Decisions

- T001 is complete as the public creation work unit. T002 remains separate because it owns CLI parsing, partial envelopes/exits, configured logging, command metadata and command acceptance tests.
- `domain.ValidateBusinessID` is creation-only: nil allowed, empty/malformed UTF-8 rejected, maximum 256 UTF-16 units. v88/v89 reject any non-nil identity as unsupported at adapter entry before validation; bulk code must not preempt that precedence.
- No generated client, default version, interface signature, shared pool, or command renderer changes were needed for T001. README describes public API behavior and unkeyed-literal compatibility; `make docs-content` regenerated only `docs/index.md`.

## Gotchas

- Existing `run_processinstance.go` still discards partial results at its error branch and renders human creation results through stdout. T002 must repair that full path; service/facade preservation alone does not complete CLI partial-failure support.
- The existing counted service summary uses requested-minus-created as failed; T002 must suppress inappropriate service detail and produce bounded aggregate diagnostics using actual evidence, especially under fail-fast/cancellation. Preserve independent DEBUG diagnostics and avoid raw joined error output.
- v88 service test construction requires both strict Camunda and strict Operate mocks. v89/v810 need only the Camunda mock.
- BusinessID test inventory is nonempty in domain, core process-instance service, all three adapters, and process facade. Traversal/waiter/walker have no new BusinessID tests; their full existing suites passed separately.

## Reusable Commands

- Use the `isolated` environment-sanitizing helper in `quickstart.md` before all validation. This iteration used an equivalent temporary Python wrapper outside the repository; do not rely on that temporary file surviving.
- Targeted: `go test ./internal/domain ./internal/services/processinstance/... ./c8volt/process -run 'Test.*BusinessID|Test.*Create.*ProcessInstance' -count=1`.
- Targeted race: `go test -race ./internal/services/processinstance/... ./c8volt/process -run 'Test.*BusinessID|Test.*Create.*ProcessInstance' -count=1`.
- Broader affected packages: `go test ./internal/domain ./internal/services/processinstance/... ./c8volt/process -count=1`; repository compilation: `go build ./...`.
- Documentation: `make docs-content`; formatting: targeted `gofmt`; whitespace: `git diff --check`.

## Do Not Repeat

- Do not restore nil-on-error collection facade returns or return the fixed-length pool slots directly: regression tests demonstrated lost successful predecessors and fabricated zero-key slots.
- Do not infer business IDs from creation inputs or related resources; do not normalize opaque values or validate observed empty strings as input.
- Do not run live validation, discovery, cluster inspection or cleanup. All T001 checks used mocks/owned HTTP fixtures. Actual engine behavior is unverified; the full repository race suite remains assigned to T006.

## Current Handoff

- Next dependency-ready work is T002 [US1]: implement `run pi --bid`, dedicated creation views, partial-result envelope/exit handling and configured diagnostic logging with command tests/docs. Recheck tasks.md before selection; T001 is complete and no dependency blocks T002. T003–T006 remain in their scheduled order.
