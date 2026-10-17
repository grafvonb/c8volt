# Bug Assessment: Race in process-instance total test request capture

- **Slug**: process-instance-total-race
- **Created**: 2026-10-17
- **Source**: Pasted GitHub CI output supplied as `Pasted text.txt`
- **Verdict**: valid
- **Severity**: medium
- **Assessed revision**: `f47a9f05581d5143734dd89af6475f53d080c54e` on `312-pd-watch-indicators`
- **Location override**: User explicitly requested `specs/bugs/` because `.specify/` is ignored by Git. The slug was suggested by the assistant and accepted by the user.

## Report (verbatim or summarized)

The supplied CI output starts with:

```text
mkdir -p .coverage
GORACE=" atexit_sleep_ms=0" go test ./... -race -covermode=atomic -coverprofile=.coverage/coverage.out
```

It reports two data races involving the slice header and backing storage at the same handler append:

```text
WARNING: DATA RACE
Read at 0x00c000016db0 by goroutine 5720:
  github.com/grafvonb/c8volt/cmd.TestGetProcessInstanceTotalOutput.func2.1.1()
      /home/runner/work/c8volt/c8volt/cmd/get_processinstance_paging_test.go:113 +0xfe
Previous write at 0x00c000016db0 by goroutine 5709:
  github.com/grafvonb/c8volt/cmd.TestGetProcessInstanceTotalOutput.func2.1.1()
      /home/runner/work/c8volt/c8volt/cmd/get_processinstance_paging_test.go:113 +0x1b7
```

The second report includes `runtime.growslice` and the same source line. Both goroutines originate in `net/http.(*Server).Serve`. The failing subtest is:

```text
--- FAIL: TestGetProcessInstanceTotalOutput
    --- FAIL: TestGetProcessInstanceTotalOutput/incident_detail_filters_count_matching_direct_incidents
        --- FAIL: TestGetProcessInstanceTotalOutput/incident_detail_filters_count_matching_direct_incidents/direct_incidents_only
            testing.go:1712: race detected during execution of test
FAIL github.com/grafvonb/c8volt/cmd
```

The original attachment is `/Users/adam.boczek/.codex/attachments/acafd1a5-eb6f-4e77-a700-cbad5c33a6ea/Pasted text.txt`; the excerpts above preserve the decisive evidence without requiring access to that local attachment.

No GitHub issue or workflow-run URL was supplied. No external page was fetched; URL host classification and fetch policy are not applicable. “On GitHub” is treated as the reported execution environment, not an instruction to publish an issue.

## Symptom

The GitHub race-enabled coverage job fails because concurrent requests to a test HTTP server mutate its shared request recorder without synchronization. The test should safely record all three requests, assert the unordered request set, print `1\n` on stdout, and leave stderr empty.

This blocks CI intermittently. The supplied evidence does not demonstrate a race in production state or an incorrect production total.

## Reproduction

1. Use the current repository with its Go toolchain (the CI trace identifies Go 1.26.2).
2. Run the existing failing subtest with race detection and atomic coverage instrumentation:

   ```sh
   go test ./cmd -race -covermode=atomic -run '^TestGetProcessInstanceTotalOutput$/^incident_detail_filters_count_matching_direct_incidents$/^direct_incidents_only$' -count=20
   ```

3. When the two incident requests overlap at the unsynchronized append, the race detector reports `cmd/get_processinstance_paging_test.go:113`. The race is scheduling-dependent; a clean invocation does not prove the shared slice is safe.

**Local assessment result**: The command above passed all 20 repetitions (`ok`, 2.204s, coverage 10.4%). No local race was reproduced. The supplied GitHub trace directly reports the conflicting accesses, and the current code still contains them.

The reported broader invocation is the repository's `make cover` target. `.github/workflows/go.yml:36` invokes `make cover`, and `Makefile:326` runs the race-enabled coverage command. The full suite was not rerun during this assessment because the trace and focused inspection already identify the unsafe recorder.

## Suspected Code Paths

