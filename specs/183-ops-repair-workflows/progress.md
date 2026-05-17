# Progress: Ops Repair Workflows

## Traceability

- Issue: #183
- Branch: `183-ops-repair-workflows`
- Mandatory implementation context: `specs/ralph-implementation-rules.md`
- Commit subject suffix: `#183`

## Clarification Gate

- 2026-05-17: No critical ambiguities detected worth formal clarification before planning. The issue defines command targets, input modes, frozen target behavior, job applicability, variable update scope, dry-run, reports, architecture constraints, and out-of-scope behavior.

## Codebase Patterns

- `cmd/ops_repair.go` already defines the grouping command and must remain free of top-level target `--key` semantics.
- `cmd/ops_purge_processinstances_with_incidents.go` demonstrates incident-filtered ops workflow flags, report handling, pre-mutation planning, confirmation, automation metadata, and deterministic rendering patterns.
- `internal/services/ops/api.go` currently injects process-instance, incident, process-definition, resource, and cluster services; this feature must add job service injection for repair without bypassing resource services.
- `c8volt/ops/model.go` already has shared workflow statuses, but repair requires adding `not_applicable`.
- Existing incident primitives live in `internal/services/incident`; process-instance search and variable updates live in `internal/services/processinstance`; job lookup and update live in `internal/services/job`.

## Validation Notes

- Planning artifacts only so far; no implementation validation has been run.
