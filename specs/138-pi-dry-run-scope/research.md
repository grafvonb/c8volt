# Research: Process-Instance Dry Run Scope Preview

## Decision: Use `DryRunCancelOrDeletePlan` as the scope source of truth

**Rationale**: The facade already returns the exact data required by the issue: resolved roots, collected affected keys, traversal outcome, warning text, and missing ancestors. Cancel and delete preflight already call this plan before confirmation and mutation, so reusing it keeps dry run aligned with real execution.

**Alternatives considered**: Recomputing ancestry and descendants in the command layer was rejected because it would duplicate existing traversal rules and risk diverging from real preflight behavior.

## Decision: Branch after shared preflight planning, before confirmation and mutation

**Rationale**: The command helpers `cancelProcessInstancesWithPlan` and `deleteProcessInstancesWithPlan` already gather impact counts before prompting. Dry run should return immediately after plan rendering so it cannot submit cancel/delete requests or wait for completion.

**Alternatives considered**: Adding dry-run checks before planning was rejected because it would skip the dependency expansion the feature specifically needs to preview.

## Decision: Keep search-mode dry run inside existing page orchestration

**Rationale**: `processPISearchPagesWithAction` already applies search paging, page limiting, continuation, and per-page preflight callbacks for cancel/delete. Reusing that path ensures dry run previews the same selected pages and per-page scope calculation as real execution.

**Alternatives considered**: A separate search-only dry-run loop was rejected because it would recreate continuation and limit behavior already tested for process-instance search.

## Decision: Use a dedicated command-level dry-run payload

**Rationale**: Cancel and delete reports describe submitted mutations. During dry run there are no mutation reports, so structured output needs a payload that represents preview data directly: requested keys, roots, family keys, counts, traversal outcome, warnings, and missing ancestors.

**Alternatives considered**: Returning empty cancel/delete reports with metadata was rejected because it would be ambiguous for automation and could imply a mutation pathway ran.

## Decision: Return aggregate structured search output with nested per-page previews

**Rationale**: Search-mode dry run can process more than one page, and the command must preserve the existing per-page scope calculation while still giving automation an easy top-level summary. One aggregate object with nested page previews exposes total requested/root/affected counts and keeps page-level traversal outcomes, warnings, and missing ancestors inspectable.

**Alternatives considered**: Returning only a flat array of per-page previews was rejected because automation would have to reconstruct totals. Returning only one de-duplicated aggregate preview was rejected because it would hide page-level traversal outcomes and warnings.

## Decision: Regenerate docs from command metadata after implementation

**Rationale**: `--dry-run` changes command help and examples. The repository constitution requires README and generated CLI docs to match shipped behavior, and the existing docs path is `make docs-content`.

**Alternatives considered**: Hand-editing generated CLI docs was rejected because repository guidance prefers changing command metadata and regenerating derived output.
