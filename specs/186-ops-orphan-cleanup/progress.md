# Progress: Ops Purge Orphan Process Instances

**Issue**: https://github.com/grafvonb/c8volt/issues/186  
**Feature**: `186-ops-orphan-cleanup`  
**Mandatory Implementation Context**: `specs/ralph-implementation-rules.md`

## Ralph Rules

- Every Ralph launch must include `--implementation-context specs/ralph-implementation-rules.md`.
- Every implementation iteration must read and apply `specs/ralph-implementation-rules.md`.
- Each iteration must complete only the current Ralph work unit.
- Do not stage or commit unless the Ralph workflow reaches its commit step and validation passes.
- Commit subjects must follow Conventional Commits and end with `#186`.

## Codebase Patterns

- Ops purge audit reports are written by command-layer helpers after the facade returns its structured report; the command enriches the report with CLI build/config identity and writes requested files before rendering normal stdout.
- Automation-safe destructive ops commands should plan/discover first, then block `--automation` without `--auto-confirm` before confirmation or mutation when concrete targets exist; zero-target and dry-run flows remain allowed.
- Confirmed orphan purge keeps discovery immutable by allowing the command to pass a frozen `DiscoveredKeys` set into the ops facade after an interactive pre-plan; the service skips rediscovery when that set is present.
- `internal/services/ops/orphan_purge.go` now submits deletion through `internal/services/processinstance.DeleteProcessInstances`, preserving worker/fail-fast/no-wait/force option behavior and using the dry-run expansion as the mutation scope.
- `cmd/ops_purge_orphan_processinstances.go` reuses process-instance delete plan validation (`rejectDeletePlanRequiringForce`) and root confirmation helpers for non-`--auto-confirm` destructive runs; `--no-wait` and `--force` are exposed for parity with the delete path.
- Human rendering for orphan purge now distinguishes dry-run previews, confirmed no-op runs, submitted deletion requests, and deleted outcomes.
- `internal/services/processinstance/orphan_discovery.go` now owns paged orphan-child discovery for ops workflows: it forces child-only selection with `HasParent=true`, applies the caller's compatible filters, honors batch size/limit, and delegates parent existence checks to `FilterProcessInstanceWithOrphanParent`.
- `internal/services/ops/orphan_purge.go` orchestrates purge discovery, delete-plan validation, dry-run skipping, and confirmed deletion while keeping resource-specific traversal and deletion mechanics in process-instance services.
- `cmd/ops_purge_orphan_processinstances.go` uses the public ops facade rather than process command helpers directly; dry-run and confirmed purge share one request/result model.
- Adding a concrete child under a discovery-only ops grouping command makes the parent capability contract `limited` while the grouping command itself remains automation-unsupported.
- `cmd/ops.go`, `cmd/ops_execute.go`, and `cmd/ops_repair.go` already define discovery-only grouping commands and shared ops foundation from issue #197.
- This feature should add `cmd/ops_purge.go` for destructive cleanup workflows while preserving `ops execute` for non-purge playbooks such as smoke tests.
- `cmd/ops_contract.go` already defines shared ops workflow step statuses and report-format primitives.
- `cmd/ops_contract_test.go` protects the shared ops step status vocabulary and report-format inference; extend it when report behavior changes.
- `cmd/get_processinstance_filtering.go` owns process-instance search filter conversion and local orphan-child filtering through `FilterProcessInstanceWithOrphanParent`.
- `cmd/get_processinstance_paging.go` owns shared process-instance search paging, limits, continuation states, progress output, and automation-aware continuation behavior.
- `cmd/delete_processinstance.go` owns existing process-instance delete dry-run planning, destructive confirmation, and deletion submission through the process facade.
- `deleteProcessInstancesWithPlanAndRender` can validate/delete a frozen key set while deferring dry-run rendering, which matches the orphan purge plan orchestration need.
- `c8volt/process/dryrun.go` exposes thin facade methods over `internal/services/processinstance` delete/cancel dry-run planning.
- `internal/services/processinstance` owns version-neutral process-instance service contracts and versioned service implementations.
- No `c8volt/ops` or `internal/services/ops` packages exist yet; the foundational work unit must create them and wire `c8volt/client.go` plus `c8volt/contract.go`.
- `cmd/command_contract.go` records mutation, contract, automation, output-mode, and required-flag metadata through Cobra annotations consumed by capabilities tests.
- Generated CLI docs live under `docs/cli/` and must be refreshed through `make docs-content`.
- `internal/domain/ops_orphan_purge.go` owns the version-neutral orphan purge request, result, step, outcome, and audit report models for this feature.
- `internal/services/ops` is the workflow service boundary over process-instance primitives; the foundational implementation preserves the request and returns unsupported until story tasks add orchestration.
- `c8volt/ops` is the public facade boundary and is embedded into the top-level `c8volt.API` through `c8volt/client.go` and `c8volt/contract.go`.

