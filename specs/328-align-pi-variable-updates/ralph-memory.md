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

## Gotchas

- Camunda 8.10 cursor requests use `*EndCursor`; 8.8 and 8.9 use a value `EndCursor`.
- `hasMoreTotalItems` describes a capped total, not by itself whether another result page exists.

## Reusable Commands

- `go test ./internal/services/variable/... -count=1`
- `go test ./internal/services/processinstance/... -run 'Variable|Enrich' -count=1`
- `go test ./... -run '^$'`
- `git diff --check`

## Do Not Repeat

- Do not reintroduce one-page `limit=1000` reads in callers or infer completion from a short/sparse retained page.
- Do not decode variable values through generated models; they omit raw value/truncation details needed by planning and verification.

## Current Handoff
- Start T002 (US1): implement collection planning and thin facade models/conversions on top of the complete root-variable reader. Preserve the T001 page contract and return no executable partial plan after any target failure.
