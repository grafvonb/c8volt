# Ralph Memory

Feature: 286-tenant-pd-ordering
Started: 2026-08-31T11:37:22Z

## Codebase Patterns
- Focused baseline commands from `specs/286-tenant-pd-ordering/quickstart.md` passed before implementation changes in iteration 1.
- `internal/domain/processdefinition.go` now owns `CompareProcessDefinitionsCanonical` and `SortProcessDefinitionsCanonical`; downstream services should reuse that order instead of duplicating tenant/BPMN/version/key comparisons.
- `internal/services/processdefinition.SearchProcessDefinitionsPages` now applies final canonical sorting only to the returned accumulated ordinary collection; page visitor cumulative counts and limit selection still reflect traversal order before final normalization.
- `c8volt/process/client_test.go` now covers ordinary facade search preserving the service-provided canonical sequence and paged facade search returning the service-normalized final sequence while visitor pages remain in arrival order.
- `internal/services/processdefinition/v87/service.go` now sends Operate sort fields `tenantId ASC`, `bpmnProcessId ASC`, `version DESC`, `key ASC` for ordinary search and normalizes `SearchProcessDefinitions` output with `domain.SortProcessDefinitionsCanonical`.
- `internal/services/processdefinition/v88/service.go` now sends Camunda v2 ordinary sort fields `tenantId ASC`, `processDefinitionId ASC`, `version DESC`, `processDefinitionKey ASC` and normalizes `SearchProcessDefinitions` output with `domain.SortProcessDefinitionsCanonical`.
- `internal/services/processdefinition/v89/service.go` now manually JSON-encodes Camunda v2 ordinary sort fields `tenantId ASC`, `processDefinitionId ASC`, `version DESC`, `processDefinitionKey ASC` and normalizes `SearchProcessDefinitions` output with `domain.SortProcessDefinitionsCanonical`.
- `internal/services/processdefinition/v810/service.go` now manually JSON-encodes Camunda v2 ordinary sort fields `tenantId ASC`, `processDefinitionId ASC`, `version DESC`, `processDefinitionKey ASC` and normalizes `SearchProcessDefinitions` output with `domain.SortProcessDefinitionsCanonical`.
- `c8volt/process/client_test.go` now includes facade coverage that `SearchProcessDefinitions` preserves the service-provided canonical key sequence while carrying each process definition's statistics through public conversion without cross-key association drift.
- `cmd/cmd_views_processdefinition_test.go` now uses a five-row canonical renderer fixture and asserts identical process-definition key sequences across list human output, JSON envelope arrays, keys-only output, and watch refresh rendering.
- `internal/services/processdefinition/search_test.go` now has watch-snapshot service coverage proving broad snapshots reuse the paged canonical collection order and that statistics-only refresh changes retain row positions while updating per-key statistics.
- `cmd/get_processdefinition_watch_test.go` now has ten-refresh command coverage proving volatile statistics count changes render in place without moving process-definition rows.
- `internal/services/processdefinition/v88/service_test.go` now compares non-stat and with-stat ordinary search over the same shuffled collection, asserting identical canonical key order and per-key statistics association after enrichment.
- `internal/services/processdefinition/v88/service.go` documents that statistics are attached to each definition struct before final canonical sorting, so sorted results preserve per-key association.
- `internal/services/processdefinition/v89/service_test.go` now compares non-stat and with-stat ordinary search over the same shuffled collection, asserting identical canonical key order and per-key statistics association after enrichment.
- `internal/services/processdefinition/v89/service.go` documents that statistics are attached to each definition struct before final canonical sorting, so sorted results preserve per-key association.
- `internal/services/processdefinition/v810/service_test.go` now compares non-stat and with-stat ordinary search over the same shuffled collection, asserting identical canonical key order and per-key statistics association after enrichment.
- `internal/services/processdefinition/v810/service.go` documents that statistics are attached to each definition struct before final canonical sorting, so sorted results preserve per-key association.
- `internal/services/processdefinition/search.go` now treats `ProcessDefinitionSearchRequest.Latest` as a shared traversal intent: it sets `IsLatestVersion` for page adapters, traverses all available pages without applying the user limit early, reduces by exact tenant/BPMN group with version-desc/key-asc winner rules, canonically sorts, then applies the latest limit.
- `CollectProcessDefinitionWatchSnapshot` now routes latest snapshots through `SearchProcessDefinitionsPages` with `Latest: true`, preserving direct-key snapshots and ordinary broad snapshot paging behavior.
- `c8volt/process.ProcessDefinitionSearchRequest` and `internal/domain.ProcessDefinitionSearchRequest` now carry additive `Latest bool`; `c8volt/process/convert.go` maps it to the domain request.
- `c8volt/process.SearchProcessDefinitionsLatest` now delegates to `pdsvc.SearchProcessDefinitionsPages` with `Latest: true`, preserving facade error conversion and returning the shared service's complete, reduced, canonically ordered result sequence.
- `internal/services/processdefinition/v87/service_test.go` now covers v8.7 canonical Operate sort on paged requests, the retained 1000-definition compatibility fetch cap, and local latest selection by exact tenant/BPMN group with lexical key tie handling.
- `internal/services/processdefinition/v87/service.go` now groups v8.7 local latest emulation by exact tenant ID plus BPMN process ID, resolves equal-version ties by lowest opaque key text, sorts selected rows canonically, and documents the 1000 visible-definition compatibility window.
- `internal/services/processdefinition/v88/service_test.go` now covers native `isLatestVersion`, tenant-then-process latest sort, cursor continuation requests, lower-bound/exact page totals, and multi-page latest metadata.
- `internal/services/processdefinition/v88/service.go` now sends latest-page sort fields `tenantId ASC`, `processDefinitionId ASC` and classifies follow-up cursor pages from native `HasMoreTotalItems` metadata instead of offset arithmetic.
- `internal/services/processdefinition/v89/service_test.go` now covers native `isLatestVersion`, tenant-then-process latest sort, cursor continuation requests, lower-bound/exact page totals, and multi-page latest metadata.
- `internal/services/processdefinition/v89/service.go` now sends latest-page sort fields `tenantId ASC`, `processDefinitionId ASC` and classifies follow-up cursor pages from native `HasMoreTotalItems` metadata instead of offset arithmetic.
- `internal/services/processdefinition/v810/service_test.go` now covers native `isLatestVersion`, tenant-then-process latest sort, cursor continuation requests, lower-bound/exact page totals, and multi-page latest metadata.
- `internal/services/processdefinition/v810/service.go` now sends latest-page sort fields `tenantId ASC`, `processDefinitionId ASC` and classifies follow-up cursor pages from native `HasMoreTotalItems` metadata instead of offset arithmetic.
- `cmd/get_processdefinition.go` now routes broad list and broad `--latest` through `SearchProcessDefinitionsPages`, preserving command page-size/progress wiring while carrying `Latest` on the facade request.
- `cmd/process_definition_selector_validation.go` now validates selectors through `SearchProcessDefinitionsPages` with `Latest` set for latest-aware checks; near-match and visible recovery listings also use the paged facade collection path.
- `cmd/get_processdefinition_test.go` now covers broad latest paged dispatch, `--batch-size` propagation, all-tenant filter clearing, and identical latest keys across page sizes 1, 2, and 1000.
- `cmd/process_definition_selector_validation_test.go` now proves latest selector validation uses the paged collection request rather than the legacy latest facade call.

