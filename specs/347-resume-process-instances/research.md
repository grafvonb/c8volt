# Research: Process-Instance Resumption

Date: 2026-10-27. Scope: issue #347. Research is based on repository source, two delegated read-only reviews (services/facades and CLI/validation), and the official documentation below. No runtime or live-environment validation was performed.

## 1. Supported operation and adapter

**Decision:** Use the existing generated Camunda 8.10 single-instance resumption operation with an empty typed body. Add the method to the local generated-client interface; do not regenerate clients. Reject configured 8.8/8.9 through capability validation before preview, discovery, or mutation.

**Rationale:** `internal/clients/camunda/v810/camunda/client.gen.go` already defines `ResumeProcessInstanceJSONRequestBody`, its optional operation reference, and `ResumeProcessInstanceWithResponse`. The [8.10 endpoint documentation](https://docs.camunda.io/docs/apis-tools/orchestration-cluster-api-rest/specifications/resume-process-instance/) identifies the single-instance operation, suspended-only eligibility, independent relatives, 204 success, and `SUSPEND_PROCESS_INSTANCE` on `PROCESS_DEFINITION`.

**Alternatives considered:** Client regeneration adds no needed capability. Native batch endpoints, direct HTTP outside generated adapters, and version probing expand scope or bypass repository conventions.

## 2. One collection workflow

**Decision:** Implement one internal `ResumeProcessInstances` collection executor used by CLI single/multiple-key and tree modes. Public singular resumption delegates a singleton to this collection and mechanically extracts its result. The versioned single-target method remains the primitive called by collection workers.

**Rationale:** `internal/services/processinstance/suspend.go` already preserves order, prevalidates keys/timing, initializes unattempted slots, uses `toolx.DetermineNoOfWorkers` and `toolx/pool.ExecuteSlice`, and retains partial evidence. `c8volt/process/suspend.go` establishes thin conversion and `ferrors.FromDomain` boundaries. The singleton wrapper must not introduce independent execution mechanics.

**Alternatives considered:** A separate single-key orchestration path would duplicate controls; public-facade worker pools or CLI polling violate layering. A generic mutation framework is unnecessary.

## 3. Submission evidence and retry policy

**Decision:** Follow `v810/suspension_submission.go` in a focused `resumption_submission.go`: reuse `RetryCamundaMutation`, request-rate attempt tracking, context, configured backoff, and strict documented response validation. Accept only a valid empty 204 response. Track cumulative uncertainty and last valid submission status outside retry return values.

**Rationale:** A local request-rate admission failure is not a dispatched request. Transport failure or malformed success can leave acceptance unknown; a later rejection does not erase earlier ambiguity. A valid later acceptance can establish acceptance. Read status must never replace submission status. No fresh mutation may originate in confirmation polling.

**Alternatives considered:** Accepting any nil Go error, generic 2xx response, or later observed state would falsely establish acceptance. New unbounded retries or shared HTTP-policy changes are out of scope. Keep protocol interpretation operation-specific while reusing existing retry/rate primitives.

## 4. Resumption confirmation

**Decision:** Add `waiter/resumption.go`, patterned on `waiter/suspension.go`, with a read-only interface. After acceptance, matching `ACTIVE` or `COMPLETED` confirms; `CANCELED`/`TERMINATED` returns conflict and stops. Persistent suspension, absence, and unknown states remain bounded observations. Authorization/precondition and other unsuccessful reads preserve prior evidence and return according to existing read policy.

**Rationale:** Immediate completion must succeed without an intermediate active observation. Cancellation is not normal completion. The existing waiter protects identity, context/deadline, attempts, nested-read logging, and last observation. The [suspension semantics documentation](https://docs.camunda.io/docs/components/concepts/process-instance-suspension/) explains buffered work on resumption and possible failures that leave an instance suspended; acceptance alone therefore cannot establish the requested observed outcome.

**Alternatives considered:** Reusing the suspension success predicate or accepting every terminal state gives wrong outcomes. Missing suspension dates alone and observations after deadline are not proof. No-wait bypasses all confirmation reads.

## 5. Frozen tree planning

**Decision:** Add `PlanProcessInstanceResumption` in `internal/services/processinstance/resume_plan.go`, returning a complete immutable-by-convention snapshot with seed keys, observations/classifications, and ordered eligible keys. Use existing `DescendantsResult`, traverse before state filtering, and select only `SUSPENDED`. Reuse structural traversal validation by narrowly extracting the suspension validator if needed; eligibility stays operation-specific.

**Rationale:** `suspend_plan.go`, `walker/walker.go`, and `traversal/result.go` already support selected-plus-descendants, DFS first-seen ordering, overlap deduplication, identity validation, and explicit partial/unresolved outcomes. `v810/service.go` retains 1,000 direct children per lookup. Continue through active/terminal parents. Keep `WithIgnoreTenant` for explicit-key discovery, as current suspension does: backend authorization remains authoritative. Search/read permission is still required.

**Alternatives considered:** Calling the walk CLI, escalating roots, applying a suspended-only search before traversal, or rediscovering after confirmation changes scope. Repeated discovery, pagination changes, atomic execution, and reconstructing prior suspension ownership are excluded.

## 6. Shared validation and evidence models

**Decision:** Narrowly rename/extract operation-independent key normalization from `common/processinstance_suspension.go` for both suspension and resumption, preserving behavior and tests. Introduce resumption-specific domain/public result and plan types with the existing suspension field vocabulary. Do not change suspension's public types or classifications.

**Rationale:** Exactly 16 decimal digits, trimming, and stable identity deduplication are already implemented. Results need independent submission/confirmation evidence; a single success boolean is inadequate. Cancellation fits `status=failed`, `submission=accepted`, `confirmation=unconfirmed`, the observed cancellation state, and conflict error. Tree classifications need only `eligible` and `excluded`; there is no already-resumed success.

**Alternatives considered:** Duplicating validation risks drift; reusing suspension-named public models misstates semantics. A new generic workflow/state hierarchy is disproportionate.

## 7. CLI and output boundaries

**Decision:** Mirror the focused suspension file split: command construction, collection runner, tree runner, and dedicated resumption views. Reuse explicit-admin input options, confirmation, activity, envelope, bounded-error, and exit helpers. Model failures with raw machine evidence but allowlist human reasons; cancellation gets a distinct compact reason.

**Rationale:** `cmd/suspend_processinstance{,_batch,_tree}.go` and `cmd/cmd_views_processinstance_suspend{,_tree}.go` already separate prompts, information, machine results, and post-render exits. `renderHumanLine` honors configured logging. JSON emits one envelope, errors take precedence over accepted no-wait, and no-op success must derive from a successful plan. `collectExplicitPIAdminInputOptions` preserves established tenant and option semantics.

**Alternatives considered:** Direct stderr error prints, raw `err.Error()` human logs, or invoking a second error renderer after a partial JSON envelope violate repository contracts. Sanitizing machine evidence to hide a human-rendering bug would break the existing schema intent.

## 8. Fresh observations

**Decision:** Preserve existing instance conversion and map each successful observation afresh, replacing rather than merging `SuspendedDate`.

**Rationale:** `v810/convert.go` and `convert_test.go` already support date omission. A successful subsequent response without the date clears stale evidence; failed reads retain the last successful observation. Add workflow regression coverage without redesigning inspection commands.

**Alternatives considered:** Caching the old date, inventing a resumption timestamp, or clearing timestamps before a successful read loses or fabricates evidence.

## 9. Validation and fixture boundary

**Decision:** Use colocated isolated service, facade, command/subprocess, and real-terminal tests. Adapt existing suspension suites, including full logging/activity/output matrices. Update relevant integration fixture source where CLI resumption coverage is intended, but never execute it during implementation. Use documentation checks only for this planning change.

**Rationale:** `AGENTS.md` and constitution v2.0.0 require isolated dependencies and proportional checks. `integration/cli` uses the integration build tag; Makefile integration targets invoke environment-backed workflows. Those targets are explicitly excluded. Runtime check commands must clear inherited Go flags and live-test/profile variables and supply test-owned configurations.

**Alternatives considered:** Live target discovery, local Camunda, smoke targets, or fixture execution are not authorized. Renderer-only tests cannot prove configured logger routing, prompt routing, partial exits, or duplicate suppression.

## 10. Cohesive implementation sequencing

**Decision:** Keep interfaces, all affected implementations/test doubles, domain/public models, and their mapping/guard validation together. Add complete collection and waiter behavior before command wiring; add tree planning as a separate independently verifiable outcome; attach tests/formatting/docs to their owning work units. Detailed task generation is the next phase.

**Rationale:** Interface expansion can break unrelated fake implementations in resource, process-definition, ops, and process tests. Collection support is required from the outset, so a single-only implementation followed by a bulk rewrite is inappropriate.

**Alternatives considered:** Standalone context-reading tasks, hypothetical parallel tasks, or separate interface-only changes would not leave cohesive compiling increments.

## Resolution

All planning unknowns are resolved. No new dependency, schema migration, client regeneration, or architecture exception is required. The spec's behavioral scope and the issue's implementation constraints remain aligned with repository rules.
