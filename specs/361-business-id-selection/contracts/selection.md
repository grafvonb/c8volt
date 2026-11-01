# Contract: Safe Process Selection

**Requirements**: FR-007–FR-016, FR-020–FR-023 | **Model**: [data-model.md](../data-model.md)

## Proposed public/service boundary

The public process facade gains thin counterparts to these narrowly scoped internal operations:

- `ResolveProcessInstanceSelection(ctx, request, options...)`: convert a collection or unique-start search into copied criteria, distinct selected observations/keys, tenant evidence and explicit completion.
- `PlanProcessInstanceSelectedMutation(ctx, request, options...)`: resolve a cancel/delete collection, expand its union and return the complete copied plan.
- `ExecuteProcessInstanceCancellationPlan`: validate the plan and delegate frozen roots to existing bulk cancellation without BID rediscovery.
- `ExecuteProcessInstanceDeletionPlan`: validate the plan and execute its frozen affected members in observed children-before-parent order, reusing exact-key submission/state/force/retry/wait mechanics without recursively discovering additional delete targets.

Names are proposed implementation identifiers. Public signatures use public models and `foptions.FacadeOption`; service signatures use domain models/options. Keep all backend mechanics in internal services. Add interface implementations and affected stubs together. No new resolver API is added independently to each command.

Add the no-I/O internal adapter contract `ValidateProcessInstanceSearchFilter`. Invoke it before both fresh selection and frozen-plan consumption, even for empty plans or plans created using another adapter. A BID plan passed to v88/v89 must fail unsupported with zero mutation requests.

## Discovery rules

1. Validate selector shape, capability and value before any bypass or empty return.
2. Perform BID discovery using configured tenant scope. Do not apply explicit-key ignore-tenant options before selection.
3. Apply all supported predicates, continue across pages and deduplicate by technical key before counting the documented limit.
4. Track authoritative exhaustion, deliberate collection bound, visitor stop and incomplete/failing discovery distinctly.
5. Freeze executable scope only after the intended bounded collection and required expansion are complete. Return partial observations with original failure evidence, never a mutation-ready plan.

Use trustworthy continuation metadata, raw progress and supported terminal probes; fail if completeness cannot be established safely. Missing ancestry, failure on later child pages or traversal cancellation prevents BID mutation. A seed limit must not clip required relatives.

## Cardinality and per-command scope

| Command | Cardinality | Execution scope |
| --- | --- | --- |
| `get pi` | Query collection | Existing search/page/enrichment behavior |
| `walk pi` | One distinct match proven by exhaustion | Existing full-family, parent or children walk choices |
| `cancel pi` | Collection | Existing ancestry-to-root family cancellation rules; see [cancellation.md](cancellation.md) |
| `delete pi` | Collection | Existing family expansion, state preconditions and force behavior |
| `update pi` | Collection | Existing variable-update scope for each selected instance; no family widening |
| `resolve pi` | Collection | Existing process resolution expansion and incident workflow; reviewed mutation targets remain frozen |
| `suspend pi` / `resume pi` | Collection | Exact instances by default; `--tree` expands descendants of selected seeds only |

Walk fails ambiguity as soon as two distinct keys are established, even within one family. Duplicate observations of one key are not ambiguity. A single observed result plus a limit, stop or failure is not uniqueness. Exhausted zero results produce successful empty output and no walk invocation. Existing walk enrichment must work with the resolved seed without requiring the user to supply its key separately.

## Frozen-plan invariants

Criteria, selected observations, roots, affected members and tenant evidence are copied. The plan retains selection origin so explicit-key and search authorization paths remain deliberate. Consuming services validate operation, matching criteria/context and actual completion evidence; a caller-provided readiness flag alone is insufficient. A plan is not authorization to bypass backend checks.

Confirmation refers to the same frozen submitted keys and observed expanded impact used by execution. Revalidation may report changed state or authorization failure, but must not silently discover new BID matches, replace roots or rebuild an enlarged client-selected target set. Preserve original plan versus later observed outcome evidence.

The existing recursive `DeleteProcessInstance` workflow cannot consume a frozen plan unchanged: it discovers children again and submits their deletion. The plan-aware executor must consume observed membership/order instead. A child appearing after approval receives no unplanned client delete request; a resulting backend conflict is an explicit failure. Retention and orphan purge reuse this executor. Cancellation remains a native cascading root action with the timing boundary in [cancellation.md](cancellation.md).

For resolution, any incident/job/variable actions shown for approval must be captured in the owner service's plan before approval. Do not simulate plan execution by running discovery again with a dry-run flag switched off. No direct incident-native BID selector is introduced.

## Empty, abort and failure semantics

Completed empty discovery performs no confirmation or mutation in normal/dry-run, auto-confirm, automation or no-wait combinations supported by that command. No-wait with zero submissions is successful, not accepted/pending. Key-only empty output contains zero bytes; JSON uses exactly one existing command-appropriate empty envelope. Rendering never adds discovery requests.

User decline/EOF retains the existing command's abort behavior. Failed, interrupted or malformed discovery retains its original failure classification and partial evidence. Neither path is a successful no-op. Failures after submission preserve existing partial results, waiting, fail-fast and exit semantics.

## Required isolated evidence

Exercise every listed command's actual dispatch; verify tenant-scoped discovery followed by correct key execution options, selector conflicts, empty-result modes, sparse/cyclic/failed pages, duplicate-limit accounting, and unchanged explicit-key previews. Use real terminal stdin to prove prompt-free empty selection and stable nonempty prompt routing. Include full family traversal beyond one backend child page and exact versus descendant-only suspend/resume tests.
