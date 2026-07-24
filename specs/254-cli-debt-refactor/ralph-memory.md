# Ralph Memory

Feature: 254-cli-debt-refactor
Started: 2026-07-24T04:28:25Z

## Codebase Patterns
- Basic read searches in `cmd/get_job_search.go`, `cmd/get_element_search.go`, `cmd/get_incident_search.go`, and `cmd/get_processinstance_search.go` currently own page walking, limit trimming, prompt/auto-continue decisions, incremental rendering, and total fallback. Job/element are the simplest duplicated offset loops; incident adds cursor fallback and process-instance-key projection; process-instance adds local filtering, direct incident-index strategy, enrichment boundaries, and warning-stop progress.
- Process-instance cancel/delete share `processPISearchPagesWithAction` for command-level search-selected paging, but delete has a separate non-dry-run planning pass that freezes the full delete scope before one confirmation and mutation. Preserve that safety difference.
- Ops workflows under `internal/services/ops` already own frozen discovery, planning, and execution for repair, purge, retention, smoke test, and slow analysis. Similar loops carry workflow-specific counts and safety semantics, so do not replace them with a generic ops discovery abstraction without matching behavior.
- Root/activity/capability plumbing: `cmd/root.go` installs `toolx/logging.ActivityWriter` on stderr through context; `cmd/command_contract.go` derives capability metadata from Cobra annotations; ops renderers show user-limited discovery in compact output and hide complete page details unless verbose.
- US1 progress policy lives in `specs/254-cli-debt-refactor/progress-policy.md`. Basic paged read progress uses `page size`, `current page`, `total so far`, `more matches`, and `next step`; it is printed to stderr only for `--verbose` one-line output and suppressed for JSON, keys-only, quiet, and non-verbose modes.

## Decisions
- Phase 1 found no conflict between `specs/ralph-implementation-rules.md` and `specs/254-cli-debt-refactor/spec.md`.
- Phase 2 checked in the full 55-command assessment table in `specs/254-cli-debt-refactor/assessment.md` with all required columns and validation tests.
- The command tree count is guarded in `cmd/command_contract_test.go` by comparing the live capability tree to assessment rows; docsgen also validates assessment sections and row count.
- US1 added a shared basic-search progress formatter in `cmd/get_processinstance_paging.go` and wired it through job, element, and incident search loops. Process-instance progress now also respects `--quiet`.
- Ops discovery status rendering is centralized in `cmd/cmd_views_ops_processinstance_scope.go`; user-limited discovery stays visible in compact output and complete discovery page details remain verbose-only.
- US2 T026 added job facade and v8.8 service tests around service-owned job page collection. v8.9 already had equivalent service coverage; command-owned job paging remains for interactive prompts and verbose progress until the broader job ownership slice decides how to preserve that output contract.
- US2 T027 added element facade and v8.8 service tests around service-owned element page collection, offset traversal, limit-capped page sizing, and result mapping.
- US2 T028 added incident facade and v8.8 service tests around local-filter page collection and caller-cap trimming. Incident facade paging currently triggers through `internal/services/incident.SearchIncidents` for error-message and creation-time filters, while v8.8 service-owned collection handles broader local compatibility filters below the facade.
- US2 T029-T030 added process-instance facade/v8.8 service tests for paged search metadata, cursor paging, parent/incident compatibility filter forwarding, and shared dry-run planner option forwarding. Delete search-page tests now pin the frozen-scope safety contract: all page plans are collected before one confirmation and one aggregate delete call.
- US2 T031-T033 added job paged-search visitor/result contracts across `internal/domain/job.go` and `c8volt/job/model.go`, moved job offset advancement, page-size capping, user-limit trimming, and total fallback into `internal/services/job/v88` and `v89`, and reduced `cmd/get_job_search.go` to rendering, verbose progress, and prompt decisions through `SearchJobsPages`.
- US2 T034-T036 added element paged-search visitor/result contracts across `internal/domain/element.go` and `c8volt/element/model.go`, moved element offset advancement, page-size capping, user-limit trimming, and total fallback into `internal/services/element/v88` and `v89`, kept v8.7 explicitly unsupported, and reduced `cmd/get_element_search.go` to rendering, verbose progress, and prompt decisions through `SearchElementsPages`.
- US2 T037-T039 added incident paged-search visitor/result contracts across `internal/domain/incident.go` and `c8volt/incident/model.go`, moved incident page advancement, cursor-aware request building, caller-limit trimming, local compatibility filtering, and total fallback into `internal/services/incident/v88` and `v89`, kept v8.7 explicitly unsupported, and reduced `cmd/get_incident_search.go` to rendering, verbose progress, process-instance-key output, and prompt decisions through `SearchIncidentsPages`.
- US2 T040-T041 added process-instance paged-search visitor/result and total contracts across `internal/domain/processinstance.go` and `c8volt/process/model.go`, moved process-instance page traversal, user-limit trimming, local relationship/incident compatibility filtering, direct incident-index collection, and total fallback into `internal/services/processinstance.SearchProcessInstancesPages` / `SearchProcessInstancesTotal`, and reduced `cmd/get_processinstance_search.go` plus `cmd/get_processinstance_total.go` to request construction, rendering, progress, and prompt decisions.
- US2 T042-T045 added process-instance mutation planning page contracts across `internal/domain/processinstance.go` and `c8volt/process/model.go`, moved search-selected cancel/delete page traversal, limit trimming, and page-level dry-run dependency planning into `internal/services/processinstance.PlanProcessInstanceMutationPages`, exposed it through `c8volt/process.PlanProcessInstanceMutationPages`, removed the old command-owned `processPISearchPagesWithAction` loop, and updated the assessment/quickstart after `go test ./cmd ./c8volt/job ./c8volt/element ./c8volt/incident ./c8volt/process -count=1` passed.
- US3 T046/T052 added fake-latency facade tests for process-instance incident enrichment and traversal incident-detail lookup, then moved those two independent incident lookup paths to `toolx/pool.ExecuteSlice` in `internal/services/processinstance/enrichment.go`. Result ordering remains deterministic because pool result slots follow input order; lookup call order is no longer deterministic.
- US3 T047 added internal fake-latency coverage in `internal/services/processinstance/dryrun_test.go` for direct dry-run dependency expansion and search-selected mutation page planning. The tests pin concurrent ancestry/descendant traversal through the existing bounded pool path and confirm option pass-through plus deterministic plan ordering.

