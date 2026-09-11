# CLI Contract: Empty Selector Delete/Cancel Results

## Applicability

Commands: `c8volt delete process-instance` and `c8volt cancel process-instance`, including existing aliases such as `pi`, when selectors successfully discover zero instances. Applies to normal execution and `--dry-run`.

No flag, alias, selector rule, shared envelope schema, or exit-code mapping is added or removed.

## Output matrix

| Selected mode | Normal execution | Dry-run |
| --- | --- | --- |
| `--json` | Exactly one successful shared envelope with an empty operation-specific report payload | Exactly one successful shared envelope with an empty aggregate preview payload |
| `--keys-only` | Zero stdout bytes | Zero stdout bytes |
| Ordinary human | Exactly `found: 0` followed by a newline on stdout | Exactly `found: 0` followed by a newline on stdout |
| Quiet human | No informational empty-result message on either stream | No informational empty-result message on either stream |
| Quiet + JSON | Same JSON contract | Same JSON contract |
| Quiet + keys-only | Zero stdout bytes | Zero stdout bytes |

Resolve output mode using existing precedence. Existing auto-confirm and automation options retain that resolved mode; automation does not authorize emitting human text into machine output. Supported no-wait combinations still report `succeeded`, never `accepted`, for zero-match scopes.

## Normal JSON example

For `c8volt delete pi --state terminated --json --auto-confirm` with no matching instances, the result has the following core fields (existing optional tenant context may also appear):

```json
{
  "outcome": "succeeded",
  "command": "delete process-instance",
  "payload": {}
}
```

Cancel uses `cancel process-instance`. The empty `items` field is omitted by the existing report type. Do not introduce an item or change this container's serialization rules.

## Dry-run JSON example

Core fields for empty `cancel pi --state active --dry-run --json`:

```json
{
  "outcome": "succeeded",
  "command": "cancel process-instance",
  "payload": {
    "operation": "cancel",
    "requestedCount": 0,
    "resolvedRootCount": 0,
    "affectedCount": 0,
    "selectedFinalStateCount": 0,
    "selectedFinalState": null,
    "requiresCancelBeforeDeleteCount": 0,
    "requiresCancelBeforeDelete": null,
    "traversalOutcome": "complete",
    "scopeComplete": true,
    "warning": "",
    "missingAncestors": null,
    "previews": null,
    "mutationSubmitted": false
  }
}
```

Delete uses the corresponding command and operation. Existing optional tenant context is retained without additional discovery. JSON whitespace and property order are not contractual; a second decoded JSON value or any human text outside the envelope is forbidden.

## Side effects and completion

- Successful exit status remains unchanged.
- No confirmation prompt or mutation request occurs for the empty scope.
- No additional discovery request is introduced by rendering. Baseline discovery may include definition validation when that selector requires it.
- No new tenant lookup, progress message, or logger behavior is introduced.
- Quiet suppression refers to the empty-result information; it does not redesign unrelated diagnostics.
- Errors, user aborts, nonempty operations, explicit keys, and sparse pages retain their existing behavior.

## Regression obligations

Cover all four command/execution combinations, the output matrix, and auto-confirm/automation/no-wait overlays. Capture stdout and stderr separately. For JSON decode one envelope and require EOF afterward; check command, succeeded outcome, and the exact command-appropriate empty payload. For keys-only assert byte length zero. For human output assert exactly one unchanged summary. Verify real-terminal prompt-free completion and unchanged discovery counts. See [quickstart](../quickstart.md).
