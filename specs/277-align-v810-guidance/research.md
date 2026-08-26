# Research: Align Camunda 8.10 Guidance

## Decision 1: Use the Delivered #273/#275 Contract as Baseline

**Decision**: Use issue #273 as the authority for the V810 identity, aliases, baseline lifecycle, native service behavior, and live-integration boundary. Apply issue #275 as the authoritative replacement for #273's original embedded-fixture mapping, and apply #277 as the intentional replacement of #273's former V88 default with V89. Validate these decisions against current version, baseline, gateway-diagnostic, and fixture-selection behavior.

**Rationale**: The detailed #273 contracts and current behavior agree, while #275 explicitly finalizes #273 by replacing the temporary C89 selection with native C810 definitions. This yields one coherent current model without reopening unrelated decisions.

**Alternatives considered**: Treating #273 and #275 as independent feature lines was rejected because it would allow mutually exclusive fixture mappings. Treating every older feature artifact as current policy was rejected because completed feature history describes the scope that existed when it was delivered.

## Decision 2: Classify Guidance Before Editing It

**Decision**: Divide relevant artifacts into active normative guidance, generated derivatives, integration-only guidance, and historical implementation records. Correct active guidance, regenerate derived documentation from its source, preserve the intentionally stable-only integration scope, and retain historical completed tasks, progress, and Ralph memory with their existing supersession context.

**Rationale**: A text search alone cannot distinguish an active C89 fallback from an accurate record that C89 was once implemented, nor can it distinguish a stale product-support list from an intentional live-integration matrix that ends at 8.9.

**Alternatives considered**: Replacing every 8.7–8.9 or C89 reference globally was rejected because it would falsify history and expand live-integration scope. Ignoring historical records entirely was rejected because reviewers need a clear rule for interpreting them.

## Decision 3: Include Current Architecture Memory in the Maintainer Sweep

**Decision**: Update `AGENTS.md`, `specs/ralph-implementation-rules.md`, `.specify/memory/architecture.md`, and `.specify/memory/architecture-repo-facts.md` wherever they enumerate the supported service/client lines or name the newest supported runtime.

**Rationale**: The current architecture synthesis and repository-fact source still describe support only through 8.9. They are active project guidance and feed future planning, so correcting only AGENTS and Ralph rules would leave a second source of stale architectural truth.

**Alternatives considered**: Limiting edits to the two paths named in the issue problem statement was rejected because the specification requires all normative maintainer inventories to agree. Rewriting historical numbered feature plans was rejected because they accurately describe their original delivery scope.

## Decision 4: Preserve Gateway Diagnostics and Clarify Normative Language

**Decision**: Change #273 FR-006 and SC-002 so they state that a different gateway major/minor is not accepted as a compatibility match and emits the established mismatch diagnostic, while empty or malformed values are unverifiable and emit the established unrecognizable diagnostic. Clarify the authored `config test-connection` help with the same matrix. Do not introduce command failure.

**Rationale**: The #273 research, version-selection contract, current diagnostic behavior, and tests agree: observed 8.10 patch/prerelease values match by major/minor; another line produces a mismatch warning; empty or malformed values produce an unverifiable warning. Only the top-level specification and abbreviated command help are ambiguous or incomplete.

**Alternatives considered**: Changing runtime behavior to hard-fail was rejected because issue #277 is a guidance refinement and the detailed contract already defines diagnostics. Hand-editing the generated CLI page was rejected because authored command metadata is its source.

## Decision 5: Keep C810 Active and C89 Historical or Derivational

**Decision**: Require C810 as the only active V810 embedded family. Preserve C89 only as the stable V89 family, the behavioral derivation source for corresponding C810 definitions, a rejected alternative in design research, or a superseded historical #273 implementation record.

**Rationale**: Issue #275 expressly overwrote only fixture selection and already corrected the active #273 design set. The existing #275 note in #273 tasks makes the checked C89 tasks, progress, and Ralph memory interpretable without rewriting history.

**Alternatives considered**: Deleting or rewriting historical C89 records was rejected because it would falsify delivery history. Suppressing all C89 mentions was rejected because derivation and stable V89 ownership are legitimate current facts.

## Decision 6: Correct the Shipped Configuration Template at Its Source

**Decision**: Extend the supported-version comment in `config/templates/config.example.yaml` to include 8.10 and identify 8.9 as the default, while retaining the example's explicit configured value. Assert the rendered V89 selection close to existing config command tests and verify the source-only supported-version comment with a content scan because YAML rendering strips comments.

**Rationale**: The template is operator-facing and currently lists only 8.7–8.9. It is rendered through existing config commands, so correcting the source and protecting the rendered result covers both file users and CLI users without changing configuration behavior.

**Alternatives considered**: Changing the example value from 8.9 was rejected as unnecessary because an explicit example selection is not a statement of the default. Updating only README was rejected because README is already correct and does not repair the shipped template.

## Decision 7: Keep Generated Documentation Source-Driven

**Decision**: Review README, API guidance, the authored CLI landing-page support matrix, the generated documentation homepage, root CLI reference, version reference, and config test-connection reference against the V810 contract. Change authored sources only for concrete drift, then run `make docs-content` and retain only source-driven generated changes.

**Rationale**: README, API guidance, root CLI help, and version output must state the correct supported versions, aliases, prerelease baseline, and V89 default. Generated documentation must remain derived from README and command metadata.

**Alternatives considered**: Rewriting all operator pages was rejected because most are already aligned. Skipping generated-document review was rejected because issue #277 requires one coherent operator contract.

## Decision 8: Validate Content and Protect Runtime Scope

**Decision**: Combine focused configuration/help tests, explicit repository scans, documentation regeneration review, protected-path diffs, `git diff --check`, and `make test`. Protect runtime service code, generated clients, embedded BPMN definitions, and live integration assets from feature changes.

**Rationale**: Content scans prove the exact version and fixture statements, existing tests prove rendered documentation behavior, and the full test gate guards against accidental regressions. Diff guards prove the refinement did not expand into runtime or integration changes.

**Alternatives considered**: Manual review alone was rejected because stale enumerations are easy to reintroduce. Adding a new validation framework was rejected because repository-native tests and search tools are sufficient.

## Research Resolution

All technical and scope decisions are resolved for Phase 1 design; no open research questions remain.
