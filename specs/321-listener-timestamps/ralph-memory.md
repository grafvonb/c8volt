# Ralph Memory

Feature: 321-listener-timestamps
Started: 2026-09-16T02:43:39Z

## Codebase Patterns

- Command execution tests can use `testx.RunCmdSubprocessInDirWithSeparateOutputs`; HTTP fixtures use `testx.WriteTestConfigForVersion` and `testx.NewIPv4Server`.
- Concurrent request observations use `testx.SafeSlice` and `testx.AtomicCounter`.
- Domain listener projection copies optional timestamp pointers directly; table-driven tests in `internal/domain/job_test.go` cover independent absence, offsets, and non-active deadlines.
- Versioned job adapters map optional generated `CreationTime` and `EndTime` pointers directly in `fromJobSearchResult`; the existing get/search fixtures are the narrow regression seam for all supported versions.

## Decisions

- The active feature pointer, branch, task artifacts, and plan agree on `321-listener-timestamps`; no conflict with `AGENTS.md`, constitution v2.0.0, or `specs/ralph-implementation-rules.md` was found.
- Use the repository-declared Go 1.26 / go1.26.2 toolchain and proportionate targeted validation; setup-only artifact changes do not require runtime tests.
- Preserve supplied timestamps without version cutoffs in v88, v89, and v810; missing/null values remain nil, while v87 retrieval stays explicitly unsupported.

## Gotchas

- The issue number is present in the feature name and tasks, but the branch begins with `codex/`; Ralph `commit.issue: auto` therefore does not infer a commit-subject suffix.

## Reusable Commands

- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./internal/domain -run 'RuntimeListenerJob|Timestamp' -count=1`
- `go test ./internal/services/job/... -run 'TestSearchJobsByKey|TestService_SearchJobs|Timestamp' -count=1`
- `go test ./internal/services/job/v87 -run 'TestService_GetJob_Unsupported|TestService_SearchJobs_Unsupported' -count=1 -v`

## Do Not Repeat

- Do not run runtime tests for setup-only or documentation-only task-state changes; use structural and whitespace checks.

## Current Handoff

- Begin US1 with T008–T013; author listener-row and element execution regressions before adding the element facade timestamp mapping and shared human timestamp-column helper.
