# Quickstart: Preserve High-Level Activity

## Prerequisites

- Go toolchain from `go.mod`.
- Repository dependencies available.
- Optional Camunda integration profile only for manual smoke checks; unit and command tests should prove the core behavior without a real cluster.

## Targeted Validation

Run activity hierarchy and HTTP fallback tests:

```bash
go test ./toolx/logging ./internal/services/httpc -count=1
```

Expected outcome:

- Lower-priority activity does not replace higher-priority activity.
- HTTP activity is treated as fallback.
- Known c8volt endpoints have resource-aware labels.
- Unknown endpoint fallbacks remain available.

Run representative command and service tests:

```bash
go test ./cmd -run 'Activity|Progress|Indicator|RunProcessInstance|DeleteProcessInstance|Ops|GetProcessInstance' -count=1
go test ./internal/services/processinstance/... ./internal/services/processdefinition/... ./internal/services/resource/... -count=1
```

Expected outcome:

- Process-instance delete progress is not overwritten by nested loads or waits.
- Bulk run progress is not overwritten by individual create/load activity.
- Ops analysis progress is not overwritten by nested element search activity.
- Process-instance search/total/orphan progress remains stable.
- Deployment wait activity is not overwritten by lower-level request activity.

Run output-mode safety checks:

```bash
go test ./cmd -run 'JSON|KeysOnly|Quiet|Automation|Machine' -count=1
```

Expected outcome:

- JSON output remains valid JSON.
- Keys-only output remains one key per line.
- Quiet and automation modes do not emit transient activity text.

## Full Validation

Run the repository test suite:

```bash
make test
```

Expected outcome:

- Full race-enabled test suite passes.

## Optional Manual Smoke Checks

Use a configured Camunda profile and an interactive terminal.

Process-instance deletion:

```bash
c8volt delete pi -b <process-id> --dry-run
```

Expected outcome:

- The spinner stays on the delete impact or delete workflow message while nested resource loads occur.

Slow-process analysis:

```bash
c8volt ops analyse slow-process-instances -b <process-id> --with-full-timeline
```

Expected outcome:

- Discovery or frozen-scope progress remains visible while runtime element searches run.

Simple fallback command:

```bash
c8volt get tenant --limit 1
```

Expected outcome:

- When activity is visible and no broader workflow is active, fallback text names the resource family instead of generic API wording.

## Documentation Review

Review README and generated CLI docs after implementation:

```bash
rg -n 'indicator|activity|progress|submitting Camunda API request|loading Camunda API data' README.md docs/cli cmd
```

Expected outcome:

- Any documented activity/progress behavior matches the new hierarchy.
- Generated docs are refreshed only if command help text changes.

## Final Validation Notes

Recorded 2026-07-28 12:05 during Ralph iteration 6.

- Documentation search passed with existing README and generated CLI wording still matching the output contract; no README changes were needed.
- `make docs-content` was not run because this feature did not change command help or generated documentation inputs during polish.
- Focused quickstart validation passed:
  - `go test ./toolx/logging ./internal/services/httpc -count=1`
  - `go test ./cmd -run 'Activity|Progress|Indicator|RunProcessInstance|DeleteProcessInstance|Ops|GetProcessInstance' -count=1`
  - `go test ./internal/services/processinstance/... ./internal/services/processdefinition/... ./internal/services/resource/... -count=1`
  - `go test ./cmd -run 'JSON|KeysOnly|Quiet|Automation|Machine' -count=1`
- Full validation passed with `make test`.
- Optional manual smoke checks were not run; they require a configured Camunda profile and interactive terminal. Automated coverage validates the activity hierarchy, fallback labels, and script-safe output contracts.