- `cmd/get_processinstance_paging_test.go:111` — declares the shared plain `[]string` recorder inside the affected subtest.
- `cmd/get_processinstance_paging_test.go:113` — appends to that recorder in each HTTP handler without a lock; both conflicting stacks identify this line.
- `cmd/get_processinstance_paging_test.go:118` — fixture returns two process instances, keys `123` and `124`, both with incidents.
- `cmd/get_processinstance_paging_test.go:138` — compares the captured request set after executing the command; the replacement must read through synchronized snapshot access.
- `internal/services/processinstance/search.go:76` — `SearchProcessInstancesTotal` applies local filtering when counting matching results.
- `internal/services/processinstance/search.go:249` — `filterProcessInstancesWithDirectIncidents` delegates incident detail enrichment.
- `internal/services/processinstance/enrichment.go:35` — `EnrichProcessInstancesWithIncidents` uses a worker pool at line 40 and issues per-instance incident searches at line 42, explaining concurrent handlers.
- `testx/safe_slice.go:10` — existing mutex-protected collector; `Append` at line 15 serializes writes and `Snapshot` at line 21 returns a protected copy. Its documented purpose is concurrent HTTP test request capture.

## Root Cause Hypothesis

**Confidence: high.** The direct-incident total test assumes that request capture is serial, but incident enrichment intentionally uses concurrent workers. Both incident handlers execute `requests = append(requests, ...)` on the same slice. Concurrent appends race on its header and potentially on its backing array, matching both CI reports. The test does not need `t.Parallel()` for this to occur: `net/http` serves requests concurrently. `require.ElementsMatch` already allows nondeterministic request ordering, but it does not make collection safe.

Git blame attributes the recorder declaration and append to `01ff5d140` dated 2026-08-10. They were not introduced by the recent #312 watch-output changes. This assessment does not establish when overlapping execution first became possible or when CI first observed it.

## Proposed Remediation

**Preferred**: Replace only the affected subtest's plain slice with `testx.SafeSlice[string]`. Record requests with `requests.Append(r.Method + " " + r.URL.Path)` and pass `requests.Snapshot()` to the existing `require.ElementsMatch`. Add the existing `testx` package import to the test file. Retain the two-instance fixture, normal worker concurrency, all three expected requests, exact stdout count, and empty stderr assertion.

This reuses the repository's established concurrency-safe test recorder. It fixes both collection and observation without changing command behavior, services, schemas, output, or production concurrency. Keep the patch focused; unrelated capture helpers can be assessed separately if evidence identifies another race.

**Alternative**: Protect append and copied reads with a local `sync.Mutex`. This works but duplicates `testx.SafeSlice`, which already implements the required behavior. Do not force one worker, remove incident requests, disable race detection, or add sleeps to hide the overlap.

**Files likely to change**:

- `cmd/get_processinstance_paging_test.go` — import, recorder declaration, append, and final snapshot assertion in the affected subtest.

**Tests to add or update**:

- Preserve and update the existing subtest; no duplicate implementation-mirroring test is needed for the already established collector.
- Rerun the focused race command above after the fix, preserving multiple iterations and atomic coverage instrumentation.
- Run `go test ./cmd -race -run '^TestGetProcessInstanceTotalOutput$' -count=1` to cover the enclosing total-output scenarios.
- Let the normal GitHub `make cover` job confirm the original full-suite failure is resolved. A focused test-only patch does not by itself require another local full-suite run.
- Run `gofmt` on the touched test file and `git diff --check`. No CLI documentation regeneration is required because production behavior is unchanged.

## Risks & Considerations

- **Severity rationale**: Medium because this causes nondeterministic CI failures and can corrupt the test's recorded request set, blocking otherwise valid changes. No customer-data loss or production-state race is established.
- Request ordering remains intentionally nondeterministic; keep `ElementsMatch` rather than introducing an ordered comparison.
- Synchronize reads as well as writes by using `Snapshot`; do not expose the collector's internal slice.
- Local reproduction was negative, so the validation report must distinguish a clean local run from the positive CI race evidence.
- No API breakage, migration, production performance cost, or output-contract change is expected from the preferred fix.
- No source changes or commits were made during assessment. This assessment is under version-controlled `specs/` as requested, but has not been committed or published to GitHub.

## Open Questions

- The exact GitHub workflow-run URL and failing commit SHA were not supplied. They would establish CI provenance, but are not required to fix the unsafe recorder demonstrated by the attached trace and current code.
