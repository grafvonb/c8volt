# Feature Specification: Walk PI Elements

**Feature Branch**: `251-walk-pi-elements`

**Created**: 2026-07-22

**Status**: Draft

**Input**: User description: "GitHub issue #251: feat(walk pi): add --with-elements to show runtime element instances under walked process instances"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect Runtime Elements During Process Walk (Priority: P1)

As a Camunda operator investigating a process-instance family, ancestry, or descendant tree, I can request runtime element details directly in `walk process-instance` / `walk pi` output so I can understand where work is active, completed, or incidented without issuing separate per-instance inspection commands.

**Why this priority**: This is the core value of the feature. `walk pi` is the natural diagnostic command for related process instances, and operators need runtime element context while preserving the traversal view.

**Independent Test**: Can be fully tested by walking a process-instance family with `--with-elements` and confirming that each walked process instance shows only its own element rows while the traversal order and relationships remain unchanged.

**Acceptance Scenarios**:

1. **Given** a process-instance family with runtime elements on multiple process instances, **When** the operator runs `walk pi --key <key> --with-elements`, **Then** the output includes an `elements:` section under each process-instance row that has elements.
2. **Given** a walked process instance with no runtime elements, **When** the operator requests `--with-elements`, **Then** the process-instance row remains visible and no misleading element rows are shown.
3. **Given** a runtime element with incident information, **When** the operator views walked output with elements, **Then** the element row clearly indicates the incident state in the same compact grammar used by existing process-instance element output.

---

### User Story 2 - Preserve Traversal Modes With Elements (Priority: P2)

As an operator, I can combine `--with-elements` with family, children, parent, and flat walk modes so that element enrichment does not alter which process instances are selected or how traversal paths are presented.

**Why this priority**: The feature must be dependable across the existing walk modes, not only the default family view.

**Independent Test**: Can be tested by running `walk pi` with `--children --with-elements`, `--parent --with-elements`, and `--flat --with-elements` against known related process instances and verifying that selection and ordering match the same commands without element enrichment.

**Acceptance Scenarios**:

1. **Given** a selected process instance with descendants, **When** the operator runs `walk pi --key <key> --children --with-elements`, **Then** the selected process instance and descendants are enriched with their own elements.
2. **Given** a selected process instance with ancestors, **When** the operator runs `walk pi --key <key> --parent --with-elements`, **Then** ancestry rows remain in ancestry order and include element details where available.
3. **Given** a selected traversal rendered in flat mode, **When** the operator runs `walk pi --key <key> --flat --with-elements`, **Then** path separators remain readable and element sections do not obscure the process-instance paths.

---

### User Story 3 - Use Elements In Scripted Output Safely (Priority: P3)

As an automation author, I can request JSON output for walked process instances with elements and receive traversal metadata plus per-item element arrays, while invalid combinations fail before enrichment begins.

**Why this priority**: c8volt is used in scripts, and enriched output must remain stable, complete, and safe to consume.

**Independent Test**: Can be tested by running JSON and invalid flag combinations and verifying that valid JSON preserves traversal metadata and invalid keys-only enrichment fails with a clear error before remote enrichment.

**Acceptance Scenarios**:

1. **Given** an operator requests JSON output with `walk pi --key <key> --with-elements`, **When** the command succeeds, **Then** the JSON includes traversal metadata and per-item `elements` data for enriched process instances.
2. **Given** an operator combines `--with-vars`, `--with-incidents`, and `--with-elements`, **When** human output is rendered, **Then** sections appear in the order `vars:`, `incidents:`, `elements:`.
3. **Given** an operator combines `--keys-only` with `--with-elements`, **When** validation runs, **Then** the command is rejected with a clear error before element lookup or remote enrichment starts.

### Edge Cases

