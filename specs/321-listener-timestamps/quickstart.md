# Validation Guide: Listener Timestamps

## Prerequisites

Run from the repository root on `codex/321-listener-timestamps`, after implementation. Use the Go version/toolchain declared in `go.mod`. Automated tests use local fixtures and do not require a Camunda server. The commands below are implementation validation instructions, not claims that tests have already run.

## Targeted automated validation

```sh
go test ./internal/services/job/... -run 'TestSearchJobsByKey|TestService_SearchJobs|Timestamp' -count=1
go test ./internal/domain -run 'RuntimeListenerJob|Timestamp' -count=1
go test ./c8volt/job -run 'TestClient_GetJob|TestClient_SearchJobs|Timestamp' -count=1
go test ./c8volt/element ./c8volt/process ./c8volt/ops -run 'Listener|Timestamp' -count=1
go test ./internal/services/element/... ./internal/services/processinstance/... ./internal/services/ops/... -run 'Listener|SlowProcessAnalysis' -count=1
go test ./cmd -run 'Listener|Timestamp|SlowProcessAnalysis' -count=1
```

Extend existing test names or use a `Timestamp`/`Listener` name for new coverage so these patterns select the new cases. Confirm tests actually execute; a no-tests match is not validation. Include public job page conversion where its existing test name falls outside these patterns by running that test explicitly.

Expected proofs:

1. v88/v89/v810 mapping retains distinct source times; missing-field fixtures still succeed; v87 unsupported behavior stays intact.
2. Every public mapping preserves both values independently; absent fields disappear from JSON while non-active deadlines remain.
3. All four commands obey the [output contract](contracts/listener-timestamps.md). Use command execution fixtures to detect mapping loss, not just constructed view objects.
4. Row tests cover both listener kinds, completed/activated/canceled/created/failed/blank/unknown states, all creation/end presence combinations, absent deadlines, and mixed rows with error fields. Assert tag order, stable alignment, no fabricated values, and existing offset formatting.
5. Command JSON tests capture stdout and stderr separately, decode exactly one existing result envelope and require EOF. Preserve requested-empty versus unrequested arrays, mode validation, error behavior, and ordinary output without listeners.
6. Reuse walk default/children/parent/flat coverage and slow-analysis fixtures to assert unchanged grouping, durations, and analysis outcomes with timestamp-bearing listener data.

## Optional live inspection

Use an existing configured Camunda environment that supports listener lookup and has known element/process keys with retained listener jobs. Set shell variables to those keys and a valid configuration file. Select a slow-analysis target that satisfies existing analysis selection criteria. Live environments may omit historical timestamps; controlled fixtures above remain authoritative for the full matrix.

```sh
make build
./bin/c8volt --config "$C8VOLT_CONFIG" get element --key "$ELEMENT_KEY" --with-listeners
./bin/c8volt --config "$C8VOLT_CONFIG" get process-instance --key "$PROCESS_KEY" --with-elements --with-listeners
./bin/c8volt --config "$C8VOLT_CONFIG" walk process-instance --key "$PROCESS_KEY" --with-elements --with-listeners
./bin/c8volt --config "$C8VOLT_CONFIG" ops analyse slow-process-instances --key "$PROCESS_KEY" --with-listeners
./bin/c8volt --config "$C8VOLT_CONFIG" --json get element --key "$ELEMENT_KEY" --with-listeners
```

Repeat each affected command with `--json` and compare the timestamps in its existing nested result structure. Use a copy of the configuration with `app.show_timezone_offset: true` to verify numeric offsets, then compare with the default false setting. Do not interpret absent `e:` as a fabricated completion or display a retained completed-job deadline as `d:`. Inspect equivalent results without listener enrichment to confirm unchanged element/process durations.

## Documentation and final checks

Update README and the four command descriptions/examples, then run:

```sh
make docs-content
git diff --check
```

