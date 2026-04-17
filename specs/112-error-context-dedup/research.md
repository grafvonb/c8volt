# Research: Preserve Concise CLI Error Breadcrumbs

## Decision 1: Keep `ferrors` as the shared class and exit boundary, and fix duplication in upstream wrappers

- **Decision**: Preserve `c8volt/ferrors/errors.go` as the only shared classification and exit-code boundary, and remove duplicated prose in the command, helper, and service wrappers that currently feed it.
- **Rationale**: `ferrors.Normalize` intentionally preserves wrapped text and adds the class prefix once through `wrap(...)`. Changing that behavior to strip duplicated prose would make normalization parse or rewrite arbitrary error strings. The lower-risk, repository-native fix is to tighten the wrappers that currently build repeated messages before `HandleAndExit` logs them.
- **Alternatives considered**:
  - Deduplicate inside `ferrors.Normalize`: rejected because it would make the shared error model depend on string surgery and could alter unrelated error paths.
  - Introduce a new rendering layer parallel to `ferrors`: rejected because it would duplicate the repository’s existing CLI error boundary.

## Decision 2: Treat duplication as a pattern-family problem, not just a single process-instance bug

- **Decision**: Organize the audit and tasking around duplication-pattern families that feed user-facing CLI output, starting with the reported process-instance walk path and expanding to every matching repository seam found during investigation.
- **Rationale**: The reported example comes from a specific walk/ancestry/get chain, but the codebase shows the same nested-wrapper style in multiple places: walker breadcrumbs (`get %s`, `ancestry fetch`), versioned process-instance services (`fetching process instance with key %s`), command wrappers (`error fetching ...`), and a smaller set of single-resource and cluster fetch commands.
- **Alternatives considered**:
  - Fix only the original `walk pi` path: rejected by the spec clarification that widened scope to every duplicated CLI error path found during investigation.
  - Sweep every `fmt.Errorf(": %w")` call indiscriminately: rejected because many wrappers add useful context and do not create the duplication pattern this feature targets.

## Decision 3: Preserve the shared class prefix and deduplicate only the wrapped details after it

- **Decision**: Keep existing shared class prefixes such as `resource not found:` and `unsupported capability:` exactly as they are, and remove duplication only from the wrapped detail text that follows those prefixes.
- **Rationale**: The clarified spec explicitly chose this behavior. It preserves the established `ferrors` contract for scripts and users while cleaning up the noisy tail of the message.
- **Alternatives considered**:
  - Remove the shared class prefix when the root error is already descriptive: rejected because it would blur the stable CLI contract and change externally observable semantics.
  - Let each command choose its own final top-level phrasing: rejected because it would fragment the shared error model again.

## Decision 4: Use stage-only breadcrumb wrappers once a lower layer already owns the root failure detail

- **Decision**: When a lower layer already returns the root failure detail, upper layers should add only stage context such as ancestry, descendants, wait, cancel, or render stage labels instead of restating the same identifier or failure sentence.
- **Rationale**: The current noisy example is caused by multiple layers all restating both the key and the same not-found meaning. The clean target shape keeps the call-path breadcrumbs but lets the deepest relevant layer own the specific resource detail.
- **Alternatives considered**:
  - Preserve every current wrapper verbatim and only trim the deepest error: rejected because most duplication originates in the upper layers that keep rephrasing the same failure.
  - Remove breadcrumbs entirely: rejected because the issue and clarified spec both say the breadcrumbs are useful context.

## Decision 5: Allow equivalent breadcrumb shortening, but not semantic drift

- **Decision**: Breadcrumb labels may be shortened only when the failing stage remains clearly identifiable and the wording stays semantically equivalent.
- **Rationale**: Some current labels are themselves partly repetitive, especially when they echo the same key or restate the same operation twice. Allowing small, equivalent wording reductions keeps the final message concise without breaking the user’s mental model of the call path.
- **Alternatives considered**:
  - Freeze every breadcrumb label verbatim: rejected because it would leave avoidable noise in the exact place this feature is meant to improve.
  - Freely rewrite breadcrumbs for readability: rejected because it would create broader user-visible churn and weaker regression expectations.

