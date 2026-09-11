# Ralph Memory

Feature: 303-standard-tenant-logging
Started: 2026-09-11T16:49:28Z

## Codebase Patterns

- The planned correction is CLI rendering only: both tenant-context emitters live in `cmd/processinstance_mutation_progress.go` and should reuse `printOpsDurableLine` from `cmd/ops_progress_render.go`.
- Attached logger tests construct `logging.New(logging.LoggerConfig{...})`, install it with `logging.ToContext`, and keep its writer separate from command stdout.
- Tests that mutate process-instance command globals call `resetProcessInstanceCommandGlobals` before the test and register it with `t.Cleanup`; real-terminal checks use `testx.NewCmdTerminalRunner`.

## Decisions

- Feature artifacts and the mandatory Ralph implementation rules are aligned; no conflict blocks implementation.

## Gotchas

- Attached-logger tests are required because no-logger fallback tests cannot prove severity, formatting, or threshold filtering.

## Reusable Commands

- Baseline: `go test ./cmd -run 'Test(ProcessInstanceMutation|CancelProcessInstance|DeleteProcessInstance|Confirmation)' -count=1`
- Focused US1 regression: `go test ./cmd -run '^TestProcessInstanceMutationTenantSeverity$' -count=1`

## Do Not Repeat

- Do not treat a passing pre-change baseline as proof of the new logger behavior.

## Current Handoff
- Begin US1 with T003: add the attached-logger tenant severity regression cases and confirm they fail against the two current raw-write emitters before T004.
