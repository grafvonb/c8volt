# Ralph Progress Log

Feature: 170-process-incident-expect
Started: 2026-05-05 08:05:19

## Codebase Patterns

- `cmd/expect_processinstance.go` is currently state-only: it requires `--state` through Cobra, parses with `process.ParseStates`, reads stdin keys through `readKeysIfDash`, merges them with `mergeAndValidateKeys`, then delegates to `cli.WaitForProcessInstancesState`.
- Process-instance expectation command metadata is already read-only and automation-unsupported through `setCommandMutation`, `setContractSupport`, and `setAutomationSupport`; future command changes should preserve those calls.
- Command invalid-input tests use helper subprocess entrypoints in `cmd/expect_test.go` with `testx.RunCmdSubprocess`, then assert exit codes and envelope fields for JSON mode.
- Facade waits map public `process.States` to domain `d.States`, pass facade options through `options.MapFacadeOptionsToCallOptions`, and convert domain `StateResponse` values to public reports in `c8volt/process/client.go` and `bulk.go`.
- Shared process-instance waiting lives in `internal/services/processinstance/waiter`; it owns polling, backoff, max-retry, activity logging, `--state absent` not-found handling, and canceled/terminated compatibility.
- Versioned process-instance services expose the same API surface and delegate state waits to the shared waiter; v8.7 maps incident through the Operate `Incident` pointer, while v8.8/v8.9 map `HasIncident`.
- Process-instance expectation docs are driven by command examples/help and generated CLI docs; README and `docs/index.md` currently list only state-based `expect pi` examples.
- Public process facade expansion requires matching methods on command and facade test stubs (`cmd/process_api_stub_test.go`, `c8volt/process/client_test.go`) because both assert full API conformance.
- Combined process-instance expectation contracts now use public request/report types in `c8volt/process` and mirrored domain request/response types in `internal/domain` because internal service APIs cannot import the public facade package.
- Foundational combined waiter scaffolding delegates state-only expectation requests to the existing state waiter and returns unsupported for incident requests until the US1 wait loop is implemented.
- Incident expectation waits use full `GetProcessInstance` polling through the shared waiter so the matcher can evaluate both state and the observed incident marker from the same fetched instance.
- `expect pi --incident true` must bypass Cobra's old unconditional required `--state` flag and instead rely on explicit "at least one expectation" validation before reading/merging keys.
- Command-level incident waits route through `WaitForProcessInstancesExpectation`; state-only waits still use `WaitForProcessInstancesState` to preserve existing report shape and state-only behavior.
- Incident false semantics are enforced by requiring a present process instance before `processInstanceExpectationMatches` can succeed; missing instances retry until timeout/max-retries instead of mapping to incident-free.
- Facade incident reports preserve observed `false` values through non-nil incident pointers in `fromDomainIncidentExpectation`, so JSON output can distinguish `false` from an omitted incident observation.
- Combined state and incident waits require both expectations to match on the same present process-instance fetch; canceled/terminated compatibility still flows through `stateIn`/`statesEquivalent`, while state-only command execution remains on `WaitForProcessInstancesState`.

---

---
## Iteration 1 - 2026-05-05 08:06:38 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Inspect current process-instance expectation command validation, stdin key handling, and read-only/automation metadata
- [x] T002: Inspect existing expect command tests and helper subprocess patterns
- [x] T003: Inspect process facade wait APIs and reports
- [x] T004: Inspect shared process-instance waiter behavior and tests
- [x] T005: Inspect versioned process-instance service contracts and incident mappings
- [x] T006: Inspect process-instance expectation docs
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/170-process-incident-expect/tasks.md
- specs/170-process-incident-expect/progress.md
**Learnings**:
- The command-layer change should start by replacing the required `--state` constraint with at-least-one expectation validation after strict `--incident` parsing is available.
- A combined expectation path can reuse the existing waiter polling/backoff shape, but incident matching needs to fetch full process instances so missing instances do not satisfy `--incident false`.
- Future tests need stub API methods added in both command and facade test stubs once combined wait methods are introduced.
---

---
## Iteration 2 - 2026-05-05 08:15:31 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**:
- [x] T007: Add an incident expectation value type and strict parser accepting only `true` and `false`
- [x] T008: Add combined process-instance expectation request/report types for optional state and incident expectations
- [x] T009: Extend the public process facade API with combined process-instance expectation wait methods
- [x] T010: Extend the internal process-instance service API with combined expectation wait methods
- [x] T011: Extend versioned service contracts to include combined expectation wait methods
- [x] T012: Add shared matcher helpers for state and incident expectations
- [x] T013: Add command-level flag storage and strict validation scaffolding for `--incident`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/expect_processinstance.go
- cmd/process_api_stub_test.go
- c8volt/process/api.go
- c8volt/process/bulk.go
- c8volt/process/client.go
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- internal/domain/processinstance.go
- internal/services/processinstance/api.go
- internal/services/processinstance/waiter/waiter.go
- internal/services/processinstance/v87/contract.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/service.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/service.go
- specs/170-process-incident-expect/tasks.md
- specs/170-process-incident-expect/progress.md
**Learnings**:
- Registering a new facade method requires test stub updates in every package that asserts the full facade or service API.
- The `--incident` flag can be parsed strictly now, but the command should not switch away from the existing state wait path until US1 wires the combined waiter behavior.
- Use `GOCACHE=/tmp/c8volt-go-build-cache` for Go tests in this sandbox; the default macOS Go build cache is outside the writable roots.
---

