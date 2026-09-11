# Ralph Memory

Feature: 303-standard-tenant-logging
Started: 2026-09-11T16:49:28Z

## Codebase Patterns

- The planned correction is CLI rendering only: both tenant-context emitters live in `cmd/processinstance_mutation_progress.go` and should reuse `printOpsDurableLine` from `cmd/ops_progress_render.go`.
- Attached logger tests construct `logging.New(logging.LoggerConfig{...})`, install it with `logging.ToContext`, and keep its writer separate from command stdout.
- Tests that mutate process-instance command globals call `resetProcessInstanceCommandGlobals` before the test and register it with `t.Cleanup`; real-terminal checks use `testx.NewCmdTerminalRunner`.
- Both mutation tenant emitters now route existing `tenantContextHumanLine` values through `printOpsDurableLine`; the rendered marker remains before emission, so filtering cannot cause replay.
- Attached-logger coverage uses a shared emitter table and validates the full 2 emitters × 2 formats × 3 thresholds matrix. Plain records parse `logging.PlainTimestampLayout`; JSON records decode `time`, `level`, and `msg` individually and require EOF.
- Cancel execution coverage attaches the real logger to independently captured stderr, keeps stdout uncontaminated, and counts planning and mutation calls around tenant emission. Existing selector tests already own the broad empty/sparse/direct-key/abort/error matrix, so T008 extends those cases instead of duplicating command scaffolding.
- Delete execution coverage follows the same pattern: positive attached-logger paths assert clean stdout and unchanged planning/deletion counts, direct execution decodes one JSON envelope through EOF, and the expanded empty-selector table retains exact human, JSON, keys-only, quiet, automation, dry-run, verbose, auto-confirm, and no-wait contracts.

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
- US3 cancel suite: `go test -race ./cmd -run 'TestCancelProcessInstance' -count=1`
- US3 delete suite: `go test -race ./cmd -run 'TestDeleteProcessInstance' -count=1`

## Do Not Repeat

- Do not treat a passing pre-change baseline as proof of the new logger behavior.

## Current Handoff
- Continue US3 with T010: extend the real-terminal confirmation fixtures for both commands, including configured/inherited stderr, attached plain/JSON logging, prompt acceptance/abort, prompt-free empty scope, and keys-only paging.
