# Contract: Process-Instance Total-Only Output

## Command Surface Contract

`get process-instance` gains a `--total` flag that is valid only for search/list usage.

| Combination | Contract |
|--------|----------|
| `get pi --total` | Return only the numeric count of matching process instances |
| `get pi --total --key ...` | Reject as invalid |
| `get pi --total --json` | Reject as invalid |
| `get pi --total --keys-only` | Reject as invalid |
| `get pi --total --with-age` | Reject as invalid |

## Output Contract

- `--total` must emit only the numeric count value and no instance-detail rows.
- Zero matches must emit `0`.
- `--total` must not silently fall back to one-line, keys-only, or JSON detail output.

## Shared Metadata Contract

The shared process-instance page model must carry enough metadata for the command layer to distinguish:

| Metadata | Contract |
|--------|----------|
| Reported total value | Numeric total returned by the backend when available |
| Exact vs lower-bound state | Indicates whether the reported total is authoritative or capped |
| Overflow state | Continues to drive paging behavior independently of the reported total |

`OverflowState` alone is not sufficient to satisfy the `--total` contract.

## Version Semantics Contract

| Version | Reported total behavior |
|--------|--------------------------|
| `v8.7` | Surface the best available backend-reported total from the current Operate search payload when present |
| `v8.8` | Surface `totalItems` and whether it is capped |
| `v8.9` | Surface `totalItems` and whether it is capped |

When the backend marks the total as capped or lower-bound, `--total` must print that numeric lower-bound value unchanged.

## Default Behavior Preservation Contract

- Without `--total`, `get process-instance` must preserve existing detail output behavior.
- Existing filters, paging prompts, and keyed lookup semantics must remain unchanged unless explicitly required by the new validation rules.
- Direct `--key` lookup remains a strict single-resource path and must keep normal not-found behavior.

## Documentation Contract

The feature is incomplete unless:

- `README.md` explains `--total` and its count-only intent
- generated CLI docs include the new flag and its purpose
- the docs mention or reflect the lower-bound total behavior where it materially affects user expectations

## Regression Contract

The feature is incomplete unless tests prove:

- `--total` prints only a numeric value for search/list usage
- zero-match searches print `0`
- capped backend totals remain numeric lower bounds
- conflicting flag combinations are rejected explicitly
- default non-`--total` output remains unchanged
- shared page metadata conversions preserve reported total semantics across versions
