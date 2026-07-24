# CLI Progress Policy

Feature: 254-cli-debt-refactor
Created: 2026-07-24

## Purpose

This policy defines where c8volt may show progress during long-running and paged CLI workflows. It keeps human operators informed without corrupting JSON, keys-only, quiet, automation, and report-oriented output.

## Activity Indicators

- Use the shared `toolx/logging.ActivityWriter` for transient activity only.
- Activity indicators write to stderr and must be disabled by `--no-indicator`, `--quiet`, automation mode, and JSON log format.
- Activity labels should describe work semantically, such as discovery, planning, deletion, repair, lookup, or cleanup. They must not print URLs, HTTP methods, cursors, generated-client method names, or per-key lifecycle chatter.
- Normal log or command output must clear transient activity text before writing durable lines.

## Verbose Durable Progress

- Paged read commands may emit durable progress only when `--verbose` is set and the active render mode is one-line.
- Durable page progress writes to stderr, not stdout.
- Basic paged searches use the stable shape:

```text
page size: <N>, current page: <N>, total so far: <N>, more matches: <yes|no|unknown>, next step: <prompt|auto-continue|complete|limit-reached|warning-stop|partial-complete>
```

- Process-instance progress may append a short detail or warning clause when the next action needs operational explanation, such as limit reached, prompt, partial completion, or an indeterminate backend result.
- `--batch-size` is always the per-page request size. `--limit` is the total user cap for read searches and the frozen-scope cap for mutation/discovery workflows that document that behavior.

## Prompts

- Interactive paged one-line output may prompt only after rendering the current page and only when more matches are known.
- JSON output, automation mode, and explicit confirmation modes must not prompt unexpectedly.
- Prompt text should include the current page count, cumulative count, known total when available, whether more matching resources remain, and the action being confirmed.

## Ops Discovery Summaries

- User-limited discovery must be visible in compact human output:

```text
discovery user-limited: limit <N>; pages <N>; batch size <N>
```

- Complete discovery page details are verbose-only for compact ops output unless the workflow already has a specific audited reason to show them by default.
- JSON payloads and report files must retain complete discovery fields such as limit, batch size, pages, candidates seen, candidates frozen, duplicate candidates, skipped candidates, notices, and errors where the workflow tracks them.

## Machine Output Silence

- JSON stdout must be exactly one parseable document or shared result envelope.
- Keys-only stdout must contain one key per line and no progress, prompt, spinner, found-summary, or row text.
- Quiet mode suppresses progress and activity output, while preserving the command result shape for commands that still produce data.
- Automation mode must not prompt and must not place progress or activity text in machine stdout.
- `--no-indicator` disables transient activity only; it must not change durable command results.
