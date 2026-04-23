# Contract: Process Definition Incident Statistics

## Command Contract

The feature applies to:

- `get process-definition --stat`
- `get pd --stat`

The command remains read-only and keeps the existing selectors and output modes. The only contract change is the meaning and presence of the `in:` segment in stats rendering.

## Incident Count Contract

| Semantic | Contract |
|--------|----------|
| Supported versions | `in:` is the number of process instances that currently have at least one active incident |
| Counting rule | Each affected process instance is counted at most once |
| Supported zero | Render `in:0` |
| Unsupported versions | Omit `in:` entirely |

The shared process-definition statistics model records this boundary with:

- `Incidents`: the numeric incident-bearing process-instance count
- `IncidentCountSupported`: `true` when `Incidents` is authoritative for rendering, `false` when `in:` must be omitted

Raw incident totals are not a valid implementation of this contract.

## Version Contract

| Version | `ac/cp/cx` | `in:` behavior |
|--------|-------------|----------------|
| `v8.7` | Reject `--stat`; no equivalent native generated-client statistics endpoint is available | Omit `in:` entirely |
| `v8.8` | Source `ac` from native process-definition instance-version statistics; preserve existing `cp/cx` behavior | Render incident-bearing active process-instance count from the same native statistics |
| `v8.9` | Source `ac` from native process-definition instance-version statistics; preserve existing `cp/cx` behavior | Render incident-bearing active process-instance count from the same native statistics |

The feature is incomplete if `v8.7` still shows `in:` or if `v8.8`/`v8.9` still show `in:-` for supported stats.

## Service Contract

| Version | Planned stats source rule |
|--------|----------------------------|
| `v8.7` | Keep current unsupported boundary; do not synthesize counts through process-instance enumeration |
| `v8.8` | Keep `cp/cx` from existing process-definition stats enrichment and source `ac/in` from native process-definition instance-version statistics by exact definition key |
| `v8.9` | Keep `cp/cx` from existing process-definition stats enrichment and source `ac/in` from native process-definition instance-version statistics by exact definition key |

## Rendering Contract

When `Statistics` is not requested:

- no bracketed stats segment is shown

When `Statistics` is requested on supported versions:

- bracketed stats remain in the existing order
- `ac:` is the active process-instance count for the exact definition version
- `in:` is present
- `in:0` is allowed and must not collapse to `-`

When `Statistics` is requested on unsupported versions:

- the other stat fields keep their existing meaning
- `in:` is omitted entirely

## Documentation Contract

- `README.md` must describe the visible `get process-definition --stat` behavior at least at a high level.
- Generated CLI docs under `docs/cli/` must reflect the same behavior after regeneration.

## Regression Contract

The feature is incomplete unless tests prove:

- the supported-version incident value counts affected process instances, not raw incident totals
- supported zero values render as `in:0`
- unsupported versions omit `in:`
- `ac`, `cp`, and `cx` remain unchanged
- docs are updated and regenerated to match the shipped command behavior
