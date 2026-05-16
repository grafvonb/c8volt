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

## Validation Log

- Pending: Ralph implementation iterations will record targeted validation and final `make test` results here as work units complete.
