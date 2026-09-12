# CLI Contract: Selector Tenant Diagnostics

Applies to the existing progress and non-dry-run confirmation tenant emitters shared by `cancel process-instance` and `delete process-instance`.

## Severity and Format

| Existing message marker | INFO threshold | WARN threshold | ERROR threshold |
| --- | --- | --- | --- |
| Warn=false | INFO record | Suppressed | Suppressed |
| Warn=true | WARN record | WARN record | Suppressed |

Use the configured standard record format. Plain logs retain the standard severity prefix; JSON logs are standard structured records with message and severity fields. The message value equals the existing tenant line text. No bespoke format or schema is introduced. Default `plain-time` and supported `text` formats inherit the shared helper's existing behavior.

## Eligibility and Streams

- Existing human/verbose/debug progress guards remain unchanged; quiet, automation, and machine-result suppression cannot be bypassed by logger configuration.
- Confirmation-context guards remain unchanged and are not replaced by progress guards.
- Emitted diagnostic records use the existing attached logger's destination (stderr in normal command setup). Without a logger, fallback uses configured command stderr.
- JSON results retain the existing envelope, payload, and omission/null rules. Keys-only results retain one key per line and zero bytes for an empty result.
- Dry-run preview rendering keeps its existing output destination and policy. The correction adds no diagnostic records to result stdout.
- Interactive control text remains plain and uses configured/inherited command stderr; selecting JSON log format does not convert prompts into JSON records.
- Preserve output-mode precedence and supported auto-confirm/no-wait combinations. Auto-confirm skips the question under existing policy; it does not grant additional logging eligibility.

## Ordering and Execution

Configured-tenant provenance, override, scope, affected-tenant summaries, and additional warnings retain their producer-defined order and warning flags. The existing rendered marker is set before emission even when all records are filtered. No logging change affects discovery continuation, target selection, confirmation, mutation requests, completion reporting, or exit status.

## Acceptance Evidence

Both emitters require attached-logger plain/JSON tests with all three thresholds, separate streams, exact message order and severity, and repeated calls. Command regressions cover both delete/cancel and existing mode combinations. JSON result tests decode the one expected envelope and require EOF; keys-only tests assert exact bytes. Real-terminal fixtures verify prompt placement, acceptance/abort, and prompt-free empty-scope completion. Existing request assertions prove that rendering adds no backend work.
