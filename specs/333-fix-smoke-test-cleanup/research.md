# Research: Complete Smoke-Test Definition Cleanup

## Deployment scope and main identity

**Decision**: Preserve the full deployment definition collection, deduplicate exact keys in response order, and retain the main fixture as a separately identified projection.

**Rationale**: `smokeTestDeploymentUnits` supplies three fixtures while `smokeTestDeploymentResult` discards dependencies. Response identities are the available authority for scope. Missing keys, conflicting metadata for a duplicate key, or a missing expected fixture identity must make scope completeness false; do not claim complete cleanup. Fixture metadata can validate expected identities but must never invent mutation keys. Missing main identity prevents starting the wrong process.

**Alternatives considered**: Searching by BPMN names risks other versions and tenants; tracking only instance-used definitions misses unused deployed resources; hard-coded cleanup keys are unavailable before deployment.

## Cleanup owner and response attribution

**Decision**: Keep a deterministic per-definition loop in the ops service and reuse `FindUnrelatedProcessInstancesForDefinition` and the existing single-key `DeleteProcessDefinitions` wrapper. Preserve the response and error from each invocation.

**Rationale**: Generic batch deletion returns a joined error and a response can remain `Ok` after acceptance even when the confirmation wait fails. A batch response alone cannot identify each confirmed result reliably. Existing instance cleanup provides the owned family scope. Blocked definitions are skipped independently. Without fail-fast, independent errors do not prevent attempting later safe definitions; with fail-fast or cancelled context, remaining entries stay unattempted.

**Alternatives considered**: Batch deletion loses precise per-key error attribution; modifying the generic API widens scope; a new worker pool adds synchronization for only three resources. Worker flags are maxima and remain forwarded to existing helpers. Sequential definition cleanup is acceptable for this bounded deployment.

## Submission, confirmation, and absence

**Decision**: Derive submission from an accepted resource response, not slice length. Confirmation additionally requires the helper to complete without error. An empty response list is not successful cleanup. Preserve existing not-found semantics; do not silently convert an error into confirmed deletion.

**Rationale**: Attempted calls, accepted mutations, and observed absence are different facts. No-wait opts skip confirmation only, not eligibility or normal delete guards. Existing safety discovery must continue across sparse pages and retain its tenant configuration. Existing force cleanup can re-expand instance scope after eligibility; retain those established semantics and do not claim atomic protection against concurrent new workloads.

**Alternatives considered**: Treating every returned response as submitted or every error-free empty list as confirmed creates misleading success. Broad force cleanup of unrelated instances is forbidden.

## Public compatibility

**Decision**: Add optional collection fields to the existing v1 report and public facade; keep singular fields tied to the main fixture. Add one authoritative aggregate definition-cleanup summary. Rendering and final workflow decisions consume the collection summary whenever available.

**Rationale**: Existing consumers retain field names, types, omission behavior, and meaning. Additional optional fields are a compatible extension. New consumers can account for dependencies. Legacy singleton-only values remain renderable through an explicit fallback, but new executions must never use that fallback to excuse incomplete deployment evidence.

**Alternatives considered**: Repurposing singular fields as aggregates changes semantics; replacing the schema with v2 is unnecessary for an additive extension; leaving output singleton-only would conceal the original bug.

## Outcome and progress

**Decision**: Preserve existing outcome vocabulary. Errors yield `partially_failed` after deployment; intentional or safety skips yield `passed_cleanup_skipped`; all accepted no-wait requests retain `passed` with submitted cleanup and explicit no-wait evidence. Only waited, fully confirmed results say removed. Emit one definition stage total and per-definition completion facts using existing progress mechanisms.

**Rationale**: No-wait already permits success without deletion confirmation. Making that qualification explicit preserves automation while meeting operational-proof requirements. Counts must cover the full deployment and may not restart for each nested call.

**Alternatives considered**: A new top-level pending outcome would break established consumers; treating safety skips as errors changes policy; reporting passed alone without submitted status hides the opt-out.

## Evidence and research scope

Reviewed repository models, converters, smoke-test workflow and views, generic process-definition deletion, fixture selection, constitution, and Ralph guidance. A read-only research agent independently checked cleanup control flow and response attribution. No external technology selection or dependency changes are needed. The published issue remains unavailable; no claim of verification is made.
