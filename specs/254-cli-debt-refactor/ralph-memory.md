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

## Gotchas
- `delete process-instance` search mode intentionally plans all selected pages before confirmed mutation; treating it like page-by-page cancel would weaken frozen-scope confirmation behavior.
- `get process-instance --direct-incidents-only` is a command-owned alternate query strategy that only applies with `--limit` and compatible incident filters.
- Do not print basic-search verbose progress for JSON or keys-only modes; machine-output tests use job search as the representative paged-read path.
- Incident v8.8 still sends only tenant-safe top-level filters and applies richer compatibility filters locally; v8.9 sends safe server filters and still locally filters root-process-instance-key and error-message semantics.

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
- `git diff --check`

## Do Not Repeat
- Do not infer that similar ops discovery loops are equivalent until the candidate counts, frozen scope, report fields, force behavior, and confirmation prompt semantics are compared.

## Current Handoff
- Continue Phase 4 / US2 at T040. Start the process-instance search ownership slice by moving query strategy, page traversal, total fallback, and local compatibility filtering below command ownership, while preserving `--direct-incidents-only`, enrichment boundaries, warning-stop behavior, and machine-output silence.
