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
