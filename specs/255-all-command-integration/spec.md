# Feature Specification: All-Command Integration Suite

**Feature Branch**: `255-all-command-integration`

**Created**: 2026-07-25

**Status**: Draft

**Input**: User description: "Create a destructive all-command integration suite for c8volt that covers all 55 commands and all flags/version behavior against real local Camunda clusters, using the default `$HOME/.config/c8volt` configuration only. The suite must tolerate clean or dirty disposable clusters, may mutate existing data, prefer c8volt commands for setup, prefer embedded BPMN models, record missing command/API and fixture needs as proposals, validate command help and generated CLI examples, and remain isolated under `integration/` so normal Speckit and Ralph implementation runs do not load it."

**Reusable Context**: `integration/assets/all-command-go-integration-rules.md` is context-only guidance for this feature. It is not normal Speckit or Ralph implementation context.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Discover Complete Command Coverage (Priority: P1)

Release validators can prove that every command currently exposed by c8volt has an explicit integration coverage entry.

**Why this priority**: Without a complete command inventory, the suite can silently miss newly added or refactored commands.

**Independent Test**: Can be tested by running the suite inventory check and verifying it compares the live command contract against the suite coverage manifest.

**Acceptance Scenarios**:

1. **Given** the current c8volt command contract reports 55 command nodes, **When** the coverage inventory check runs, **Then** each command node has an explicit suite coverage entry.
2. **Given** a new command appears in the command contract, **When** the coverage inventory check runs before the suite is updated, **Then** the suite fails with the missing command path.
3. **Given** a command is removed or renamed, **When** the coverage inventory check runs, **Then** the suite reports the stale coverage entry rather than silently passing.

---

### User Story 2 - Validate Real Local Profiles (Priority: P2)

Release validators can run the suite against real local Camunda profiles from the operator's default c8volt configuration.

**Why this priority**: The suite is intended to prove real operator behavior and must not depend on generated test configuration or altered authentication behavior.

**Independent Test**: Can be tested by running the profile gate and verifying it uses only profiles available from the default local c8volt configuration.

**Acceptance Scenarios**:

1. **Given** usable local profiles exist for required Camunda versions, **When** the suite starts, **Then** it confirms connectivity and version before any destructive scenario runs.
2. **Given** a required local profile is missing or unreachable, **When** the suite starts, **Then** it fails early with a clear environment readiness message.
3. **Given** a test attempts to use a private generated configuration, **When** the suite validates its startup contract, **Then** the run is rejected.

---

### User Story 3 - Seed And Reuse Disposable Cluster Data (Priority: P3)

Release validators can run command-family scenarios in both clean and dirty disposable clusters without brittle assumptions about existing data.

**Why this priority**: The real integration target may contain no data or substantial unrelated data, and both states must be valid.

**Independent Test**: Can be tested by running the seeding and discovery checks against an empty cluster and a cluster with unrelated process data.

**Acceptance Scenarios**:

1. **Given** the selected cluster has no relevant data, **When** the suite prepares a scenario, **Then** it creates the required data using c8volt commands whenever available.
2. **Given** the selected cluster already contains unrelated data, **When** search, read, mutation, purge, or repair scenarios run, **Then** unrelated data does not cause false failures.
3. **Given** a command scenario requires destructive behavior, **When** the suite runs against the disposable target, **Then** it may mutate seeded or pre-existing data and records what happened.

---

### User Story 4 - Exercise Command Families And Flags (Priority: P4)

Release validators can prove each command family works across supported version behavior, output modes, aliases, local validations, and destructive confirmation paths.

**Why this priority**: c8volt's operator value depends on stable command behavior across reads, mutations, and high-level ops workflows.

**Independent Test**: Can be tested by running one command-family group and verifying it records coverage for every command-local flag and expected output mode in that family.

**Acceptance Scenarios**:

1. **Given** a command has aliases, required flags, optional flags, and output modes, **When** its family scenario runs, **Then** the suite records coverage for each supported surface.
2. **Given** a command has version-specific support or unsupported behavior, **When** the suite runs against configured version profiles, **Then** the suite verifies the expected behavior for each applicable version.
3. **Given** a command is destructive, **When** the suite executes the destructive scenario, **Then** it covers preview behavior where available and the real confirmed mutation path.

---

### User Story 5 - Report Setup Gaps As Product Proposals (Priority: P5)

Maintainers can see where integration coverage required direct Camunda setup or additional embedded models, so those gaps can become future c8volt improvements.

**Why this priority**: The suite should prefer c8volt commands and embedded assets, but it must still be able to cover states c8volt cannot currently create.

**Independent Test**: Can be tested by running a scenario that requires unavailable setup support and verifying the suite records a proposal rather than hiding the fallback.

**Acceptance Scenarios**:

1. **Given** no c8volt command can create a required test state, **When** the suite uses direct Camunda setup, **Then** it records the missing command or command-extension proposal.
2. **Given** existing embedded BPMN models cannot create a required workflow condition, **When** the suite uses another setup path, **Then** it records the missing embedded model proposal.
3. **Given** proposal records are generated, **When** maintainers review suite output, **Then** each proposal identifies the coverage need, affected command area, affected versions, and operator value.

---

### User Story 6 - Validate Help And Example Trustworthiness (Priority: P6)

Maintainers can verify that command help and generated CLI examples remain executable and that destructive examples are clearly warned.

**Why this priority**: c8volt examples guide real operators and automation users; stale or unsafe examples erode trust.

**Independent Test**: Can be tested by running example validation and verifying read-only examples execute, mutating examples use disposable targets, and destructive examples carry warnings.

**Acceptance Scenarios**:

