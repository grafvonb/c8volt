# Data Model: Complete Smoke-Test Definition Cleanup

All additions exist in `internal/domain/ops_smoke_test_model.go` with equivalent public types in `c8volt/ops/model.go` and mechanical conversion in `convert.go`. No persistence migration is required.

## Deployed definition

`SmokeTestDeployedDefinition` contains `ProcessDefinitionKey`, `BpmnProcessID`, `ProcessDefinitionVersion`, `TenantID`, and `ResourceName`. Public JSON uses `processDefinitionKey`, `bpmnProcessId`, `processDefinitionVersion`, `tenantId`, and `resourceName` with existing optional-field conventions.

Add `ProcessDefinitions []SmokeTestDeployedDefinition` (`processDefinitions,omitempty`) and `DefinitionScopeComplete bool` (`definitionScopeComplete,omitempty`) to `SmokeTestDeploymentResult`. Existing singular deployment fields continue describing the main runnable fixture. Existing deployment errors carry missing/conflicting-evidence explanations.

Deduplicate nonempty keys in first-seen order. Conflicting nonempty identity/tenant metadata for a duplicate is an evidence error. Missing tenant information stays unknown rather than being fabricated; retain existing deployment tenant fallback rules. Missing key entries retain available identity evidence for reporting but are never mutation targets. Validate that the expected fixture identities are represented; fixture inventory supplies validation, never replacement keys.

## Per-definition cleanup

`SmokeTestDefinitionCleanupItem` contains:

- `Definition`: deployed identity evidence.
- `Eligibility`: existing `SmokeTestCleanupEligibility` with blockers/errors.
- `Cleanup`: existing `SmokeTestProcessDefinitionCleanupResult` with responses, submitted/confirmed flags, no-wait, and errors.
- `Disposition`: `confirmed`, `submitted`, `retained`, `failed`, or `unattempted`.
- `Reason`: compact explanation for retention, missing identity, or unattempted work.

Add `DefinitionResults []SmokeTestDefinitionCleanupItem` (`definitionResults,omitempty`) to `SmokeTestCleanupResult`. Results follow the deployment collection order. Initialize entries before cleanup so early returns preserve resource accounting.

State transitions: unattempted -> retained for no-cleanup or safety blocker; unattempted -> failed for eligibility/identity error; eligible attempt -> submitted after acceptance without wait; eligible attempt -> confirmed only after successful confirmation; eligible attempt -> failed on submission or confirmation error. A failed confirmation can retain `Submitted=true` while `Confirmed=false`. Never overwrite completed entries on later errors.

## Aggregate summary

Add optional `DefinitionSummary *SmokeTestDefinitionCleanupSummary` (`definitionSummary,omitempty`) containing `Status`, `Total`, `Confirmed`, `Submitted`, `Retained`, `Failed`, and `Unattempted` counts. Counts are mutually exclusive disposition counts; `Submitted` counts unconfirmed submitted entries, not every historical submission. Total equals the result-entry count. Identity errors must be represented and cannot disappear from the total. If an expected fixture is absent from the response entirely, record the scope error without inventing a deployed resource entry; scope incompleteness independently prevents confirmed aggregate status. A zero total never overrides a scope error.

Status precedence: failed for errors or unexpected unattempted work; skipped for explicit no-cleanup or safety retention without errors; submitted when otherwise successful but confirmation is skipped; confirmed only when the entire valid nonempty scope is confirmed. Mixed skipped/submitted collections retain their separate counts. Dry-run has no actual deployment/result entries and no fabricated submitted resources.

Existing singular eligibility, cleanup, and retained-definition fields remain main-fixture projections. Aggregate `Errors` includes dependency errors. The new summary, not the legacy projection, drives the overall cleanup stage and final outcome. Process-instance cleanup retains its existing model and behavior.

## Relationships and invariants

One deployment has many definition entries; each entry has exactly one cleanup result once deployment evidence is available. Report and result share the same collection facts through existing construction/mapping. No renderer performs discovery. A nil summary permits legacy render fallback only; freshly executed deployments populate summary and scope completeness explicitly. No-cleanup preserves incomplete evidence and must not imply a complete inventory when evidence is missing.