## Decision 6: Apply the same prefix-preserving dedup rule to other shared error classes when the pattern matches

- **Decision**: If the same duplication pattern appears on other shared error classes, preserve their existing normalized class prefixes and deduplicate the wrapped details the same way.
- **Rationale**: The clarified spec explicitly chose a cross-class rule. This keeps the repository cleanup coherent and avoids special-casing not-found-only behavior when the same wrapper pattern also affects unsupported or other normalized failures.
- **Alternatives considered**:
  - Limit the cleanup to not-found failures only: rejected because the clarified spec broadened the contract to matching non-not-found cases.
  - Allow class-specific custom phrasing rules per command: rejected because it would reintroduce inconsistency.

## Decision 7: Use representative regression anchors per duplication-pattern family

- **Decision**: Add or update representative tests for each affected duplication-pattern family rather than requiring one regression test per every discovered command path.
- **Rationale**: The clarified spec selected representative coverage per pattern family. The repository already has strong seams for this: subprocess and in-process tests under `cmd/`, helper tests under `internal/services/processinstance/walker`, and shared classification tests under `c8volt/ferrors`.
- **Alternatives considered**:
  - Add a test for every discovered path: rejected as too broad for one feature and unnecessary once pattern families are covered.
  - Test only the originally reported process-instance walk case: rejected because it would not prove the repo-wide audit outcome.

## Decision 8: Keep documentation impact explicit but conditional

- **Decision**: Treat documentation as likely unchanged unless the audit finds README or docs examples that depend on the old duplicated phrasing.
- **Rationale**: This feature changes user-visible output, so the constitution requires an explicit documentation decision. However, the current change is a semantic no-op for failure class and command usage, and there is no evidence yet that the docs depend on the noisy message text.
- **Alternatives considered**:
  - Unconditionally update README and generated docs: rejected because the repository’s user docs do not appear to document the exact duplicated strings today.
  - Ignore documentation review entirely: rejected because the constitution requires an explicit decision.

## Audit Inventory: Confirmed Duplication-Pattern Families

### Implementation Boundary Refresh

The setup audit confirmed that the feature should only touch wrapper seams that contribute to final CLI rendering. The following owner layers now define the implementation boundary for later tasks:

| Pattern family | Owner layer | Confirmed wrapper seams |
|--------|-------------|-------------------------|
| Process-instance lookup and traversal | `internal/services/processinstance/walker` and versioned process-instance services | `walker.go` wrappers `get %s`, `list children of %s`, `ancestry fetch`; `v88`/`v89` `GetProcessInstance` wrappers `fetching process instance with key %s`; the equivalent ancestry/family wrappers in `v87` |
| Process-instance mutation and wait follow-up | Versioned process-instance services and waiter-backed follow-up paths | `CancelProcessInstance`, `DeleteProcessInstance`, `GetProcessInstanceStateByKey`, and `waiting for ... failed` wrappers across `v87`, `v88`, and `v89` |
| Single-resource command fetch wrappers | `cmd/` command handlers | `error fetching resource by id %s`, `error fetching topology`, `error fetching cluster license`, `error fetching process definition...`, and process-instance list/fetch command wrappers |
| Resource/client orchestration wrappers | `c8volt/resource` helpers that bubble into CLI output | `waiting for process definition %s removal failed` in `c8volt/resource/client.go` and adjacent orchestration wrappers that forward lower-layer failures |

This audit did not justify changes inside `c8volt/ferrors`, transport clients, or success-path logging. Those areas remain fixed unless later implementation finds a user-facing duplication seam that still routes through the same CLI rendering contract.

### Pattern family 1: Process-instance lookup and traversal

- [`internal/services/processinstance/walker/walker.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker.go) adds stage breadcrumbs such as `get %s`, `list children of %s`, and `ancestry fetch`.
- [`internal/services/processinstance/v88/service.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/v88/service.go) wraps lookup failures repeatedly with `fetching process instance with key %s`.
- The reported issue example is created when those layers compose on top of a normalized `resource not found` root failure.

