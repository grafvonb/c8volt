# Data Model: Element Scope Variables

This design uses existing in-memory domain/facade conventions. No persistence or migration is introduced. Names below are proposed new types; existing process-variable page/item types remain reusable internally.

## Selected element and scoped query

| Field | Meaning and invariant |
|-------|-----------------------|
| `ElementInstanceKey` / plan `ElementKey` | Concrete runtime element key; nonblank, never a BPMN ID or user-task key |
| `ProcessInstanceKey` | Actual owning process from element lookup/discovery; required and independent of element key |
| `ScopeKey` | Exactly the selected element key |
| `TenantId` | Observed tenant evidence; existing absent/empty evidence remains unknown |
| Selection index | Original first-occurrence key order, preserved in plan and outcomes |

`ScopeVariableQuery` contains `ProcessInstanceKey`, `ScopeKey`, and optional `ExpectedTenantId`. Nil means no known expected tenant; only present evidence can be compared. Do not derive expected identity from a configured discovery tenant for explicit keys. Query validation rejects missing owner/scope before variable requests. A selected call-activity element never implies its called process root.

## Local variable observation

Reuse `domain.ProcessInstanceVariable` internally and introduce an element-owned facade `Variable` projection with these JSON fields:

| JSON field | Representation |
|------------|----------------|
| `name` | Variable name |
| `value` | Raw received serialized value string, preserving existing inspection conventions |
| `variableKey` | Optional backend variable key |
| `processInstanceKey` | Required owning process |
| `scopeKey` | Required selected local scope |
| `tenantId` | Optional observed tenant |
| `apiTruncated` | Boolean, always present; backend completeness evidence |

`Element.Variables` is `*[]Variable` with `json:"variables,omitempty"`; the domain field follows the corresponding internal item type. Nil omits the field. A successful empty read points to an allocated zero-length slice and serializes as `[]`, never `null`. This is independent of the existing optional `listeners` field. Mapping must preserve both.

A complete collection has passed the shared traversal's cursor, total, offset, and cancellation checks. Raw item count precedes defensive filtering. Unrelated scopes may be discarded, but a row claiming the selected scope with inconsistent owner/known tenant is a discovery failure. Backend-truncated values may be displayed with evidence; they cannot establish unchanged or confirmed status.

## Requested object

`map[string]any` represents the validated shared variable-update payload. Preserve the existing parser's number/value semantics. The top-level object must be non-null; exactly one payload source is required. `{}` is valid. A property with `null` means assigning JSON null, not deleting a variable. Clone nested maps and slices at facade and execution boundaries.

## Prepared local plan

`ElementVariablePlan` contains `ElementKey`, `ProcessInstanceKey`, `ScopeKey`, known `TenantId`, and the following collections. Classification order follows nearby variable-plan conventions and remains deterministic.

| Collection | Contents |
|------------|----------|
| `Additions` | Requested names absent locally; each has name/value/completeness metadata |
| `Changes` | Locally present requested names without complete equality; each has name/before/after/truncation evidence |
| `UnchangedRequested` | Requested names proven equal by complete local observations |
| `Untouched` | Locally observed names absent from the request |
| `MutationVariables` | Exactly additions and changed after-values |
| `VerificationValues` | Exactly all requested values, including unchanged requested names |

A same-name parent value is irrelevant to classification. Truncated local observations are changes when requested, even if the visible text appears equal. Do not fabricate a complete before-value from truncated data.

`ElementVariableUpdatePlan` contains `RequestedKeys`, `RequestedValues`, `Elements`, `RequestedCount`, `UpdateCount`, the four variable classification counts, `MutationSubmitted=false`, and existing `TenantContext`. One entry exists for each unique requested key, in order. An empty requested object still produces identity/local-read plans for the selected elements. No partial successful plan is returned after a planning error.

Execution validation requires unique ordered keys; exact key/scope agreement; nonblank owner; consistent counts and tenant evidence; disjoint, complete classifications; mutation-map equality with additions/changes; verification-map equality with the original request; and `MutationSubmitted=false`. Deep-copy before any worker starts. Invalid/tampered plans yield zero writes. This protects the API contract, not against concurrent changes in the external engine.

## Submission evidence

