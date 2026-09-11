# Ralph Memory

Feature: 303-standard-tenant-logging
Started: 2026-09-11T16:49:28Z

## Codebase Patterns

- The planned correction is CLI rendering only: both tenant-context emitters live in `cmd/processinstance_mutation_progress.go` and should reuse `printOpsDurableLine` from `cmd/ops_progress_render.go`.
- Attached logger tests construct `logging.New(logging.LoggerConfig{...})`, install it with `logging.ToContext`, and keep its writer separate from command stdout.
- Tests that mutate process-instance command globals call `resetProcessInstanceCommandGlobals` before the test and register it with `t.Cleanup`; real-terminal checks use `testx.NewCmdTerminalRunner`.
- Both mutation tenant emitters now route existing `tenantContextHumanLine` values through `printOpsDurableLine`; the rendered marker remains before emission, so filtering cannot cause replay.

## Decisions

- Feature artifacts and the mandatory Ralph implementation rules are aligned; no conflict blocks implementation.

## Gotchas

- Attached-logger tests are required because no-logger fallback tests cannot prove severity, formatting, or threshold filtering.
- With `commit.issue: auto`, branch `codex/303-standard-tenant-logging` has no leading numeric prefix, so Ralph commit subjects must omit an issue suffix.

## Reusable Commands

- Baseline: `go test ./cmd -run 'Test(ProcessInstanceMutation|CancelProcessInstance|DeleteProcessInstance|Confirmation)' -count=1`
- Focused US1 regression: `go test ./cmd -run '^TestProcessInstanceMutationTenantSeverity$' -count=1`
- US1 suite: `go test ./cmd -run 'TestProcessInstanceMutation(Tenant|Progress)' -count=1`

## Do Not Repeat

- Do not treat a passing pre-change baseline as proof of the new logger behavior.

## Current Handoff
- Begin US2 with T006: extend both attached-logger emitters across plain/JSON formats and INFO/WARN/ERROR thresholds, reusing the T004 integration without further production changes.
