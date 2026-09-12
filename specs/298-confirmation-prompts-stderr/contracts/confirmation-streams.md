# CLI Contract: Confirmation Streams

## Output routing

When existing rules require a confirmation question, its complete existing text is written to the command's configured stderr destination, falling back to standard stderr. It has no timestamp, severity prefix, or logger decoration. Result stdout contains only the existing result format; keys-only output has one key per line.

Both default-no command/paging confirmation and default-yes selector recovery follow this routing contract. User-directed merging of the two streams is outside the separation guarantee.

## Decisions and eligibility

| Input after existing normalization | Default-no | Default-yes |
|---|---|---|
| `y` or `yes` | Accept | Accept |
| Empty line | Abort | Accept |
| Any other line | Abort | Abort |
| EOF/failed scan | Abort | Abort |

Normalization remains case-insensitive with surrounding whitespace trimmed. Auto-confirm and supported automation skip applicable prompts exactly as before. Non-terminal stdin retains existing helper behavior. Unsupported automation retains existing rejection.

Selector recovery additionally requires terminal stdout and excludes JSON, keys-only, auto-confirm, and automation. Redirecting stdout must not enable selector recovery merely because its question now uses stderr.

The helpers retain the existing abort error. Mutation callers preserve their error/exit semantics; paging callers that currently convert decline/EOF to a normal stop continue doing so. No extra result page may be fetched after a stop decision. Existing operational success verification is unchanged.

## Compatibility

No command, flag, alias, result schema, default, input policy, or backend interface changes. The intentional observable change is that prompt consumers must capture stderr rather than stdout. Prompt-write failure behavior remains unchanged; this feature introduces no new exit or retry policy.

## Acceptance boundaries

- Real terminal stdin with redirected result stdout proves default-no and direct default-yes routing.
- Complete command paging proves only keys reach stdout across repeated questions.
- Custom and inherited command stderr capture prove writer wiring.
- Eligible selector recovery and suppressed redirected-stdout recovery are separate scenarios.
- Tests of pipe input or mocked terminal detection supplement but cannot replace terminal-path tests.
