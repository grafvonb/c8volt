# Bug Verification: Race in process-instance total test request capture

- **Slug**: process-instance-total-race (explicit user input)
- **Tested**: 2026-10-17
- **Assessment**: [assessment.md](assessment.md)
- **Fix**: [fix.md](fix.md)
- **Result**: verified
- **Scope**: Local verification; reports stored under `specs/bugs/` as requested.

## Summary

The assessment's focused reproduction passed 20 repetitions with race detection and atomic coverage, and the enclosing total-output scenarios passed with race detection. Inspection confirms the affected recorder uses synchronized writes and a synchronized snapshot while retaining concurrent enrichment and all existing assertions.

## Checks Performed

| Check | Command / Action | Result | Notes |
| --- | --- | --- | --- |
| Reproduction and updated test | `go test ./cmd -race -covermode=atomic -run '^TestGetProcessInstanceTotalOutput$/^incident_detail_filters_count_matching_direct_incidents$/^direct_incidents_only$' -count=20` | pass | Exit 0; all 20 repetitions passed without a race report. |
| Regression scenarios | `go test ./cmd -race -run '^TestGetProcessInstanceTotalOutput$' -count=1` | pass | Exit 0; enclosing total-output tests passed. |
| Formatting and whitespace | `gofmt -l cmd/get_processinstance_paging_test.go && git diff --check` | pass | Exit 0; no output. Read-only checks. |
| Fix scope | Review `git diff -- cmd/get_processinstance_paging_test.go` | pass | Existing `testx.SafeSlice[string]` replaces the affected plain slice; handlers use `Append`, assertion uses `Snapshot`. Fixture, unordered three-request assertion, exact stdout, empty stderr, and production concurrency remain intact. |
| Full GitHub coverage job | `make cover` | skipped | Left to normal CI, as recommended by the assessment for this focused test-only change. No GitHub run was triggered or inspected. |

## Output Excerpts

Focused reproduction:

```text
ok  github.com/grafvonb/c8volt/cmd  1.962s  coverage: 10.4% of statements
```

Enclosing regression scenarios:

```text
ok  github.com/grafvonb/c8volt/cmd  1.700s
```

## Residual Risks

- The race is scheduling-dependent. The pre-fix local run also passed, so this is not a demonstrated local failing-to-passing reproduction. The original GitHub trace and inspection of the synchronized replacement support the fix alongside the passing checks.
- Verification exercised the assessment's automated reproduction locally; it does not establish that the original Linux GitHub full-suite coverage job now passes.
- Source code was not changed during verification. The fix and reports remain uncommitted.

## Recommendation

The fix is verified within the local scope recommended by the assessment. Commit and push when requested, then confirm the normal GitHub `make cover` job passes before treating the original CI failure as confirmed resolved there.

No `before_bug_test` or `after_bug_test` hooks are registered in `.specify/extensions.yml`.
