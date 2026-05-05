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
