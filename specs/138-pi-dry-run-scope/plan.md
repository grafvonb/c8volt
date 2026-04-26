# Implementation Plan: Process-Instance Dry Run Scope Preview

**Branch**: `138-pi-dry-run-scope` | **Date**: 2026-04-25 | **Spec**: [spec.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/138-pi-dry-run-scope/spec.md)
**Input**: Feature specification from `/specs/138-pi-dry-run-scope/spec.md`

## Summary

Add `--dry-run` to `cancel process-instance` and `delete process-instance` so operators can preview the same family-aware destructive scope currently computed during command preflight. The implementation should reuse `c8volt/process.DryRunCancelOrDeletePlan`, surface roots, affected family keys, traversal outcome, warnings, and missing ancestor keys in human and structured output, return aggregate summary plus nested per-page previews for structured search-mode dry runs, and bypass confirmation, cancel/delete mutations, and wait polling when dry run is enabled.

## Technical Context

**Language/Version**: Go 1.26
**Primary Dependencies**: standard library, `github.com/spf13/cobra`, existing command helpers in `cmd/`, process facade in `c8volt/process`, shared traversal types in `c8volt/process/api.go`, versioned process-instance services in `internal/services/processinstance/*`, generated docs pipeline via repository make targets
**Storage**: N/A
**Testing**: `go test`, focused package tests under `cmd/` and `c8volt/process`, final `make test`; docs regeneration through `make docs-content` when command metadata changes
**Target Platform**: CLI for local and CI use against supported Camunda process-instance cancel/delete flows
**Project Type**: CLI
**Performance Goals**: Dry run should use the same traversal calls real preflight already needs; avoid submitting mutations or running waiter polling; preserve current paged search behavior and stop/continuation rules
**Constraints**: Reuse `DryRunCancelOrDeletePlan` as the scope source of truth; preserve orphan-parent partial-result behavior; keep real cancel/delete behavior unchanged; ensure dry run works for direct keys and search pages; expose machine-readable output without requiring warning-text parsing; structured search-mode output must include an aggregate summary plus nested per-page previews; update README and generated CLI docs in the same change
**Scale/Scope**: Two command surfaces (`cancel process-instance`, `delete process-instance`), shared preflight helpers in `cmd/cancel_processinstance.go` and `cmd/delete_processinstance.go`, shared page orchestration in `cmd/get_processinstance.go`, facade dry-run types in `c8volt/process`, focused command/facade tests, README examples, and generated CLI docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Operational Proof Over Intent**: Pass. The feature adds a non-mutating proof step that reports exactly what real destructive execution would target before any action is submitted.
- **CLI-First, Script-Safe Interfaces**: Pass. Behavior is exposed through explicit Cobra flags and structured output fields that automation can consume without parsing prompts.
- **Tests and Validation Are Mandatory**: Pass. The plan requires keyed, search/paged, escalation, full-family, partial orphan, no-mutation, aggregate structured output, and docs/help regression coverage.
- **Documentation Matches User Behavior**: Pass. `--dry-run` is a user-visible command change, so README examples and generated CLI docs must be refreshed.
- **Small, Compatible, Repository-Native Changes**: Pass. The design extends existing cancel/delete preflight and dry-run facade contracts instead of introducing a parallel traversal or planning subsystem.

## Project Structure

### Documentation (this feature)

```text
specs/138-pi-dry-run-scope/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── process-instance-dry-run.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── cancel_processinstance.go
├── cancel_test.go
├── delete_processinstance.go
├── delete_test.go
├── get_processinstance.go
├── cmd_views_*.go
├── process_api_stub_test.go
└── root.go

c8volt/process/
├── api.go
├── dryrun.go
└── client_test.go

README.md
docs/cli/
├── c8volt_cancel_process-instance.md
└── c8volt_delete_process-instance.md
```

**Structure Decision**: Keep implementation in the current command-layer preflight path. `cancelProcessInstancesWithPlan` and `deleteProcessInstancesWithPlan` already call `DryRunCancelOrDeletePlan` before prompting and mutating; dry run should branch after that shared plan is computed, render the plan, and return without confirmation, mutation, or wait behavior. Search-mode dry run should continue using `processPISearchPagesWithAction` so page limiting and continuation behavior stay consistent with real destructive preflight.

