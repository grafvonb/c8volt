# Ralph Memory

Feature: 312-pd-watch-indicators
Started: 2026-10-16T14:34:26Z

## Codebase Patterns

- `processDefinitionDisplay` in `cmd/cmd_views_processdefinition.go` is the owned, unstyled projection shared by ordinary PD rows and watch comparison; compare these values before alignment.
- Watch history is invocation-local in `processDefinitionWatchBaseline`; build a candidate after complete collection and commit it only after repaint, summary, rows, and footer all write successfully.
- Watch lifecycle tests use `executeGetProcessDefinitionWatchHarnessForTest` with injected sleep/clock functions and split refresh bodies on `processDefinitionWatchRepaintControlSequenceForTest`.
- Selective styling computes widths from plain `flatRow` values, then decorates only changed semantic tokens; terminal capability is derived from the command's configured or inherited output writer plus `TERM`/`NO_COLOR`.

## Decisions

- Plain US1 output always renders the exact change summary, a fixed two-character marker prefix, existing aligned rows, and the existing `found: N` footer.
- `processDefinitionChangedFields` already preserves individual visible-value identity for T002; new rows have no changed-field mask and must not be bolded.
- Bold spans are watch-only and limited to replacement values on `*` rows; markers, labels, brackets, padding, new rows, summary, and footer remain unstyled.

## Gotchas

- A successful empty snapshot is represented by `initialized=true` plus an empty map; do not infer initialization from map length.
- Unsupported incident counts remain excluded from display identity even when the raw incident count changes.
- `docs/index.md` build metadata changes when `make docs-content` runs and is part of the generated-doc update.
- Command help focus rejects implementation-facing `stdout` wording; describe nonterminal fallback as redirected output in help text.

## Reusable Commands

- `go test ./cmd -run 'Test(GetProcessDefinition|ProcessDefinitionWatch|ListProcessDefinitions|ProcessDefinitionView)' -count=1`
- `go test ./cmd -count=1`
- `make docs-content`
- `git diff --check`

## Do Not Repeat

- Do not restore watch/list byte parity: watch intentionally wraps unchanged ordinary row content with the summary and marker column.
- Do not retain `ProcessDefinition` or statistics pointers in the baseline; fixtures and callers may reuse and mutate them.

## Current Handoff
- Feature complete; no handoff required.
