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
- Real-terminal process-instance coverage uses the root command with `--no-indicator` and a stub HTTP server. It exercises cancel/delete across plain/JSON logging, configured/inherited root stderr, acceptance/abort, exact multiline prompts, stdout isolation, and mutation-call presence or absence. Existing terminal fixtures retain prompt-free empty scopes and one-key-per-line paging.
- The README tenant-context paragraph now states that eligible selector-based delete/cancel diagnostics use standard INFO/WARN logging, honor configured log format/level, retain existing mode eligibility, and remain separate from JSON command results.
- Cancel/delete command `Long` descriptions now repeat that concise logging contract; `TestProcessInstanceHelp_DocumentsTenantContract` locks the shared wording for both commands before docs regeneration.
- `make docs-content` regenerates the two process-instance command pages plus README-derived `docs/index.md`; the only content change is the tenant-logging clarification, while the homepage build line receives the generator's expected current commit metadata.
- All ten feature-touched Go files are gofmt-clean. The three quickstart targeted suites, `git diff --check`, and the repository-wide race-enabled `make test` pass; no environmental or terminal-fixture blockers remain.
- Final review against the tenant-logging contract and FR-001–FR-009 confirms the only production behavior change is the two planned durable-line emission calls; all remaining changes are adjacent tests, source documentation, generated documentation, or feature records. The focused 12-case emitter matrix and three targeted acceptance suites pass, and the prior full race-enabled validation remains recorded in `quickstart.md`.

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
- Feature complete; no handoff required.
