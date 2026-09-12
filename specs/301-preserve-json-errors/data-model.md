# Data Model: Existing Command Error Contract

This feature changes delivery of existing records, not their schema. No storage, migration, domain model, or facade type is introduced.

## ResultEnvelope

Source: `cmd/command_contract.go`; constructed for errors in `cmd/cmd_views_contract.go`.

| Field | Existing type | Error behavior |
| --- | --- | --- |
| `outcome` | `Outcome` string | `invalid` for invalid input; otherwise existing `failed` outcome |
| `class` | string, optional | Existing normalized classification |
| `command` | string | Canonical command path from the actual Cobra command |
| `tenantContext` | optional tenant context | Attached by the existing renderer only when already available |
| `payload` | generic, optional | Omitted for these errors; do not add empty operation reports |
| `detail` | optional ResultDetail | Present for these errors with normalized message and class |

## ResultDetail

| Field | Existing type | Rule |
| --- | --- | --- |
| `message` | string | Trimmed normalized error text; preserve contextual meaning and corrective guidance |
| `class` | optional string | Same existing classification as envelope class |
| `suggestion` | optional string | Existing behavior leaves it absent on this path; do not move guidance into a new field |

The class is intentionally present at envelope and detail levels. “No duplication” concerns redundant diagnostics/message prefixes or multiple results, not removal of established schema fields.

## Contract eligibility and render mode

The actual command identifies its effective `ContractSupport`; only `ContractSupportFull` with selected JSON mode uses the shared envelope. `pickMode` selects JSON before keys-only, then ordinary one-line output. Quiet does not replace that mode selection. Capability annotations and output declarations remain unchanged.

All 12 current shared stdin-key callers are full-contract. A fixture with non-full metadata verifies the existing fallback without adding a new production caller.

## Key input

Existing flag keys precede stdin keys in the merged result. Validation of stdin preserves existing ordering and error precedence. Human-readable `filter: ` text receives its specific corrective error. Other bad keys retain their quoted content and zero-based index in the processed stdin slice, after blank lines have been removed. Caller-owned deduplication remains where it already exists.

## Failure transition

1. Existing validation or runtime action produces an error.
2. The command handler selects the existing JSON/full-contract route or human fallback.
3. The existing normalization policy determines detail, class, outcome, and exit classification.
4. One result or human diagnostic is emitted and execution terminates.
5. `--no-err-codes` substitutes the existing zero exit status but does not permit further work or change the error outcome.

No mutation state transition is introduced. Existing failed/local-precondition behavior for an empty embedded-file scope is preserved.
