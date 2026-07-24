# Feature Specification: CLI Debt Refactor

**Feature Branch**: `254-cli-debt-refactor`

**Created**: 2026-07-24

**Status**: Draft

**Input**: User description: "https://github.com/grafvonb/c8volt/issues/254"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Consistent Output During Large CLI Workflows (Priority: P1)

As a c8volt operator running commands against large Camunda estates, I need progress, prompts, summaries, and machine-readable output to behave consistently so I can trust interactive runs and automation equally.

**Why this priority**: The most visible risk is operator confusion or broken scripts when long-running commands page through thousands of resources, prompt for more work, or emit progress while returning JSON or keys-only output.

**Independent Test**: Can be tested by running representative paged read and bulk mutation commands in human, verbose, quiet, JSON, keys-only, and automation modes and verifying that output remains predictable for each mode.

**Acceptance Scenarios**:

1. **Given** an operator runs a paged command in human verbose mode, **When** the command retrieves multiple pages, **Then** durable progress includes enough context to understand page size, pages processed, cumulative results, whether more data exists, and the next action.
2. **Given** an operator runs a command in JSON, keys-only, quiet, or automation mode, **When** the command performs long-running work, **Then** progress, spinner, prompt, and diagnostic text do not corrupt the machine-consumable output.
3. **Given** a workflow discovers candidates for an ops operation, **When** discovery completes or stops at a user limit, **Then** the user receives a stable summary that distinguishes complete discovery from user-limited discovery.

---

### User Story 2 - Clear Ownership of Paging and Discovery Behavior (Priority: P2)

As a maintainer, I need command code to focus on command concerns while shared backend mechanics are owned by reusable service behavior so future commands can be changed safely and consistently.

**Why this priority**: The current command tree has duplicated page walking, result trimming, continuation, search strategy, and local filtering behavior across basic commands and mutations, which increases maintenance cost and regression risk.

**Independent Test**: Can be tested by reviewing the checked-in assessment and verifying that changed commands still pass behavior tests while duplicated paging and discovery ownership is reduced for the highest-debt basic commands.

**Acceptance Scenarios**:

1. **Given** the command tree contains 55 command nodes, **When** the assessment is complete, **Then** every node is classified by command family, paging behavior, mutation behavior, automation support, progress behavior, output constraints, ownership, execution style, and performance risk.
2. **Given** a basic read command retrieves multiple pages, **When** its internals are refactored, **Then** page traversal, continuation, total fallback, result trimming, and query strategy are no longer reimplemented independently in each command while the command still owns user-facing rendering and prompts.
3. **Given** an ops workflow has domain-specific discovery semantics, **When** the workflow is reviewed, **Then** similar-looking code is extracted only when the behavior and ownership are actually the same.

---

### User Story 3 - Faster High-Volume Operations Without Safety Loss (Priority: P3)

As an operator processing thousands of process instances and related resources, I need high-volume searches, discovery, enrichment, planning, and mutations to complete efficiently while still respecting safety limits and confirmation rules.

**Why this priority**: c8volt is used for operational work at scale, so refactoring must improve or preserve throughput and must not trade safety for cosmetic simplification.

**Independent Test**: Can be tested with fake-latency, benchmark-style, or targeted smoke scenarios that compare high-volume behavior before and after changed workflows.

**Acceptance Scenarios**:

1. **Given** a workflow processes many independent resources, **When** work can safely run in parallel, **Then** throughput improves or remains stable while respecting configured worker, batch, limit, fail-fast, and no-worker-limit behavior.
2. **Given** a destructive workflow is planned from discovered resources, **When** discovery or validation partially fails, **Then** confirmation safety, partial-completion reporting, and deterministic exit behavior remain explicit.
3. **Given** an operator sets a user limit or batch size, **When** a high-volume command runs, **Then** the command interprets those values consistently and reports the resulting scope clearly where appropriate.

---

### User Story 4 - Help and Documentation Match Behavior (Priority: P4)

As an operator or automation author, I need command help and generated documentation to describe limits, batch sizing, progress, and output behavior accurately so I can choose the right flags without reading source code.

**Why this priority**: Documentation drift makes scripts brittle and weakens trust, especially where `--batch-size`, `--limit`, automation mode, and progress output differ by command family.

**Independent Test**: Can be tested by comparing command help, generated CLI docs, and command behavior for every changed command and confirming that terminology is intentional and consistent.

**Acceptance Scenarios**:

1. **Given** a changed command exposes `--batch-size`, **When** a user reads help or generated documentation, **Then** the text clearly describes it as per-page request size unless the command intentionally documents a narrower meaning.
2. **Given** a changed command exposes `--limit`, **When** a user reads help or generated documentation, **Then** the text clearly describes it as the total user cap or frozen-scope cap as appropriate.
3. **Given** command support metadata is generated, **When** capability output is requested, **Then** automation and output-contract support accurately reflect the changed command behavior.

### Edge Cases

