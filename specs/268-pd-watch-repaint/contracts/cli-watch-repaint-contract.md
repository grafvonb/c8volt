# CLI Contract: Process Definition Watch Repaint

## Scope

This contract defines observable behavior for `c8volt get process-definition --watch` and aliases `get pd --watch` / `get pds --watch` after the repaint fix.

## Repaint Contract

- Each successful refresh attempts to repaint one terminal view before rendering the current result.
- Refresh output must not append a growing sequence of prior result blocks in interactive terminal use.
- Repaint control is part of human terminal presentation, not part of a machine-output contract.
- If terminal repaint cannot be applied in the current environment, the command still renders the current human result body without watch-only labels.

## Result Body Contract

For a given selector and display option set, the default watch result body must match the equivalent non-watch human output.

Example non-watch command:

```bash
c8volt get process-definition --bpmn-process-id invoice --latest
```

Example body:

```text
2251799813685255 tenant invoice v3
found: 1
```

Equivalent watch refresh body:

```text
2251799813685255 tenant invoice v3
found: 1
```

Rules:

- Do not print `snapshot 1:`, `snapshot 2:`, or any other watch-only snapshot label in the result body.
- Do not print watch-only `found:` rows; `found: N` remains only because normal human list output already includes it.
- Do not add tick numbers, timestamps, endpoint names, request bodies, cursors, or per-page lifecycle detail to the result body.
- Empty refreshes use the same normal human empty-list result shape as the equivalent non-watch command.
- Statistics rows use the same normal human row format as the equivalent non-watch command with statistics enabled.

## Slow Refresh Contract

- The command measures each refresh duration.
- A refresh is slow when collection plus rendering duration exceeds the configured `--watch-interval`.
- Default human mode warns once per continuous slow-refresh streak.
- Default human mode suppresses repeated warnings while refreshes continue exceeding the interval.
- A refresh that completes within the interval resets the warning streak, allowing a later slow streak to warn again.
- Slow-refresh warnings stay outside the result body wherever the terminal environment allows it.
- Verbose mode may report per-refresh timing/status outside the result body.

## Serial Refresh Contract

- Watch refreshes never overlap.
- The next refresh begins only after the previous refresh completes and the command observes its configured wait behavior.
- Broad searches and statistics-heavy refreshes may delay visible updates, but they must not spawn concurrent refresh collection.

## Incompatible Output Contract

Watch mode remains human-only.

Rules:

- `--watch --json` is rejected before lookup work.
- `--watch --keys-only` is rejected before lookup work.
- `--watch --xml` is rejected before lookup work.
- `--watch --quiet` is rejected before lookup work.
- `--watch --automation` is rejected before lookup work.
- Rejection errors are clear local validation errors.
- Non-watch invocations keep existing human, JSON, keys-only, XML, quiet, and automation behavior.

## Documentation Contract

Help, command metadata, README guidance, and generated CLI docs must describe:

- Watch mode repaints one terminal view instead of appending snapshot blocks.
- Watch result body matches normal human output.
- Watch-specific snapshot labels are absent from default output.
- Slow refreshes produce concise default warnings and more detailed verbose timing/status.
- JSON, keys-only, XML, quiet, and automation remain incompatible with watch.