## Validation Log

- Planning artifacts created on 2026-05-11.

---
## Iteration 1 - 2026-05-11 20:40:24 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Record mandatory Ralph context and issue traceability in `specs/186-ops-orphan-cleanup/progress.md`
- [x] T002: Inspect existing ops foundation from issue #197, process-instance search, process-instance delete, command contract, and docs generation patterns; record reusable discoveries in `specs/186-ops-orphan-cleanup/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/186-ops-orphan-cleanup/tasks.md
- specs/186-ops-orphan-cleanup/progress.md
**Learnings**:
- Issue traceability is explicit in `spec.md`, `plan.md`, and `tasks.md`; every commit subject for this feature must end with `#186`.
- The current work unit is tracking/setup only; no production code changes were needed before foundational ops APIs.
- Existing process-instance delete helpers already support dry-run planning, confirmation, and mutation against an immutable key set, so later orphan purge work should reuse those semantics instead of duplicating delete mechanics.
- Validation passed with `git diff --check` and `go test ./cmd -run 'TestOps|TestCommandContract' -count=1`.
---

---
## Iteration 2 - 2026-05-11 20:47:26 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T003: Define internal orphan-purge request/result domain models in `internal/domain/ops_orphan_purge.go`
- [x] T004: Define public ops facade request/result models in `c8volt/ops/model.go`
- [x] T005: Define public ops facade API in `c8volt/ops/api.go`
- [x] T006: Define internal ops service interface and constructor in `internal/services/ops/api.go`
- [x] T007: Implement public ops facade conversions in `c8volt/ops/convert.go`
- [x] T008: Implement thin public ops facade client in `c8volt/ops/client.go`
- [x] T009: Wire ops facade creation and API embedding in `c8volt/client.go` and `c8volt/contract.go`
- [x] T010: Add foundational ops facade wiring tests in `c8volt/ops/client_test.go`
- [x] T011: Add foundational internal ops service tests in `internal/services/ops/orphan_purge_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- internal/domain/ops_orphan_purge.go
- internal/services/ops/api.go
- internal/services/ops/orphan_purge_test.go
- c8volt/ops/api.go
- c8volt/ops/model.go
- c8volt/ops/convert.go
- c8volt/ops/client.go
- c8volt/ops/client_test.go
- c8volt/client.go
- c8volt/client_test.go
- c8volt/contract.go
- specs/186-ops-orphan-cleanup/tasks.md
- specs/186-ops-orphan-cleanup/progress.md
**Learnings**:
- The current iteration intentionally adds only the ops boundary and unsupported placeholder behavior; `ops purge` command behavior starts with US1.
- The public facade maps process-instance filter and dry-run plan fields explicitly because process facade conversion helpers are package-private.
- Validation passed with `go test ./c8volt/ops ./internal/services/ops ./c8volt -count=1`, `go test ./internal/domain -count=1`, and `go test ./cmd -run 'TestOps|TestCommandContract' -count=1`.
---

---
## Iteration 3 - 2026-05-11 20:59:21 CEST
**User Story**: User Story 1 - Preview Orphan Cleanup Safely
**Tasks Completed**:
- [x] T012: Add dry-run command tests for discovered orphan keys and no delete request in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T013: Add no-target dry-run command test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T014: Add compatible filter narrowing test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T015: Add orphan discovery and delete-plan service tests in `internal/services/ops/orphan_purge_test.go`
- [x] T016: Add process-instance orphan discovery primitive or reuse wrapper in `internal/services/processinstance/orphan_discovery.go`
- [x] T017: Implement dry-run discovery and plan orchestration in `internal/services/ops/orphan_purge.go`
- [x] T018: Map dry-run ops facade inputs and outputs in `c8volt/ops/client.go`
- [x] T019: Add `ops purge` grouping command in `cmd/ops_purge.go` and `ops purge orphan-process-instances` Cobra command, dry-run flag handling, and compatible selection flags in `cmd/ops_purge_orphan_processinstances.go`
- [x] T020: Add human and JSON rendering for dry-run purge results in `cmd/cmd_views_ops_purge_orphan_processinstances.go`
- [x] T021: Mark US1 tasks complete and record validation notes in `specs/186-ops-orphan-cleanup/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- internal/domain/ops_orphan_purge.go
- internal/services/processinstance/orphan_discovery.go
- internal/services/ops/api.go
- internal/services/ops/orphan_purge.go
- internal/services/ops/orphan_purge_test.go
- c8volt/ops/model.go
- c8volt/ops/convert.go
- c8volt/ops/client_test.go
- cmd/ops_purge.go
- cmd/ops_purge_orphan_processinstances.go
- cmd/cmd_views_ops_purge_orphan_processinstances.go
- cmd/ops_purge_orphan_processinstances_test.go
- cmd/capabilities_test.go
- specs/186-ops-orphan-cleanup/tasks.md
- specs/186-ops-orphan-cleanup/progress.md
**Learnings**:
- Dry-run purge can compose existing process-instance primitives without shelling out or duplicating delete mechanics: discover once, freeze the orphan key set, and validate through `DryRunCancelOrDeletePlan`.
- The command package test suite must use `GOCACHE=/tmp/go-build-cache` in this sandbox; the default macOS Go cache path can be unreadable.
- `go test ./c8volt ./c8volt/ops ./internal/services/ops -count=1` is blocked by the existing `TestNew_V89WiresSupportedRuntime` localhost connection under this sandbox, but the touched package pass ran successfully.
- Validation passed with `GOCACHE=/tmp/go-build-cache go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance ./internal/domain -count=1` and `git diff --check`.
---

