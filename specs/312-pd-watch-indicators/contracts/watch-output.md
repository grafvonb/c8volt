# CLI Contract: `get pd --watch` Change Indicators

## Invocation and compatibility

Existing aliases, selectors, refresh interval and consecutive retry limits remain unchanged. Watch runs indefinitely by default; `--watch-timeout` optionally bounds the session (`0` means unlimited, negative values are invalid, and explicitly supplying it requires `--watch`). HTTP request and retry-loop timeouts do not bound the session. An individual request deadline is retryable while the session context remains active. Session expiry and Ctrl-C stop cleanly; fatal errors and retry exhaustion remain failures. Existing rejection of `--json`, `--keys-only`, `--xml`, `--quiet`, and `--automation` with watch remains unchanged. Ordinary `get pd` output is unchanged in all modes. This feature does not add terminal eligibility restrictions.

## Successful refresh on the terminal

After existing repaint controls, render:

```text
16:42:08.123 INFO 1 updated, 1 new, 0 removed
<existing definition with changed values> *
<new definition> +
<unchanged definition>
found: 3
```

Angle-bracket descriptions illustrate existing row content, not literal output. Each changed/new row ends with one separating space and its marker after the last displayed value. Unchanged rows have no suffix or added trailing spaces. Retain the existing aligned PD fields, canonical order, and `found: N` footer. Do not sort by marker or show removed rows.

The summary is one timestamped INFO log line on configured stderr after repaint and before rows, using the existing `HH:MM:SS.mmm` plain-time logging format on every successful refresh. It ends after the removed count, with no marker legend or trailing spaces. Marker meanings remain documented in command help. On the first successful refresh, all markers are blank and all change counts are zero. An empty successful refresh still prints the summary and `found: 0`.

## Comparison semantics

Compare keys and displayed values with the previous successfully displayed complete refresh. New rows count only as new; changed existing rows count once as updated; absent prior keys count as removed. Failed refreshes do not advance history. A later identical success clears markers and highlights. A key reappearing after a successful absence is new. Padding, order, and undisplayed fields cannot trigger changes.

## Bold text

When the configured stdout writer is a terminal, `TERM` is nonempty and not `dumb`, and `NO_COLOR` is empty, bold only changed visible values of existing rows. Use ANSI bold start/reset around value spans, excluding padding and unchanged statistics tokens. New rows rely on `+`. Plain and styled variants have identical visible widths, markers, counts, and ordering.

With bold unavailable, no bold controls are emitted. Existing repaint controls are unaffected by this contract. This distinction preserves current watch behavior while keeping indicators understandable without styling.

## Streams and failures

The timestamped INFO summary belongs on configured stderr; rows and footer belong on configured stdout. There is no `Changes:` label. Reuse the plain logging handler and propagate write errors so failed log output cannot advance the baseline. Existing retries, slow-refresh warnings, verbose status, and stop diagnostics stay on configured stderr. No confirmation is introduced. Collection failure produces no candidate view; repaint/view failure is returned and does not commit the baseline. Partial collection results never imply removals.

## Documentation

Explain markers, selective bold, zero-count initial baseline, clearing on unchanged success, failure-baseline behavior, and removed counts in command help and README. Update command output metadata that currently claims watch/list parity. Generate CLI reference and homepage from those sources.
