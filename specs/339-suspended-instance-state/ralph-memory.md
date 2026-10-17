# Ralph Memory

Feature: 339-suspended-instance-state
Started: 2026-10-17T11:26:30Z

## Codebase Patterns

- `SuspendedDate` is an optional string in domain/public instances, mapped through v810 `formatTimePtr` and both facade directions. Fresh conversion clears absent metadata; nil/zero dates omit JSON without changing state. Existing nested incident/variable/element payloads inherit the field; human renderers need no change.
- `c8volt/process/state.go` owns canonical parsing: trim whitespace, accept case variants and `cancelled`, and reject UNKNOWN/unrecognized inputs with the UNKNOWN fallback. `validStates` also supplies CLI invalid-value feedback.
- `ValidateProcessInstanceStates` is mandatory on shared/versioned process-instance APIs. Shared page/total traversal validates before direct-incident discovery; adapters validate page search and single/bulk waits before requests/workers. v88/v89 reject trimmed case-insensitive suspension; v810 permits it. Generic waiters stay capability-neutral.
- Lookup and single-state waits parse observed states; instance, combined-expectation and bulk-state report conversions preserve raw states. Extending the parser fixes suspended lookup reports without changing converters or facade orchestration.
- `cmd/cmd_views_processinstance.go` already renders the state in its existing column and JSON payload. No renderer or schema change is needed for recognition. Empty quiet summaries need guards in both incremental search and the collected list view; existing nonempty quiet output is intentionally retained (see `TestGetProcessInstanceSearch_AllTenantsProtectedOutputModesStayClean`).
- `cmd/processinstance_state_completion.go` supplies bootstrap-free candidates from public constants: search/analysis offers ALL, expectations offer ABSENT. Suspension carries an 8.10 description; typed prefix case is preserved. Analysis still rejects ABSENT.

## Decisions

- T001 adds domain/public `StateSuspended` and public parsing only. Preserve terminal sets: domain ABSENT is terminal, public ABSENT is not; suspension is nonterminal in both.
- Creation visibility includes SUSPENDED through the cloned domain observable-state list. All adapters keep using the generic waiter directly; explicit-request capability guards remain outside it, preserving 8.8/8.9 creation defaults.
- Update get/expect state flag descriptions with the 8.10 limitation because the parser expands exposed valid-value feedback. Regenerate CLI docs and the README-derived homepage with `make docs-content`.

## Gotchas

- Direct service-factory HTTP tests must include `/v2` in `config.APIs.Camunda.BaseURL`; CLI bootstrap supplies this path separately. Older adapters intentionally leave `SuspendedDate` empty, even if a response contains the unfamiliar field.
- Creation help is asserted by `docsgen/TestGeneratedRunProcessInstanceDocsDocumentPipeline` as well as command help tests; keep its state list aligned.

- `StartConfirmedAt` belongs to internal creation results; public ProcessInstance/JSON does not expose it. Keep confirmation timestamp checks at the adapter layer. CLI no-wait emits `accepted`, omits unobserved state and makes no lookup; confirmed creation emits `succeeded` without implying execution completion.

- Do not normalize every report uniformly: existing unknown/raw and cancellation-alias behavior differs between parsed state reports and direct casts. `TestClient_ProcessInstanceStateReports` captures those contracts.
- Interface updates also affect `c8volt/resource/client_test.go` in addition to process facade and dry-run stubs. Embedded nil API stubs may compile while panicking if a newly called method is not implemented.
- The existing domain unsupported sentinel text is “unsupported media type”; preserve the sentinel and wrapping so public `ferrors` still classifies these as unsupported capability.
- Public `StateReport` has no `Ok` field and preserves UNKNOWN on failed waits. Combined expectation reports expose `Ok`; do not change either schema.
- State help text is asserted in both `cmd/expect_test.go` and `cmd/command_contract_test.go`; update both when the vocabulary changes.
- The generated v810 client/provenance stays at 8.10.3; live acceptance later requires observed server 8.10.4.
- v810 serializes the state equality filter as a plain string (`"SUSPENDED"`), not a `$eq` object. ALL may omit the entire filter. Empty get JSON omits both `total` and `items`; require one envelope followed by EOF. Quote `"8.10"` in raw YAML test fixtures to avoid YAML numeric normalization to 8.1.
- Slow-analysis state help is also asserted in `TestCommandContractOpsAnalyseSlowProcessInstances`; keep it aligned when changing vocabulary.

