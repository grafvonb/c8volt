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
- `cmd/run_processinstance.go` now routes `run pi` JSON output through the shared mutation envelope while routing non-JSON output through `listProcessInstancesView`, preserving observed process-instance state in normal output.
- `run pi --keys-only` reaches the shared process-instance keys-only renderer through `listProcessInstancesView`; it must not emit `found:`, state text, or JSON envelope fields on stdout.
- `expect pi` state waits remain strict and should keep using explicit user-requested states, not the broader creation-confirmation state set.
- `README.md`, `cmd/*` help text, `docsgen`, and generated `docs/cli/*` must stay aligned for user-facing command changes.
- `specs/ralph-implementation-rules.md` is binding for every Ralph implementation iteration and should be read before nearby source/tests for the current work unit.
- `internal/domain/state.go` now owns `ObservableProcessInstanceCreationStates`, which returns a copy of the shared `ACTIVE`, `COMPLETED`, `CANCELED`, and `TERMINATED` confirmation state set for later process-instance services.
- `internal/domain.ProcessInstanceCreation` carries the observed creation-confirmation state, and `c8volt/process` maps it onto the public `ProcessInstance.State` for later command rendering.
- Camunda 8.7 creation responses still do not provide a usable process-instance key; key-based v8.7 confirmation uses the tenant-filtered Operate search path when a caller already has a key.

## Validation Log

- 2026-05-23 19:36 CEST: `go test ./internal/domain -run 'State' -count=1` passed.
- 2026-05-23 19:36 CEST: `go test ./internal/domain -count=1` passed.
- 2026-05-23 19:44 CEST: `go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'CreateProcessInstance|WaitForProcessInstance|GetProcessInstanceStateByKey' -count=1` passed.
- 2026-05-23 19:44 CEST: `go test ./internal/domain ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'State|CreateProcessInstance|WaitForProcessInstance' -count=1` passed.
- 2026-05-23 19:44 CEST: `go test ./c8volt/process -run 'CreateProcessInstances' -count=1` passed.
- 2026-05-23 19:44 CEST: `go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1` passed.
- 2026-05-23 19:49 CEST: `go test ./cmd -run 'RunProcessInstance|ProcessInstancesView|DeployProcessDefinition|EmbedDeploy' -count=1` passed.
- 2026-05-23 19:56 CEST: `go test ./cmd ./docsgen -run 'RunProcessInstance|ExpectProcessInstance|CommandContract|Capabilities|Generated' -count=1` passed.

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
## Iteration 3 - 2026-05-23 19:44 CEST
**User Story**: User Story 1 - Confirm Fast Process Instance Runs
**Tasks Completed**:
- [x] T005: Add v8.7 creation confirmation tests for ACTIVE, COMPLETED, terminal observable state, and absent/not-found rejection
- [x] T006: Add v8.8 creation confirmation tests for ACTIVE, COMPLETED, terminal observable state, and absent/not-found rejection
- [x] T007: Add v8.9 creation confirmation tests for ACTIVE, COMPLETED, terminal observable state, and absent/not-found rejection
- [x] T008: Use the shared observable creation state set when v8.7 waits after process instance creation
- [x] T009: Use the shared observable creation state set when v8.8 waits after process instance creation
- [x] T010: Use the shared observable creation state set when v8.9 waits after process instance creation
- [x] T011: Preserve the observed state and confirmation timestamp on returned created process instances
- [x] T012: Run targeted service validation
- [x] T013: Update progress with US1 implementation notes and validation results
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- c8volt/process/convert.go
- internal/domain/processinstance.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v87/service_test.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/service.go
- internal/services/processinstance/v89/service_test.go
- specs/225-run-observable-keys/tasks.md
- specs/225-run-observable-keys/progress.md
**Learnings**:
- v8.8 and v8.9 creation waits can accept `ACTIVE`, `COMPLETED`, and terminal observable states by reusing `ObservableProcessInstanceCreationStates`; not-found observations still retry and fail instead of confirming creation.
- v8.7 cannot observe a newly created instance without a key from the create response, so the existing no-key compatibility path remains unchanged; key-based state confirmation is now backed by tenant-filtered Operate search.
- Observed confirmation state is stored on the internal creation result and mapped through the public process facade for subsequent `run pi` rendering work.
- Validation passed with the targeted US1 command, focused facade conversion coverage, and full v8.7/v8.8/v8.9 process-instance service package tests.
---
## Iteration 4 - 2026-05-23 19:49 CEST
**User Story**: User Story 2 - See The Observed Process Instance State
**Tasks Completed**:
- [x] T014: Add `run pi` normal-output state rendering tests
- [x] T015: Add `run pi` JSON envelope state rendering tests
- [x] T016: Add process-instance list view state regression coverage
- [x] T017: Render `run pi` non-JSON results through the existing process-instance list view
- [x] T018: Ensure `run pi` JSON output keeps the shared full-contract envelope and includes observed process instance `state`
- [x] T019: Verify `deploy --run` and `embed deploy --run` continue without separate state expectation flags
- [x] T020: Run targeted command validation
- [x] T021: Update progress with US2 implementation notes and validation results
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/run_processinstance.go
- cmd/run_test.go
- cmd/cmd_views_get_test.go
- cmd/deploy_test.go
- cmd/embed_test.go
- specs/225-run-observable-keys/tasks.md
- specs/225-run-observable-keys/progress.md
**Learnings**:
- `renderCommandResult` is intentionally JSON-envelope-only, so normal `run pi` output must call the shared process-instance list view after creation.
- Keeping `run pi` JSON on `renderCommandResult` preserves the state-changing accepted/succeeded envelope behavior while still carrying observed state in payload items.
- `deploy --run` and `embed deploy --run` remain shared creation-confirmation consumers and do not expose a run-specific state expectation flag.
---
## Iteration 5 - 2026-05-23 19:56 CEST
**User Story**: User Story 3 - Pipe Created Keys Into Strict Expectations
**Tasks Completed**:
- [x] T022: Add `run pi --keys-only` output tests
- [x] T023: Add command contract/capabilities coverage for `run pi` keys-only support
- [x] T024: Add strict `expect pi` regression coverage for mismatched explicit state expectations
- [x] T025: Add generated docs coverage for the `run pi --keys-only | expect pi --state <state> -` example
- [x] T026: Ensure `run pi --keys-only` uses the shared process-instance keys-only renderer
- [x] T027: Update `run pi` help text and examples with completed and active pipeline patterns
- [x] T028: Update README pipeline examples and wording
- [x] T029: Regenerate generated CLI documentation
- [x] T030: Run targeted validation
- [x] T031: Update progress with US3 implementation notes and validation results
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- README.md
- cmd/capabilities_test.go
- cmd/command_contract_test.go
- cmd/expect_test.go
- cmd/run.go
- cmd/run_processinstance.go
- cmd/run_test.go
- docs/cli/c8volt_run.md
- docs/cli/c8volt_run_process-instance.md
- docs/index.md
- docsgen/main_test.go
- specs/225-run-observable-keys/tasks.md
- specs/225-run-observable-keys/progress.md
**Learnings**:
- US2's `renderRunProcessInstanceResult` routing already lets keys-only mode reuse the shared process-instance key renderer; the implementation work made that behavior explicit with tests and docs.
- Command capability discovery infers keys-only support from inherited root flags after Cobra flag reset, so contract coverage should inspect the live `run process-instance` capability.
- `expect pi` mismatched explicit states still timeout/fail instead of accepting another observable lifecycle state, preserving the strict pipeline boundary.
- Generated CLI docs were refreshed with `make docs-content`, which also synced `docs/index.md` from the README.
---
