# Ralph Memory

Feature: 235-get-pi-elements
Started: 2026-07-16T15:20:23Z

## Codebase Patterns
- `cmd/get_processinstance.go` owns the process-instance Cobra command, flag registration, validation calls, keyed/list orchestration, and render dispatch. Existing enrichment branches call helpers from `cmd/get_processinstance_enrichment.go` and then render through `cmd/cmd_views_processinstance_activity.go`.
- `cmd/get_processinstance_enrichment.go` wraps facade enrichment calls with activity labels and preserves zero-row behavior by still calling the facade path.
- `cmd/cmd_views_processinstance_activity.go` is the shared human/JSON activity renderer for process-instance rows with `vars:` and `incidents:` sections; future element rendering should extend `processInstanceActivityItem` and section formatting there.
- `cmd/process_api_stub_test.go` defines `stubProcessAPI`; new process facade methods must be added there or command tests will fail at the `process.API` interface assertion.
- Runtime element reuse points already exist in `c8volt/element` and `internal/services/element` with public/domain fields matching the feature contract and `SearchElements`/`SearchElementsPage` methods.
- Process facade enrichment currently delegates to `internal/services/processinstance` using thin conversion in `c8volt/process/client.go` and `c8volt/process/convert.go`; element enrichment should follow the incident/variable pattern.
- Element-enriched process-instance contracts now exist as `domain.ElementEnrichedProcessInstances`, `process.ElementEnrichedProcessInstances`, and `process.ProcessInstanceElement`; conversion helpers in `c8volt/process/convert.go` map `domain.Element` into the process facade model.
- `c8volt.New` wires the element service into the process facade with `process.NewWithElements(pdAPI, piAPI, incAPI, eAPI, log)`. The existing `process.New` remains for tests/backcompat and constructs a facade with no element service dependency.

## Decisions
- Treat Phase 1 as a setup work unit only. Do not start Phase 2 or US1 implementation in iteration 1.
- Use the resolved iteration commit policy (`conventional`, scope `ralph`, issue `auto`) for the work-unit commit, despite the plan note that references GitHub issue #242.
- Iteration 2 completed only Phase 2 foundation. `EnrichProcessInstancesWithElements` is present on `process.API` but intentionally returns an unsupported placeholder until US1 implements the service attachment workflow and facade delegation.

## Gotchas
- `plan.md` records GitHub issue #242, but branch-prefix issue inference on `235-get-pi-elements` produces `#235` under `commit.issue: auto`.
- `--with-elements` must reject `--total` and `--keys-only`; validation should be added beside the existing `--with-incidents` and `--with-vars` validation paths.
- `flagGetPIWithElements` is registered and reset, but command orchestration does not call the element enrichment facade yet. US1 must add validation, keyed invocation, activity wrapper, and rendering before relying on the flag behavior.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd -run 'TestGetProcessInstanceHelp_DocumentsPagingAndAutomationSurface' -count=1`
- `go test ./c8volt/process ./internal/services/processinstance -count=1`
- `go test ./internal/services/processinstance ./c8volt/process ./cmd -count=1`

## Do Not Repeat
- Do not duplicate element lookup logic in `cmd`; reuse the element service through the process facade/service enrichment boundary.
- Do not add a process-instance-specific generated-client path for runtime element lookup.

## Current Handoff
- Next iteration starts Phase 3 / US1. Begin with tests T014-T017, then replace the placeholder `process.EnrichProcessInstancesWithElements` implementation by adding `processinstance.EnrichProcessInstancesWithElements`, calling the injected element service, and wiring keyed `--with-elements` rendering.