- Walked process instances may have zero, one, or many runtime elements.
- A process-instance row may have both detail sections and child process instances; child rows must not appear nested under an `elements:` section.
- Element lookup may fail for one or more walked process instances; the command must fail rather than presenting partially enriched success output.
- Camunda 8.7 environments do not support the required element-search capability and must return the same unsupported-capability outcome used by existing process-instance element enrichment.
- Existing `--with-vars` and `--with-incidents` validation and rendering behavior must remain unchanged when `--with-elements` is absent.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow operators to request runtime element enrichment for `walk process-instance` and `walk pi` using `--with-elements`.
- **FR-002**: The system MUST keep traversal selection unchanged when element enrichment is requested.
- **FR-003**: The system MUST resolve the normal walk result before enriching walked process instances with runtime elements.
- **FR-004**: The system MUST attach each runtime element only to the walked process instance that owns it.
- **FR-005**: Human output MUST preserve existing walk order and tree structure when element details are present.
- **FR-006**: Human element rows MUST use the same compact row grammar already used by process-instance output with elements.
- **FR-007**: When variables, incidents, and elements are requested together, human output MUST render detail sections in the order `vars:`, `incidents:`, `elements:`.
- **FR-008**: JSON output with elements MUST preserve traversal metadata and include per-item element data for each enriched activity item.
- **FR-009**: JSON output MUST allow the same walked item to include variables, incidents, and elements when those enrichments are requested together.
- **FR-010**: The system MUST reject `--keys-only --with-elements` with a clear validation error before any remote element enrichment occurs.
- **FR-011**: The system MUST preserve existing validation for `--with-vars` and `--with-incidents`.
- **FR-012**: Element lookup failures MUST fail the command rather than reporting partially enriched success.
- **FR-013**: Camunda 8.7 usage MUST return the existing unsupported element-search capability outcome used by current element enrichment.
- **FR-014**: The feature MUST NOT add element-specific filters, listener enrichment, job enrichment, metrics enrichment, traversal-selection changes, parent/child discovery changes, or missing-ancestor behavior changes.

### Key Entities *(include if feature involves data)*

- **Walked Process Instance**: A process instance returned by a walk traversal, including its key, definition identity, tenant, lifecycle state, dates, parent/root relationship, and optional enrichment sections.
- **Runtime Element Instance**: An element instance owned by a process instance, including its element instance key, element identity, element name when available, type, state, dates, duration when available, tenant, definition references, and incident status.
- **Traversal Result**: The selected walk outcome, including mode, root key, keys, relationships between walked process instances, and ordered activity items.
- **Activity Item**: The display and JSON unit that combines a walked process instance with optional variables, incidents, and elements.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can inspect runtime elements for every process instance in a walked family with one command instead of running separate element inspection commands per process instance.
- **SC-002**: In representative family, children, parent, and flat walks, 100% of process instances selected without `--with-elements` are still selected with `--with-elements`.
- **SC-003**: In tree output containing both detail sections and child process instances, 100% of child process-instance rows remain visually attached to the correct process-instance parent rather than to a detail section.
- **SC-004**: In combined enrichment output, 100% of rows with variables, incidents, and elements render detail sections in the documented order.
- **SC-005**: Scripted JSON consumers can access traversal metadata and per-item element arrays from a single successful command response.
- **SC-006**: Invalid `--keys-only --with-elements` requests fail before remote enrichment in 100% of validation tests.
- **SC-007**: Existing walk behavior without `--with-elements` remains unchanged across the covered human, JSON, keys-only, and validation test cases.

## Assumptions

- `walk process-instance` and `walk pi` are equivalent user-facing command paths for this feature.
- The existing process-instance element enrichment behavior defines the canonical element row grammar, JSON element shape, and unsupported-capability outcome.
- Element enrichment is requested only for keyed walk behavior, consistent with existing walk command structure.
- If an enriched process instance has no elements, the command should not invent placeholder element rows.
- Documentation and generated CLI references will be updated during implementation if command flags or user-visible output change.