## Phase 0: Research

Research findings are captured in [research.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/138-pi-dry-run-scope/research.md).

- Confirm `DryRunCancelOrDeletePlan` is the authoritative source for roots, collected affected keys, traversal outcome, warning text, and missing ancestors.
- Confirm keyed cancel/delete command paths already compute preflight plans before confirmation and can return dry-run output from the same helper before mutation.
- Confirm search-mode cancel/delete paths already process each selected page through the same helper, so dry run can reuse the per-page action callback without duplicating pagination logic.
- Confirm the structured output model should be a command-level dry-run payload rather than overloading cancel/delete report payloads, because no mutation reports exist during dry run.
- Confirm structured search-mode dry-run output should use one aggregate summary with nested per-page previews so automation can inspect totals and page-level traversal outcomes.
- Confirm documentation should be regenerated after command metadata examples/help mention `--dry-run`.

## Phase 1: Design & Contracts

Design artifacts are captured in:

- [data-model.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/138-pi-dry-run-scope/data-model.md)
- [quickstart.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/138-pi-dry-run-scope/quickstart.md)
- [contracts/process-instance-dry-run.md](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/specs/138-pi-dry-run-scope/contracts/process-instance-dry-run.md)

- Add `--dry-run` to both process-instance destructive commands with examples that show direct-key and search-mode preview.
- Introduce a small shared command payload for dry-run output containing requested keys, resolved roots, affected family keys, requested count, root count, affected count, traversal outcome, warning text, missing ancestors, and an explicit mutation-submitted flag set to false.
- For structured search-mode dry runs that process multiple pages, return an aggregate summary with nested per-page previews rather than only a flat list or only de-duplicated aggregate keys.
- Refactor the existing cancel/delete preflight helpers so both real execution and dry run call one plan-building path, then branch only after scope calculation.
- For keyed dry run, render the single calculated plan and exit successfully when actionable scope exists.
- For search-mode dry run, process each selected page through the same page action path and aggregate dry-run payloads into one structured summary with nested per-page previews.
- Preserve `printDryRunExpansionWarning` semantics for human output while also including missing ancestor details in structured output.
- Ensure no dry-run path calls `CancelProcessInstances`, `DeleteProcessInstances`, confirmation prompts, or waiter behavior.
- Update `cmd/process_api_stub_test.go` support as needed so command tests fail on unexpected mutation calls.
- Update README examples and run `make docs-content` after command metadata changes.
- Update the active plan pointer in `AGENTS.md` so downstream Speckit steps target this feature.

## Phase 2: Task Planning Approach

Task generation should break the work into dependency-ordered slices:

1. Establish the command-level dry-run output contract and focused facade/command tests around existing `DryRunCancelOrDeletePlan` data.
2. Add `--dry-run` flag wiring and help/examples for `cancel process-instance`, then implement keyed cancel dry-run without mutations or prompts.
3. Add the matching delete keyed dry-run path using the same shared helper and tests.
4. Extend search-mode/paged dry-run handling for cancel/delete while preserving existing page scope calculation and limit behavior.
5. Add orphan-parent partial and unresolved regression coverage for human and structured output.
6. Update README, regenerate CLI docs, and run targeted tests plus final `make test`.

## Post-Design Constitution Check

- **Operational Proof Over Intent**: Still passes. The design proves planned scope and explicitly avoids mutation/wait paths during preview.
- **CLI-First, Script-Safe Interfaces**: Still passes. The dry-run flag and structured payload are stable CLI-facing surfaces, including an aggregate-plus-per-page shape for paged structured previews.
- **Tests and Validation Are Mandatory**: Still passes with command/facade coverage and final repository validation.
- **Documentation Matches User Behavior**: Still passes with README and generated CLI docs updates.
- **Small, Compatible, Repository-Native Changes**: Still passes. The work stays inside existing command/facade preflight layers.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |
