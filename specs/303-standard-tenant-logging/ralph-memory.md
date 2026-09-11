# Ralph Memory

Feature: 303-standard-tenant-logging
Started: 2026-09-11T16:49:28Z

## Codebase Patterns

- The planned correction is CLI rendering only: both tenant-context emitters live in `cmd/processinstance_mutation_progress.go` and should reuse `printOpsDurableLine` from `cmd/ops_progress_render.go`.

## Decisions

- Feature artifacts and the mandatory Ralph implementation rules are aligned; no conflict blocks implementation.

## Gotchas

- Attached-logger tests are required because no-logger fallback tests cannot prove severity, formatting, or threshold filtering.

## Reusable Commands

- Baseline: `go test ./cmd -run 'Test(ProcessInstanceMutation|CancelProcessInstance|DeleteProcessInstance|Confirmation)' -count=1`

## Do Not Repeat

- Do not treat a passing pre-change baseline as proof of the new logger behavior.

## Current Handoff
- Complete T002: inspect the four named command/test files, run the targeted baseline, and record the result in `quickstart.md`.
