# Validation Guide: PD Watch Indicators

## Prerequisites

Use the current feature branch and the repository Go toolchain. All implementation validation uses isolated fixtures; no live Camunda cluster is required or permitted.

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

## Watch lifetime regression coverage

Run the runner tests and command timeout/diagnostic tests with isolated fakes:

```sh
go test ./toolx/watch -count=1
go test ./cmd -run 'TestGetProcessDefinitionWatch' -count=1
```

Verify absent and explicit-zero session duration provide no deadline even with a
short backoff timeout; positive duration expires during collection or interval
wait; invalid durations and use without watch fail validation. Verify wrapped
request timeouts retry, preserve the previous snapshot, reset after success, and
honor finite retry budgets. Check plain and structured configured logger routing,
compact normal/verbose errors, redacted DEBUG diagnostics, and original failure
exit status. Existing cancellation and output-mode exclusion coverage remains.

The original optional live viewing instructions are superseded by repository
implementation validation rules; no live-engine behavior is claimed here.

## Documentation and formatting

After implementation, format touched Go files with `gofmt -w`, update README and command help/output metadata, then run:

```sh
make docs-content
git diff --check
```

Review generated documentation for the marker legend, initial baseline, clearing semantics, failure behavior, and summary example. Do not hand-edit generated CLI docs. Broaden runtime tests only for shared-code changes or unresolved failures.

See [watch output contract](contracts/watch-output.md) for exact output and [data model](data-model.md) for state transitions. Planning-only validation consists of artifact/link/placeholder review and whitespace checks; these runtime steps are for implementation.