## Reusable Commands

- T005 broader validation passed: `GORACE=atexit_sleep_ms=0 go test ./internal/services/processinstance/... ./c8volt/process ./cmd ./docsgen -race -count=1`. Final repository-wide `make test` and live acceptance remain T006.
- T005 targeted checks: `go test ./internal/services/processinstance/v810 ./c8volt/process -run 'Convert|ProcessInstance|Suspended' -count=1`; `go test ./cmd -run 'Suspended|ProcessInstance' -count=1`. `TestClient_SuspendedDateSnapshots` reuses real versioned adapters/facades for present-to-absent reads/searches; `TestSuspendedDateSnapshots` checks separate streams, one envelope plus EOF, unchanged rows/keys, and one request per snapshot.
- T004 race validation: `make test` passed every package except a stale generated run-help assertion; after correcting it, `GORACE=atexit_sleep_ms=0 go test ./docsgen -race -count=1` passed. All packages now have passing race coverage; final integrated validation still belongs to T006.

- T004 targeted checks: `go test ./internal/domain ./internal/services/processinstance/... ./c8volt/process -run 'ObservableProcessInstanceCreation|CreateProcessInstance|State|Expectation|SuspendedUnsupported' -count=1`; `go test ./cmd -run 'RunHelp|RunProcessInstance|RunCommand' -count=1`. Zero-delay, bounded-attempt creation fixtures cover delayed absence-to-suspension; an already-expired context covers deadline failure without sleeps.

- `go test ./internal/domain -run 'State' -count=1`
- `go test ./c8volt/process -run 'State|ProcessInstance|Expectation' -count=1`
- `go test ./cmd -run 'TestOneLinePI|TestListProcessInstancesView|TestExpectHelp_DocumentsWaitVerificationUsage' -count=1`
- `make docs-content` regenerates CLI references and `docs/index.md` (including its build provenance line).
- T002 targeted check: `go test ./internal/services/processinstance/... ./c8volt/process -run 'State|Expectation|Search|Suspended' -count=1`. Cross-version HTTP coverage is centralized in service `search_test.go`; public adapter coverage is in `c8volt/process/suspended_test.go`.
- T002 race coverage: `make test` passed every package except an initially missing resource test-double method; after adding it, `go test ./c8volt/resource -race -count=1` passed. No runtime failures remain; successful unrelated packages were not rerun.
- T003 targeted command check: `go test ./cmd -run 'GetPI|GetProcessInstance|Expect|Completion|Suspended|OpsSlowProcess' -count=1`. All passed. Full command race run passed except a stale slow-analysis contract assertion; after correcting it, `GORACE=atexit_sleep_ms=0 go test ./cmd -race -run '^TestCommandContractOpsAnalyseSlowProcessInstances$' -count=1` passed.
- `TestSuspendedCommandHelper` accepts JSON arguments through `suspendedCommandEnv`; reuse it for isolated exit/stream tests. `decodeSuspendedEnvelope` requires EOF. The real-terminal matrix covers empty/nonempty six output modes with ordinary, auto-confirm and automation policies.

## Do Not Repeat

- Do not add facade/backend orchestration to fix state recognition; the public vocabulary is the missing piece.
- Do not test quiet suppression through a view helper alone. Use actual command execution for T003's quiet/empty/automation contracts, with real terminal stdin for applicable interactive paths.

## Current Handoff

- Next: T006, integration and final validation. T001–T005 are implemented; all four user stories now have deterministic coverage. Add the existing-harness `TestGetFamilySuspendedState` and reusable fixture helpers; do not start a new public mutation or watch workflow.
- Verify observed server patch 8.10.4 before fixture mutations, using an explicitly designated disposable profile. Keep configured release at minor `8.10`; preserve generated client/provenance baseline 8.10.3. Resume owned instances before normal cleanup and record cleanup outcomes separately.
- Record live evidence and tested revision/dirty status in `validation.md`, update integration guidance, prove test discovery, and run final `make test`. If infrastructure or permission is unavailable, record the concrete reason and leave T006 pending; a skip or different server patch is not acceptance.
