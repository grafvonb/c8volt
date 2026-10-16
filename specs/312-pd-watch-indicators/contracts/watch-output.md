# CLI Contract: `get pd --watch` Change Indicators

## Invocation and compatibility

Existing aliases, selectors, refresh interval, retries, timeout, and termination behavior remain unchanged. No flag is added. Existing rejection of `--json`, `--keys-only`, `--xml`, `--quiet`, and `--automation` with watch remains unchanged. Ordinary `get pd` output is unchanged in all modes. This feature does not add terminal eligibility restrictions.

## Successful refresh body on stdout

After existing repaint controls, render:

```text
Changes: 1 updated, 1 new, 0 removed   (* changed, + new)
* <existing definition with changed values>
+ <new definition>
  <unchanged definition>
found: 3
```

Angle-bracket descriptions illustrate existing row content, not literal output. Each actual row starts with exactly one marker character and one separating space; unchanged rows start with two spaces. Retain the existing aligned PD fields, canonical order, and `found: N` footer. Do not sort by marker or show removed rows.

The summary is exactly one line above rows, including the legend and three spaces before `(* changed, + new)`. On the first successful refresh, all markers are blank and all change counts are zero. An empty successful refresh still prints the summary and `found: 0`.

## Comparison semantics

Compare keys and displayed values with the previous successfully displayed complete refresh. New rows count only as new; changed existing rows count once as updated; absent prior keys count as removed. Failed refreshes do not advance history. A later identical success clears markers and highlights. A key reappearing after a successful absence is new. Padding, order, and undisplayed fields cannot trigger changes.

## Bold text

When the configured stdout writer is a terminal, `TERM` is nonempty and not `dumb`, and `NO_COLOR` is empty, bold only changed visible values of existing rows. Use ANSI bold start/reset around value spans, excluding padding and unchanged statistics tokens. New rows rely on `+`. Plain and styled variants have identical visible widths, markers, counts, and ordering.

With bold unavailable, no bold controls are emitted. Existing repaint controls are unaffected by this contract. This distinction preserves current watch behavior while keeping indicators understandable without styling.

## Streams and failures

The summary, rows, and footer belong on configured stdout. Existing retries, slow-refresh warnings, verbose status, and stop diagnostics stay on configured stderr. No confirmation is introduced. Collection failure produces no candidate view; repaint/view failure is returned and does not commit the baseline. Partial collection results never imply removals.

## Documentation

Explain markers, selective bold, zero-count initial baseline, clearing on unchanged success, failure-baseline behavior, and removed counts in command help and README. Update command output metadata that currently claims watch/list parity. Generate CLI reference and homepage from those sources.
