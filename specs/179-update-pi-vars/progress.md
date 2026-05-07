# Ralph Progress Log

Feature: 179-update-pi-vars
Started: 2026-05-07 17:46:04

## Codebase Patterns

- Update process-instance target selection should let `mergeAndValidateKeys(...).Unique()` feed the facade bulk method directly; command-local single-key caps conflict with shared repeated-flag and stdin key behavior.
- CLI stdin-key tests can replace `os.Stdin` with an `os.Pipe`; `readKeysIfDash` intentionally checks the real stdin file descriptor with `term.IsTerminal`.
- Facade bulk update behavior is owned by `UpdateProcessInstancesVariables`, which deduplicates `typex.Keys`, computes worker count with `toolx.DetermineNoOfWorkers`, and maps `fail-fast`/`no-worker-limit` through shared facade options.
- Foundation command shells can expose Cobra metadata and required flags before full story behavior lands; tests should cover metadata while later story tasks own parsing, mutation, confirmation, and rendering behavior.
- Expanding facade/service interfaces requires updating strict test stubs in `cmd/process_api_stub_test.go` and `c8volt/process/client_test.go` with panic-on-unexpected-call methods to preserve unrelated test signal.
- Supported-version service API additions can compile with temporary domain-level unsupported stubs in `v88`/`v89` until story-specific tasks wire generated client calls.
- Root command families register with Cobra globals in `cmd/<verb>.go`, call `rootCmd.AddCommand(...)` in `init`, bind shared backoff flags via `addBackoffFlagsAndBindings`, and set state-changing metadata with `setCommandMutation(..., CommandMutationStateChanging)`.
- Leaf command metadata uses `setContractSupport(..., ContractSupportFull)` and `setAutomationSupport(..., AutomationSupportFull, "...")` when shared machine output and unattended execution are supported.
- Process-instance commands use `validateOptionalDashArg(args)`, `readKeysIfDash(args)`, `mergeAndValidateKeys(...)`, and `typex.Keys.Unique()` for repeated `--key` plus stdin `-` target handling.
- Worker controls are shared through global flags `flagWorkers`, `flagFailFast`, and `flagNoWorkerLimit`; `collectOptions()` maps fail-fast/no-worker-limit/no-wait into facade call options.
- `get process-instance --with-vars` confirms process-scope variables through `EnrichProcessInstancesWithVariables` -> `SearchProcessInstanceVariables`, filters returned variables where `ProcessInstanceKey == key && ScopeKey == key`, sorts by name, and renders via `cmd_views_processinstance_activity.go` / `cmd_views_processinstance_vars.go`.
- Process facade/service additions should extend `c8volt/process.API`, `internal/services/processinstance.API`, and all versioned `v87`/`v88`/`v89` service contracts with compile-time assertions preserved.
- v8.8 and v8.9 generated clients expose `CreateElementInstanceVariablesWithResponse(ctx, elementInstanceKey, body, ...)` for `PUT /element-instances/{elementInstanceKey}/variables`; responses have status/status-code helpers and no success JSON payload.
- Camunda 8.7 unsupported behavior is represented by errors wrapping `domain.ErrUnsupported`, with message text naming the unsupported operation before mutation.
- CLI docs are generated from Cobra metadata by `go run -ldflags "$(LDFLAGS)" ./docsgen -out ./docs/cli -format markdown`; `make docs-content` also syncs `docs/index.md` from `README.md`.
- Generated element-instance variable update responses should be validated with `httpc.HttpStatusErr`; successful calls can return `204 No Content` with an empty body.
- Single-key update confirmation can reuse `SearchProcessInstanceVariables` and the existing process-scope filter helper, then compare only requested variable names after normalizing JSON values.
- Shared JSON command envelopes are emitted by `renderCommandResult`; human output needs an explicit view helper because `renderCommandResult` intentionally does nothing outside machine-readable modes.

