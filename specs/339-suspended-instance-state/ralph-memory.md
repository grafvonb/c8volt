# Ralph Memory

Feature: 339-suspended-instance-state
Started: 2026-10-17T11:26:30Z

## Codebase Patterns

- `c8volt/process/state.go` owns canonical parsing: trim whitespace, accept case variants and `cancelled`, and reject UNKNOWN/unrecognized inputs with the UNKNOWN fallback. `validStates` also supplies CLI invalid-value feedback.
- Lookup and single-state waits parse observed states; instance, combined-expectation and bulk-state report conversions preserve raw states. Extending the parser fixes suspended lookup reports without changing converters or facade orchestration.
- `cmd/cmd_views_processinstance.go` already renders the state in its existing column and JSON payload. No renderer or schema change is needed for recognition. Quiet suppression belongs to command execution, not `listProcessInstancesView` alone.

## Decisions

- T001 adds domain/public `StateSuspended` and public parsing only. Preserve terminal sets: domain ABSENT is terminal, public ABSENT is not; suspension is nonterminal in both.
- Keep creation-observable states unchanged until T004, after T002 adds explicit-request capability guards.
- Update get/expect state flag descriptions with the 8.10 limitation because the parser expands exposed valid-value feedback. Regenerate CLI docs and the README-derived homepage with `make docs-content`.

## Gotchas

- Do not normalize every report uniformly: existing unknown/raw and cancellation-alias behavior differs between parsed state reports and direct casts. `TestClient_ProcessInstanceStateReports` captures those contracts.
- Recognition does not enforce version capabilities. T002 must reject explicit suspended requests on v88/v89 before searches/waits, including empty/direct-incident paths; keep generic creation waiters unguarded.
- State help text is asserted in both `cmd/expect_test.go` and `cmd/command_contract_test.go`; update both when the vocabulary changes.
- The generated v810 client/provenance stays at 8.10.3; live acceptance later requires observed server 8.10.4.

## Reusable Commands

- `go test ./internal/domain -run 'State' -count=1`
- `go test ./c8volt/process -run 'State|ProcessInstance|Expectation' -count=1`
- `go test ./cmd -run 'TestOneLinePI|TestListProcessInstancesView|TestExpectHelp_DocumentsWaitVerificationUsage' -count=1`
- `make docs-content` regenerates CLI references and `docs/index.md` (including its build provenance line).
- `make test` passed for T001 (full race-enabled suite); iteration 1 log: `/tmp/c8volt-339-iteration1-make-test-final.log` (temporary, not durable evidence).

## Do Not Repeat

- Do not add facade/backend orchestration to fix state recognition; the public vocabulary is the missing piece.
- Do not test quiet suppression through a view helper alone. Use actual command execution for T003's quiet/empty/automation contracts, with real terminal stdin for applicable interactive paths.

## Current Handoff

- Next: T002 (US2), the service capability contract and all version adapters/test doubles. T003 remains in US2 after T002; neither is implemented by the T001 metadata update.
- Keep the generic waiter and creation-observable list separate from explicit-request guards. T004 owns creation visibility, T005 timestamp snapshots, and T006 live evidence plus final integrated validation.