- Commands may process zero results, exactly one page, multiple pages, or a final partial page.
- A user-specified limit may be smaller than, equal to, or larger than the requested page size.
- Discovery may be complete, intentionally user-limited, or interrupted by an error after partial progress.
- Destructive workflows may require confirmation, auto-confirmation, dry-run behavior, or automation-safe refusal to prompt.
- Machine output modes must remain clean even when work is slow enough to normally show progress or activity.
- Similar ops workflows may appear duplicate while preserving different safety, reporting, or candidate-freezing semantics.
- Performance improvements must not create uncontrolled request fan-out or bypass operator-configured limits.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The feature MUST include a checked-in assessment covering all 55 command nodes and classifying each by command family, paging behavior, mutation behavior, automation support, activity indicator use, durable progress use, JSON and keys-only constraints, behavior ownership, serial or parallel execution style, and high-volume performance risk.
- **FR-002**: The feature MUST reduce duplicated basic command paging and discovery behavior for job, element, incident, and process-instance searches while preserving existing user-visible behavior unless a tested and documented behavior change is explicitly chosen.
- **FR-003**: The feature MUST keep command responsibilities focused on flag handling, command metadata, confirmations, prompt policy, render-mode selection, and stdout or stderr presentation.
- **FR-004**: The feature MUST move repeated backend mechanics such as page traversal, cursor or offset advancement, user-limit trimming, local compatibility filtering, query strategy selection, frozen-scope discovery, and mutation planning out of command-specific ownership where doing so improves clarity without reducing behavior fidelity.
- **FR-005**: The feature MUST review process-instance cancel and delete workflows separately, preserving destructive confirmation safety, partial-completion reporting, auto-confirm behavior, automation behavior, worker controls, batch-size controls, user-limit controls, and fail-fast behavior.
- **FR-006**: The feature MUST review high-level ops workflows selectively and preserve workflow-specific discovery, candidate-freezing, dry-run, reporting, and safety semantics where they differ.
- **FR-007**: The feature MUST identify and characterize performance-sensitive paths for process-instance search and enrichment, incident search and repair, process-instance cancel and delete planning, retention-policy execution, purge workflows, slow-process analysis, and job, element, and incident list commands.
- **FR-008**: The feature MUST use bounded parallel execution for independent high-volume work when it improves or preserves throughput safely, while respecting existing operator controls for workers, batch size, limits, fail-fast behavior, and worker-limit overrides.
- **FR-009**: The feature MUST define and apply a CLI-wide progress policy covering activity indicators, verbose durable paging progress, long-running mutation progress, ops discovery summaries, prompts, and machine-output silence.
- **FR-010**: The feature MUST ensure JSON, keys-only, quiet, and automation modes remain free of spinner, prompt, progress, and incidental diagnostic output.
- **FR-011**: The feature MUST keep `--batch-size` documented as per-page request size and `--limit` documented as total user cap or frozen-scope cap, with any command-family differences stated intentionally.
- **FR-012**: The feature MUST update command help, generated CLI documentation, examples, and command capability metadata whenever changed behavior, flags, output contracts, aliases, or examples affect users.
- **FR-013**: The feature MUST add targeted command and service tests for every changed behavior and broaden validation according to the risk of the changed workflow.
- **FR-014**: The feature MUST not modify generated Camunda clients, backfill old spec artifacts, introduce a generic static audit framework, force all ops discovery into one abstraction, or accept abstractions that make high-volume operations slower without a documented reason.

### Key Entities

- **Command Node**: A CLI command or grouping node that is classified by purpose, supported output contracts, automation behavior, paging behavior, mutation behavior, progress behavior, and performance risk.
- **Command Behavior Assessment**: A checked-in record of each command node's current classification, ownership boundaries, execution style, and high-volume risk.
- **Paged Discovery Workflow**: A user-visible retrieval process that may advance across multiple result pages, honor user limits, report progress, and produce human or machine output.
- **High-Volume Workflow**: A search, enrichment, discovery, planning, mutation, or analysis workflow expected to handle thousands of process instances or related resources.
- **Progress Policy**: A CLI-wide contract defining when activity indicators, durable progress lines, discovery summaries, prompts, or silence are expected.
- **Machine Output Contract**: The requirement that JSON, keys-only, quiet, and automation output remain parseable and free from incidental user-facing noise.
- **Operator Control**: A user-provided setting such as batch size, result limit, worker count, fail-fast behavior, auto-confirmation, automation mode, or worker-limit override that bounds scope, throughput, and safety.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of the 55 command nodes are classified in the checked-in assessment before refactoring decisions are marked complete.
- **SC-002**: 100% of changed commands preserve clean JSON, keys-only, quiet, and automation output in targeted tests.
- **SC-003**: Each high-volume workflow category named in the requirements has a recorded performance characterization before or during refactoring.
- **SC-004**: No changed high-volume workflow is slower in targeted validation without a documented reason and an accepted tradeoff.
- **SC-005**: 100% of changed destructive workflows retain explicit confirmation, auto-confirm, automation, partial-completion, and deterministic exit behavior coverage in tests or documented validation.
- **SC-006**: 100% of changed command help and generated CLI documentation describes `--batch-size`, `--limit`, progress behavior, and output-contract behavior consistently with implemented behavior.
- **SC-007**: Maintainers can identify the owner of paging, discovery, query strategy, progress, rendering, and confirmation behavior for every changed command from the assessment and tests without inspecting unrelated command implementations.

## Assumptions

- The feature is scoped to issue #254 and targets milestone v4.2.0.
- Existing public command behavior is treated as the compatibility baseline unless the implementation plan documents and tests an intentional change.
- The first deliverable is an assessment and classification artifact, followed by incremental refactoring slices ordered from lower-risk basic commands to higher-risk mutation and ops workflows.
- Existing operator controls for worker counts, batch sizes, limits, automation, quiet mode, no-indicator mode, fail-fast mode, auto-confirmation, and worker-limit overrides remain available where commands already support them.
- Generated Camunda client code, old spec artifacts, and unrelated feature plans remain out of scope.
