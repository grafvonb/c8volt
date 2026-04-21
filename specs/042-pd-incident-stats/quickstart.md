# Quickstart: Report Process Definition Incident Statistics

## Planned Behavior

- `get process-definition --stat` keeps its existing command path and non-incident statistics fields.
- On `v8.8` and `v8.9`, the output includes `in:<count>` where `<count>` is the number of process instances that currently have at least one active incident for that process definition.
- On `v8.8` and `v8.9`, a supported zero value must render as `in:0`, not `in:-`.
- On `v8.7`, the `in:` segment is omitted entirely because the repository cannot derive that value reliably from the current supported surface.
- README and generated CLI docs must describe the supported-version and unsupported-version behavior explicitly.

## Implementation Notes

- Start in the existing versioned services under `internal/services/processdefinition/`; keep endpoint selection and aggregation there rather than pushing version branching into `cmd/`.
- Reuse the current `GetProcessDefinitionStatistics` enrichment for `ac`, `cp`, and `cx`.
- Source supported-version `in:` from the newer incident/process-instance statistics endpoint(s) exposed by the generated `v8.8` and `v8.9` clients.
- Adjust the shared/domain process-definition statistics model only as much as needed to represent “supported zero” versus “unsupported,” using `IncidentCountSupported` plus the existing `Incidents` count.
- Update `oneLinePD(...)` in `cmd/cmd_views_get.go` so it can:
  - keep `ac`, `cp`, and `cx` on their current formatting path
  - show `in:0` when supported
  - omit `in:` entirely when unsupported
- Keep `v8.7` truthfulness by omission rather than by placeholder text.

## Verification Focus

1. Confirm `v8.8` service tests prove the incident value uses incident-bearing process-instance count semantics rather than raw incident totals.
2. Confirm `v8.9` service tests prove the same semantics.
3. Confirm `v8.7` tests prove unsupported behavior remains explicit and does not surface `in:`.
4. Confirm `cmd/get_test.go` covers supported non-zero rendering, supported zero rendering, and unsupported omission.
5. Confirm any shared process/facade model changes remain JSON-compatible and are covered by focused tests.
6. Confirm README and generated CLI docs describe the visible behavior accurately.

## Suggested Verification Commands

```bash
go test ./c8volt/process -count=1
go test ./internal/services/processdefinition/... -count=1
go test ./cmd -count=1
make docs-content
make test
```

Run the focused suites first so model, service, and renderer failures are isolated before the full repository gate.

## Manual Smoke Ideas

Use the same process definition key or BPMN process ID across supported and unsupported configs:

```bash
./c8volt --config /tmp/c8volt-v88.yaml get pd -b C88_SimpleUserTaskWithIncident_Process --stat
./c8volt --config /tmp/c8volt-v89.yaml get pd -b C89_SimpleUserTaskWithIncident_Process --stat
./c8volt --config /tmp/c8volt-v87.yaml get pd -b C87_SimpleUserTaskWithIncident_Process --stat
./c8volt --config /tmp/c8volt-v88.yaml get pd -b C88_SimpleUserTaskWithIncident_Process --stat --json
```

Check that:

- `v8.8` and `v8.9` show `in:<count>` and keep the other stats fields unchanged
- supported zero counts render as `in:0`
- `v8.7` omits `in:` entirely
- JSON output stays aligned with the shared process-definition statistics model
- README notes and regenerated CLI docs match the shipped behavior
