# Research: Walk Process-Instance Incident Details

## Decision: Reuse issue #154 incident detail models and lookup behavior

**Rationale**: Issue #154 already introduced `ProcessInstanceIncidentDetail`, enriched process-instance output, tenant-aware incident search methods, and v8.7 unsupported behavior. Reusing those models keeps this feature aligned with existing output conventions and avoids adding a parallel incident representation for walk.

**Issue #154 review**: The persisted issue text and current code agree on the core fields needed here: incident key, process-instance key, tenant ID, state, error type, error message, flow-node/job metadata, root process-instance key, and process-definition identifiers. No field mismatch was found between the issue #154 behavior and the walk enrichment needs. Walk-specific output should add traversal metadata around the same incident detail model rather than changing the incident fields.

**Alternatives considered**:

- Create walk-specific incident models. Rejected because it would duplicate #154 and increase JSON drift.
- Fetch incidents directly from generated clients in `cmd/walk_processinstance.go`. Rejected because command code should stay thin and tenant/version behavior belongs in the facade/service layers.

## Decision: Enrich only process instances returned by traversal

**Rationale**: The issue says to fetch incident data for walked process instances returned by the walk. Using `TraversalResult.Keys` and `TraversalResult.Chain` preserves existing mode selection, ordering, partial results, and tenant boundaries.

**Alternatives considered**:

- Fetch incidents for parent or child candidates before traversal completes. Rejected because it could alter behavior and perform calls for instances not actually returned.
- Use the existing keyed get enrichment helper directly on a synthetic `ProcessInstances` value. Acceptable for internal reuse if it preserves traversal metadata; the external output must remain walk-shaped.

## Decision: Human output places incident messages under matching rows

**Rationale**: Issue #157 says to reuse #154 output conventions where possible. Issue #154 established normal process-instance row first, then indented `incident <incident-key>:` lines below it. Applying the same convention keeps diagnosis predictable.

**Alternatives considered**:

- Append incident messages inline. Rejected because long messages would make tree/path output hard to scan.
- Print incidents after the whole traversal. Rejected because users would need to mentally join incidents back to keys.

## Decision: JSON output keeps traversal metadata and enriches items

**Rationale**: Existing walk JSON includes mode, outcome, root key, ordered keys, edges, item list, warning, and missing ancestor metadata. Automation needs those fields unchanged while also receiving per-item incident details.

**Alternatives considered**:

- Return the issue #154 `IncidentEnrichedProcessInstances` payload directly. Rejected because it would drop walk-specific traversal metadata.
- Add incidents to all default `ProcessInstance` JSON. Rejected because default output must remain unchanged when `--with-incidents` is omitted.

## Decision: v8.7 remains explicit unsupported for incident enrichment

**Rationale**: Issue #157 repeats the tenant-safety requirement from issue #154. The existing v8.7 incident lookup path returns unsupported because tenant-safe keyed enrichment cannot be guaranteed from the current boundary.

**Alternatives considered**:

- Use a tenant-unsafe direct incident lookup fallback. Rejected by the issue and repository safety guidance.
- Silently omit incidents on v8.7. Rejected because it would make diagnostics misleading.

## Decision: Incident enrichment failure fails the command

**Rationale**: `--with-incidents` is an explicit request for diagnostic incident data. Rendering a successful traversal with missing or partial incident details would be easy to misread during incident response, especially in automation. Failing the command preserves the repository's operational proof standard and makes lookup problems visible.

**Alternatives considered**:

- Render traversal with a warning for failed incident lookups. Rejected because it can still look successful to scripts or operators scanning output.
- Render empty incident collections for failed lookups. Rejected because it is indistinguishable from genuinely no-incident results.
