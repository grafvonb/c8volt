# Progress: Run Confirmation Observes Real Process Instance States

**Issue**: [#225](https://github.com/grafvonb/c8volt/issues/225)
**Branch**: `225-run-observable-keys`
**Implementation Context**: Use `--implementation-context specs/ralph-implementation-rules.md` for every Ralph iteration.

## Status

- Specification generated: complete
- Clarification gate: complete; no critical ambiguities detected worth formal clarification
- Architecture grounding: reused existing architecture memory; no refresh needed
- Planning artifacts: complete
- Task generation: complete
- Ralph launch: pending explicit budget confirmation

## Codebase Patterns

- `internal/services/processinstance/v87`, `v88`, and `v89` own version-specific process-instance creation and wait behavior.
- `internal/services/processinstance/waiter` owns shared polling, absent/not-found handling, backoff, fail-fast worker fan-out, and state expectation checks.
- `cmd/run_processinstance.go` owns `run pi` flags, validation, facade calls, and result rendering dispatch.
- `cmd/cmd_views_get.go` already renders `process.ProcessInstances` in normal, JSON, and keys-only modes through shared helpers.
- `cmd/cmd_views_contract.go` renders shared JSON envelopes for full-contract commands and intentionally does not render non-JSON command results.
- `README.md`, `cmd/*` help text, `docsgen`, and generated `docs/cli/*` must stay aligned for user-facing command changes.
- `specs/ralph-implementation-rules.md` is binding for every Ralph implementation iteration and should be read before nearby source/tests for the current work unit.
- `internal/domain/state.go` now owns `ObservableProcessInstanceCreationStates`, which returns a copy of the shared `ACTIVE`, `COMPLETED`, `CANCELED`, and `TERMINATED` confirmation state set for later process-instance services.

## Validation Log

- 2026-05-23 19:36 CEST: `go test ./internal/domain -run 'State' -count=1` passed.
- 2026-05-23 19:36 CEST: `go test ./internal/domain -count=1` passed.

## Residual Risks

- The exact v8.7 behavior when creation returns no usable process instance key must be preserved.
- Keys-only mode must not emit human summaries or activity text on stdout.
- `expect pi` strictness must remain separate from run confirmation semantics.

---
## Iteration 1 - 2026-05-23 19:34 CEST
**User Story**: Phase 1 Setup (Shared Context)
**Tasks Completed**:
- [x] T001: Read implementation context, feature artifacts, task list, and progress log
- [x] T002: Record initial codebase pattern notes and current work-unit status
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/225-run-observable-keys/tasks.md
- specs/225-run-observable-keys/progress.md
**Learnings**:
- Active traceability is persisted as GitHub issue #225 in spec, plan, tasks, and progress artifacts.
- Phase 2 foundational tasks are the next incomplete work unit and block user story implementation.
- The feature must keep broader run confirmation semantics separate from strict `expect pi` state assertions.
---
## Iteration 2 - 2026-05-23 19:36 CEST
**User Story**: Phase 2 Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T003: Add a documented helper for observable process-instance creation confirmation states
- [x] T004: Add state helper tests for observable creation states
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- internal/domain/state.go
- internal/domain/state_test.go
- specs/225-run-observable-keys/tasks.md
- specs/225-run-observable-keys/progress.md
**Learnings**:
- Creation confirmation semantics are now centralized in `internal/domain` so v8.7, v8.8, and v8.9 services can share the same accepted state set.
- `ObservableProcessInstanceCreationStates` returns a copy to avoid accidental mutation of shared confirmation semantics.
- Validation passed with `go test ./internal/domain -run 'State' -count=1` and `go test ./internal/domain -count=1`.
---