## Gotchas
- `delete process-instance` search mode intentionally plans all selected pages before confirmed mutation; treating it like page-by-page cancel would weaken frozen-scope confirmation behavior.
- `get process-instance --direct-incidents-only` still only takes the direct incident-index strategy when command mode selection marks it compatible (`--limit`, incident detail filters, and no conflicting enrichment/filter flags); the incident-page traversal and unique process-instance key collection now live in `internal/services/processinstance/search.go`.
- Do not print basic-search verbose progress for JSON or keys-only modes; machine-output tests use job search as the representative paged-read path.
- Incident v8.8 still sends only tenant-safe top-level filters and applies richer compatibility filters locally; v8.9 sends safe server filters and still locally filters root-process-instance-key and error-message semantics.
- Search-selected `cancel process-instance` still mutates page by page after the preserved first-page confirmation; `delete process-instance` still freezes every selected page-level plan before one aggregate confirmation and delete call. Both now get page plans from `PlanProcessInstanceMutationPages`.
- Process-instance incident enrichment and traversal incident enrichment now perform independent lookups concurrently. Tests that observe lookup calls must use synchronized collectors such as `testx.SafeSlice` and should assert set membership rather than call order.
- Dry-run process-instance dependency traversal tests deliberately block until two worker callbacks overlap; keep callback assertions as returned errors rather than `require` calls inside worker goroutines.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd -run 'TestCommandContract|TestCapability' -count=1`
- `go test ./cmd -run 'TestCapabilityDocumentForRoot_CoversCLIDebtAssessment' -count=1`
- `go test ./docsgen -run 'TestCLIDebtRefactorAssessmentArtifactDocumentsBaseline' -count=1`
- `go test ./cmd ./toolx/logging -run 'TestGet(Job|Element|Incident|ProcessInstance).*Progress|TestPagedSearchMachineOutputCleanliness|TestOpsPurge.*Discovery|TestRenderOpsRepair.*Discovery|TestIndicatorEnabled|TestActivityWriter_DisabledSuppressesActivityOutput' -count=1`
- `go test ./cmd -run 'TestGet(Job|Element|Incident|ProcessInstance)|Test.*JSON|Test.*KeysOnly|Test.*Automation|Test.*NoIndicator|Test.*Prompt' -count=1`
- `go test ./toolx/logging -count=1`
- `go test ./c8volt/job ./internal/services/job/... -count=1`
- `go test ./cmd ./c8volt/process ./internal/services/processinstance/v88 -count=1`
- `go test ./cmd ./c8volt/incident ./c8volt/process ./internal/services/incident/... ./internal/services/ops -count=1`
- `go test ./c8volt/process -run 'TestClient_Enrich(ProcessInstances|Traversal)WithIncidents' -count=1`
- `go test ./internal/services/processinstance -run 'TestEnrich.*Incidents' -count=1`
- `go test -race ./c8volt/process ./internal/services/processinstance -count=1`
- `go test ./c8volt/process -run 'Test.*(Latency|Concurrent|Performance|HighVolume|Workers)' -count=1`
- `go test ./internal/services/processinstance -run 'Test(DryRunCancelOrDeletePlan|PlanProcessInstanceMutationPages).*(Workers|Dependency|Planning|Concurrent|Latency)' -count=1`
- `go test -race ./internal/services/processinstance -run 'Test(DryRunCancelOrDeletePlan|PlanProcessInstanceMutationPages).*(Workers|Dependency|Planning|Concurrent|Latency)' -count=1`
- `git diff --check`

## Do Not Repeat
- Do not infer that similar ops discovery loops are equivalent until the candidate counts, frozen scope, report fields, force behavior, and confirmation prompt semantics are compared.

## Current Handoff
- Continue Phase 5 / US3 at T048. T047 is complete with `internal/services/processinstance/dryrun_test.go` coverage for dry-run dependency planning; T053 remains unmarked until the implementation task is addressed or explicitly reconciled against the existing bounded pool code. Do not start US4 or polish.
