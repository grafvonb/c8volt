# Ralph Memory

Feature: 235-get-pi-elements
Started: 2026-07-16T15:20:23Z

## Codebase Patterns
- `cmd/get_processinstance.go` owns the process-instance Cobra command, flag registration, validation calls, keyed/list orchestration, and render dispatch. Existing enrichment branches call helpers from `cmd/get_processinstance_enrichment.go` and then render through `cmd/cmd_views_processinstance_activity.go`.
- `cmd/get_processinstance_enrichment.go` wraps facade enrichment calls with activity labels and preserves zero-row behavior by still calling the facade path.
- `cmd/cmd_views_processinstance_activity.go` is the shared human/JSON activity renderer for process-instance rows with `vars:`, `incidents:`, and keyed `elements:` sections. Existing walk helpers still call the non-element wrapper signatures; use `formatProcessInstanceActivityLinesWithElementsWithTimezone` only when element rows must be included.
- `cmd/process_api_stub_test.go` defines `stubProcessAPI`; new process facade methods must be added there or command tests will fail at the `process.API` interface assertion.
- Runtime element reuse points already exist in `c8volt/element` and `internal/services/element` with public/domain fields matching the feature contract and `SearchElements`/`SearchElementsPage` methods.
- Process facade enrichment currently delegates to `internal/services/processinstance` using thin conversion in `c8volt/process/client.go` and `c8volt/process/convert.go`; element enrichment now follows the incident/variable pattern through `process.EnrichProcessInstancesWithElements`.
- Element-enriched process-instance contracts now exist as `domain.ElementEnrichedProcessInstances`, `process.ElementEnrichedProcessInstances`, and `process.ProcessInstanceElement`; conversion helpers in `c8volt/process/convert.go` map `domain.Element` into the process facade model.
- `c8volt.New` wires the element service into the process facade with `process.NewWithElements(pdAPI, piAPI, incAPI, eAPI, log)`. The existing `process.New` remains for tests/backcompat and constructs a facade with no element service dependency.

## Decisions
- Treat Phase 1 as a setup work unit only. Do not start Phase 2 or US1 implementation in iteration 1.
- Use the resolved iteration commit policy (`conventional`, scope `ralph`, issue `auto`) for the work-unit commit, despite the plan note that references GitHub issue #242.
- Iteration 2 completed only Phase 2 foundation and introduced the `process.API` element-enrichment contract before US1 wired the real workflow.
- Iteration 3 completed US1 keyed element enrichment. `processinstance.EnrichProcessInstancesWithElements` calls the injected element service with `ElementSearchQuery{ProcessInstanceKey: <pi key>}`, filters broad responses by owner, sorts attached elements by `startDate` then `elementInstanceKey`, and preserves selected process-instance order.
- Keyed `get pi --key <key> --with-elements` now validates `--total`, `--keys-only`, and keyed search-filter conflicts, calls the process facade, and renders `elements:` rows under the process instance.

## Gotchas
- `plan.md` records GitHub issue #242, but branch-prefix issue inference on `235-get-pi-elements` produces `#235` under `commit.issue: auto`.
- Direct keyed lookup on Camunda 8.7 currently fails at the existing process-instance direct-lookup unsupported boundary before element lookup; command coverage asserts a clear unsupported capability result for `--with-elements` on 8.7.
- Do not assert tenant filters on keyed `--with-elements` element searches: explicit `--key` admin input uses `collectExplicitPIAdminInputOptions`, so the element search is owner-key scoped without a tenant filter.
- List/search `--with-elements` is not implemented yet. At the start of US2, avoid silently ignoring the flag in non-keyed modes by wiring bounded and incremental search paths before broad user-facing validation/docs work.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd -run 'TestGetProcessInstanceHelp_DocumentsPagingAndAutomationSurface' -count=1`
- `go test ./internal/services/processinstance ./c8volt/process ./cmd -run 'TestEnrichProcessInstancesWithElements|TestClient_EnrichProcessInstancesWithElements|TestGetProcessInstance.*Elements|TestProcessInstanceActivity.*Elements|TestGetProcessInstanceHelp_DocumentsPagingAndAutomationSurface' -count=1`
- `go test ./c8volt/process ./internal/services/processinstance -count=1`
- `go test ./internal/services/processinstance ./c8volt/process ./cmd -count=1`

## Do Not Repeat
- Do not duplicate element lookup logic in `cmd`; reuse the element service through the process facade/service enrichment boundary.
- Do not add a process-instance-specific generated-client path for runtime element lookup.

## Current Handoff
- Next iteration starts Phase 4 / US2. Implement list/search `--with-elements` only: add T024-T027 tests, then wire bounded and incremental process-instance search paths to enrich selected rows with elements while keeping process-instance limits, page prompts, and `found: N` counts process-instance scoped.
