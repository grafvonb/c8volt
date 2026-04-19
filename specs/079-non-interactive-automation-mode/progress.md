# Ralph Progress Log

Feature: 079-non-interactive-automation-mode
Started: 2026-04-19 06:53:03

## Codebase Patterns

- Root-level CLI behavior is centralized in `cmd/root.go`: persistent flags bind into Viper via `initViper`, the root long help already positions `capabilities --json` as the machine-discovery entry point, and bootstrap logging/output defaults are set before command execution.
- Shared interactive confirmation currently flows through `confirmCmdOrAbort` / `confirmCmdOrAbortFn` in `cmd/cmd_cli.go`; `--auto-confirm` is the only existing explicit non-interactive bypass and non-terminal stdin is also treated as implicit confirmation.
- Shared JSON contract rendering is centralized in `cmd/cmd_views_contract.go`: `ContractSupportFull` plus JSON mode triggers the common result envelope, and state-changing commands switch from `succeeded` to `accepted` when `--no-wait` is set.
- Search-based process-instance pagination lives in `cmd/get_processinstance.go` and is reused by `cancel`/`delete`; human modes prompt before continuing, while JSON mode currently auto-consumes pages and returns aggregated machine-readable output.
- Generated CLI reference pages under `docs/cli/` are driven by live Cobra metadata via `make docs`, and README-facing homepage content is refreshed into `docs/index.md` through `make docs-content` rather than hand-edited.

---

## Iteration 1 - 2026-04-19 09:09 CEST
**User Story**: Phase 1 Setup
**Tasks Completed**:
- [x] T001: inventoried root flags, config bindings, and machine-facing guidance in `cmd/root.go`, `README.md`, and `docs/use-cases.md`
- [x] T002: inventoried prompt, paging, and result-envelope seams in `cmd/cmd_cli.go`, `cmd/cmd_views_contract.go`, `cmd/get_processinstance.go`, `cmd/delete_processinstance.go`, and `cmd/cancel_processinstance.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/079-non-interactive-automation-mode/tasks.md
- specs/079-non-interactive-automation-mode/progress.md
**Learnings**:
- `cmd/root.go` already exposes adjacent operator-facing flags (`--auto-confirm`, `--json`, `--quiet`, `--no-err-codes`) and binds root config through `app.*` and `log.*` Viper keys, which is the natural seam for a future `--automation` flag.
- Current public docs describe script-safe behavior through `--json`, `--auto-confirm`, `--no-wait`, and `capabilities --json`, but they do not yet define one canonical non-interactive automation contract.
- `delete` and `cancel` both derive impact prompts from `DryRunCancelOrDeleteGetPIKeys` before execution, so automation-mode support can likely hook the shared confirmation seam without duplicating per-command prompt text.
- Search-driven `get`, `cancel`, and `delete` already share paging state and aggregated rendering behavior, which reduces the surface area for implementing automation-mode continuation semantics later.
---

## Iteration 2 - 2026-04-19 07:01 CEST
**User Story**: Phase 2 Foundational
**Tasks Completed**:
- [x] T003: defined the root `--automation` flag, bound it through `app.automation`, and added a shared effective-mode helper in `cmd/root.go`
- [x] T004: extended command capability metadata and the discovery surface with explicit automation support fields in `cmd/command_contract.go` and `cmd/capabilities.go`
- [x] T005: added foundational regression coverage for root automation binding and discovery metadata in `cmd/root_test.go`, `cmd/capabilities_test.go`, and `cmd/command_contract_test.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/capabilities.go
- cmd/capabilities_test.go
- cmd/command_contract.go
- cmd/command_contract_test.go
- cmd/root.go
- cmd/root_test.go
- config/app.go
- specs/079-non-interactive-automation-mode/progress.md
- specs/079-non-interactive-automation-mode/tasks.md
**Learnings**:
- Root automation state fits the existing config-resolution pattern cleanly by binding `app.automation` alongside the other persistent `app.*` keys and reading the effective value back from config context when available.
- Keeping `automationSupport` separate from `contractSupport` avoids falsely equating JSON-envelope support with automation readiness; unsupported remains the safe default until a command opts in explicitly.
- `capabilities --json` is the right place to expose automation readiness first because it already serves as the machine-discovery surface and can start telling automation callers which command paths are not ready yet without changing runtime command behavior.
---

