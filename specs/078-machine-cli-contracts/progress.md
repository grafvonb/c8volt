# Ralph Progress Log

Feature: 078-machine-cli-contracts
Started: 2026-04-17 18:14:53

## Codebase Patterns

- Root-level machine metadata should be derived from the existing Cobra command tree and persistent flag definitions in `cmd/root.go`, not from generated docs or a parallel registry.
- Shared machine-readable rendering should extend the existing `pickMode` / `itemView` / `listOrJSON` seams in `cmd/` and keep current public payload models intact.
- Generated docs anchors for CLI behavior come from Cobra metadata and `README.md`; feature research should record doc targets, but implementation should update source metadata first and regenerate output afterward.
- Human-facing CLI guidance should stay in Cobra `Long`/`Example` strings and `README.md`, then flow into generated `docs/cli/` and `docs/index.md` through `make docs` and `make docs-content` instead of hand-editing generated pages.
- Shared discovery metadata can live on Cobra command annotations and be computed from the live command tree, which keeps command paths, inherited flags, and contract support in one repository-native source of truth.
- Foundational machine-result helpers should stay in `cmd/` while `c8volt/ferrors` only exposes the bounded failure-to-outcome mapping needed to preserve exit-code authority.

---

## Iteration 1 - 2026-04-17 18:23:00
**User Story**: Phase 1 Setup
**Tasks Completed**:
- [x] T001: Inventory the current command tree, root flags, and machine-facing render seams
- [x] T002: Inventory representative command-family payload and outcome seams
- [x] T003: Confirm current automation-facing docs and generated CLI reference anchors
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/078-machine-cli-contracts/progress.md
- specs/078-machine-cli-contracts/research.md
- specs/078-machine-cli-contracts/tasks.md
**Learnings**:
- The feature can build discovery metadata directly from Cobra command registrations, aliases, and real flag definitions already present under `cmd/`.
- The current machine-facing seam is fragmented between read-family JSON render helpers and state-changing commands that rely on exit codes plus logs, so a shared envelope belongs in `cmd/` rather than in generic serialization helpers.
- README plus generated `docs/index.md` and `docs/cli/` already provide the documentation path needed for a future `capabilities` command without adding new doc infrastructure.
---

## Iteration 2 - 2026-04-17 18:22:37 CEST
**User Story**: Phase 2 Foundational
**Tasks Completed**:
- [x] T004: Define the shared capability and result-envelope types
- [x] T005: Implement shared command metadata and contract-support helpers
- [x] T006: Implement shared result-envelope rendering helpers and outcome-mapping utilities
- [x] T007: Add foundational regression coverage for capability metadata, outcome mapping, and exit-code alignment
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/command_contract.go
- cmd/cmd_views_contract.go
- cmd/command_contract_test.go
- cmd/capabilities_test.go
- cmd/root.go
- c8volt/ferrors/errors.go
- c8volt/ferrors/errors_test.go
- specs/078-machine-cli-contracts/tasks.md
- specs/078-machine-cli-contracts/progress.md
**Learnings**:
- Cobra required-flag annotations are sufficient to expose machine-readable required-flag metadata without introducing a second flag registry.
- Read-only versus state-changing discovery defaults can be inferred safely from the existing top-level command taxonomy, with command annotations reserved for future exceptions and explicit rollouts.
- The foundational envelope helpers can be validated before any command-family rollout by testing document generation and failure outcome mapping in isolation.
---

## Iteration 3 - 2026-04-17 18:30:40 CEST
**User Story**: User Story 1 - Discover Safe Command Contracts
**Tasks Completed**:
- [x] T008: Add discovery-command regression tests for top-level and nested command metadata
- [x] T009: Add command-metadata coverage for flags, output modes, and support-status reporting
- [x] T010: Implement the dedicated top-level discovery command
- [x] T011: Populate representative capability metadata for `get`, `run`, `expect`, `walk`, `deploy`, `delete`, and `cancel` command families
- [x] T012: Mark unsupported and limited machine-contract support honestly in the discovery surface for non-rolled-out commands
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/capabilities.go
- cmd/capabilities_test.go
- cmd/cmd_checks.go
- cmd/command_contract.go
- cmd/command_contract_test.go
- cmd/get.go
- cmd/get_cluster.go
- cmd/get_cluster_license.go
- cmd/get_cluster_topology.go
- cmd/get_processdefinition.go
- cmd/get_processinstance.go
- cmd/get_resource.go
- cmd/get_test.go
- cmd/run.go
- cmd/run_processinstance.go
- cmd/expect.go
- cmd/expect_processinstance.go
- cmd/walk.go
- cmd/walk_processinstance.go
- cmd/deploy.go
- cmd/deploy_processdefinition.go
- cmd/delete.go
- cmd/delete_processinstance.go
- cmd/cancel.go
- cmd/cancel_processinstance.go
- specs/078-machine-cli-contracts/tasks.md
- specs/078-machine-cli-contracts/progress.md
**Learnings**:
- Inherited root flags are not enough to infer honest machine-contract support; discovery needs explicit command annotations plus child-derived roll-up for parent nodes.
- A metadata-only discovery command should bypass the full root bootstrap path so automation can inspect the CLI surface without config or network prerequisites.
- The current repository already has enough command-local render seams to distinguish `limited` read-only JSON support from state-changing commands that should stay explicitly `unsupported` until the shared envelope lands.
---

