# Progress: Ops Execute Retention Policy

**Issue**: [#187](https://github.com/grafvonb/c8volt/issues/187)
**Feature**: `187-ops-retention-policy`
**Plan**: [plan.md](plan.md)
**Tasks**: [tasks.md](tasks.md)
**Mandatory Ralph Context**: `specs/ralph-implementation-rules.md`
**Commit Rule**: Conventional Commit subject ending in `#187`

## Codebase Patterns

- Retention policy foundation now mirrors #186 boundaries: domain models in `internal/domain/ops_retention_policy.go`, public models/API/conversions in `c8volt/ops`, and a validation-only service seam in `internal/services/ops/retention_policy.go`.
- Foundational retention service validation rejects negative retention days and explicit process-instance key selection with `domain.ErrValidation`; the public ops facade maps those through `ferrors.FromDomain` to invalid-input behavior.
- Mandatory Ralph implementation context is `specs/ralph-implementation-rules.md`; commit subjects for this feature must use Conventional Commits and end with `#187`.
- `ops execute` is currently a grouping command in `cmd/ops_execute.go`; it is state-changing metadata only, renders help from `RunE`, and currently has no child workflow commands.
- Existing #186 ops purge command wiring lives in `cmd/ops_purge_orphan_processinstances.go`: command files own Cobra flags, validation, `NewCli`, automation support checks, report-path planning validation, confirmation, facade calls, report writing, and final rendering.
- Ops workflow command metadata uses `setCommandMutation`, `setContractSupport`, `setAutomationSupport`, and `setFlagContractRequired` from `cmd/command_contract.go`; capability regression tests live in `cmd/command_contract_test.go` and `cmd/capabilities_test.go`.
- Shared ops report helpers in `cmd/ops_contract.go` own report format inference, validation, overwrite mode, and secure file writes; workflow-specific Markdown and JSON rendering currently lives beside the workflow view in `cmd/cmd_views_ops_purge_orphan_processinstances.go`.
- Human ops output follows a compact sequence of workflow status lines and uses `renderSucceededResult` for JSON payloads; detailed key lists are gated by verbose output while `keys-only` prints one key per line.
- Public ops facade additions should mirror the #186 pattern: exported models in `c8volt/ops/model.go`, API method in `c8volt/ops/api.go`, conversions in `c8volt/ops/convert.go`, thin delegation plus `ferrors.FromDomain` in `c8volt/ops/client.go`, and boundary tests in `c8volt/ops/client_test.go`.
- Internal ops service additions should mirror `internal/services/ops/orphan_purge.go`: create a result with schema/report metadata, discover or reuse a frozen key set, build delete plan, skip mutation on dry-run/no targets, block non-final affected scope unless force permits it, call process-instance deletion helpers, and finish report timestamps/duration/errors in one place.
- Process-instance discovery primitives own paging and limits in `internal/services/processinstance`; `DiscoverOrphanProcessInstances` shows the pattern for service-owned search loops, batch-size normalization, limit enforcement, cursor/from pagination, and stable key uniqueness.
- Existing delete planning is `processinstance.DryRunCancelOrDeletePlan`, which deduplicates selected keys, resolves roots through ancestry traversal, expands descendants, preserves missing ancestor/traversal warnings, records selected final-state items and non-final affected items, and validates unresolved plans.
- Existing delete execution uses `processinstance.DeleteProcessInstances`, `toolx.DetermineNoOfWorkers`, `toolx/pool.ExecuteSlice`, `services.ApplyCallOptions`, `FailFast`, `NoWorkerLimit`, `NoWait`, and logger/activity helpers; command code should pass controls rather than owning worker logic.
- Process-instance search flag conversion is centralized in `cmd/get_processinstance_filtering.go`; `populatePISearchFilterOpts` resolves relative-day flags to concrete date bounds and preserves local fallback filters for roots/children/incidents.
- Generated CLI docs are protected by `docsgen/main_test.go` and produced via `make docs-content`; generated files under `docs/cli/` and `docs/index.md` should not be hand-edited after command metadata changes.

## Status

- Speckit specification created from GitHub issue #187.
- Clarification gate completed with no critical ambiguities worth formal questioning.
- Planning artifacts generated for Ralph-sized implementation work.
- Phase 1 setup and Phase 2 foundational model/facade/service tasks are complete.

## Ralph Notes

- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, `spec.md`, `plan.md`, `tasks.md`, and this progress file before code changes.
- Each iteration should complete only the first incomplete work unit and update this file with validation results.
- Existing #186 ops purge/report code is expected to be the closest implementation pattern.

---
## Iteration 1 - 2026-05-14 12:29:00 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Record mandatory Ralph context and issue traceability in `specs/187-ops-retention-policy/progress.md`
- [x] T002: Inspect existing #186 ops purge implementation, `ops execute` command group, process-instance search, process-instance delete planning, command contract metadata, and docs generation patterns; record reusable discoveries in `specs/187-ops-retention-policy/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/187-ops-retention-policy/progress.md
- specs/187-ops-retention-policy/tasks.md
**Learnings**:
- Phase 1 is a documentation/setup work unit only; no Go source changes were needed.
- Issue traceability is persisted in `spec.md`, `plan.md`, `tasks.md`, and this progress file as `#187`.
- The next iteration should begin with Phase 2 foundational model/facade/service boundary tasks and should not start US1 until Phase 2 is complete.
---
---
## Iteration 2 - 2026-05-14 12:36:33 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T003: Define internal retention request/result domain models in `internal/domain/ops_retention_policy.go`
- [x] T004: Define public ops retention request/result models in `c8volt/ops/model.go`
- [x] T005: Extend public ops facade API for retention policy in `c8volt/ops/api.go`
- [x] T006: Extend internal ops service interface for retention policy in `internal/services/ops/api.go`
- [x] T007: Implement public/internal retention model conversions in `c8volt/ops/convert.go`
- [x] T008: Implement thin public ops facade retention method in `c8volt/ops/client.go`
- [x] T009: Add foundational ops facade wiring tests for retention policy in `c8volt/ops/client_test.go`
- [x] T010: Add foundational internal ops service tests for retention policy request validation in `internal/services/ops/retention_policy_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/ops/api.go
- c8volt/ops/client.go
- c8volt/ops/client_test.go
- c8volt/ops/convert.go
- c8volt/ops/model.go
- internal/domain/ops_retention_policy.go
- internal/services/ops/api.go
- internal/services/ops/retention_policy.go
- internal/services/ops/retention_policy_test.go
- specs/187-ops-retention-policy/progress.md
- specs/187-ops-retention-policy/tasks.md
**Learnings**:
- Retention foundation is intentionally validation-only; discovery and mutation remain unimplemented for later user-story work units.
- Targeted validation passed: `go test ./c8volt/ops -count=1`, `go test ./internal/services/ops -count=1`, `go test ./c8volt -count=1`, and `go test ./cmd -run 'TestCommandCapability|TestOpsPurge|TestRoot|TestNewCli' -count=1`.
- The next iteration can begin US1 command registration now that the retention model, facade, and service seam exist.
---
