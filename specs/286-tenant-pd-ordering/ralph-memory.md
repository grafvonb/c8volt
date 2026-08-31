# Ralph Memory

Feature: 286-tenant-pd-ordering
Started: 2026-08-31T11:37:22Z

## Codebase Patterns
- Focused baseline commands from `specs/286-tenant-pd-ordering/quickstart.md` passed before implementation changes in iteration 1.
- `internal/domain/processdefinition.go` now owns `CompareProcessDefinitionsCanonical` and `SortProcessDefinitionsCanonical`; downstream services should reuse that order instead of duplicating tenant/BPMN/version/key comparisons.
- `internal/services/processdefinition.SearchProcessDefinitionsPages` now applies final canonical sorting only to the returned accumulated ordinary collection; page visitor cumulative counts and limit selection still reflect traversal order before final normalization.
- `c8volt/process/client_test.go` now covers ordinary facade search preserving the service-provided canonical sequence and paged facade search returning the service-normalized final sequence while visitor pages remain in arrival order.

## Decisions
- Treat T001 as a validation-only setup work unit; no production code changed in iteration 1.
- T002 and T003 were completed as one foundational work unit because T002's comparator tests require the T003 API to compile while quality gates require a green commit.
- T004 and T015 were paired in iteration 3 because the new service final-order test requires the service-level final normalization to pass.
- T005 was completed as a facade regression test-only work unit; no public facade implementation change was needed because conversion already preserves service slice order.

## Gotchas
- Shell wrapper note: zsh has special parameters named `status` and `commands`; use neutral variable names or run validation loops under `/bin/bash`.

## Reusable Commands
- `go test ./internal/domain -run 'Test(CompareProcessDefinitionsCanonical|SortProcessDefinitionsCanonical|.*ProcessDefinition.*(Sort|Order|Latest))' -count=1`
- `go test ./internal/domain -count=1`
- `go test ./internal/services/processdefinition -run 'Test.*(SearchProcessDefinitionsPages|Latest|WatchSnapshot|Order)' -count=1`
- `go test ./internal/services/processdefinition/v87 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort)' -count=1`
- `go test ./internal/services/processdefinition/v88 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort|Stat)' -count=1`
- `go test ./internal/services/processdefinition/v89 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort|Stat)' -count=1`
- `go test ./internal/services/processdefinition/v810 -run 'Test.*ProcessDefinition.*(Search|Latest|Order|Sort|Stat)' -count=1`
- `go test ./c8volt/process -run 'TestClient_(SearchProcessDefinitions|SearchProcessDefinitionsLatest|SearchProcessDefinitionsPages|CollectProcessDefinitionWatchSnapshot)' -count=1`
- `go test ./cmd -run 'Test.*ProcessDefinition.*(Order|Latest|Paging|Stat|JSON|Keys|Watch|XML)|TestCommandContract' -count=1`

## Do Not Repeat
- Do not use zsh variable names `status` or `commands` in validation-loop scripts.

## Current Handoff
- Next iteration starts at T006: add command-level filtered and `--all-tenants` canonical ordering tests to `cmd/get_processdefinition_test.go`.
