# Release Integration Command Matrix

This matrix describes the intended suite coverage. The executable source of
truth is `integration/scripts/run-suite.sh`; update this document whenever the
suite meaningfully changes.

## Preflight

| Command family | Coverage intent |
| --- | --- |
| `version` | Record c8volt build/version evidence. |
| `config validate` | Prove config shape before cluster calls. |
| `config show --json` | Capture effective profile/config evidence. |
| `config test-connection` | Prove connectivity and health. |
| `get cluster version` | Gate C88/C89 minor before mutation. |
| `get cluster topology` | Capture broker/partition evidence. |
| `capabilities --json` | Capture command contract and automation support. |

## Inventory

| Command family | Coverage intent |
| --- | --- |
| root/help | Catch stale top-level UX and grammar. |
| `get`, `run`, `cancel`, `delete`, `update`, `resolve` help | Catch daily workflow wording and flag drift. |
| `ops`, `ops execute`, `ops purge`, `ops repair` help | Discover nested ops surfaces and UX drift. |

## Data Setup

| Command family | Coverage intent |
| --- | --- |
| dirty-cluster baseline | Accept existing local-cluster resources as normal test input. |
| `embed list` | Verify fixture availability. |
| `embed deploy --file` | Deploy version-specific fixture data even when older versions/resources already exist. |
| `run pi -n ... --keys-only` | Generate run-owned volume data for paging, batches, stdin/key workflows, and mutation checks. |

## Read Workflows

| Command family | Coverage intent |
| --- | --- |
| `get pd --latest --limit` | Verify process-definition discovery. |
| `get pi --state --limit` | Verify bounded process-instance search. |
| `get pi --total` | Verify count/total behavior. |
| `get pi --with-vars` | Verify variable enrichment. |
| `get pi --var-exists` and `--var` | Verify native variable search. |
| `get incident` and `get job` | Capture runtime availability and UX; optional because fixture state may vary. |

## Mutating Workflows

| Command family | Coverage intent |
| --- | --- |
| `update pi --dry-run` then real `update pi` | Verify preview, confirmation bypass, mutation, and post-check. |
| `cancel pi --dry-run` then real `cancel pi` | Verify destructive preview, auto-confirm, and state observation. |
| `delete pi --dry-run` then real `delete pi` | Verify final-state deletion and absent post-check. |
| key pipelines | Keep `--keys-only` output suitable for downstream mutation commands. |

## Ops Workflows

| Command family | Coverage intent |
| --- | --- |
| `ops execute smoke-test` dry-run and real | Exercise the built-in end-to-end workflow and command report writing. |
| `ops execute retention-policy --dry-run` | Verify planning output. |
| `ops purge orphan-process-instances --dry-run` | Verify purge discovery and bounded output. |
| `ops purge process-instances-with-incidents --dry-run` | Verify incident-oriented purge planning. |
| real `ops purge ...` with `--auto-confirm`/`--force` | Verify full-force dirty-cluster mutation behavior where supported. |
| `ops repair process-instance --dry-run` | Verify repair planning when fixture data exists. |
| `ops repair incident --dry-run` | Verify incident repair planning when fixture data exists. |

## Version-Specific Checks

| Target | Coverage intent |
| --- | --- |
| C88 | `delete pd --force --auto-confirm` and broad `ops purge all-process-definitions --force --auto-confirm` must fail before mutation with clear unsupported C89 requirement. |
| C89 | Scoped and broad `ops purge all-process-definitions` dry-run/real execution validate resource deletion behavior in a dirty disposable cluster. |

## Cleanup

| Command family | Coverage intent |
| --- | --- |
| `cancel pi --state active` and fixture-scoped variants | Exercise broad/full-force cancellation without preserving existing resources. |
| `delete pi --state canceled` and fixture-scoped variants | Exercise broad/full-force deletion without preserving existing resources. |
