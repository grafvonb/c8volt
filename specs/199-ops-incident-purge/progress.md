# Progress: Ops Purge Process Instances With Incidents

**Issue**: [#199](https://github.com/grafvonb/c8volt/issues/199)
**Feature**: `199-ops-incident-purge`
**Plan**: [plan.md](plan.md)
**Tasks**: [tasks.md](tasks.md)
**Mandatory Ralph Context**: `specs/ralph-implementation-rules.md`
**Commit Rule**: Conventional Commit subject ending in `#199`

## Codebase Patterns

- Mandatory Ralph implementation context is `specs/ralph-implementation-rules.md`; commit subjects for this feature must use Conventional Commits and end with `#199`.
- Existing #186 ops purge command wiring lives in `cmd/ops_purge_orphan_processinstances.go`: command files own Cobra flags, validation, `NewCli`, automation support checks, report-path planning validation, confirmation, facade calls, report writing, and final rendering.
- Existing #187 retention workflow in `cmd/ops_execute_retention_policy.go`, `internal/services/ops/retention_policy.go`, and `cmd/cmd_views_ops_execute_retention_policy.go` is the closest pattern for freeze-then-delete-plan behavior.
- Ops workflow command metadata uses `setCommandMutation`, `setContractSupport`, `setAutomationSupport`, and `setFlagContractRequired` from `cmd/command_contract.go`; capability regression tests live in `cmd/command_contract_test.go` and related command tests.
- Shared ops report helpers in `cmd/ops_contract.go` own report format inference, validation, overwrite mode, and secure file writes; workflow-specific Markdown and JSON rendering lives beside each workflow view.
- Incident command behavior and filters are centered in `cmd/get_incident.go`, `cmd/get_incident_search.go`, `c8volt/incident`, and `internal/services/incident`; #199 should reuse those semantics for candidate discovery.
- Existing delete planning is `processinstance.DryRunCancelOrDeletePlan`, which deduplicates selected keys, resolves roots through ancestry traversal, expands descendants, preserves missing ancestor/traversal warnings, records selected final-state items and non-final affected items, and validates unresolved plans.
- Existing delete execution uses `processinstance.DeleteProcessInstances`, `toolx.DetermineNoOfWorkers`, `toolx/pool.ExecuteSlice`, `services.ApplyCallOptions`, `FailFast`, `NoWorkerLimit`, `NoWait`, and logger/activity helpers; command code should pass controls rather than owning worker logic.
- Human ops output follows a compact sequence of workflow status lines; detailed key lists are gated by verbose output while JSON and reports keep complete structured data.
- Generated CLI docs are protected by `docsgen/main_test.go` and produced via `make docs-content`; generated files under `docs/cli/` and `docs/index.md` should not be hand-edited after command metadata changes.

## Status

- Speckit specification created from GitHub issue #199.
- Clarification gate completed with no critical ambiguities worth formal questioning.
- Planning artifacts generated for Ralph-sized implementation work.

## Ralph Notes

- Each Ralph iteration must read `specs/ralph-implementation-rules.md`, `spec.md`, `plan.md`, `tasks.md`, and this progress file before code changes.
- Each iteration should complete only the first incomplete work unit and update this file with validation results.
- Existing #186/#187 ops purge/report/delete-plan code is expected to be the closest implementation pattern.
