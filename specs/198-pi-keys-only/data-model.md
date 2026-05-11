# Data Model: get incident process-instance key output

## Incident Result

Represents one incident selected by direct lookup or search/list mode.

Fields used by this feature:

- `incidentKey`: Existing incident identity emitted by `--keys-only`.
- `processInstanceKey`: Existing process instance identity emitted by `--pi-keys-only` when present.
- `errorMessage`, `state`, `errorType`, process context, and flow-node context: Existing fields preserved for non-key-only output and filtering.

Validation rules:

- `--pi-keys-only` emits no line for an incident result with an empty `processInstanceKey`.
- Duplicate non-empty `processInstanceKey` values remain valid and are emitted once per incident result.

## Pipeline Key Output

Represents a line-oriented machine output mode for shell composition.

Rules:

- Output contains only keys, one per line.
- No `found:` footer is emitted.
- Human row formatting and message truncation do not apply.
- Local validation rejects incompatible output modes before incident lookup/search.

## Delete Process-Instance Key Input

Represents merged `delete pi` keys from repeated `--key` flags and stdin `-`.

Rules:

- Duplicate numeric keys are accepted as input.
- The command boundary may collapse duplicates before dry-run planning or mutation.
- This dedupe does not change `get incident --pi-keys-only` output.
