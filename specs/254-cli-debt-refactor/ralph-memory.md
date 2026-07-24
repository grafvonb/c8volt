# Ralph Memory

Feature: 254-cli-debt-refactor
Started: 2026-07-24T04:28:25Z

## Codebase Patterns
- Basic read searches in `cmd/get_job_search.go`, `cmd/get_element_search.go`, `cmd/get_incident_search.go`, and `cmd/get_processinstance_search.go` currently own page walking, limit trimming, prompt/auto-continue decisions, incremental rendering, and total fallback. Job/element are the simplest duplicated offset loops; incident adds cursor fallback and process-instance-key projection; process-instance adds local filtering, direct incident-index strategy, enrichment boundaries, and warning-stop progress.
- Process-instance cancel/delete share `processPISearchPagesWithAction` for command-level search-selected paging, but delete has a separate non-dry-run planning pass that freezes the full delete scope before one confirmation and mutation. Preserve that safety difference.
- Ops workflows under `internal/services/ops` already own frozen discovery, planning, and execution for repair, purge, retention, smoke test, and slow analysis. Similar loops carry workflow-specific counts and safety semantics, so do not replace them with a generic ops discovery abstraction without matching behavior.
- Root/activity/capability plumbing: `cmd/root.go` installs `toolx/logging.ActivityWriter` on stderr through context; `cmd/command_contract.go` derives capability metadata from Cobra annotations; ops renderers show user-limited discovery in compact output and hide complete page details unless verbose.

## Decisions
- Phase 1 found no conflict between `specs/ralph-implementation-rules.md` and `specs/254-cli-debt-refactor/spec.md`.
- Phase 2 checked in the full 55-command assessment table in `specs/254-cli-debt-refactor/assessment.md` with all required columns and validation tests.
- The command tree count is guarded in `cmd/command_contract_test.go` by comparing the live capability tree to assessment rows; docsgen also validates assessment sections and row count.

## Gotchas
- `delete process-instance` search mode intentionally plans all selected pages before confirmed mutation; treating it like page-by-page cancel would weaken frozen-scope confirmation behavior.
- `get process-instance --direct-incidents-only` is a command-owned alternate query strategy that only applies with `--limit` and compatible incident filters.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd -run 'TestCommandContract|TestCapability' -count=1`
- `go test ./cmd -run 'TestCapabilityDocumentForRoot_CoversCLIDebtAssessment' -count=1`
- `go test ./docsgen -run 'TestCLIDebtRefactorAssessmentArtifactDocumentsBaseline' -count=1`
- `git diff --check`

## Do Not Repeat
- Do not infer that similar ops discovery loops are equivalent until the candidate counts, frozen scope, report fields, force behavior, and confirmation prompt semantics are compared.

## Current Handoff
- Next iteration should start Phase 3 / US1 at T013-T017 test work. Use `specs/254-cli-debt-refactor/assessment.md` as the baseline for output/progress behavior and keep machine-output silence for JSON, keys-only, quiet, automation, and no-indicator modes.
