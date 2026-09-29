# Ralph Memory

Feature: 324-user-task-integration
Started: 2026-09-23T09:27:07Z

## Codebase Patterns

- Scenario-specific start variables can be added without changing existing callers by keeping `runSeededProcessInstance` as a marker-only wrapper around a labeled typed-variable variant.
- Distinct fixture deployments and starts need distinct scenario labels because command log paths are keyed by scenario name.

## Decisions

- Superseding user decision: remove both live `$notIn` cases across versions, retain unit encoding tests, and validate the reduced live matrix. Other-version behavior is assumed, not proven.

## Historical observations (predate the coverage adjustment)

- On 2026-09-23, selected profile `c89local` (Camunda 8.9) returned HTTP 500 `The search server was unable to process the request` for `--var 'incident.$notIn=["99"]'`. This historical failure led to the later user-approved live coverage exclusion.
- Iteration 3 reran only `TestGetFamilyUserTaskVariables` for `c89local`; it still failed at the filter-table assertion after readiness and the earlier operator cases, so no T001 work could be validated.
- Iteration 4 found no prerequisite or profile change after iteration 3 and intentionally did not repeat the known-blocked live run; T001 was then treated as indivisible under the earlier six-operator requirement.
- Iteration 5 queried all three configured profiles read-only; `c88local`, `c89local`, and `c810local` still report gateway version `8.9.16`. The known-blocked live assertion was not repeated because neither the backend version nor required profile coverage changed.
- Iteration 6 began two minutes after that read-only check with the same branch, HEAD, harness diff, and task contract; it intentionally reused the still-valid result instead of repeating the known-failing live run.
- Iteration 7 queried all three profiles with the current source; `c88local`, `c89local`, and `c810local` still report gateway version `8.9.16`, so the version prerequisites remain unchanged and the known-failing live mutation run was not repeated.
- Iterations 8-76 reran the read-only `TestProfiles` gate: `c89local` remained ready, while `c88local` and `c810local` continued to report gateway version `8.9.16`; no prerequisite changed, so the known-failing live mutation run was not repeated.
- The same live run reached readiness for ordinary and incident fixtures: process `incident=99`, local effective `incident=1`, task/process ownership, structured Unicode data, and run marker scopes were observed. Equality, existence, inequality, and `$in` cases passed before `$notIn` failed.
- On 2026-09-23, configured profiles `c88local` and `c810local` both reported gateway version `8.9.16`; readiness rejected them before mutation because they do not provide the expected 8.8 and 8.10 coverage.

## Reusable Commands

- Typed payload check: `go test -tags=integration ./integration/cli -run '^(TestSeededVariablePayload|TestEmbeddedFixturePrefix)$' -count=1`
- Live standalone case: `go test -tags=integration ./integration/cli -run '^TestGetFamilyUserTaskVariables$' -count=1 -v -timeout=60m`

## Do Not Repeat
- Do not restore live `$notIn` cases or restart blocked profile polling. The user removed those cases across all versions; unit encoding coverage remains.

## Current Handoff
- Ralph remains stopped. Direct recovery implemented all coverage. The approved reduced matrix and complete get slice passed on c89local; see validation.md; do not claim other versions or live `$notIn` were verified.
