# Feature Specification: Tenant Context Before Ops Mutations

**Feature Branch**: `295-tenant-context-before-mutations`

**Created**: 2026-09-10

**Status**: Draft

**Input**: [GitHub issue #295](https://github.com/grafvonb/c8volt/issues/295), “fix(ops): show tenant context before mutations with auto-confirm”. Operators need to see selection scope before discovery and affected tenants before destructive or repair work begins, including when confirmation is automatic.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See tenant scope before auto-confirmed work (Priority: P1)

As an operator running an ops command with `-y` or `--auto-confirm`, I want tenant context before discovery and mutation so I can understand the scope while the operation is running, without waiting for its final report.

**Why this priority**: Automatic confirmation currently bypasses early tenant reporting, leaving operators without scope visibility until changes have already happened.

**Independent Test**: Run each of the six affected commands with auto-confirm and record output alongside discovery and mutation events. Verify selection context precedes discovery and validated affected-tenant context precedes the first mutation.

**Acceptance Scenarios**:

1. **Given** a named effective tenant, **When** any affected command runs with auto-confirm, **Then** `selection scope: <tenant> only` appears before discovery starts and before discovery scope or progress output, and affected tenants appear after impact validation but before the first mutation.
2. **Given** unfiltered selection, **When** discovery finds resources in one tenant, **Then** the pre-discovery selection scope remains `unfiltered across accessible tenants`, while the later affected-tenant summary identifies that one tenant.
3. **Given** an explicit tenant override, **When** discovery is about to begin, **Then** the established override information or warning and effective selection scope have already appeared; clearing a named configured tenant through `--tenant ""` or `--all-tenants` retains its broadening warning.
4. **Given** supported explicit resource keys, **When** the operation starts, **Then** selection context states that the tenant filter is not applied before resource resolution, and subsequent affected-tenant reporting uses actual resolved evidence rather than the configured tenant.
5. **Given** the same selection and resources as before this correction, **When** auto-confirm is used, **Then** no question is asked, the same resources are targeted, and tenant reporting adds no discovery pass.

---

### User Story 2 - Review complete context once before confirming (Priority: P1)

As an operator confirming an operation interactively, I want the same early context and all applicable warnings before the question, without repeated summaries obscuring progress or the final result.

**Why this priority**: Interactive and automatic confirmation must provide consistent scope visibility, and warnings must remain prominent when duplicate output is removed.

**Independent Test**: Run interactive operations against single-tenant, multiple-tenant, unknown-tenant, and empty scopes; check context ordering, warning severity, prompt placement, and occurrence counts.

**Acceptance Scenarios**:

1. **Given** a nonempty validated scope, **When** interactive execution reaches confirmation, **Then** selection context appeared before discovery and affected-tenant context appears before the prompt and any mutation.
2. **Given** targets in multiple known tenants, **When** impact validation finishes, **Then** one warning-level affected-tenant summary lists every distinct known tenant in deterministic order, without a second informational copy.
3. **Given** targets with unknown tenant metadata, including a mix with multiple known tenants, **When** the scope is reported, **Then** one non-blocking unknown-tenant warning appears before confirmation or mutation, in addition to the applicable known-tenant summary.
4. **Given** tenant context already displayed during execution, **When** confirmation and final human results are rendered, **Then** they do not repeat that context or its warnings, while ordinary confirmation and result information remains available.
5. **Given** an operator declines confirmation, **When** the command ends, **Then** the context was available before the decision and no mutation occurs.

---

### User Story 3 - Preserve automation output and audit evidence (Priority: P2)

As an automation or audit consumer, I want corrected human reporting to preserve my selected output contract and retain complete tenant evidence in reports.

**Why this priority**: Early human visibility must not corrupt scripted consumption or remove information needed to review an operation later.

**Independent Test**: Exercise the supported quiet, structured, automation, and other protected modes for the affected commands, and compare their output contracts and audit tenant evidence with equivalent human executions.

**Acceptance Scenarios**:

1. **Given** a protected output mode, **When** an affected command executes, **Then** no unintended human tenant labels or warnings enter its output; structured output remains parseable and quiet behavior remains suppressed according to existing rules.
2. **Given** an audit report is produced, **When** human summaries are no longer repeated in final output, **Then** the report still records complete applicable selection context, resolved known tenants, unknown-tenant evidence, and tenant warnings under its existing contract.
3. **Given** an automation execution with a defined confirmation policy, **When** tenant reporting moves earlier, **Then** no new prompt, exit behavior, or output-format change is introduced.
4. **Given** the published command guidance, **When** an operator reads the affected command examples and references, **Then** they describe selection context before discovery and affected tenants before mutation, including auto-confirm behavior.

### Edge Cases

- An empty validated scope must retain selection visibility and existing no-work behavior, without inventing an affected tenant or treating zero targets as unknown metadata.
- A scope containing only unknown tenant metadata must warn without presenting the configured tenant as an actual affected tenant.
- The default tenant, when identified in actual target evidence, uses its established display value; an empty selection filter still means unfiltered discovery.
- Repeated tenant values in target evidence appear only once in the affected-tenant summary.
- A named override identical to configuration, an absent override, or `--all-tenants` with already-unfiltered configuration must not gain unnecessary override warnings.
- Failure during discovery or impact validation must not present partial evidence as a validated mutation scope; existing failure and no-mutation behavior remains intact.
- Invalid input must retain existing validation behavior; early reporting must not start discovery or mutation for rejected input.
- Dry-run and no-work paths retain their existing preview and outcome contracts, with no repeated applicable tenant context.
- A mutation that later fails must not erase the context already shown before it; any report produced retains the evidence available under the existing report contract.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The correction MUST apply to `ops purge all-process-definitions` (including `apd`), `ops purge orphan-process-instances`, `ops purge process-instances-with-incidents`, `ops execute retention-policy`, `ops repair incident`, and `ops repair process-instance`.
- **FR-002**: For human modes that allow tenant context, each affected command MUST display its effective selection scope and applicable tenant override information or warnings before discovery or explicit-resource resolution begins, including before discovery scope and progress output.
- **FR-003**: Selection context MUST preserve established distinctions among named tenant filtering, unfiltered accessible-tenant selection, and explicit resource keys for which tenant filtering is not applied. Existing `--tenant` and `--all-tenants` meanings and override warning rules MUST remain unchanged.
- **FR-004**: Once discovery and impact validation establish the mutation scope, each affected command MUST report the known affected tenants and applicable unknown-tenant warnings before its first mutation, and before the confirmation prompt when one is required.
- **FR-005**: Exactly one known affected tenant MUST produce the established informational summary. Multiple distinct known tenants MUST produce one warning-level summary with unique tenant values in deterministic order. Unknown tenant metadata MUST produce its existing non-blocking warning, including when a multiple-tenant warning also applies.
- **FR-006**: Interactive and auto-confirm execution MUST follow the same reporting order. Auto-confirm MUST continue to skip the question without suppressing context permitted by the selected output mode.
- **FR-007**: Each applicable human selection summary, affected-tenant summary, override message, and tenant warning MUST appear once per execution. Confirmation and final human output MUST omit repetitions of context already displayed.
- **FR-008**: Audit reports MUST retain complete applicable tenant context and warnings under their existing contracts, regardless of human-output deduplication. Existing structured tenant evidence MUST also remain intact.
- **FR-009**: Human, quiet, JSON, automation, and every other supported output mode MUST preserve existing formatting, stream, suppression, prompt, and exit contracts, apart from the intended timing and deduplication of human tenant context. Supported keys-only output MUST remain one key per line without tenant commentary.
- **FR-010**: Tenant reporting MUST use existing discovery and validated scope evidence, with no extra discovery pass or metadata retrieval solely for reporting. It MUST NOT infer unknown tenants from configured selection values.
- **FR-011**: Empty scopes and discovery or validation failures MUST preserve existing no-work and failure behavior, without invented affected tenants, misleading validated scope claims, or newly enabled mutations.
- **FR-012**: Command-level acceptance coverage MUST include all six affected commands with auto-confirm, interactive ordering, and supported protected modes. Coverage MUST exercise named and unfiltered selection, explicit named and empty tenant overrides, `--all-tenants`, explicit keys where supported, and single-tenant, multiple-tenant, unknown-tenant, and empty scopes.
- **FR-013**: User-facing documentation, examples, and generated CLI references MUST describe the corrected reporting order and auto-confirm behavior consistently.
- **FR-014**: Tenant filtering, authorization, mutation targets, cancellation, deletion, repair, retries, and worker behavior MUST remain unchanged.

### Key Entities

- **Selection Context**: The effective selection scope and applicable override provenance before discovery, including the distinct meaning of explicit keys.
- **Validated Mutation Scope**: The targets established by existing discovery and impact validation, with known tenant identities and any unknown tenant metadata.
- **Tenant Context Evidence**: Selection and affected-tenant facts and warnings presented to operators or retained in existing structured and audit reports.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All six affected commands show selection context before discovery and validated affected-tenant context before the first mutation in auto-confirm acceptance scenarios that permit human context.
- **SC-002**: In 100% of interactive acceptance scenarios with a confirmation prompt, applicable scope information and tenant warnings are visible before the operator is asked to decide.
- **SC-003**: Every applicable human tenant summary or warning appears exactly once across each tested execution, including confirmation and final output.
- **SC-004**: Every tested multiple-tenant or unknown-tenant mutation scope produces the applicable warning before mutation, and every tested empty scope names no invented affected tenant.
- **SC-005**: All supported protected-mode regression scenarios preserve their output contracts, and all tested audit reports retain their complete applicable tenant evidence.
- **SC-006**: Across the acceptance scenarios, reporting adds zero discovery passes and causes zero changes to selected mutation targets or operation outcomes.
- **SC-007**: Documentation review finds no contradictory reporting-order guidance in the affected command examples or generated references.

## Assumptions

- Existing tenant resolution, override rules, explicit-key semantics, warning wording, and tenant representations remain authoritative; this feature corrects when and how often their human explanation appears.
- Existing discovery and impact validation provide the available tenant evidence. Missing metadata remains unknown rather than triggering additional retrieval.
- Output protections take precedence over displaying human preflight text; the feature does not introduce new output modes or audit schemas.
- This feature depends on the existing tenant-context and all-tenants behavior described in features 283 and 282. It does not reopen those features' selection semantics.
- Scope is limited to the six listed ops commands. General progress or output redesign and changes to mutation mechanics are excluded.
