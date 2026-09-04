# Implementation Plan: Force-Cleanup Progress During Process-Definition Purge

**Branch**: `develop` (unchanged) | **Date**: 2026-09-04 | **Spec**: [spec.md](spec.md)

**Feature ID**: `291-force-cleanup-progress`

**Input**: Feature specification from `specs/291-force-cleanup-progress/spec.md`, issue #291.

The setup script reports the resolved feature label `291-force-cleanup-progress` as BRANCH; `git branch --show-current` confirms the actual checkout is `develop`. No branch creation or switch is part of this plan.

## Summary

Expose the real cancellation, draining, process-instance history deletion, and process-definition deletion stages of forced all-process-definitions purge. Add service-owned stage-entry facts and mechanically carry them through public callbacks. Replace this command's definition-only progress wrapper with a focused coordinator that reuses existing aggregation and output support, maintains one activity and one 10-second pacing lifetime, and keeps counters separate by stage. Preserve all backend operations, safeguards, final results, and machine contracts.

## Technical Context

**Language/Version**: Go 1.26; repository toolchain `go1.26.2`.

**Primary Dependencies**: Existing Cobra 1.10.2, pflag 1.0.10, repository progress policy/renderers, `toolx/logging`, and existing service callbacks. No new dependencies.

**Storage**: Transient in-memory stage state only; no new persistence or report fields.

**Testing**: Go testing and testify 1.11.1; `testx/activitysink`, IPv4 fake HTTP servers, synchronized collectors, deterministic clocks, subprocess tests for exit paths, and race detection.

**Target Platform**: Existing CLI terminal and scripted environments. APD full-history capability remains supported on existing 8.9/8.10 paths; earlier-version rejection remains unchanged.

**Project Type**: Go CLI with public facade libraries and version-neutral service workflows.

**Performance Goals**: Update activity on each matching completion; one paced informational milestone per 10 seconds; immediate failure warnings; zero new backend calls or timers. Store at most three mutation aggregates plus one current waiting-stage marker.

**Constraints**: Separate root-tree and definition totals; no guessed affected coverage; one workflow activity; synchronized callback processing; no CLI-owned backend loops; no generated-client changes; preserve requests, workers, ordering, confirmation, and output modes.

**Scale/Scope**: One existing command and its nested force-cleanup callback path, including hundreds of affected instances. No new supported mode or resource operation.

## Constitution Check

Initial review before design: PASS. Post-design review: PASS, subject to the implementation validation gates below.

| Principle / gate | Design evidence | Required implementation proof |
|---|---|---|
| Operational proof over intent | Stage entries report execution, while completion dispositions retain submitted/confirmed/failed semantics. Draining remains a real service wait. | No premature confirmed wording; retain deletion absence verification and early-failure tests. |
| CLI-first, script-safe interfaces | Existing flags, final rendering, envelopes, reports, and supported modes stay intact. | Nested mode matrix, prompt checks, final-result and exit regressions. |
| Mandatory tests and validation | Tests at domain/conversion, service, coordinator, and real command paths. | Targeted checks first, then `make test` before implementation completion or commit. |
| Documentation matches behavior | Update APD help, README, and the authored ops guide, then regenerate documentation. | `make docs-content`, help assertions, review generated differences. |
| Small compatible repository-native changes | Add one stage event and focused command lifecycle; reuse reducers/policies/renderers. | Review package boundaries, single-scope reporter regressions, no new dependencies. |
| Repository layering and file cohesion | Service emits facts; facades map; command renders. Coordinator lives in its own progress file. | Declaration/ownership review; no worker, retry, polling, or discovery mechanics moved upward. |

No conflict with `specs/ralph-implementation-rules.md` was found. Its mandatory reading and implementation discipline remain part of future tasks. Any Ralph invocation must include `--implementation-context specs/ralph-implementation-rules.md`.

## Project Structure

### Documentation (this feature)

```text
specs/291-force-cleanup-progress/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/force-cleanup-progress.md
```

`tasks.md` is the next phase's output and is not created by this command.

### Source Code (repository root)

```text
internal/domain/ops_progress.go                   # additive stage fact and envelope
internal/domain/ops_progress_test.go
internal/services/processdefinition/delete.go     # actual stage entry boundaries
internal/services/processdefinition/delete_test.go
internal/services/ops/all_process_definitions_purge_test.go
c8volt/ops/progress_model.go                      # public stage payload
c8volt/ops/convert.go                             # mechanical conversion
c8volt/ops/model_test.go
c8volt/ops/client_test.go
c8volt/foptions/options.go                        # second public callback mapping
c8volt/foptions/options_test.go
cmd/ops_purge_all_processdefinitions.go            # wiring, help, confirmation
cmd/ops_purge_all_processdefinitions_progress.go   # new stage coordinator
cmd/ops_purge_all_processdefinitions_progress_test.go
cmd/ops_purge_all_processdefinitions_test.go       # real command regression path
cmd/ops_semantic_progress.go                      # small shared aggregate reducer
cmd/ops_semantic_progress_test.go
README.md
docs/ops/purge-all-process-definitions.md
```

