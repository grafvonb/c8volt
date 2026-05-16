# Progress: Ops Purge All Process Definitions

## Mandatory Context

- GitHub issue: https://github.com/grafvonb/c8volt/issues/208
- Feature branch/directory: `208-purge-process-definitions`
- Ralph launch context: every Ralph iteration MUST include `--implementation-context specs/ralph-implementation-rules.md`.
- Commit rule: every commit subject MUST follow Conventional Commits and end with `#208`.

## Setup Notes

- The feature spec, plan, research, data model, quickstart, and CLI contract were generated from issue #208.
- The existing ops purge workflows for orphan process instances and process instances with incidents provide the closest command, report, JSON, automation, and docs patterns.
- The existing process-definition delete source of truth lives in the resource delete path used by `delete pd`; the purge workflow should delegate active-instance impact checks, `--force`, history cleanup, process-definition deletion, wait/no-wait, worker, fail-fast, and no-worker-limit behavior there.

## Codebase Patterns

- Before every implementation iteration, read and apply `specs/ralph-implementation-rules.md` in addition to the feature artifacts. Stop and surface any conflict between those rules and the feature plan.
- Keep command behavior in `cmd/`, public orchestration in `c8volt/ops`, version-neutral workflow behavior in `internal/services/ops`, process-definition discovery through existing process-definition services, and process-definition delete planning/deletion through the existing resource delete source of truth.
- Reuse existing ops purge/report/output patterns before adding new helpers. Do not shell out to `c8volt get pd` or `c8volt delete pd`.
- Preserve `get pd` and `delete pd` behavior while adding the high-level workflow.
- Model the command orchestration after `cmd/ops_purge_processinstances_with_incidents.go`: validate static flags before remote work, call `requireAutomationSupport`, use `shouldImplicitlyConfirm(cmd)`, preflight report-file paths before discovery, run a dry-run planning pass before interactive destructive confirmation, freeze discovered candidate keys into the execution request, then render/write reports through shared ops helpers.
- Model the internal workflow after `internal/services/ops/incident_purge.go`: create a result at start, validate service dependencies, perform discovery once unless frozen candidates were provided, skip planning/deletion for no-target discovery, build a delete plan from unique frozen candidates, and finish through a report-populating helper with workflow step statuses.
- Process-definition selection should reuse `cmd/get_processdefinition.go` / facade search semantics: supported filters are `--key`, `--bpmn-process-id`, `--pd-version`, `--pd-version-tag`, and `--latest`; display-only flags such as `--xml` and `--stat` stay out of the purge command.
- Process-definition deletion safety already lives in `internal/services/resource/workflow.go`: `PreviewDeleteProcessDefinitions` deduplicates keys and computes active-instance impact, while `DeleteProcessDefinitions` delegates force cancellation, active-instance drain waiting, process-instance history cleanup, process-definition delete submission, worker count, fail-fast, no-worker-limit, and no-wait behavior.
- Public ops facade changes should stay thin: add the API method, request/result models, conversions, and client delegation in `c8volt/ops`, converting service errors through `ferrors.FromDomain`.
- Command contract metadata should mirror existing state-changing ops commands: set mutation to state-changing, contract support to full, automation support to full with concrete notes, and output modes for JSON/machine support where the command renders shared envelopes.
- Generated CLI docs are refreshed via `make docs-content`; do not hand-edit `docs/cli/*` or `docs/index.md` when command metadata/help changes.

## Validation Log

- Pending: Ralph implementation iterations will record targeted validation and final `make test` results here as work units complete.

---
## Iteration 1 - 2026-05-16 18:26:19 CEST
**User Story**: Phase 1: Setup (Shared Infrastructure)
**Tasks Completed**:
- [x] T001: Record mandatory Ralph context and issue traceability in `specs/208-purge-process-definitions/progress.md`
- [x] T002: Inspect existing #186/#187/#199 ops purge/report flows, `get pd` selection, `delete pd` preflight/deletion, command contract metadata, and docs generation patterns; record reusable discoveries in `specs/208-purge-process-definitions/progress.md`
**Tasks Remaining in Story**: None - story complete
**Commit**: Recorded in Git history for this iteration
**Files Changed**:
- specs/208-purge-process-definitions/tasks.md
- specs/208-purge-process-definitions/progress.md
**Learnings**:
- The closest command/report template is the incident-based purge workflow because it already implements frozen candidate discovery, report preservation on local aborts, `shouldImplicitlyConfirm(cmd)`, and full automation metadata.
- The process-definition delete source of truth is the resource workflow; future tasks should call into that layer through facade/service boundaries rather than duplicating impact checks or deletion mechanics.
- `get pd` owns the exact process-definition filter names and latest/non-latest branching; future purge command tests should protect that display-only flags remain unsupported on the purge command.
- Documentation generation is Makefile-driven through `make docs-content`, which runs `go run ./docsgen -out ./docs/cli -format markdown`.
---
