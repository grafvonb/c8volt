# Bug Fix: Race in process-instance total test request capture

- **Slug**: process-instance-total-race
- **Fixed**: 2026-10-17
- **Assessment**: [assessment.md](assessment.md)
- **Status**: applied
- **Location override**: `specs/bugs/` as explicitly requested by the user.

## Summary

Replaced the affected test's unsynchronized request slice with the existing `testx.SafeSlice[string]`. Concurrent HTTP handlers now append safely, and the request assertion reads a synchronized snapshot.

## Changes

| File | Change | Notes |
| --- | --- | --- |
| `cmd/get_processinstance_paging_test.go` | Modified test recorder | Added `testx` import, used `SafeSlice.Append` and `Snapshot`, and documented why synchronization is necessary. |

## Tests Added or Updated

- `cmd/get_processinstance_paging_test.go::TestGetProcessInstanceTotalOutput/incident_detail_filters_count_matching_direct_incidents/direct_incidents_only` — retains the two-instance fixture, concurrent incident enrichment, unordered assertion of all three requests, exact `1\n` stdout, and empty stderr. Only recorder synchronization changed.
- No new duplicate test or production behavior change was needed.

## Local Verification

- `go test ./cmd -race -covermode=atomic -run '^TestGetProcessInstanceTotalOutput$/^incident_detail_filters_count_matching_direct_incidents$/^direct_incidents_only$' -count=20` → passed all 20 runs; `ok`, 2.238s, coverage 10.4%.
- `go test ./cmd -race -run '^TestGetProcessInstanceTotalOutput$' -count=1` → passed; `ok`, 2.010s.
- `gofmt -w cmd/get_processinstance_paging_test.go` → completed.
- `git diff --check` → passed.
- Diff review confirms scope is limited to the assessed subtest and its import. No workers were serialized and no assertions or race detection were removed.
- The pre-fix local run also passed; these clean post-fix runs are not a local failing-to-passing reproduction. The GitHub trace identifies the race, and the synchronized recorder removes the conflicting accesses.

## Deviations from Assessment

None. The assessment remains unchanged. Reports use the user-approved `specs/bugs/` directory instead of the skill's default ignored directory.

## Follow-ups

- Run `$speckit-bug-test slug=process-instance-total-race` using this directory for a verification report.
- Confirm the normal GitHub `make cover` job passes after the fix is committed and pushed. The full suite was not rerun locally for this focused test-only patch.
- Changes are uncommitted. No CLI documentation generation is required because runtime behavior is unchanged.
