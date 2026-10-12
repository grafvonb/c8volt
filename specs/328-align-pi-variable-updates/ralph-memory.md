# Ralph Memory

Feature: 328-align-pi-variable-updates
Started: 2026-10-12T04:49:40Z

## Codebase Patterns

- Variable adapters keep generated request/filter construction version-specific, while `internal/services/variable/search` owns complete traversal through a narrow page-reader contract.
- Raw variable response decoding must use pointer fields to distinguish absent required fields from valid empty strings and absent optional total metadata from zero.

## Decisions

- Complete root-variable reads prefer an advancing `endCursor`; without one they advance the offset by the backend raw item count, not the locally retained root-only count.
- Exact totals terminate at the observed raw count. Capped totals remain monotonic lower bounds. Missing totals remain genuinely absent and traversal probes until an empty page without a cursor.
- Adapter-local root filtering retains only records whose process-instance and scope keys both match the selected root, while `RawItemCount` records all backend items.
- Prepared PI updates use an additive composed `processinstance.VariableUpdateAPI`; legacy facade constructors inject nil and the new method returns a normalized precondition error.
- Collection planning reads all selected-instance identities first, then complete root variables in stable key order, and returns the zero plan on any later target failure.
- The PI command retains the public frozen plan beside a mechanically adapted legacy preview; tenant context now comes directly from the plan instead of command-side variable evidence reconstruction.
- Prepared PI execution validates and deep-copies the entire plan before I/O, schedules only changed roots through the bounded pool, and reconstructs ordered unchanged/failed/skipped outcomes against selected-key order.
- Accepted changed roots use the shared variable waiter exactly once with all frozen requested values; no-wait bypasses verification, and confirmation failure retains mutation acceptance.
- The PI command carries the same public prepared plan through approval into facade execution; successful keys-only output is derived from `MutationAccepted`, not success status.
- PI partial execution errors use the UT-style retained-result renderer before `exitAfterRenderedResult`; JSON receives one failure envelope, while human and keys-only output retain available per-key outcomes without a second generic error render.
- PI dry-runs and completed all-unchanged plans return before approval and execution; explicit machine modes survive quiet, JSON takes precedence over keys, and rendering adds no backend reads.

## Gotchas

- Camunda 8.10 cursor requests use `*EndCursor`; 8.8 and 8.9 use a value `EndCursor`.
- `hasMoreTotalItems` describes a capped total, not by itself whether another result page exists.
- Plan values must be copied independently into requested, mutation, verification, classification, and facade representations so caller or service mutation cannot broaden a later payload.
- The planning dependency can remain a narrow complete-reader interface: `NewVariableUpdates` discovers the optional exact scope writer by interface assertion, preserving planning-only test doubles and legacy composition behavior.

## Reusable Commands

- `go test ./internal/services/variable/... -count=1`
- `go test ./internal/services/processinstance/... -run 'Variable|Enrich' -count=1`
- `go test ./... -run '^$'`
- `go test -race ./internal/services/processinstance ./internal/services/variable/waiter ./c8volt/process -run 'Variable' -count=1`
- `git diff --check`

## Do Not Repeat

- Do not reintroduce one-page `limit=1000` reads in callers or infer completion from a short/sparse retained page.
- Do not decode variable values through generated models; they omit raw value/truncation details needed by planning and verification.

## Current Handoff
- Feature complete; no handoff required.
