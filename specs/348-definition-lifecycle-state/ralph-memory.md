# Ralph Memory

Feature: 348-definition-lifecycle-state
Started: 2026-10-27T16:55:42Z

## Codebase Patterns

- Definition lifecycle is an optional `State string` (`json:"state,omitempty"`) on domain/public models. v810 copies `string(r.State)`; the facade copies the domain string. v88/v89 leave it empty. Shared conversion supplies collections, pages/visitors, final traversal, latest selection and watch snapshots.
- Human semantic cells are key=0, tenant=1, BPMN ID=2, version=3, lifecycle=4, optional statistics=5. Always reserve cell 4; the existing flat renderer omits all-empty columns. Watch statistics formatting now uses index 5.
- `handleCommandError` accepts an optional bounded human message without changing JSON evidence or exit classification. Ordinary key reads now use `boundedCommandFailureReason`; the configured logger owns diagnostics.

## Decisions

- T001/US1 is complete. No generated-client, request, service interface, sorting, selection, mutation, or waiting mechanics changed. Returned DELETED remains an ordinary item; unknown lifecycle strings survive verbatim in public results.
- Added bounded ordinary key-read diagnostics after the required not-found regression exposed raw HTTP error leakage. Tests preserve the original JSON error evidence and exit code, plain/structured routing, quiet/verbose behavior and independently enabled DEBUG diagnostics.
- T002/US2 is complete. Isolated lookup/list human/JSON tests preserve DRAINING through (active,suspended) observations (0,2), (2,0), and (0,0), retain all other counts/support flags, omit unavailable state, and align mixed/unfamiliar rows. Existing production projection/enrichment needed no change. Source help/README and regenerated docs explain suspended draining and non-atomic observations.
- T003 state change classification/highlighting/watch logger changes remain pending. The shared projection already displays lifecycle with statistics and in watch rows.

## Gotchas

- `definitionStatisticsFixture` independently supplies `lifecycle(read, key)` and `population(read)`; optional `keys`, completed/canceled/incident counts and `countStatus` support mixed rows and failed enrichment. It rejects unexpected endpoints/mutations. Exact-total 8.10 observations use one definition read plus six count reads per definition; older versions use four count reads.
- A forbidden (403) count read maps through `ferrors.FromDomain` to `local_precondition`, exit code 1. Lookup and listing JSON fail without a payload after one count request; never publish the available lifecycle as a successful partial result.

- v810 search re-decodes response `Body`; populate both serialized body and `JSON200` in generated-client mocks.
- Empty public collection JSON payload is `{}`: both `items` and zero `total` are omitted. Empty keys-only output is zero bytes. Quiet nonempty reads retain rows; quiet empty reads suppress the summary.
- Existing human renderers trim trailing row whitespace. Conversion and JSON preserve unfamiliar lifecycle whitespace/case; do not normalize the model to match human row compaction.
- Quote `"8.10"` in raw YAML fixtures; an unquoted numeric 8.10 is parsed as 8.1. Prefer `writeTestConfigForVersion` when custom logging is unnecessary.
- `Execute()` resets root stdout/stderr. Caller-owned inherited/configured destinations must execute the Cobra tree directly in subprocess helpers. Plain logger format includes a full timestamp; structured verbose reads also retain useful key attributes.

## Reusable Commands

All checks use `env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS=` to exclude inherited live settings/build tags.

- `go test ./internal/services/processdefinition/... -run 'ProcessDefinition' -count=1` (all packages passed in T002, including lifecycle retention through sparse/capped statistics paging and exact request counts).
- `go test ./c8volt/process -run 'ProcessDefinition' -count=1`.
- `go test ./cmd -run 'ProcessDefinition' -count=1` (passed in T002, including the expanded shared statistics fixture and existing watch/terminal regressions; T001 recorded 733 passing events with no skips).
- `go test -race ./cmd -run 'ProcessDefinitionLifecycle|CommandErrorEnvelopeProcessDefinition' -count=1` (targeted race check for changed subprocess fixtures).
- `make docs-content`; compile only `go test -tags=integration -c ./integration/cli -o <temporary-directory>/cli.test`; never execute that binary during implementation.
- `gofmt` on changed Go files and `git diff --check`.

## Do Not Repeat

- Do not fetch lifecycle separately, infer it from counts, change state availability on older versions, or merge a missing observation into an earlier one.
- Do not run integration tests/TestMain or inspect/contact a live Camunda environment. The draining observer was changed to bounded CLI JSON reads and compiled only; live transitions remain unverified.
- Do not broaden to `make test` without shared runtime/concurrency changes or unresolved focused failures. No such broader impact was introduced in T001 or T002.

## Current Handoff

- Next dependency-ready work is T003 / US3. Recheck tasks.md before selection; T001 and T002 are complete. No blockers.
- Complete lifecycle comparison/highlighting and configured watch summary/warning/timing logging together with baseline, failure, and terminal regression coverage. Reuse the independent lifecycle/statistics fixture; counts must not supply state.
- Audit ordinary search/selector failure diagnostics as required by T003. Their raw-error expectations are not evidence of logger compliance. T002 changed no production diagnostic/error paths; its added failure execution checks the existing single-envelope JSON contract only.
