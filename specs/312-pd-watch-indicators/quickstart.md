# Validation Guide: PD Watch Indicators

## Prerequisites

Use branch `312-pd-watch-indicators` and the repository Go toolchain. Automated command tests need no live Camunda cluster. Manual viewing requires an existing configured Camunda connection and permission to read process definitions; do not create or delete deployments solely for this check.

## Automated behavior validation after implementation

From the repository root:

```sh
go test ./cmd -run 'Test(GetProcessDefinition|ProcessDefinitionWatch|ListProcessDefinitions|ProcessDefinitionView)' -count=1
```

Extend existing harness tests to cover this deterministic sequence:

| Refresh | Complete result or failure | Expected delta |
| --- | --- | --- |
| 1 | Keys A and B | Blank markers; 0 updated, 0 new, 0 removed |
| 2 | A changes one statistic; B unchanged; C appears | A `*`, B blank, C `+`; 1 updated, 1 new, 0 removed |
| 3 | Same visible values, reordered input | All blank; all-zero counts; preserve established output ordering |
| 4 | Partial data plus retryable error | No new view; baseline remains refresh 3 |
| 5 | A unchanged, C changes, B absent | A blank, C `*`; 1 updated, 0 new, 1 removed |
| 6 | Empty complete result | No rows; 0 updated, 0 new, 2 removed; `found: 0` |
| 7 | Empty complete result | No rows; all-zero counts |
| 8 | A reappears | A `+`; 0 updated, 1 new, 0 removed |

Also exercise initial failures before baseline, a successful empty first refresh followed by additions, multiple changed values counted once, fresh-session reset, undisplayed field changes, unsupported incident values, optional values appearing/disappearing, and width changes caused by a long neighboring row. A collection failure must not repaint. A repaint/view writer error must propagate without committing the candidate baseline.

Capture stdout and stderr separately. Assert exact summary/markers/footer and no diagnostics on stdout. Preserve selector/request counts and existing retry/termination coverage. Compare plain and styled visible alignment after removing bold controls; verify only changed values, not unchanged statistics or padding, are bold. Test an actual configured terminal writer and buffered/nonterminal writers, plus `TERM` empty/`dumb` and nonempty `NO_COLOR`. Restore environment and package globals; tests that modify them must not run in parallel.

Preserve existing non-watch human, JSON, keys-only, and XML regression tests. Retain watch rejection checks for incompatible modes. Update tests that previously equated the entire watch body with an ordinary list to assert the deliberate additions and unchanged field content instead.

## Manual read-only viewing

Build the CLI and run against an already configured environment:

```sh
go build -o /tmp/c8volt-312 .
/tmp/c8volt-312 get pd --watch --watch-interval 2s
```

The first refresh shows zero counts, blank marker prefixes, and the existing footer. Stop with Ctrl-C. If normal environment activity changes displayed values during the session, verify `*` and bold on changed values, `+` for new keys, and removed counts without stale rows. An unchanged successful refresh clears indicators. Deterministic automated fixtures above remain the authoritative proof when the live environment is idle.

Repeat with bold disabled:

```sh
NO_COLOR=1 /tmp/c8volt-312 get pd --watch --watch-interval 2s
```

Markers and counts remain readable without bold; existing repaint behavior remains. Compare ordinary `get pd`, `get pd --json`, and `get pd --keys-only` with their existing output contracts; no change summary or marker prefix is added.

## Documentation and formatting

After implementation, format touched Go files with `gofmt -w`, update README and command help/output metadata, then run:

```sh
make docs-content
git diff --check
```

Review generated documentation for the marker legend, initial baseline, clearing semantics, failure behavior, and summary example. Do not hand-edit generated CLI docs. Broaden runtime tests only for shared-code changes or unresolved failures.

See [watch output contract](contracts/watch-output.md) for exact output and [data model](data-model.md) for state transitions. Planning-only validation consists of artifact/link/placeholder review and whitespace checks; these runtime steps are for implementation.