---
## Iteration 4 - 2026-05-11 21:21:48 CEST
**User Story**: User Story 2 - Run Confirmed Orphan Purge
**Tasks Completed**:
- [x] T022: Add confirmed cleanup command test for exact discovered-key deletion in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T023: Add immutable discovered-set service test in `internal/services/ops/orphan_purge_test.go`
- [x] T024: Add no-target confirmed cleanup test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T025: Add process-instance delete delegation test in `c8volt/ops/client_test.go`
- [x] T026: Implement confirmed delete orchestration through existing process-instance delete behavior in `internal/services/ops/orphan_purge.go`
- [x] T027: Reuse destructive confirmation and delete-plan validation from process-instance command helpers in `cmd/ops_purge_orphan_processinstances.go`
- [x] T028: Render deletion execution and final outcome in `cmd/cmd_views_ops_purge_orphan_processinstances.go`
- [x] T029: Mark US2 tasks complete and record validation notes in `specs/186-ops-orphan-cleanup/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- internal/domain/ops_orphan_purge.go
- internal/services/ops/orphan_purge.go
- internal/services/ops/orphan_purge_test.go
- c8volt/ops/model.go
- c8volt/ops/convert.go
- c8volt/ops/client_test.go
- cmd/ops_purge_orphan_processinstances.go
- cmd/cmd_views_ops_purge_orphan_processinstances.go
- cmd/ops_purge_orphan_processinstances_test.go
- specs/186-ops-orphan-cleanup/tasks.md
- specs/186-ops-orphan-cleanup/progress.md
**Learnings**:
- Confirmed deletion can reuse the process-instance bulk delete service once the orphan key set is frozen and the dry-run dependency expansion has validated the mutation scope.
- The command can preserve interactive confirmation semantics by planning with dry-run first, validating with existing delete-plan helpers, then passing the frozen discovered keys into the confirmed service call.
- Validation passed with `GOCACHE=/tmp/go-build-cache go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`; final iteration validation runs after this progress entry.
---

---
## Iteration 5 - 2026-05-11 21:27:08 CEST
**User Story**: User Story 3 - Run Cleanup In Automation
**Tasks Completed**:
- [x] T030: Add automation-without-auto-confirm pre-mutation failure test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T031: Add `--automation --json --auto-confirm` deterministic stdout test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T032: Add command contract metadata test for state-changing and automation support in `cmd/command_contract_test.go`
- [x] T033: Add automation validation and pre-mutation guard in `cmd/ops_purge_orphan_processinstances.go`
- [x] T034: Set mutation, contract, output-mode, and automation metadata in `cmd/ops_purge_orphan_processinstances.go`
- [x] T035: Ensure JSON rendering uses existing shared result envelope in `cmd/cmd_views_ops_purge_orphan_processinstances.go`
- [x] T036: Mark US3 tasks complete and record validation notes in `specs/186-ops-orphan-cleanup/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/command_contract_test.go
- cmd/ops_purge_orphan_processinstances.go
- cmd/ops_purge_orphan_processinstances_test.go
- specs/186-ops-orphan-cleanup/tasks.md
- specs/186-ops-orphan-cleanup/progress.md
**Learnings**:
- `--automation` remains an automation-mode signal but is not destructive confirmation for orphan purge when targets are found; the command now requires explicit `--auto-confirm` before mutation.
- JSON rendering for orphan purge already uses the shared result envelope through `renderSucceededResult`; US3 coverage now protects the envelope shape for `--automation --json --auto-confirm`.
- Validation passed with `GOCACHE=/tmp/go-build-cache go test ./cmd -run 'TestOpsPurgeOrphanProcessInstances|TestCommandCapabilityForCommand_OpsPurgeOrphanProcessInstancesContract|TestCapabilitiesCommand_OpsCommandFamilyMetadata' -count=1`, `GOCACHE=/tmp/go-build-cache go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`, and `git diff --check`.
---