Extend the internal scope-writer response with a `SubmissionStatus` enum while retaining its existing acceptance/status fields. The new element result exposes `submissionStatus`. Existing PI/UT facade schemas remain unchanged.

| Evidence | Required interpretation |
|----------|-------------------------|
| `not_sent` | No physical attempt was dispatched; also used for unchanged/skipped items |
| `rejected` | Explicit backend rejection and no earlier ambiguous dispatch |
| `unknown` | A dispatched attempt has no conclusive response and no later accepted response |
| `accepted` | Documented successful mutation response observed |

Evidence accumulation is per logical destination, retained outside the retry callback. An unsent retry preserves earlier rejected/unknown evidence. Unknown remains sticky through subsequent rejection; accepted supersedes uncertainty when success is observed. Confirmation never changes accepted evidence. Use request-rate attempt observations to identify positively known local rejection, including client-timeout substitution; do not infer unsent from a generic context error alone. Preserve original error classification independently.

For the current endpoint, recognize a well-formed documented `400 InvalidData` response without transport/decoding error as conclusive rejection. The generated endpoint response contracts in v88/v89/v810 also expose 500/503/504; these server/gateway failures do not prove rejection. Missing/malformed responses, generic proxy failures, unexpected statuses, and response-plus-transport-error cases remain unknown unless stronger evidence establishes acceptance. Do not classify every non-2xx as rejected. Adding other conclusive rejection cases requires documented endpoint evidence and focused tests; conservative unknown remains valid. Preserve existing PI/UT public behavior while adding this evidence for element outcomes.

## Ordered execution outcomes

`ElementVariableUpdateResults.Items` contains one `ElementVariableUpdateResult` per unique planned element, in selection order. Each result has:

| Field / JSON name | Meaning |
|-------------------|---------|
| `Key` / `key` | Selected element key |
| `ProcessInstanceKey` / `processInstanceKey` | Frozen owning process |
| `ScopeKey` / `scopeKey` | Frozen local destination |
| `Status` / `status` | Workflow status below |
| `SubmissionStatus` / `submissionStatus` | Physical submission evidence above |
| `MutationAccepted` / `mutationAccepted` | True exactly when submission evidence is accepted |
| `ConfirmationStatus` / `confirmationStatus` | `not_required`, `skipped`, `not_started`, `confirmed`, or `failed` |
| `StatusCode` / `statusCode` | Optional observed mutation HTTP status; never invented from a transport error |
| `Message`, `Error` / `message`, `error` | Optional existing-style machine evidence, excluded from human diagnostics unless reduced by an allowlist |
| `Variables` / `variables` | Optional frozen mutation values, using nearby result conventions |

| Workflow status | Submission | Confirmation | Interpretation |
|-----------------|------------|--------------|----------------|
| `unchanged` | not_sent | not_required | Completed planning proves no write needed |
| `skipped` | not_sent | not_required | Required work was not started because of failure/cancellation |
| `mutation_failed` | not_sent, rejected, or unknown | not_required | Started work failed before acceptance; retain exact evidence and error class |
| `submitted` | accepted | skipped | No-wait deliberately skipped read-back |
| `submitted` | accepted | not_started | Cancellation/fail-fast prevented confirmation from starting; operation remains failed |
| `confirmed` | accepted | confirmed | Complete exact-local read proves every requested value |
| `confirmation_failed` | accepted | failed | Read-back failed/timed out/canceled; acceptance remains known |

Normal transition: planned changed → mutation attempt → accepted → confirmation → confirmed. No-wait ends at submitted with confirmation skipped. Accepted work whose confirmation never starts retains submitted/accepted with confirmation not_started and the interruption error; it is not a successful ordinary update or skipped/unsubmitted work. Outcome success predicates must distinguish this case from deliberate no-wait. Failed submission ends at mutation_failed. Fail-fast stops new scheduling but preserves in-flight outcomes. No state means rolled back. Worker-attempt markers distinguish an unscheduled item from a locally paced failure after work began.

Preview and no-op use plan payloads and an explicit succeeded envelope; they do not manufacture execution results or accepted statuses. Empty element-search selection uses the existing get payload. Keys-only execution tests `MutationAccepted`, so confirmation failures remain visible and unknown acceptance does not become a falsely accepted key.
