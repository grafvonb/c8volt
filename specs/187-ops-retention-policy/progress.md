# Progress: Ops Execute Retention Policy

**Issue**: [#187](https://github.com/grafvonb/c8volt/issues/187)
**Feature**: `187-ops-retention-policy`
**Plan**: [plan.md](plan.md)
**Tasks**: [tasks.md](tasks.md)
**Mandatory Ralph Context**: `specs/ralph-implementation-rules.md`
**Commit Rule**: Conventional Commit subject ending in `#187`

## Codebase Patterns

- Retention audit reporting now mirrors the #186 orphan purge report lifecycle: command flags reuse shared ops report format/path/write helpers, existing report files are rejected before preflight unless confirmation already permits overwrite, post-discovery failures write the structured report when possible, and human output prints `report: written <path>` only after a successful report write.
- Retention confirmed deletion now follows the #186 ops purge confirmation shape: manual runs first execute a dry-run planning call, reject non-final blockers locally before mutation, prompt through `shouldImplicitlyConfirm(cmd)`, then pass frozen `DiscoveredKeys` into the execution call so a second discovery cannot change the submitted root set.
- Retention execution controls are command flags mapped through existing facade options (`--workers`, `--fail-fast`, `--no-worker-limit`, `--no-wait`, `--no-state-check`, `--force`); the internal ops service delegates deletion to `processinstance.DeleteProcessInstances` and records submitted roots, reports, no-wait, confirmation, and final retention outcome.
- Retention dry-run/no-target command output now explicitly reports skipped deletion and `outcome: planned; no changes applied`; detailed retention seed/root/affected key lists remain gated behind `--verbose`, while JSON output keeps the complete structured model through the shared result envelope.
- Retention delete planning now calls the existing `processinstance.DryRunCancelOrDeletePlan` from the ops service after discovery, maps the frozen seeds into `RetentionDeletePlan`, preserves duplicate resolved roots via `DryRunPIKeyExpansion.DuplicateRoots`, and blocks non-dry-run mutation with `domain.ErrPrecondition` when non-final affected instances exist without force.
- Retention selection filters now reuse shared process-instance search flags and validation from `cmd/get_processinstance_*`: command code registers only compatible filters, rejects explicit `--key` before client construction, maps flags through `populatePISearchFilterOpts`, and then overwrites `EndDateBefore` with the required retention boundary.
- Human retention output prints `result.Discovery.Filters.String()` when filters are available; JSON and report-ready output already carry filters through the structured retention discovery/report model.
- Retention policy foundation now mirrors #186 boundaries: domain models in `internal/domain/ops_retention_policy.go`, public models/API/conversions in `c8volt/ops`, and a validation-only service seam in `internal/services/ops/retention_policy.go`.
- Foundational retention service validation rejects negative retention days and explicit process-instance key selection with `domain.ErrValidation`; the public ops facade maps those through `ferrors.FromDomain` to invalid-input behavior.
- Mandatory Ralph implementation context is `specs/ralph-implementation-rules.md`; commit subjects for this feature must use Conventional Commits and end with `#187`.
- `ops execute` remains a grouping command in `cmd/ops_execute.go`; `retention-policy` is registered from `cmd/ops_execute_retention_policy.go`, owns local `--retention-days` validation before facade calls, and changes the execute group contract from unsupported to limited through its child.
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
- Retention discovery now has a service-owned paged helper in `internal/services/processinstance/retention_discovery.go`; it accepts an already-normalized `EndDateBefore` filter from command/ops orchestration, skips items without `EndDate`, freezes unique seed keys, honors batch-size/limit semantics, and is exposed through a narrow `RetentionDiscoveryAPI` in `internal/services/processinstance/api.go`.

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
---
## Iteration 3 - 2026-05-14 12:41:53 CEST
**User Story**: User Story 1 - Register Retention Policy Command
**Tasks Completed**:
- [x] T011: Add command registration and help tests for `ops execute retention-policy` in `cmd/ops_execute_retention_policy_test.go`
- [x] T012: Add invalid missing/negative/non-integer `--retention-days` subprocess tests in `cmd/ops_execute_retention_policy_test.go`
- [x] T013: Add command contract metadata tests for state-changing and automation support in `cmd/command_contract_test.go`
- [x] T014: Add `ops execute retention-policy` Cobra command, summary, examples, and required retention flag in `cmd/ops_execute_retention_policy.go`
- [x] T015: Wire retention-policy command into the existing execute group in `cmd/ops_execute.go`
- [x] T016: Implement local retention flag validation and invalid-input error mapping in `cmd/ops_execute_retention_policy.go`
- [x] T017: Set mutation, output-mode, required-flag, and automation metadata in `cmd/ops_execute_retention_policy.go`
- [x] T018: Mark US1 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/capabilities_test.go
- cmd/cmd_views_ops_execute_retention_policy.go
- cmd/command_contract_test.go
- cmd/get_processinstance_test.go
- cmd/ops_execute.go
- cmd/ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy_test.go
- cmd/ops_test.go
- specs/187-ops-retention-policy/progress.md
- specs/187-ops-retention-policy/tasks.md
**Learnings**:
- `ops execute` capability discovery now becomes limited because it has a full-contract child command, while the grouping command itself still does not claim automation support.
- `--retention-days` uses command-owned validation plus `setFlagContractRequired`; Cobra parse errors are mapped with `useInvalidInputFlagErrors` and semantic errors go through `failBeforeCli`.
- Targeted validation passed with sandbox-local Go cache: `go test ./cmd -count=1` and `go test ./c8volt/ops ./internal/services/ops -count=1`.
---
---
## Iteration 4 - 2026-05-14 12:51:18 CEST
**User Story**: User Story 2 - Discover Retention Seeds
**Tasks Completed**:
- [x] T019: Add process-instance retention discovery service tests in `internal/services/processinstance/retention_discovery_test.go`
- [x] T020: Add ops service dry-run discovery tests for seed freezing and no delete calls in `internal/services/ops/retention_policy_test.go`
- [x] T021: Add command dry-run discovery output tests in `cmd/ops_execute_retention_policy_test.go`
- [x] T022: Add process-instance retention discovery primitive using existing end-date older-days search semantics in `internal/services/processinstance/retention_discovery.go`
- [x] T023: Expose retention discovery through the process-instance service interface in `internal/services/processinstance/api.go`
- [x] T024: Implement dry-run discovery orchestration and seed freezing in `internal/services/ops/retention_policy.go`
- [x] T025: Map dry-run discovery request and result through `c8volt/ops/client.go` and `c8volt/ops/convert.go`
- [x] T026: Render compact human and JSON discovery output in `cmd/cmd_views_ops_execute_retention_policy.go`
- [x] T027: Mark US2 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/ops/client_test.go
- cmd/cmd_views_ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy_test.go
- internal/services/ops/retention_policy.go
- internal/services/ops/retention_policy_test.go
- internal/services/processinstance/api.go
- internal/services/processinstance/retention_discovery.go
- internal/services/processinstance/retention_discovery_test.go
- specs/187-ops-retention-policy/progress.md
- specs/187-ops-retention-policy/tasks.md
**Learnings**:
- Retention discovery uses the same normalized `EndDateBefore` boundary produced from retention days, while future US3 selection filters can add to that filter without replacing the retention age constraint.
- The service discovery helper filters out no-`EndDate` items defensively, then freezes a unique seed-key set for downstream planning.
- Targeted validation passed with sandbox-local Go cache: `go test ./internal/services/processinstance ./internal/services/ops ./c8volt/ops ./cmd -run 'TestDiscoverRetentionProcessInstances|TestExecuteRetentionPolicy|TestClientExecuteRetentionPolicy|TestOpsExecuteRetentionPolicy|TestCommandContract|TestCommandCapability|TestOpsPurge|TestRoot|TestNewCli' -count=1`.
---
---
## Iteration 5 - 2026-05-14 12:59:44 CEST
**User Story**: User Story 3 - Apply Compatible Selection Filters
**Tasks Completed**:
- [x] T028: Add selection filter narrowing tests in `cmd/ops_execute_retention_policy_test.go`
- [x] T029: Add unsupported explicit `--key` invalid-input subprocess test in `cmd/ops_execute_retention_policy_test.go`
- [x] T030: Add service tests for normalized retention filters in `internal/services/ops/retention_policy_test.go`
- [x] T031: Add compatible process-instance selection flags to `cmd/ops_execute_retention_policy.go`
- [x] T032: Map selection flags into the retention request without allowing explicit keys in `cmd/ops_execute_retention_policy.go`
- [x] T033: Apply normalized filters during retention discovery in `internal/services/processinstance/retention_discovery.go`
- [x] T034: Include selected filters in human, JSON, and report-ready retention output in `cmd/cmd_views_ops_execute_retention_policy.go`
- [x] T035: Mark US3 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy_test.go
- internal/services/ops/retention_policy_test.go
- internal/services/processinstance/retention_discovery_test.go
- specs/187-ops-retention-policy/progress.md
- specs/187-ops-retention-policy/tasks.md
**Learnings**:
- Retention policy selection filters should stay on the existing process-instance search path: shared command validation owns state, batch-size, limit, process-definition selector, roots/children, and incident selector checks.
- The retention service deliberately overwrites any request `Selection.EndDateBefore` with the derived retention boundary, so compatible filters narrow discovery without replacing the required age threshold.
- Targeted validation passed with sandbox-local Go cache: `go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance -run 'TestOpsExecuteRetentionPolicy|TestClientExecuteRetentionPolicy|TestExecuteRetentionPolicy|TestDiscoverRetentionProcessInstances' -count=1` and `go test ./cmd -run 'TestOpsExecuteRetentionPolicy|TestCommandContract|TestCommandCapability|TestRoot|TestNewCli' -count=1`.
---
---
## Iteration 6 - 2026-05-14 13:06:42 CEST
**User Story**: User Story 4 - Build And Validate Delete Plan
**Tasks Completed**:
- [x] T036: Add ops service delete-plan tests for child seeds, resolved roots, affected keys, and duplicates in `internal/services/ops/retention_policy_test.go`
- [x] T037: Add non-final affected instance blocking test in `internal/services/ops/retention_policy_test.go`
- [x] T038: Add command dry-run plan rendering tests in `cmd/ops_execute_retention_policy_test.go`
- [x] T039: Reuse existing process-instance delete planning from retention seeds in `internal/services/ops/retention_policy.go`
- [x] T040: Preserve missing ancestor, traversal warning, duplicate, final-state, and non-final details in the retention result model in `internal/domain/ops_retention_policy.go`
- [x] T041: Map delete-plan details through `c8volt/ops/convert.go`
- [x] T042: Render compact delete-plan human output and complete JSON output in `cmd/cmd_views_ops_execute_retention_policy.go`
- [x] T043: Mark US4 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy_test.go
- internal/domain/processinstance_traversal.go
- internal/services/ops/retention_policy.go
- internal/services/ops/retention_policy_test.go
- internal/services/processinstance/dryrun.go
- specs/187-ops-retention-policy/progress.md
- specs/187-ops-retention-policy/tasks.md
**Learnings**:
- Retention planning should stay on the existing process-instance delete planning path; the only shared primitive extension needed for US4 was exposing duplicate resolved roots from the existing ancestry results.
- Non-final affected instances are preserved in the retention delete plan and block non-dry-run mutation with `domain.ErrPrecondition` until a later execution story wires the force/delete controls.
- Targeted validation passed with sandbox-local Go cache: `go test ./internal/services/ops -run 'TestExecuteRetentionPolicy' -count=1`, `go test ./internal/services/processinstance -run 'Test.*DryRun|TestDiscoverRetentionProcessInstances' -count=1`, `go test ./cmd -run 'TestOpsExecuteRetentionPolicy' -count=1`, and `go test ./c8volt/ops -run 'TestClientExecuteRetentionPolicy' -count=1`.
---
---
## Iteration 7 - 2026-05-14 13:12:36 CEST
**User Story**: User Story 5 - Support Dry Run Output
**Tasks Completed**:
- [x] T044: Add no-target dry-run success test in `cmd/ops_execute_retention_policy_test.go`
- [x] T045: Add `--dry-run --json` deterministic output test in `cmd/ops_execute_retention_policy_test.go`
- [x] T046: Add dry-run no prompt/no mutation test in `cmd/ops_execute_retention_policy_test.go`
- [x] T047: Implement dry-run command orchestration through the ops facade in `cmd/ops_execute_retention_policy.go`
- [x] T048: Add planned/skipped/final outcome status handling for dry-run and no-target flows in `internal/services/ops/retention_policy.go`
- [x] T049: Keep detailed key lists behind verbose output while preserving complete JSON data in `cmd/cmd_views_ops_execute_retention_policy.go`
- [x] T050: Mark US5 tasks complete and record validation notes in `specs/187-ops-retention-policy/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy_test.go
- internal/services/ops/retention_policy_test.go
- specs/187-ops-retention-policy/progress.md
- specs/187-ops-retention-policy/tasks.md
**Learnings**:
- Retention dry-run command coverage now exercises no-target success, JSON envelope structure, and matching-seed no-mutation behavior through HTTP fakes.
- Dry-run human output now has an explicit no-target line plus skipped deletion/final outcome wording, while existing verbose-only key-list behavior remains intact.
- Targeted validation passed with sandbox-local Go cache: `go test ./cmd ./internal/services/ops -run 'TestOpsExecuteRetentionPolicy|TestExecuteRetentionPolicy' -count=1`, `go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance -run 'TestOpsExecuteRetentionPolicy|TestClientExecuteRetentionPolicy|TestExecuteRetentionPolicy|TestDiscoverRetentionProcessInstances' -count=1`, and `go test ./cmd -count=1`.
---
---
## Iteration 8 - 2026-05-14 13:20:51 CEST
**User Story**: User Story 6 - Execute Confirmed Deletion
**Tasks Completed**:
- [x] T051: Add confirmed deletion command test for exact frozen-plan root submission
- [x] T052: Add execution-control mapping tests for workers, fail-fast, no-wait, no-state-check, and force
- [x] T053: Add `--automation --json` without `--auto-confirm` success test
- [x] T054: Add local-precondition failure subprocess tests for post-planning blockers and exit code
- [x] T055: Add compatible delete execution control flags and facade option mapping
- [x] T056: Execute deletion through existing process-instance deletion service
- [x] T057: Use `shouldImplicitlyConfirm(cmd)` for destructive confirmation decisions
- [x] T058: Preserve no-wait, confirmation, per-key status, and final outcome
- [x] T059: Render deletion execution and final outcome
- [x] T060: Mark US6 tasks complete and record validation notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/ops/convert.go
- c8volt/ops/model.go
- cmd/cmd_views_ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy_test.go
- internal/domain/ops_retention_policy.go
- internal/services/ops/retention_policy.go
- internal/services/ops/retention_policy_test.go
- specs/187-ops-retention-policy/progress.md
- specs/187-ops-retention-policy/tasks.md
**Learnings**:
- Manual confirmed retention deletion must freeze discovered seeds from the planning call into `DiscoveredKeys`; otherwise a second discovery during execution could submit a different root set than the operator reviewed.
- The service can reuse the orphan purge deletion status helpers for step status, but retention needs its own outcome mapping because the outcome enum is retention-specific.
- Targeted validation passed with sandbox-local Go cache: `go test ./internal/services/ops -run 'TestExecuteRetentionPolicy' -count=1`, `go test ./c8volt/ops -run 'TestClientExecuteRetentionPolicy' -count=1`, `go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance -run 'TestOpsExecuteRetentionPolicy|TestClientExecuteRetentionPolicy|TestExecuteRetentionPolicy|TestDiscoverRetentionProcessInstances' -count=1`, and `go test ./cmd -run 'TestOpsExecuteRetentionPolicy|TestCommandContract|TestCommandCapability|TestRoot|TestNewCli' -count=1`. Local-listener command integration cases were skipped by the sandbox network policy during command test runs.
---
---
## Iteration 9 - 2026-05-14 13:28:48 CEST
**User Story**: User Story 7 - Write Audit Reports
**Tasks Completed**:
- [x] T061: Add report format inference and validation tests for retention
- [x] T062: Add Markdown retention report rendering test
- [x] T063: Add JSON retention report rendering test
- [x] T064: Add existing report-file preservation tests for dry-run, unconfirmed, and locally blocked runs
- [x] T065: Add post-discovery failure report-write test
- [x] T066: Reuse shared ops report-file validation, format inference, overwrite safety, and file writing
- [x] T067: Extend report model/rendering for retention-specific discovery, plan, deletion, and outcome fields
- [x] T068: Ensure reports render from the stable structured retention model
- [x] T069: Print compact `report: written <path>` human output after report writes
- [x] T070: Mark US7 tasks complete and record validation notes
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cmd_views_ops_execute_retention_policy.go
- cmd/get_processinstance_test.go
- cmd/ops_contract_test.go
- cmd/ops_execute_retention_policy.go
- cmd/ops_execute_retention_policy_test.go
- specs/187-ops-retention-policy/progress.md
- specs/187-ops-retention-policy/tasks.md
**Learnings**:
- Retention reports should stay command-owned for file validation/write orchestration while rendering from the `RetentionAuditReport` already populated by the ops service.
- Existing report files are protected before discovery for dry-run, unconfirmed, and locally blocked flows; post-discovery failures with a new report path write failure status, discovery counts, deletion status, and errors.
- Targeted validation passed with sandbox-local Go cache: `go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance -run 'TestOpsExecuteRetentionPolicy|TestOpsWorkflowReport|TestValidateOpsWorkflowReport|TestWriteOpsWorkflowReport|TestClientExecuteRetentionPolicy|TestExecuteRetentionPolicy|TestDiscoverRetentionProcessInstances' -count=1` and `go test ./cmd -count=1`.
---
