# Tasks: Process Definition Watch Change Indicators

## Watch lifetime follow-up (2026-10-21)

The user-authorized [watch lifetime follow-up](spec.md#watch-lifetime-follow-up-2026-10-21)
on `345-bulk-job-completion` supersedes the original no-new-flags and unchanged-timeout
constraints below. It adds an unlimited default, optional `--watch-timeout`, request
timeout recovery, configured lifecycle logging, and isolated regression coverage.


**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [output contract](contracts/watch-output.md), and [quickstart.md](quickstart.md).

**Issue / Branch**: #312 / `312-pd-watch-indicators`

**Organization**: Two sequential work units, one per story. Tests required by FR-012, documentation, formatting, and targeted validation belong in the same unit as the behavior. All paths are repository-relative. Ralph execution must follow `AGENTS.md` and `specs/ralph-implementation-rules.md`.

## Setup and Foundation

Existing watch infrastructure and test harnesses are sufficient. No standalone setup or foundation tasks are needed. Read context, verify the branch, and track progress within each work unit. Keep presentation state in `cmd/get_processdefinition_watch.go` and final rendering in `cmd/cmd_views_processdefinition.go`. No new dependencies, flags, public models, backend requests, ordering algorithms, or retry policies.

## User Story 1 — Spot changes between successful refreshes (Priority: P1, MVP)

**Goal**: Deliver plain indicators with exact counts and a baseline that survives failures and empty results.

**Independent test**: Execute baseline → update/add → unchanged/reordered → partial failure(s) → recovery/removal → empty → unchanged empty → reappearance. Assert exact markers, counts, rows, footer, and preserved ordinary output.

- [x] T001 [US1] Implement complete plain watch indicators and baseline handling in `cmd/get_processdefinition_watch.go` and `cmd/cmd_views_processdefinition.go`, with lifecycle and view tests in `cmd/get_processdefinition_watch_test.go` and `cmd/cmd_views_processdefinition_test.go`; update `cmd/get_processdefinition.go` and `README.md`, regenerate docs with `make docs-content`, and complete the validation below.

Implementation:

- Project owned, unstyled displayed values by definition key: tenant, BPMN process ID, combined version/tag, optional statistics presence, and individually identifiable active/completed/canceled/supported-incident values. Preserve existing zero-as-minus formatting and ordinary row bytes. Do not retain mutable model/statistics pointers or compare undisplayed metadata.
- Keep invocation-local initialized state separate from an empty baseline map. Compare by key before alignment, preserve snapshot row order, count each changed existing definition once, and keep new/updated categories exclusive. Removed keys contribute counts without retained rows.
- Build a candidate only after complete successful collection. Commit it only after successful repaint and error-returning view writes. Failed/partial collection and repaint/summary/row/footer write failures must not advance history. Reuse the existing runner and collection path; rendering must add no requests.
- Render the exact summary from `contracts/watch-output.md`, then rows with ` *` or ` +` suffixes, or no suffix when unchanged, followed by existing `found: N`. Do not add a marker or trailing spaces on the first/unchanged refresh. Empty success prints summary and `found: 0`; first success always reports zero changes.
- Keep per-statistic change identity ready for T002; do not add bold yet. Update watch/list parity assertions to the new contract while preserving ordinary rendering behavior.

Required coverage in this unit:

- All story-1 acceptance scenarios and the full eight-refresh quickstart sequence, including repeated/initial failures, partial results with errors, recovery against the last success, successful empty first result, nonempty → empty → empty, removed keys reappearing as new, and fresh-session reset.
- Multiple changed values counted once; unchanged marker clearing; reordered input and neighboring width changes without false updates; undisplayed and unsupported-incident changes ignored; optional values/statistics appearing and disappearing.
- Owned baseline values even when fixtures mutate and reuse model/statistics objects. Repaint and summary/row/footer writer errors propagate without candidate commitment; use focused state tests where needed rather than new public APIs or production globals.
- Exact stdout and separate stderr, collector counts, no repaint on collection failure, and existing selector/retry/timeout/cancellation/slow-warning/verbose behavior. Preserve sparse-page/service coverage and ordinary human/JSON/keys-only/XML and incompatible-watch-mode regressions in `cmd/get_processdefinition_test.go` and related tests.

Documentation and validation:

- Explain markers, summary, baseline, failure behavior, clearing, and removals in README/help/output metadata; remove claims that the whole watch view matches ordinary list output. Describe only delivered plain behavior. Regenerate `docs/cli/` and `docs/index.md`; do not hand-edit them.
- Add purpose comments to new/materially changed functions and tests, run `gofmt` on touched Go files, then `go test ./cmd -run 'Test(GetProcessDefinition|ProcessDefinitionWatch|ListProcessDefinitions|ProcessDefinitionView)' -count=1`, `make docs-content`, and `git diff --check`. Review generated changes. Tests modifying globals/environment must restore state and must not run in parallel.

**Checkpoint**: Complete plain-marker MVP, including all baseline correctness and failure-recovery coverage. No later task is needed to establish trustworthy comparisons.

## User Story 2 — Read indicators with or without styling (Priority: P2)

**Goal**: Add selective bold values while preserving visible alignment, plain fallback, and ordinary output.

**Independent test**: Compare styled and plain views for identical changes, verify individual changed spans and identical visible layout, and exercise actual configured terminal and nonterminal output writers.

- [x] T002 [US2] Add watch-only capability detection and selective bold rendering in `cmd/cmd_views_processdefinition.go`, with lifecycle wiring in `cmd/get_processdefinition_watch.go` only as needed; extend `cmd/cmd_views_processdefinition_test.go` and `cmd/get_processdefinition_watch_test.go` or a focused `cmd/get_processdefinition_watch_terminal_test.go`, complete guidance in `cmd/get_processdefinition.go` and `README.md`, regenerate docs with `make docs-content`, and complete the validation below.

Implementation and coverage:

- Detect capability on `cmd.OutOrStdout()` using existing `golang.org/x/term`. Bold requires a terminal writer, nonempty/non-`dumb` `TERM`, and empty `NO_COLOR`. Preserve existing watch eligibility and repaint behavior; do not assume terminal stdin means terminal stdout.
- Compute widths/padding from plain columns using existing helpers, then bold only changed visible values on updated rows. Keep padding, labels/brackets, unchanged statistics, summary, and footer outside bold spans. New rows use `+` without bold. A removed optional field still produces `*` but has no replacement token to bold. Do not extend shared flat formatting into a generic ANSI framework.
- Assert individual statistics highlighting, combined version/tag and other displayed field changes, zero-as-minus transitions, optional presence/support changes, no bold for first/new/unchanged rows, and clearing after an identical successful refresh. Removing bold controls must yield exactly the plain output, including alignment and marker suffixes.
- Test an actual configured terminal stdout writer, nonterminal buffers/files, inherited output destinations where applicable, and `TERM` empty/`dumb` plus nonempty `NO_COLOR`. Reuse existing terminal-test infrastructure and explicit style parameters for deterministic view tests. Restore globals/environment and avoid parallel execution for such tests.
- Retain T001 lifecycle and ordinary-output regressions. Complete README/help/metadata with selective bold, fallback, and the exact summary example; regenerate `docs/cli/` and `docs/index.md` and review them.
- Add required comments, format touched Go files, then run `go test ./cmd -run 'Test(GetProcessDefinition|ProcessDefinitionWatch|ListProcessDefinitions|ProcessDefinitionView)' -count=1`, `make docs-content`, and `git diff --check`.

**Checkpoint**: Both stories and the complete output contract are satisfied.

## Cross-Cutting Completion

No standalone polish or validation task. Each unit must compile and pass its relevant checks before being marked complete. During Ralph execution, record actual results in `specs/312-pd-watch-indicators/progress.md` and follow its commit discipline; issue-associated Conventional Commit subjects end with ` #312`.

Run shared-helper/consumer tests only if those helpers change. Use `make test` if shared runtime changes or unresolved failures justify the full race suite. Optional live read-only viewing supplements deterministic fixtures; do not create or delete deployments for this feature.

## Dependencies and Parallel Opportunities

`T001 [US1]: complete plain indicators and recovery → T002 [US2]: selective styling`

T002 depends on T001's display projection and baseline. No `[P]` tasks: both units touch the same renderer, runner, and tests. Within US1, projection/runner/view signatures and acceptance coverage must move together; within US2, capability/rendering/tests describe one feature increment. There are no useful parallel implementation pairs.

## Consolidation Review

Reduced from three tasks to two. The former recovery task mostly tested behavior already required by the core task and reused the same fixtures and files. Its complete acceptance coverage now ships in T001; the specification merges recovery into story 1 so Ralph still processes one story per iteration. Styling is renumbered as story 2/T002.

Keep these two units separate: terminal capability detection and ANSI-safe layout are a distinct, independently verifiable enhancement to a usable plain MVP. Combining them would increase iteration scope without removing duplicate implementation. Documentation and checks remain attached to each delivered behavior rather than becoming standalone tasks.

## Requirement Coverage

| Requirements / outcomes | Work unit and proof |
| --- | --- |
| FR-001–FR-008; SC-001–SC-003, SC-005 | T001 markers/counts/order, complete baseline state machine, failure/write-error recovery, empty results, clearing, and display-only comparison; T002 highlight clearing |
| FR-009; SC-004 | T002 selective bold/capability/alignment; T001 readable plain markers |
| FR-010; SC-006 | T001 ordinary-output/flags/selectors preserved; T002 regression coverage retained |
| FR-011; SC-006 | README/help/metadata/generated docs accompany both increments |
| FR-012 | All required fixtures and targeted tests included in T001 and T002 |

## Implementation Strategy

Deliver T001 as the full plain MVP, then T002 for styling. Build shared behavior once. The merger preserves every acceptance scenario and meaningful test from the original three-task list; it removes the separate hardening pass, not its coverage.