## Decisions
- Treat T001 as a validation-only setup work unit; no production code changed in iteration 1.
- T002 and T003 were completed as one foundational work unit because T002's comparator tests require the T003 API to compile while quality gates require a green commit.
- T004 and T015 were paired in iteration 3 because the new service final-order test requires the service-level final normalization to pass.
- T005 was completed as a facade regression test-only work unit; no public facade implementation change was needed because conversion already preserves service slice order.
- T006 was completed as command regression coverage for tenant-filtered and `--all-tenants` broad listing; no production command change was needed because both cases already consume the shared paged collection path.
- T007 and T011 were paired in iteration 6 because the new v8.7 request-sort regression requires the adapter sort tuple and canonical result normalization to pass.
- T008 and T012 were paired in iteration 7 because the new v8.8 request-sort regression requires the adapter sort tuple and canonical result normalization to pass.
- T009 and T013 were paired in iteration 8 because the new v8.9 request-sort regression requires the adapter sort tuple and canonical result normalization to pass.
- T010 and T014 were paired in iteration 9 because the new v8.10 request-sort regression requires the adapter sort tuple and canonical result normalization to pass.
- T016 was completed as a facade regression test-only work unit; no implementation change was needed because the existing conversion maps the service slice in order and maps statistics from the same element.
- T018 was completed as a service regression test-only work unit; no production change was needed because broad watch snapshots already delegate to `SearchProcessDefinitionsPages`, whose final result is canonically sorted.
- T019 was completed as command regression test-only work; no production command change was needed because watch rendering consumes the snapshot slice in order and updates statistics in the same rows.
- T020 was completed as v8.8 adapter parity coverage plus an implementation comment; no behavior change was needed because enrichment mutates each definition before the final canonical sort.
- T021 was completed as v8.9 adapter parity coverage plus an implementation comment; no behavior change was needed because enrichment mutates each definition before the final canonical sort.
- T022 was completed as v8.10 adapter parity coverage plus an implementation comment; no behavior change was needed because enrichment mutates each definition before the final canonical sort.
- T023, T030, and T031 were completed together because the shared service tests require an additive latest request intent and the service-owned latest traversal/reduction implementation to pass.
- T024 and T032 were paired because the facade latest regression tests intentionally reject the old direct `SearchProcessDefinitionsLatest` service call and require the shared paged latest traversal to pass.
- T025 and T033 were paired because the new v8.7 latest grouping regression requires the local compatibility selector implementation to pass.
- T026 and T034 were paired because the new v8.8 latest page regression requires tenant/process native sort order and cursor continuation metadata changes to pass.
- T027 and T035 were paired because the new v8.9 latest page regression requires tenant/process native sort order and cursor continuation metadata changes to pass.
- T028 and T036 were paired because the new v8.10 latest page regression requires tenant/process native sort order and cursor continuation metadata changes to pass.
- T029 and T037 were paired because the new CLI latest and selector tests require broad `--latest` and selector validation to share the paged canonical facade collection path.

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
- US3 is complete. Next iteration should start Phase 6 at T038: update canonical-order wording, latest grouping, exact comparison rules, and the Camunda 8.7 compatibility note in `cmd/get_processdefinition.go` and `README.md`; after source wording changes, T039 should regenerate CLI docs with `make docs-content`.