---
## Iteration 3 - 2026-05-05 08:22:40 CEST
**User Story**: User Story 1 - Wait For Incident Presence
**Tasks Completed**:
- [x] T014: Add command test for `c8volt expect pi --key <key> --incident true` waiting until incident true
- [x] T015: Add facade test proving incident true expectation maps through the process client
- [x] T016: Add waiter test proving incident true waits across false-to-true polling
- [x] T017: Implement combined expectation waiting for a single process instance
- [x] T018: Implement combined expectation waiting for multiple process instances with existing worker controls
- [x] T019: Delegate versioned services to the shared combined waiter
- [x] T020: Map combined expectation wait results through the public process facade
- [x] T021: Wire `expect process-instance` to call the combined expectation wait path when `--incident true` is provided
- [x] T022: Run targeted US1 Go tests and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/expect_processinstance.go
- cmd/expect_test.go
- c8volt/process/client_test.go
- internal/services/processinstance/waiter/waiter.go
- internal/services/processinstance/waiter/waiter_test.go
- specs/170-process-incident-expect/tasks.md
- specs/170-process-incident-expect/progress.md
**Learnings**:
- Full process-instance polling is required for incident expectations; state-only polling cannot observe the incident marker directly.
- Versioned v8.7/v8.8/v8.9 services already delegate combined expectation waits to the shared waiter, so US1 needed the shared waiter behavior rather than version-specific loops.
- Validation run: `GOCACHE=/tmp/c8volt-go-build-cache go test ./cmd ./c8volt/process ./internal/services/processinstance/waiter -run 'TestExpect|TestClient_.*Incident|TestWait.*Incident' -count=1`.
- Additional compile proof: `GOCACHE=/tmp/c8volt-go-build-cache go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -count=1`.
---

---
## Iteration 4 - 2026-05-05 08:25:47 CEST
**User Story**: User Story 2 - Wait For Incident Absence
**Tasks Completed**:
- [x] T023: Add command test for `c8volt expect pi --key <key> --incident false` succeeding for a present incident-free instance
- [x] T024: Add waiter test proving a missing process instance does not satisfy `--incident false`
- [x] T025: Add facade test proving incident false expectation preserves present-instance semantics
- [x] T026: Update incident matcher behavior so false requires a present process instance
- [x] T027: Ensure facade report mapping preserves observed incident false status
- [x] T028: Run targeted US2 Go tests and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/expect_test.go
- c8volt/process/client_test.go
- internal/services/processinstance/waiter/waiter_test.go
- specs/170-process-incident-expect/tasks.md
- specs/170-process-incident-expect/progress.md
**Learnings**:
- The existing combined expectation matcher already blocks missing instances for `--incident false`; US2 locked that behavior with explicit command, waiter, and facade tests.
- Validation run: `GOCACHE=/tmp/c8volt-go-build-cache go test ./cmd ./c8volt/process ./internal/services/processinstance/waiter -run 'TestExpect|TestClient_.*Incident|TestWait.*Incident' -count=1`.
---

---
## Iteration 5 - 2026-05-05 08:30:04 CEST
**User Story**: User Story 3 - Combine State And Incident Expectations
**Tasks Completed**:
- [x] T029: Add command test for combined `--state active --incident true` waiting until both match
- [x] T030: Add waiter tests preserving `--state absent` and canceled/terminated compatibility with combined expectation matching
- [x] T031: Add facade test for combined state and incident expectation requests
- [x] T032: Update combined matcher logic to require all requested expectations for each selected instance
- [x] T033: Preserve state-only wait delegation and reporting compatibility
- [x] T034: Update command status/log messages for incident-only and combined expectations
- [x] T035: Run targeted US3 Go tests and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/expect_processinstance.go
- cmd/expect_test.go
- c8volt/process/client_test.go
- internal/services/processinstance/waiter/waiter_test.go
- specs/170-process-incident-expect/tasks.md
- specs/170-process-incident-expect/progress.md
**Learnings**:
- The existing combined matcher already enforced all requested expectations, so US3 primarily added regression coverage around mismatched state/incident attempts and compatibility semantics.
- Command logs now distinguish incident-only waits from combined state and incident waits without changing state-only command output.
- Validation run: `GOCACHE=/tmp/c8volt-go-build-cache go test ./cmd ./c8volt/process ./internal/services/processinstance/waiter -run 'TestExpect|TestClient_.*Wait|TestWaitForProcessInstanceState|TestWait.*Expectation' -count=1`.
---