1. **Given** a command help example contains placeholders, **When** example validation runs, **Then** placeholders are substituted with suite-created keys, resources, or embedded files before execution.
2. **Given** a generated CLI example is read-only, **When** example validation runs, **Then** the example executes successfully against the selected profile.
3. **Given** an example performs mutation, **When** example validation inspects help or generated documentation, **Then** the example is marked with a clear warning about potentially dangerous actions.

### Edge Cases

- The selected cluster may be completely clean and require all scenario data to be created during the run.
- The selected cluster may already contain unrelated definitions, instances, incidents, jobs, variables, tenants, and resources.
- Broad destructive workflows may alter or delete pre-existing data in the selected disposable target.
- Some Camunda versions may not support specific commands, flags, or backend states.
- Search results may include unrelated data, so assertions must not rely on exact global counts.
- No-match scenarios must use impossible selectors rather than assuming an empty cluster.
- Cleanup may fail or intentionally retain data; reporting must preserve this evidence without hiding command results.
- Command help or generated docs may contain examples that require dynamic substitution before execution.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The suite MUST derive or validate its command inventory from the live c8volt command contract.
- **FR-002**: The suite MUST require explicit coverage entries for all 55 current command nodes and fail when live command inventory and coverage entries diverge.
- **FR-003**: The suite MUST use the operator's default local c8volt configuration and MUST NOT rely on a generated private config file.
- **FR-004**: The suite MUST support selecting existing local profiles for Camunda 8.7, 8.8, and 8.9 validation where those profiles are available.
- **FR-005**: The suite MUST confirm profile connectivity and expected Camunda version before destructive scenarios run.
- **FR-006**: The suite MUST tolerate both clean and dirty disposable cluster states.
- **FR-007**: The suite MUST be allowed to mutate, cancel, delete, resolve, repair, or purge seeded and pre-existing data in the selected disposable cluster.
- **FR-008**: The suite MUST prefer c8volt commands for test data creation before using direct Camunda setup.
- **FR-009**: The suite MUST prefer embedded c8volt BPMN models before using external process models or direct setup.
- **FR-010**: Historical 255 proposal output recorded direct Camunda setup fallbacks; new work MUST track such gaps in spec-owned artifacts rather than generating backlog proposals during test execution.
- **FR-011**: Historical 255 proposal output recorded missing embedded-model capabilities; new work MUST track such gaps in spec-owned artifacts rather than generating backlog proposals during test execution.
- **FR-012**: The suite MUST group coverage by c8volt command family, including explicit groups for high-level ops commands.
- **FR-013**: Each leaf command coverage entry MUST include aliases, required flags, command-local flags, relevant persistent flags, supported output modes, success behavior, validation behavior, and applicable version behavior.
- **FR-014**: Parent and grouping command coverage MUST include help/discovery behavior and no-argument behavior.
- **FR-015**: Destructive command coverage MUST exercise preview behavior where supported and confirmed mutation behavior where required.
- **FR-016**: The suite MUST validate command help examples and generated CLI documentation examples.
- **FR-017**: Mutating examples MUST be clearly marked with a warning about potentially dangerous actions.
- **FR-018**: The suite MUST produce reusable evidence that distinguishes seeded, pre-existing, mutated, retained, and cleanup-failed data.
- **FR-019**: The suite context and artifacts MUST remain isolated under `integration/` and MUST NOT become normal Speckit, Ralph, public docs, or generated docs inputs.
- **FR-020**: Suite reports MUST distinguish product failures, harness setup failures, missing fixture support, missing command support, and cluster/environment availability failures.

### Key Entities

- **Command Inventory**: The complete set of command paths reported by c8volt's live command contract for the version under test.
- **Coverage Entry**: The suite's explicit declaration that a command path and its expected flags, aliases, output modes, and version behavior are covered.
- **Integration Profile**: A named local c8volt profile from the default configuration that identifies a disposable Camunda target.
- **Run Marker**: A unique value attached to suite-created data so results and cleanup evidence can distinguish seeded resources from pre-existing resources.
- **Evidence Record**: A per-command result containing inputs, profile/version context, output, exit status, resource keys, and data ownership classification.
- **Legacy Proposal Record**: A documented gap where direct Camunda setup or missing embedded BPMN support was required to complete coverage. This generated-test-output pattern is deprecated for new work.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The inventory check reports 100% explicit coverage for the 55 current command nodes.
- **SC-002**: The suite rejects runs with missing or unreachable required profiles before any destructive scenario starts.
- **SC-003**: The suite completes its seeded-data preparation successfully on both clean and dirty disposable clusters.
- **SC-004**: Every command-local flag in the coverage manifest is exercised by at least one recorded scenario.
- **SC-005**: Every high-level ops command has a dedicated recorded scenario, including preview and real execution where applicable.
- **SC-006**: Example validation reports every checked example as executable or reports an actionable failure with the command path and source location.
- **SC-007**: Every direct setup fallback and missing embedded-model need appears in the proposal report with enough detail for maintainers to triage it.
- **SC-008**: The suite writes reusable evidence for each command group, including stdout, stderr, exit status, selected profile/version, and data ownership classification.

## Assumptions

- The selected integration clusters are disposable and are not shared, production, customer, or otherwise protected environments.
- The existing `integration/` folder is the correct home for suite rules, prompts, scripts, generated evidence conventions, and future suite code.
- Normal feature implementation, docs generation, Speckit tasks, and Ralph iterations will not read the suite rules file unless explicitly directed to work on this integration suite.
- Version coverage depends on local profiles being available for the corresponding Camunda versions.
- Some proposal records may identify future product work but do not themselves authorize product behavior changes.
