# Research: Keyed Process-Instance Incident Details

## Decision: Use generated process-instance incident search for v8.8 and v8.9

**Rationale**: Both generated Camunda clients expose `SearchProcessInstanceIncidentsWithResponse(ctx, processInstanceKey, body, ...)`, plus `IncidentSearchQuery`, `IncidentFilter`, and `IncidentResult.ErrorMessage`. This matches the issue requirement and keeps incident lookup in the tenant-aware Camunda API path.

**Alternatives considered**: General incident search was rejected as the primary path because the issue calls out process-instance incident search APIs. Direct incident get-by-key was rejected because the command starts from process-instance keys and must avoid tenant-unsafe lookups.

## Decision: Treat v8.7 incident enrichment as unsupported

**Rationale**: The existing v8.7 process-instance direct lookup path is already constrained by tenant-safety limits. Even though older generated clients expose incident search shapes, this feature must not create a tenant-unsafe enrichment path for keyed lookup.

**Alternatives considered**: Falling back to general incident search or Operate direct incident lookup was rejected because it could undermine the tenant-safety boundary called out in the issue.

## Decision: Use an enriched output wrapper only when the flag is present

**Rationale**: Current default JSON serializes `process.ProcessInstances` and current human output uses `oneLinePI`. Adding incident details directly to the base process-instance model would risk changing default output for callers that did not request the feature. Human-readable incident messages should instead render as indented `incident:` lines directly below the matching row when enrichment is requested.

**Alternatives considered**: Adding `incidents` directly to `ProcessInstance` was rejected because it would make absent-vs-not-requested harder to distinguish and could change default JSON output.

## Decision: Validate `--with-incidents` before lookup side effects

**Rationale**: The flag is scoped to keyed lookup only. Failing early for missing `--key` or search-mode combinations keeps the CLI predictable and prevents accidental broad incident searches.

**Alternatives considered**: Ignoring the flag in search mode was rejected because silent no-op behavior would be surprising and hard to catch in automation.

## Decision: Preserve search-mode incident filters

**Rationale**: `--incidents-only` and `--no-incidents-only` already describe search filtering, not message enrichment. The new flag should not alter those existing workflows.

**Alternatives considered**: Extending `--with-incidents` to search mode was rejected as out of scope for issue 154 and significantly larger because paged search enrichment would need separate performance, continuation, and output contracts.
