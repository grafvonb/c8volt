# Ralph Memory

Feature: 303-standard-tenant-logging
Started: 2026-09-11T16:49:28Z

## Codebase Patterns

- The planned correction is CLI rendering only: both tenant-context emitters live in `cmd/processinstance_mutation_progress.go` and should reuse `printOpsDurableLine` from `cmd/ops_progress_render.go`.
- Attached logger tests construct `logging.New(logging.LoggerConfig{...})`, install it with `logging.ToContext`, and keep its writer separate from command stdout.
- Tests that mutate process-instance command globals call `resetProcessInstanceCommandGlobals` before the test and register it with `t.Cleanup`; real-terminal checks use `testx.NewCmdTerminalRunner`.
- Both mutation tenant emitters now route existing `tenantContextHumanLine` values through `printOpsDurableLine`; the rendered marker remains before emission, so filtering cannot cause replay.
- Attached-logger coverage uses a shared emitter table and validates the full 2 emitters × 2 formats × 3 thresholds matrix. Plain records parse `logging.PlainTimestampLayout`; JSON records decode `time`, `level`, and `msg` individually and require EOF.

## Decisions

- Feature artifacts and the mandatory Ralph implementation rules are aligned; no conflict blocks implementation.

## Gotchas

- Attached-logger tests are required because no-logger fallback tests cannot prove severity, formatting, or threshold filtering.
- With `commit.issue: auto`, branch `codex/303-standard-tenant-logging` has no leading numeric prefix, so Ralph commit subjects must omit an issue suffix.

## Reusable Commands

- Baseline: `go test ./cmd -run 'Test(ProcessInstanceMutation|CancelProcessInstance|DeleteProcessInstance|Confirmation)' -count=1`
- Focused US1 regression: `go test ./cmd -run '^TestProcessInstanceMutationTenantSeverity$' -count=1`
- US1 suite: `go test ./cmd -run 'TestProcessInstanceMutation(Tenant|Progress)' -count=1`
- US2 suite: `go test ./cmd -run 'TestProcessInstanceMutationTenant' -count=1 -v`

## Do Not Repeat

- Do not treat a passing pre-change baseline as proof of the new logger behavior.

## Current Handoff
- Begin US3 with T008: extend cancel command execution coverage with attached logging while preserving output modes, streams, and request counts.