---
## Iteration 6 - 2026-05-11 21:34:39 CEST
**User Story**: User Story 4 - Produce Audit Reports
**Tasks Completed**:
- [x] T037: Add report format inference and validation tests in `cmd/ops_contract_test.go`
- [x] T038: Add Markdown report rendering test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T039: Add JSON report rendering test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T040: Add post-discovery failure report-write test in `cmd/ops_purge_orphan_processinstances_test.go`
- [x] T041: Extend stable ops audit report model in `cmd/ops_contract.go`
- [x] T042: Implement Markdown and JSON report renderers in `cmd/cmd_views_ops_purge_orphan_processinstances.go`
- [x] T043: Implement `--report-file` and `--report-format` command flags and write path in `cmd/ops_purge_orphan_processinstances.go`
- [x] T044: Mark US4 tasks complete and record validation notes in `specs/186-ops-orphan-cleanup/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/ops_contract.go
- cmd/ops_contract_test.go
- cmd/ops_purge_orphan_processinstances.go
- cmd/cmd_views_ops_purge_orphan_processinstances.go
- cmd/ops_purge_orphan_processinstances_test.go
- cmd/get_processinstance_test.go
- specs/186-ops-orphan-cleanup/tasks.md
- specs/186-ops-orphan-cleanup/progress.md
**Learnings**:
- Report format inference now matches the feature contract: explicit `json` or `markdown` wins, `.json` infers JSON, `.md`/`.markdown` infer Markdown, and unknown extensions default to Markdown.
- The command can still write a requested audit report when failures occur after discovery by using the partial facade result and marking local pre-mutation failures as blocked or confirmation_failed.
- Validation passed with `GOCACHE=/tmp/go-build-cache go test ./cmd -run 'TestOpsWorkflowReportFormatForPath|TestOpsPurgeOrphanProcessInstances.*Report|TestOpsPurgeOrphanProcessInstancesWritesReportAfterPostDiscoveryFailure' -count=1`, `GOCACHE=/tmp/go-build-cache go test ./cmd ./c8volt/ops ./internal/services/ops -count=1`, and `git diff --check`.
---
