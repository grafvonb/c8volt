# Implementation Plan: Update User-Task Variables

**Branch**: `codex/326-update-user-task-vars` | **Date**: 2026-09-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/326-update-user-task-vars/spec.md`, issue #326.

## Summary

Add `update user-task` with aliases `ut`, `uts`, and `user-tasks`, following `update pi` grammar and controls. Read each explicit task's complete effective variables; update existing names at their returned scopes and create absent names in the task's element scope. Freeze a deduplicated plan before confirmation, submit scope-local writes, and confirm through the same effective-variable reader. Keep orchestration in `internal/services/usertask`, generated request mapping in the existing variable adapters, and public facade methods mechanical.

No task-attribute mutations, search selection, scope flags, new dependencies, client regeneration, or unrelated PI refactor are planned.

## Technical Context

**Language/Version**: Go 1.26; repository toolchain go1.26.2.

**Primary Dependencies**: Existing Cobra/pflag, facade options/errors, generated Camunda clients, `toolx/pool`, service retry helpers, configured backoff. No dependency additions.

**Storage**: Existing Camunda variable scopes; in-memory plans and results only.

**Testing**: Existing Go/testify, HTTP fixtures, command subprocess helpers, and real-terminal runners. Targeted suites, then the issue-required `make test` race suite for the completed cross-package runtime change.

**Target Platform**: Existing supported CLI platforms and Camunda 8.8, 8.9, 8.10. Explicit unsupported operation on 8.7; unchanged default runtime version.

**Project Type**: Existing CLI and public Go facade library.

**Performance Goals**: Reuse complete variable paging; one logical mutation per unique scope/name, grouped into one payload per scope. Bounded execution using existing worker policy; no new latency target, discovery fan-out, or total-count pass.

**Constraints**: Explicit-key authorization, stdout purity, truthful no-op/accepted/confirmed results, same flag compatibility as PI, immutable target selection after confirmation, no rollback promise. Apply [repository rules](../../AGENTS.md) and [Ralph rules](../ralph-implementation-rules.md).

**Scale/Scope**: One or more explicit keys, same payload for all tasks. Memory proportional to selected tasks, their complete variable views, and unique target scopes; use existing bounds and context cancellation.

## Constitution Check

| Principle | Before research | After design |
|---|---|---|
| Operational proof | Pass: ordinary execution must confirm value and scope | Pass: service-owned confirmation; no-wait reports submission only |
| CLI-first, script-safe | Pass: PI baseline and explicit UT scope semantics | Pass: one-envelope errors/results, zero-byte no-op keys, stderr prompts |
| Proportionate validation | Pass: documentation checks now, executable tests later | Pass: focused package checks then full race suite for shared service/facade/concurrency changes |
| Documentation matches behavior | Pass: help, metadata, README and generated docs required | Pass: included in delivery and quickstart, no generator run for planning alone |
| Small compatible changes | Pass: existing areas and dependencies | Pass: one focused update workflow within usertask, scope-write extension in variable, no general framework |

No gate exceptions. The old PI command-local planning and its no-op rendering are not copied where they conflict with current rules. Existing PI behavior remains unchanged. The constitution does not require runtime tests for these planning documents; issue #326's full test requirement applies to implementation.

## Project Structure

### Documentation (this feature)

```text
specs/326-update-user-task-vars/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cli.md
│   └── facade-service.md
└── checklists/requirements.md
```

`tasks.md` is produced by the subsequent tasks workflow, not this planning phase.

### Source Code (repository root)

```text
cmd/
  update.go                              # parent help/examples
  update_usertask.go                      # command, flags, validation, dispatch
  update_usertask_variables.go            # payload/confirmation policy if needed
  cmd_views_usertask_update.go            # all final/preview formatting
  command_contract_test.go                # metadata and flags
c8volt/
  client.go                              # compose service dependency
  task/{api.go,model.go,convert.go,client.go}
internal/
  domain/usertask_update.go               # plan, scope targets, outcomes
  services/usertask/
    update.go                            # composed service, planning/execution
    update_wait.go                       # scope-aware confirmation
  services/variable/
    api.go
    v87/variables.go                     # explicit unsupported write
    v88/variables.go
    v89/variables.go
    v810/variables.go                     # local scope writes
README.md
docs/cli/                                # regenerated from command metadata
```

Tests live next to the changed owners, using focused `update_*_test.go` files where helpful. Existing variable adapter generated-client contracts already expose `CreateElementInstanceVariablesWithResponse`. Update interface doubles and constructor fixtures affected by the new methods. No new mode is required: waiting is service behavior, not a CLI mode.

**Structure Decision**: Reuse the existing task and variable areas. A small composed update service in the usertask package owns task-reader, variable-writer, configuration, and logger dependencies; this avoids coupling low-level read adapters to workflow state and keeps configuration out of public operation signatures. A compatible `task.NewWithVariableUpdates` constructor supplements the existing constructor following the nearby process facade pattern.

## Design and Delivery

1. Add domain plan/results and the scoped variable-write operation; implement v8.10 first, then explicit v8.9/v8.8 mappings and v8.7 unsupported behavior. Existing PI write semantics remain unchanged.
2. Implement the usertask plan/execution service with complete discovery, scope/name deduplication, frozen targets, worker scheduling, and scope-aware confirmation. Expose thin facade methods and wire the composed dependency through `c8volt.New`.
3. Add the command and focused views. Reuse PI input parsing by extracting only the genuinely shared payload functions into a concrete variable-payload file with unchanged PI behavior; no generic command framework. Match PI JSON/confirmation restrictions with task-specific wording.
4. Add end-to-end command coverage for the full output/no-op/failure/terminal matrix, metadata, and PI regressions. Update help and README, regenerate docs, then run the completed runtime validation.

Execution is consolidated into eight tested work units in [tasks.md](tasks.md). Build single-task and multi-task planning/execution together; the three specification stories remain acceptance criteria rather than separate rebuilds of the same workflow. Context review, fixtures, formatting, and focused checks are included in each substantive unit.

Research choices are recorded in [research.md](research.md); schema and lifecycle details in [data-model.md](data-model.md); exact interfaces and rendering in [contracts](contracts/cli.md); executable validation in [quickstart.md](quickstart.md).

## Complexity Tracking

No constitutional violations. Shared scope targets require an association from each task to its target scopes so one write can serve multiple tasks and failures remain truthful. A flat target list plus task references is sufficient; no graph engine, persistent plan store, rollback framework, or new CLI surface is needed.