Existing `cmd/ops_progress_mode.go`, `cmd/ops_progress_render.go`, `cmd/ops_progress_milestones.go`, `cmd/cmd_views_ops_purge_all_processdefinitions.go`, and `toolx/logging` remain reuse points. Inspect actual nearby file names before editing. Generated CLI markdown and homepage are refreshed with `make docs-content`, not hand-edited.

**Structure Decision**: Keep backend state transitions in the existing process-definition service. Add no service interface, versioned adapter, generated client, global registry, or new CLI mode. Shared changes remain limited to callback representation and reducer reuse; APD-specific lifecycle and wording stay under its focused command progress file.

## Phase 0: Research Result

[research.md](research.md) records evidence, decisions, and rejected alternatives. Existing nested completion facts are sufficient for all mutation counts; additive stage entries supply the missing pre-completion visibility and drain state. All unknowns are resolved.

## Phase 1: Design

### Service events and public mapping

Add `stage` as an event kind with a typed stage payload, optional work total, and optional planned affected count. The [data model](data-model.md) and [contract](contracts/force-cleanup-progress.md) define phase identifiers and invariants. Emit entries at the owning service boundaries before cancellation, draining, history deletion, and the first definition-delete request. Cover both definition deletion paths: the force/preplanned `DeleteProcessDefinitionResources` path and the ordinary `DeleteProcessDefinitions` worker path through `deleteProcessDefinition`. For the latter, use a private per-run synchronized once-only entry hook immediately before the first validated resource deletion, preserving existing per-item checks and scheduling. Do not reroute ordinary deletion through the force/preplanned path. Skip entries for work never entered. Preserve existing FrozenScope and Completion emissions and do not count both. Extend both ops and foptions public conversion paths without serializing progress into final command results.

### Command lifecycle and activity

Construct a dormant coordinator with the request callback. Discovery and preflight continue using existing renderers. Remove the eager definition-only start after confirmation. Keep the existing outer activity wrapper for dry-run/preview calls only. For a real execution call, the coordinator owns one generic workflow activity from call entry, including discovery/revalidation. The first actual stage-entry event relabels that activity and starts the mutation pacing clock. Replace the frozen-candidate shortcut that currently labels revalidation as definition deletion.

The coordinator serializes stage selection, aggregate reduction, transient updates, and durable emissions. Explicit stage entry alone changes current stage. Matching completion facts update stage-local state; unrelated/empty phases and non-completion counters do not increment work or replace current activity. Do not open the old outer workflow activity around a real execution call in addition to the coordinator. Preview activity stops before confirmation; real execution owns a fresh activity handle. Stop the mutation activity before final result/error rendering on every return path, without relying on deferred cleanup after an exit helper.

### Counters, pacing, and closure

Use unique root totals supplied by the service for cancellation and history deletion, and definition totals for the final stage. Extract the existing aggregate reducer for shared use; retain nil affected coverage and failure semantics. Planned affected scope is distinct from completed affected work.

Use one clock across stage transitions. Default milestones follow matching completions at the existing 10-second interval; failures remain immediate and verbose/debug uses per-item outcomes. Keep historical dirty stage snapshots. Close emits at most one compact final record covering unreported stage aggregates if durable progress was activated, without changing current activity or declaring success. Stage transitions do not flush. The detailed deterministic rules are in the contract.

### Validation design and traceability

| Requirement group | Primary proof |
|---|---|
| FR-001–FR-007; SC-001/SC-002 | Real nested service sequence, callback conversion, barrier-controlled command activity, shared-root totals and drain visibility. |
| FR-008–FR-010; SC-003/SC-004 | Fake-clock tests across stages, exact 10-second boundary, idle drain, failure before threshold, verbose/debug outcomes, one final historical record, idempotent close. |
| FR-011–FR-013; SC-005/SC-006 | JSON/automation/quiet policy, unsupported keys-only behavior, confirmation and dry-run paths, request/worker/result/report regression assertions. |
| FR-014/FR-015 | Actual Cobra-to-facade-to-service command tests, service failures/interruptions, documentation regeneration and full race-enabled suite. |

Required service coverage includes non-force stage entry on the ordinary path and preserves the first serial definition deletion probe, request shape, existing preview rechecks, wait/no-wait semantics, deduplication, worker arguments, fail-fast behavior, and skipped stages after error. Cross-stage count/clock tests must fail if only definition completions are forwarded. Do not make flag-mutating command tests parallel.

Use [quickstart.md](quickstart.md) for runnable validation. Artifact validation is performed during planning; implementation tests and backend runs are future work, not claimed as passed here.

## Complexity Tracking

No constitution violations or exceptions. The added stage event distinguishes unquantified waiting from completion; the focused coordinator is needed because existing single-scope reporters each own their own activity and pacing clock. Extending those reporters into a general workflow framework is not justified.
