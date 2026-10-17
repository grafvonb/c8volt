# Ralph Memory

Feature: 339-suspended-instance-state
Started: 2026-10-17T11:26:30Z

## Codebase Patterns

- `c8volt/process/state.go` owns canonical parsing: trim whitespace, accept case variants and `cancelled`, and reject UNKNOWN/unrecognized inputs with the UNKNOWN fallback. `validStates` also supplies CLI invalid-value feedback.
- `ValidateProcessInstanceStates` is mandatory on shared/versioned process-instance APIs. Shared page/total traversal validates before direct-incident discovery; adapters validate page search and single/bulk waits before requests/workers. v88/v89 reject trimmed case-insensitive suspension; v810 permits it. Generic waiters stay capability-neutral.
- Lookup and single-state waits parse observed states; instance, combined-expectation and bulk-state report conversions preserve raw states. Extending the parser fixes suspended lookup reports without changing converters or facade orchestration.
- `cmd/cmd_views_processinstance.go` already renders the state in its existing column and JSON payload. No renderer or schema change is needed for recognition. Empty quiet summaries need guards in both incremental search and the collected list view; existing nonempty quiet output is intentionally retained (see `TestGetProcessInstanceSearch_AllTenantsProtectedOutputModesStayClean`).

- `cmd/processinstance_state_completion.go` supplies bootstrap-free candidates from public constants: search/analysis offers ALL, expectations offer ABSENT. Suspension carries an 8.10 description; typed prefix case is preserved. Analysis still rejects ABSENT.

## Decisions

- T001 adds domain/public `StateSuspended` and public parsing only. Preserve terminal sets: domain ABSENT is terminal, public ABSENT is not; suspension is nonterminal in both.
- Keep creation-observable states unchanged until T004. T002 explicit-request guards stay outside the generic waiter, so creation can continue calling it directly.
- Update get/expect state flag descriptions with the 8.10 limitation because the parser expands exposed valid-value feedback. Regenerate CLI docs and the README-derived homepage with `make docs-content`.

## Gotchas

- Do not normalize every report uniformly: existing unknown/raw and cancellation-alias behavior differs between parsed state reports and direct casts. `TestClient_ProcessInstanceStateReports` captures those contracts.
- Interface updates also affect `c8volt/resource/client_test.go` in addition to process facade and dry-run stubs. Embedded nil API stubs may compile while panicking if a newly called method is not implemented.
- The existing domain unsupported sentinel text is “unsupported media type”; preserve the sentinel and wrapping so public `ferrors` still classifies these as unsupported capability.
- Public `StateReport` has no `Ok` field and preserves UNKNOWN on failed waits. Combined expectation reports expose `Ok`; do not change either schema.
- State help text is asserted in both `cmd/expect_test.go` and `cmd/command_contract_test.go`; update both when the vocabulary changes.
- The generated v810 client/provenance stays at 8.10.3; live acceptance later requires observed server 8.10.4.

- v810 serializes the state equality filter as a plain string (`"SUSPENDED"`), not a `$eq` object. ALL may omit the entire filter. Empty get JSON omits both `total` and `items`; require one envelope followed by EOF. Quote `"8.10"` in raw YAML test fixtures to avoid YAML numeric normalization to 8.1.
- Slow-analysis state help is also asserted in `TestCommandContractOpsAnalyseSlowProcessInstances`; keep it aligned when changing vocabulary.

## Reusable Commands

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

- Next: T004 (US3), creation visibility when SUSPENDED is the first observation. Extend the cloned observable-state list and prove visibility separately from execution completion on all versions.
- Keep explicit capability checks out of the generic creation waiter. T002/T003 service and CLI selection/expectation coverage is complete; preserve version errors, command-specific restrictions and existing nonempty quiet output.
- T005 owns timestamp mapping/snapshots; T006 owns live 8.10.4 evidence and final integrated validation. Neither was started. Generated v810 clients/provenance remain unchanged at 8.10.3.
