# Ralph Memory

Feature: 339-suspended-instance-state
Started: 2026-10-17T11:26:30Z

## Codebase Patterns

- `c8volt/process/state.go` owns canonical parsing: trim whitespace, accept case variants and `cancelled`, and reject UNKNOWN/unrecognized inputs with the UNKNOWN fallback. `validStates` also supplies CLI invalid-value feedback.
- `ValidateProcessInstanceStates` is mandatory on shared/versioned process-instance APIs. Shared page/total traversal validates before direct-incident discovery; adapters validate page search and single/bulk waits before requests/workers. v88/v89 reject trimmed case-insensitive suspension; v810 permits it. Generic waiters stay capability-neutral.
- Lookup and single-state waits parse observed states; instance, combined-expectation and bulk-state report conversions preserve raw states. Extending the parser fixes suspended lookup reports without changing converters or facade orchestration.
- `cmd/cmd_views_processinstance.go` already renders the state in its existing column and JSON payload. No renderer or schema change is needed for recognition. Quiet suppression belongs to command execution, not `listProcessInstancesView` alone.

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

## Reusable Commands

- `go test ./internal/domain -run 'State' -count=1`
- `go test ./c8volt/process -run 'State|ProcessInstance|Expectation' -count=1`
- `go test ./cmd -run 'TestOneLinePI|TestListProcessInstancesView|TestExpectHelp_DocumentsWaitVerificationUsage' -count=1`
- `make docs-content` regenerates CLI references and `docs/index.md` (including its build provenance line).
- T002 targeted check: `go test ./internal/services/processinstance/... ./c8volt/process -run 'State|Expectation|Search|Suspended' -count=1`. Cross-version HTTP coverage is centralized in service `search_test.go`; public adapter coverage is in `c8volt/process/suspended_test.go`.
- T002 race coverage: `make test` passed every package except an initially missing resource test-double method; after adding it, `go test ./c8volt/resource -race -count=1` passed. No runtime failures remain; successful unrelated packages were not rerun.

## Do Not Repeat

- Do not add facade/backend orchestration to fix state recognition; the public vocabulary is the missing piece.
- Do not test quiet suppression through a view helper alone. Use actual command execution for T003's quiet/empty/automation contracts, with real terminal stdin for applicable interactive paths.

## Current Handoff

- Next: T003 (US2), CLI selection/validation/completion and execution-path output/stream contracts. T002 capability guards and cross-version HTTP/facade/waiter tests are implemented; keep command-specific mutation/ops restrictions unchanged.
- T003 must cover bootstrap-free state completion, supported case variants, unsupported error/exit behavior, and empty/nonempty human/JSON/keys/quiet/automation paths, including real terminal stdin where eligible. T001 help updates alone did not complete this work.
- T004 owns creation visibility, T005 timestamp snapshots, and T006 live 8.10.4 evidence plus final integrated validation; none is started.