Review generated CLI references and README-derived documentation for consistent tag definitions and the completed-listener example. Run `gofmt` on touched Go files during implementation. Broaden to `make test` only if targeted failures or a wider actual diff justify it, as described in [plan.md](plan.md).

For planning-only edits, validate Markdown links, required artifacts, whitespace, and absence of unresolved placeholders; do not run runtime tests or regenerate command documentation.

## Recorded implementation validation

The targeted checks above were executed successfully during Ralph iterations 2–6, with these explicit additions and narrower selections where the consolidated patterns do not name every regression:

- `go test ./internal/services/job/v87 -run 'TestService_GetJob_Unsupported|TestService_SearchJobs_Unsupported' -count=1 -v` passed, preserving the unsupported-version boundary.
- `go test ./c8volt/job -run 'TestClient_SearchJobsPage_OmitsMissingTimestamps' -count=1` passed, covering the single-page conversion not uniquely selected by the consolidated job pattern.
- The element command selection included keyed and search human output, JSON output, help, validation, and the shared timestamp-column tests.
- The process/walk/slow-analysis command selection included keyed, list, family, children, parent, flat, normal-timeline, full-timeline, JSON, and v8.7 unsupported paths.
- Original facade and enrichment selections covered populated, requested-empty, and unrequested listener collections, retained deadlines, timestamp transport, and ownership/request assertions. Independent optional-field coverage was distributed across the suite; the listener-enriched analysis test checked the root duration but did not yet compare complete analysis results. The review follow-up below closes those specific gaps.
- `make docs-content` completed after the command-source guidance changes. Final review confirmed the four generated command references match their source descriptions, README and generated index use the same timestamp definitions, all touched Go files produce an empty `gofmt -d`, and `git diff --check main...HEAD` passes.

No targeted failures or broader-impact changes required `make test`, so the full race suite was not rerun. Optional live inspection was not performed; controlled fixtures remain the authoritative validation for missing and independently populated timestamps.

## Review follow-up validation

The follow-up changes tests and validation evidence only; production code and generated command documentation are unchanged.

- Deadline suppression assertions for process get and walk now reject a `d:` token anywhere on the target listener row, including after creation/end tags.
- `TestListenerTimestampCommandsHonorTimezoneConfig` executes all four commands with `app.show_timezone_offset` explicitly false and true. It verifies exact creation/end/deadline tokens, clean stderr, and two listener discovery requests for each case, including an activated listener with an end time.
- Each facade's `TestRuntimeListenerJobJSONPreservesTimestampsAndCollectionStates` now sends both, creation-only, end-only, and neither-present domain listener timestamps through its converter and JSON marshaler, checking omission, supplied values, and retained canceled-job deadlines.
- `TestSlowProcessAnalysisWithListenersAttachesOnlyMatchingElementJobs` now runs identical input with and without listener lifecycle timestamps. After clearing only those timestamp fields, it compares the complete analysis result, including process/element/transition durations, rankings, and associations for that fixture.

The following targeted checks passed after these changes:

```sh
go test ./cmd -run 'TestListenerTimestampCommandsHonorTimezoneConfig|TestGetProcessInstanceWithElementsAndListeners_Human|TestWalkProcessInstanceCommand_WithListenersFamily' -count=1
go test ./c8volt/element ./c8volt/process ./c8volt/ops -run '^TestRuntimeListenerJobJSONPreservesTimestampsAndCollectionStates$' -count=1
go test ./internal/services/ops -run '^TestSlowProcessAnalysisWithListenersAttachesOnlyMatchingElementJobs$' -count=1
```

Touched Go files were formatted and `git diff --check` passed. These are focused fixture-based checks, not an exhaustive cross-product of every command mode, lifecycle state, and timestamp combination. Full race-suite and live-server validation remain unperformed. Existing Ralph commit subjects remain unchanged; future feature commits should explicitly reference #321 rather than relying on issue-number inference from the namespaced branch.
