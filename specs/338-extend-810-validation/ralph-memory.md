# Ralph Memory

Feature: 338-extend-810-validation
Started: 2026-10-14T09:50:16Z

## Codebase Patterns

- Integration release policy is an ordinary, no-build-tag Go package at `integration/internal/validation`; keep it independent of the tagged `integration/cli` TestMain.
- Checked-in policy JSON is embedded by `integration/assets/embed.go`, so `validation.LoadDefaultPolicy` works from arbitrary working directories.
- Stable catalog scenario IDs use the existing integration test entry-point names, including `TestGetFamilyUserTask`, `TestGetFamilyUserTaskVariables`, and `TestVolumeGetFamilyUserTaskVariables` selected by the existing family regexes.

## Decisions

- Strict release mode supports canonical minors 8.9 and 8.10 and joins every applicable required catalog row to current-run, per-profile evidence. Camunda 8.8 remains exploratory baseline/volume scope.
- `passed_with_gaps` is reserved for named nonrequired proof gaps and always has `coverageComplete=false`; required rows cannot carry accepted gaps.
- Minimum proof levels are explicit and incomparable where necessary: controlled proof never satisfies live proof, and submission/dry-run proof cannot satisfy live post-state.

## Gotchas

- Child slice and observation paths must stay lexically beneath the absolute invocation root; reducers reject foreign-run, unselected-profile, duplicate, unknown, and foreign-minor records.
- Every persisted request, target, slice, scenario evidence, catalog, and summary uses schema version 1. Do not omit it when T002 creates runner records.
- Summary reasons are redacted, but T002 must also avoid persisting raw configuration, credentials, or credential-bearing route identities in manifests and child artifacts.

## Reusable Commands

- `go test ./integration/internal/validation -race -count=1`
- `go test ./integration/... -race -count=1`
- `go vet ./integration/internal/validation`

## Do Not Repeat

- Do not read policy files through repository-relative runtime paths; use `validation.LoadDefaultPolicy`.
- Do not infer completeness from child exit code or discovered files; reduce the frozen catalog against normalized current-run evidence.

## Current Handoff
- Continue with T002 in US1: implement the sequential runner against the T001 request/catalog/reducer API without wiring Make yet.