## Iteration 3 - 2026-04-19 07:07 CEST
**User Story**: User Story 1 - Run Commands Safely Without Prompts
**Tasks Completed**:
- [x] T006: added automation-mode regression coverage for supported confirmation bypass and unsupported-command rejection in `cmd/delete_test.go`, `cmd/cancel_test.go`, `cmd/expect_test.go`, and `cmd/walk_test.go`
- [x] T007: added automation-mode paging continuation regression coverage in `cmd/get_processinstance_test.go`, `cmd/delete_test.go`, and `cmd/cancel_test.go`
- [x] T008: implemented shared automation-mode support gating and implicit-confirm helpers in `cmd/cmd_cli.go` and `cmd/get_processinstance.go`
- [x] T009: wired implicit automation confirmation into representative state-changing commands in `cmd/delete_processinstance.go`, `cmd/cancel_processinstance.go`, and `cmd/delete_processdefinition.go`
- [x] T010: marked supported read flows and explicit unsupported observe flows for automation mode in `cmd/get_processinstance.go`, `cmd/expect_processinstance.go`, and `cmd/walk_processinstance.go`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel_processinstance.go
- cmd/cancel_test.go
- cmd/capabilities_test.go
- cmd/cmd_cli.go
- cmd/delete_processdefinition.go
- cmd/delete_processinstance.go
- cmd/delete_test.go
- cmd/expect_processinstance.go
- cmd/expect_test.go
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/walk_processinstance.go
- cmd/walk_test.go
- specs/079-non-interactive-automation-mode/progress.md
- specs/079-non-interactive-automation-mode/tasks.md
**Learnings**:
- Runtime automation support can stay aligned with discovery metadata by reusing per-command `automationSupport` annotations as the single source of truth for explicit `--automation` rejection.
- Supported automation flows can reuse the existing confirmation seam by passing one shared implicit-confirm helper into both destructive confirmation prompts and paged search continuation prompts.
- Parent command discovery metadata can remain conservative while leaf commands opt into automation incrementally, which keeps `capabilities --json` truthful during staged rollout.
---

## Iteration 4 - 2026-04-19 09:41 CEST
**User Story**: User Story 2 - Combine Automation Mode With Machine Output
**Tasks Completed**:
- [x] T011: added automation JSON regressions that keep capability discovery and paged process-instance output machine-readable on stdout while progress stays off stdout
- [x] T012: extended representative `--no-wait` regressions to cover automation-mode accepted outcomes for run, deploy, delete, and cancel flows
- [x] T013: factored shared machine-readable render-mode detection through the contract helpers
- [x] T014: aligned discovery/read-flow tests with automation JSON behavior for capabilities and paged process-instance reads
- [x] T015: enabled explicit automation support on representative state-changing run/deploy commands while preserving accepted-envelope behavior on `--no-wait`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel_test.go
- cmd/capabilities_test.go
- cmd/cmd_views_contract.go
- cmd/cmd_views_rendermode.go
- cmd/delete_test.go
- cmd/deploy_processdefinition.go
- cmd/deploy_test.go
- cmd/get_processinstance_test.go
- cmd/get_test.go
- cmd/run_processinstance.go
- cmd/run_test.go
- specs/079-non-interactive-automation-mode/progress.md
- specs/079-non-interactive-automation-mode/tasks.md
**Learnings**:
- The repo already routed verbose process-instance paging diagnostics to `cmd.ErrOrStderr()`, so automation JSON isolation mainly needed regression coverage rather than a new logging channel.
- Representative state-changing commands only become truthfully automation-ready when runtime gating and discovery annotations move together; enabling one without the other leaves the contract ambiguous.
- The shared envelope logic can stay simple as long as JSON detection remains the single gate for machine-readable rendering and the command-specific automation tests verify stdout cleanliness.
---

## Iteration 5 - 2026-04-19 07:21 CEST
**User Story**: User Story 3 - Preserve Human Workflows While Documenting the Automation Contract
**Tasks Completed**:
- [x] T016: added human-mode regression assertions proving root/get help and representative paged cancel/delete flows stay prompt-driven without `--automation`
- [x] T017: added discovery/help-text regression coverage for the documented automation contract in `cmd/capabilities_test.go` and `cmd/root_test.go`
- [x] T018: updated root and representative command help text for the automation contract in `cmd/root.go`, `cmd/capabilities.go`, `cmd/run_processinstance.go`, and `cmd/get_processinstance.go`
- [x] T019: updated user-facing automation guidance in `README.md` and `docs/use-cases.md`
- [x] T020: regenerated CLI reference docs under `docs/cli/` and synced `docs/index.md` from README-backed content
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- README.md
- cmd/root.go
- cmd/root_test.go
- cmd/capabilities.go
- cmd/capabilities_test.go
- cmd/command_contract_test.go
- cmd/get_processinstance.go
- cmd/get_test.go
- cmd/delete_test.go
- cmd/cancel_test.go
- cmd/run_processinstance.go
- docs/use-cases.md
- docs/index.md
- docs/cli/
- specs/079-non-interactive-automation-mode/progress.md
- specs/079-non-interactive-automation-mode/tasks.md
**Learnings**:
- The finalized automation contract fits best as documentation layered on top of existing human-mode wording: root help anchors the contract globally, while representative commands only need concise command-local guidance about `--automation`, `--json`, and `--no-wait`.
- Existing paging prompt regressions for `cancel` and `delete` already exercised the preserved human path, so this iteration mainly needed contract-specific assertions that those flows stay in `prompt` mode until callers opt into automation or `--auto-confirm`.
- Regenerating docs after Cobra help changes also refreshed many root inherited-flag pages, which is expected because the root persistent flag descriptions flow through every generated command reference page.
---
