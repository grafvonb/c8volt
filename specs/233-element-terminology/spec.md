# Feature Specification: Element Terminology Standardization

**Feature Branch**: `233-element-terminology`

**Created**: 2026-05-25

**Status**: Draft

**GitHub Issue**: [#233](https://github.com/grafvonb/c8volt/issues/233) - refactor: standardize incident and process context on element terminology

**Input**: GitHub issue #233 requires removing legacy flow-node naming from public c8volt contracts and standardizing incident, process-instance, ops, JSON, CLI, docs, and human output surfaces on Camunda v2 `element*` terminology.

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.

  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
-->

### User Story 1 - Filter Incidents With Element Terminology (Priority: P1)

Operators and scripts can filter incidents using the same element terminology already used by job-oriented workflows, without seeing legacy flow-node flags in public command contracts.

**Why this priority**: Incident filtering is the most direct public contract change and must establish the canonical flag names before related ops workflows can align.

**Independent Test**: Can be tested by invoking incident filtering with `--element-id` and `--element-instance-key`, then verifying the old public flags are rejected as unknown.

**Acceptance Scenarios**:

1. **Given** incidents exist for a BPMN element, **When** a user runs `get incident --element-id <id>`, **Then** the command filters by that element ID and exposes no `flowNodeId` public field.
2. **Given** an element instance key is known, **When** a user runs `get incident --element-instance-key <key>`, **Then** the command filters by that element instance key.
3. **Given** a user runs `get incident --flow-node-id <id>` or `get incident --fni-key <key>`, **When** the command parses flags, **Then** it rejects the legacy flag as unknown before remote work.

---

### User Story 2 - Show Canonical Incident Context (Priority: P2)

Operators inspecting incidents through direct, process-instance, walk, and ops repair/purge views see one canonical incident context vocabulary in JSON and compact human output.

**Why this priority**: Once selection uses canonical flags, rendered results must stop mixing old and new terms so humans and automation receive one consistent contract.

**Independent Test**: Can be tested by rendering incident-bearing command output in human and JSON modes and scanning for `elementId`, `elementInstanceKey`, `e:`, and `ei:` while ensuring legacy names are absent.

**Acceptance Scenarios**:

1. **Given** a command renders incident JSON, **When** incidents include element context, **Then** JSON uses `elementId` and `elementInstanceKey`, not `flowNodeId` or `flowNodeInstanceKey`.
2. **Given** a command renders human incident rows, **When** incidents include element context, **Then** the compact labels are `e:` and `ei:`, not `fn:` or `fni:`.
3. **Given** ops repair or purge workflows use incident filters, **When** a user reviews help, dry-run, JSON, or confirmation output, **Then** only canonical element terminology appears.

---

### User Story 3 - Standardize Process Context Fields (Priority: P3)

Operators and automation consuming process-instance context see parent element instance terminology consistently wherever process-instance trees, incident enrichment, and walk output expose parent execution context.

**Why this priority**: Process-instance context is adjacent to incident context and must be updated before documentation and contract discovery can be considered complete.

**Independent Test**: Can be tested by rendering process-instance JSON and walk output with parent context and verifying `parentElementInstanceKey` replaces `parentFlowNodeInstanceKey`.

**Acceptance Scenarios**:

1. **Given** process-instance JSON includes parent execution context, **When** output is rendered, **Then** it uses `parentElementInstanceKey`.
2. **Given** a user runs `get pi --with-incidents` or `walk pi --with-incidents`, **When** incident and process context are displayed, **Then** field names and labels use canonical element terminology.

---

### User Story 4 - Keep Legacy Names Behind Adapter Boundaries (Priority: P4)

Maintainers can continue supporting older generated Camunda clients while guaranteeing that legacy flow-node names never leak into public c8volt models, CLI flags, JSON contracts, docs, or human output.

**Why this priority**: Adapter containment completes the breaking cleanup while preserving supported-version behavior where upstream wire fields still use legacy names.

**Independent Test**: Can be tested by checking public models, command contracts, generated docs, and version adapters so legacy `FlowNode*` names appear only where generated-client mappings require them.

**Acceptance Scenarios**:

1. **Given** a supported runtime version exposes legacy generated fields, **When** c8volt maps adapter responses, **Then** public domain and facade results use canonical element names.
2. **Given** public contracts, docs, and command metadata are inspected, **When** legacy flow-node names are searched, **Then** they are absent from c8volt-facing surfaces.

### Edge Cases

- Generated Camunda or Operate clients may still contain `FlowNode*` fields; those names are allowed only inside generated client code and version-specific adapter mapping boundaries.
- Legacy public flags such as `--flow-node-id` and `--fni-key` must fail as unknown flags rather than acting as deprecated aliases.
- Existing JSON consumers will experience a breaking field-name change; the feature intentionally does not provide transitional compatibility aliases.
- Human output must avoid old abbreviations even in ops dry-runs, confirmations, summaries, and report-oriented output.
- Documentation and command metadata must be regenerated from source behavior so generated pages do not preserve stale legacy terminology.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose incident filtering with public flags named `--element-id` and `--element-instance-key`.
- **FR-002**: The system MUST reject `--flow-node-id` and `--fni-key` as unknown public flags.
- **FR-003**: The system MUST use `elementId` and `elementInstanceKey` in public incident JSON output.
- **FR-004**: The system MUST use `parentElementInstanceKey` in public process-instance JSON output.
- **FR-005**: The system MUST remove `flowNodeId`, `flowNodeInstanceKey`, and `parentFlowNodeInstanceKey` from public c8volt models and JSON contracts.
- **FR-006**: The system MUST render compact human incident context as `e:<elementId>` and `ei:<elementInstanceKey>`.
- **FR-007**: The system MUST remove public human output abbreviations `fn:` and `fni:`.
- **FR-008**: The system MUST align incident filter flags across `get incident`, `ops repair incident`, and `ops purge process-instances-with-incidents`.
- **FR-009**: The system MUST align incident and process context terminology across `get pi --with-incidents` and `walk pi --with-incidents`.
- **FR-010**: The system MUST keep generated legacy Camunda or Operate `FlowNode*` names isolated to generated clients and version-specific adapter mappings.
- **FR-011**: The system MUST update command contract tests to assert canonical flags and fields are present and old public flags and fields are absent.
- **FR-012**: The system MUST update README-facing documentation and generated CLI documentation from source or generation paths when user-facing command surfaces change.
- **FR-013**: The Ralph implementation launch instructions for this feature MUST include `--implementation-context specs/ralph-implementation-rules.md`.

### Key Entities *(include if feature involves data)*

- **Incident Context**: The public incident identity and location information shown in filters, JSON, human rows, and ops workflows; canonical element attributes are `elementId` and `elementInstanceKey`.
- **Process Instance Context**: The public process-instance relationship information shown in process inspection and walk flows; canonical parent attribute is `parentElementInstanceKey`.
- **Adapter Mapping Boundary**: The internal containment boundary where generated legacy upstream field names may be translated into canonical c8volt terms.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every affected public command help, contract, and generated documentation page uses the canonical element terminology for incident and process context.
- **SC-002**: Automated tests prove `--element-id` and `--element-instance-key` work for incident filtering and the legacy public flags are rejected.
- **SC-003**: Automated JSON-output tests prove old public JSON field names are absent and canonical element field names are present.
- **SC-004**: Automated human-output tests prove `e:` and `ei:` appear where element context is present and `fn:` and `fni:` are absent.
- **SC-005**: Repository validation for the affected command, facade, service, docs, and contract areas passes before tasks are marked complete.

## Assumptions

- This is an intentional breaking change; no public compatibility aliases will be kept unless a later issue reopens that decision.
- Existing supported Camunda versions remain in scope; version-specific generated wire names must be translated at adapter boundaries.
- Generated client files are not manually edited for this cleanup.
- Documentation updates follow the repository generation path rather than hand-editing derived CLI reference pages.
- Planning, task generation, and Ralph iterations must apply `specs/ralph-implementation-rules.md`.
