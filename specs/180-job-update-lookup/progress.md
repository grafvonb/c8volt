# Ralph Progress Log

Feature: 180-job-update-lookup
Started: 2026-05-08 11:45:05

## Codebase Patterns

- Cobra command files use package-level flag variables, register subcommands in `init()`, call `NewCli(cmd)` inside command execution, validate automation with `requireAutomationSupport(cmd)`, collect shared facade options with `collectOptions()`, and attach machine-contract metadata with `setCommandMutation`, `setContractSupport`, `setAutomationSupport`, and `setFlagContractRequired`.
- State-changing update commands use command-local planning before mutation, reject JSON plus verbose output, require `--auto-confirm` or `--automation` for non-dry-run JSON mutations, render dry-runs before mutation, skip prompts for no-op plans, and use `shouldImplicitlyConfirm(cmd)` plus `confirmCmdOrAbortFn` for material interactive changes.
- Process-instance variable update planning currently lives in `cmd/update_processinstance_variables.go` and rendering payloads live in `cmd/cmd_views_processinstance_update.go`; the task references to `cmd/update_processinstance_payload.go` and `cmd/update_processinstance_plan.go` are stale.
- Service packages follow `api.go`, `factory.go`, versioned `v87`/`v88`/`v89` subpackages, compile-time interface assertions, and `New(cfg, httpClient, log)` factories that switch on `cfg.App.CamundaVersion`.
- Unsupported version behavior returns domain `ErrUnsupported` from the versioned service, as seen in tenant v8.7 and process-instance variable update v8.7 paths.
- Generated Camunda v8.8 and v8.9 clients expose `SearchJobsWithResponse(ctx, JobSearchQuery, ...)`, `UpdateJobWithResponse(ctx, jobKey, JobUpdateRequest, ...)`, `JobSearchResult`, and `JobChangeset` with `Retries *int32` and `Timeout *int64`; `JobFailRequest` has `RetryBackOff`, which is separate from the update endpoint and remains out of scope.
- Waiters use `cfg.App.Backoff`, optional context deadlines, `backoff.InitialDelay` plus `backoff.NextDelay`, max-retry checks, verbose polling logs, and return confirmation failure without converting it into mutation success.
- Incident-enriched process-instance human output includes `jobKey` only when present in `incidentHumanFields`; regression coverage already asserts the full incident line containing `jobKey=...`.
- Docs are regenerated through `make docs-content`, which runs `go run -ldflags "$(LDFLAGS)" ./docsgen -out ./docs/cli -format markdown` and syncs `docs/index.md` from `README.md`.

---

---
## Iteration 1 - 2026-05-08 11:46:46 CEST
**User Story**: Phase 1: Setup (Shared Discovery)
**Tasks Completed**: 
- [x] T001: Inspect get/update command registration and command metadata patterns
- [x] T002: Inspect job-related generated client methods and types
- [x] T003: Inspect versioned service package patterns
- [x] T004: Inspect existing waiter/backoff and confirmation patterns
- [x] T005: Inspect jobKey incident output and regressions
- [x] T006: Inspect README, generated docs, and docs generation workflow
- [x] T077: Inspect current variable update dry-run, plan, confirmation, JSON, and no-op behavior
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**: 
- specs/180-job-update-lookup/tasks.md
- specs/180-job-update-lookup/progress.md
**Learnings**:
- `update job` should mirror the process-instance variable update validation sequence: JSON guardrails before planning, lookup-backed plan construction, dry-run/no-op render paths before mutation, then local confirmation before calling the facade.
- Job implementation can reuse generated v8.8/v8.9 `SearchJobs` and `UpdateJob` methods directly in versioned job services while keeping retry confirmation in a dedicated job waiter.
- The current setup discovery changed only Speckit tracking files; no source behavior was modified.
---