---
## Iteration 1 - 2026-05-07 17:47:53 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**:
- [x] T001: Inspect root command registration and mutation metadata patterns
- [x] T002: Inspect process-instance key/stdin, worker, fail-fast, and no-worker-limit behavior
- [x] T003: Inspect process-instance variable lookup and rendering paths
- [x] T004: Inspect process facade models and service interface patterns
- [x] T005: Inspect generated v8.8/v8.9 update client methods and v8.7 unsupported patterns
- [x] T006: Inspect README, generated docs, and docs generation workflow
**Tasks Remaining in Story**: None - story complete
**Commit**: No commit - sandbox cannot write `.git/index.lock`
**Files Changed**:
- specs/179-update-pi-vars/tasks.md
- specs/179-update-pi-vars/progress.md
**Learnings**:
- Setup is discovery-only; no production code changed and no targeted Go package behavior was affected.
- The future update command should reuse existing PI target selection and output metadata instead of adding command-local parsing or docs generation paths.
- Commit was blocked by filesystem permissions on `.git`; `git add` failed with `Unable to create '.git/index.lock': Operation not permitted`.
---
---
## Iteration 2 - 2026-05-07 17:55:20 CEST
**User Story**: Phase 2: Foundational (Blocking Prerequisites)
**Tasks Completed**: 
- [x] T007: Add the `update` root command with aliases, examples, backoff bindings, and state-changing metadata
- [x] T008: Add process-instance variable update request/result domain models
- [x] T009: Add facade-level process-instance variable update request/result models
- [x] T010: Extend process facade interface and client stubs for update/confirmation orchestration
- [x] T011: Extend process-instance service API and compile-time implementation assertions for variable update support
- [x] T012: Add unsupported Camunda 8.7 update method behavior
- [x] T013: Add command contract discovery tests for the new update root and process-instance metadata
- [x] T014: Run foundational targeted validation and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- cmd/update.go
- cmd/update_processinstance.go
- cmd/command_contract_test.go
- cmd/process_api_stub_test.go
- c8volt/process/api.go
- c8volt/process/bulk.go
- c8volt/process/client.go
- c8volt/process/client_test.go
- c8volt/process/convert.go
- c8volt/process/model.go
- internal/domain/processinstance.go
- internal/services/processinstance/api.go
- internal/services/processinstance/v87/contract.go
- internal/services/processinstance/v87/service.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/variables.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/variables.go
- specs/179-update-pi-vars/tasks.md
- specs/179-update-pi-vars/progress.md
**Learnings**:
- The foundational leaf command is metadata-only in this iteration; US1 still owns request parsing, service mutation calls, confirmation lookup, and output rendering.
- Interface expansion surfaced strict test stubs immediately, which is useful for keeping command and facade tests explicit about unexpected update calls.
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process ./internal/services/processinstance/v87 -run 'Test(CommandCapability|Update|Unsupported)' -count=1` passed.
- `GOCACHE=/tmp/c8volt-gocache go test ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'Test(Update|Unsupported)' -count=1` passed for compile coverage on touched versioned services.
---
---
## Iteration 3 - 2026-05-07 18:06:13 CEST
**User Story**: User Story 1 - Update Variables For One Process Instance
**Tasks Completed**: 
- [x] T015: Add command test for `update pi --key <key> --vars '{"foo":"bar"}'` submitting the v8.8 update request and confirming through variable lookup
- [x] T016: Add command test proving `update process-instance` and `update pi` behave identically for a single key
- [x] T017: Add v8.8 service test for `PUT /v2/element-instances/{elementInstanceKey}/variables` using the process instance key
- [x] T018: Add v8.9 service test for `PUT /v2/element-instances/{elementInstanceKey}/variables` using the process instance key
- [x] T019: Add facade test for normalized JSON confirmation comparing requested values to returned process-instance variables
- [x] T020: Add the executable update process-instance command surface for one key
- [x] T021: Parse and validate single-key `--vars` JSON object input before mutation
- [x] T022: Implement v8.8 variable update service call
- [x] T023: Implement v8.9 variable update service call
- [x] T024: Implement facade update and default confirmation flow
- [x] T025: Render single-key confirmed update results for human and JSON output
- [x] T026: Run the US1 targeted validation command and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- cmd/update_processinstance.go
- cmd/cmd_views_processinstance_update.go
- cmd/update_processinstance_test.go
- cmd/process_api_stub_test.go
- c8volt/process/client.go
- c8volt/process/client_test.go
- internal/services/processinstance/v88/contract.go
- internal/services/processinstance/v88/variables.go
- internal/services/processinstance/v88/service_test.go
- internal/services/processinstance/v89/contract.go
- internal/services/processinstance/v89/variables.go
- internal/services/processinstance/v89/service_test.go
- specs/179-update-pi-vars/tasks.md
- specs/179-update-pi-vars/progress.md
**Learnings**:
- The generated Camunda update method uses the process instance key as an `ElementInstanceKey` path parameter and returns no success payload, so domain results need to carry status metadata from the HTTP response.
- The prescribed US1 test regex requires test names beginning with `TestUpdateProcessInstance`, `TestUpdatePI`, `TestElementInstanceVariables`, or `TestVariableConfirmation` to avoid silently skipping new coverage.
- US1 intentionally keeps the command to one unique target key; repeated-key, stdin merge/dedup, and worker-controlled multi-key behavior remain in the next user story.
---
---
## Iteration 4 - 2026-05-07 18:12:20 CEST
**User Story**: User Story 2 - Update Multiple Selected Process Instances
**Tasks Completed**:
- [x] T027: Add command test for multiple repeated `--key` values applying one `--vars` payload to each unique key
- [x] T028: Add command test for stdin `-` keys merged and deduplicated with `--key` values
- [x] T029: Add facade test for multi-key update respecting worker count and fail-fast options
- [x] T030: Reuse existing stdin key parsing, validation, merge, and deduplication behavior for update targets
- [x] T031: Apply the same parsed variable map to every unique target key through facade/service calls
- [x] T032: Reuse existing worker, `--workers`, `--fail-fast`, and `--no-worker-limit` option mapping for multi-key updates
- [x] T033: Render multi-key human and JSON results with independent per-key statuses
- [x] T034: Run the US2 targeted validation command and fix regressions
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- cmd/update_processinstance.go
- cmd/update_processinstance_test.go
- c8volt/process/client_test.go
- specs/179-update-pi-vars/tasks.md
- specs/179-update-pi-vars/progress.md
**Learnings**:
- The US2 implementation only needed to remove the command's temporary single-key guard because the existing facade bulk path already deduplicates keys, applies worker settings, and preserves per-key results.
- The sandbox blocks local TCP listeners, so server-backed command tests in `./cmd` are skipped here; the targeted command still passes, and the non-network facade worker/dedup test exercises the executable bulk path.
- `GOCACHE=/tmp/c8volt-gocache go test ./cmd ./c8volt/process -run 'Test(UpdateProcessInstance.*(Multiple|Stdin|Dedup|Workers|FailFast))' -count=1` passed.
---