### Pattern family 2: Process-instance mutation and wait flows

- The same versioned service files wrap cancel, delete, family, ancestry, and waiter-backed follow-up failures with increasingly specific prose such as `fetching family for process instance with key %s` and `waiting for canceled state failed for %s`.
- These flows are likely to repeat the same key or root failure detail after lookup failures are deduplicated unless the wrappers are aligned to the new stage-only rule.

### Pattern family 3: Single-resource command fetch wrappers

- Commands such as [`cmd/get_processdefinition.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_processdefinition.go), [`cmd/get_resource.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_resource.go), [`cmd/get_cluster_license.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_license.go), and [`cmd/get_cluster_topology.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_cluster_topology.go) all add top-level `error fetching ...` wrappers.
- Some of these may be acceptable one-layer context, while others may duplicate lower-layer fetch wording; the audit needs to separate stage-only wrappers from repeated-detail wrappers.

### Final audit result: Cluster license/topology share the remaining non-not-found duplication seam

- The final audit confirmed one matched non-not-found family after User Stories 1 and 2: cluster license and topology failures still repeated the same fetch-stage meaning across `internal/services/cluster/common` and `cmd/get_*`.
- The shared cluster helper now returns transport and HTTP-status failures without adding `fetch cluster license` or `fetch cluster topology`, leaving the command layer to own the single user-facing stage breadcrumb.
- Representative command regressions in `cmd/get_test.go` now assert the final rendered output keeps the shared `service unavailable:` or `malformed response:` prefix while excluding the old inner fetch-stage duplication.
- Shared-prefix coverage in `c8volt/ferrors/errors_test.go` now explicitly locks the unavailable-class rendering behavior alongside the existing unsupported and not-found coverage.

### Pattern family 4: Resource/client orchestration wrappers

- [`c8volt/resource/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/resource/client.go) and resource service files layer additional operation text around process-instance validation, cancellation, and wait flows.
- These paths are user-facing through delete and deploy flows, so they are part of the audit when they repeat an already-complete lower-layer sentence.

## Regression Anchors

- [`cmd/walk_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/walk_test.go) is the primary anchor for the originally reported traversal case.
- [`cmd/cancel_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/cancel_test.go) and [`cmd/delete_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/delete_test.go) are the best anchors for mutation and wait follow-up pattern families.
- [`cmd/get_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/get_test.go) and specific `get_*` command tests are the best anchors for single-resource fetch wrapper cleanup.
- [`internal/services/processinstance/walker/walker_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/walker/walker_test.go) and `v87`/`v88`/`v89` service tests are the tightest helper-level seams for breadcrumb and root-detail composition.
- [`c8volt/ferrors/errors_test.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/ferrors/errors_test.go) should continue proving that classification and exit-code behavior are unchanged while the upstream wrappers become less repetitive.

### Regression Anchor Confirmation

| Family | Confirmed anchors | Current evidence from the audit |
|--------|-------------------|---------------------------------|
| Lookup and traversal | `cmd/walk_test.go`, `internal/services/processinstance/walker/walker_test.go` | `walk` already exercises traversal composition, and walker tests remain the narrow seam for breadcrumb ordering and root-detail ownership |
| Mutation and wait follow-up | `cmd/cancel_test.go`, `cmd/delete_test.go`, `internal/services/processinstance/v87/service_test.go`, `internal/services/processinstance/v88/service_test.go` | command tests already cover cancel/delete orchestration while service tests assert current duplicated wrapper text such as `fetching process instance with key 123` |
| Single-resource fetch wrappers | `cmd/get_test.go` and targeted `get_*` command tests | `get` tests already assert top-level wrapper text for cluster, resource, and process-definition fetch failures |
| Shared class/exit behavior | `c8volt/ferrors/errors_test.go`, `cmd/bootstrap_errors_test.go` | these tests protect normalization and bootstrap mapping, which must stay fixed while wrapper text is deduplicated |