## Iteration 4 - 2026-04-17 18:44:41 CEST
**User Story**: User Story 2 - Receive Stable Machine Results
**Tasks Completed**:
- [x] T013: Add `get` and `walk` result-envelope regression tests for confirmed successful read-only flows
- [x] T014: Add `run`, `deploy`, `delete`, and `cancel` regression tests for `accepted` versus `succeeded` behavior
- [x] T015: Add `invalid` and `failed` envelope regression tests with exit-code alignment
- [x] T016: Integrate the shared result envelope into read-only command rendering
- [x] T017: Integrate the shared result envelope into representative state-changing command families
- [x] T018: Align `accepted`, `invalid`, and `failed` envelope behavior with repository-native `--no-wait` and `ferrors` semantics
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/cancel_processinstance.go
- cmd/cancel_test.go
- cmd/capabilities_test.go
- cmd/cmd_cli.go
- cmd/cmd_views_contract.go
- cmd/cmd_views_get.go
- cmd/cmd_views_rendermode.go
- cmd/cmd_views_walk.go
- cmd/command_contract_test.go
- cmd/delete_processdefinition.go
- cmd/delete_processinstance.go
- cmd/delete_test.go
- cmd/deploy_processdefinition.go
- cmd/deploy_test.go
- cmd/expect_processinstance.go
- cmd/expect_test.go
- cmd/get_processinstance.go
- cmd/get_processinstance_test.go
- cmd/get_resource.go
- cmd/get_test.go
- cmd/run_processinstance.go
- cmd/run_test.go
- cmd/version.go
- cmd/version_test.go
- cmd/walk_processinstance.go
- cmd/walk_test.go
- specs/078-machine-cli-contracts/tasks.md
- specs/078-machine-cli-contracts/progress.md
**Learnings**:
- The shared envelope can stay honest and incremental by living at the `cmd/` render boundary while reusing existing domain payload models and `ferrors.ResolveExitCode` for process-level authority.
- `--no-wait` semantics map cleanly to `accepted` when the command already returns a repository-native payload, while read-only JSON flows can adopt `succeeded` without changing their underlying item/list render helpers.
- Process-instance search actions needed accumulated reporter data to make paged cancel/delete JSON output truthful for machine consumers instead of only covering direct-key flows.
---

## Iteration 5 - 2026-04-17 18:54:15 CEST
**User Story**: User Story 3 - Keep Human CLI Behavior Intact
**Tasks Completed**:
- [x] T019: Add compatibility regression tests that prove plain-text and keys-only behavior remain intact for representative commands
- [x] T020: Add discovery/help-text regression coverage for the new top-level command and unchanged CLI taxonomy
- [x] T021: Update machine-contract and automation guidance in README and docs content
- [x] T022: Update Cobra help text for the discovery command and affected machine-readable guidance
- [x] T023: Regenerate generated CLI reference output from the updated Cobra command metadata
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- README.md
- cmd/capabilities.go
- cmd/capabilities_test.go
- cmd/get_resource.go
- cmd/get_test.go
- cmd/root.go
- cmd/root_test.go
- cmd/run_processinstance.go
- cmd/run_test.go
- cmd/version.go
- cmd/version_test.go
- cmd/walk_processinstance.go
- cmd/walk_test.go
- docs/cli/c8volt.md
- docs/cli/c8volt_capabilities.md
- docs/cli/c8volt_get_resource.md
- docs/cli/c8volt_run_process-instance.md
- docs/cli/c8volt_version.md
- docs/cli/c8volt_walk_process-instance.md
- docs/index.md
- docs/index.md
- specs/078-machine-cli-contracts/progress.md
- specs/078-machine-cli-contracts/tasks.md
**Learnings**:
- The version command tests cannot use `t.Parallel()` because the shared Cobra root and flag reset helpers mutate global command state under `-race`.
- User-facing automation guidance is safest when it reinforces the new machine contract without changing default plain-text or `--keys-only` behavior for existing operator flows.
- Generated CLI reference coverage for new top-level commands comes entirely from Cobra metadata, so adding help text plus `make docs` is sufficient to publish them.
---

## Iteration 6 - 2026-04-17 18:58:30 CEST
**User Story**: Phase 6 - Polish & Cross-Cutting Concerns
**Tasks Completed**:
- [x] T024: Refresh implementation and verification notes in quickstart, research, and plan
- [x] T025: Run focused machine-contract validation with `go test ./c8volt/ferrors -count=1` and `go test ./cmd -count=1`
- [x] T026: Run documentation regeneration validation with `make docs` and `make docs-content`
- [x] T027: Run repository validation with `make test`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- docs/index.md
- specs/078-machine-cli-contracts/plan.md
- specs/078-machine-cli-contracts/progress.md
- specs/078-machine-cli-contracts/quickstart.md
- specs/078-machine-cli-contracts/research.md
- specs/078-machine-cli-contracts/tasks.md
**Learnings**:
- The final polish pass only regenerated tracked docs metadata in `docs/index.md`; the machine-contract command reference pages were already in sync with the committed Cobra help text.
- Keeping the focused contract suites ahead of `make test` makes it obvious whether a regression belongs to shared outcome mapping, command behavior, or a broader repository interaction.
- `make docs-content` is still part of the required validation path because README-sourced homepage metadata changes can advance even when the CLI reference pages themselves do not.
---
