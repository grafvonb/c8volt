# Feature Specification: Add Cluster License Command

**Feature Branch**: `63-cluster-license`  
**Created**: 2026-03-18  
**Status**: Draft  
**Input**: User description: "https://github.com/grafvonb/c8volt/issues/63"

## Clarifications

### Session 2026-03-18

- Q: Should this feature add only the nested command `c8volt get cluster license`, or also a legacy-style direct path like `c8volt get cluster-license`? → A: Add only `c8volt get cluster license` under `get cluster`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Retrieve cluster license details (Priority: P1)

As an operator, I want to run `c8volt get cluster license` so that I can inspect the connected cluster's license status from the CLI without making raw API calls.

**Why this priority**: The issue's primary value is exposing already-supported cluster license information through a discoverable CLI command.

**Independent Test**: Run `c8volt get cluster license` against a configured environment and confirm it returns the expected license details with the same success and failure semantics as other `get cluster` commands.

**Acceptance Scenarios**:

1. **Given** the connected cluster returns license information, **When** the operator runs `c8volt get cluster license`, **Then** the command completes successfully and prints the license details in the CLI's standard structured output format.
2. **Given** the connected cluster returns license details with optional fields omitted, **When** the operator runs `c8volt get cluster license`, **Then** the command still succeeds and represents the available license information without inventing missing values.

---

### User Story 2 - Understand failures and command usage (Priority: P2)

As an operator, I want command help and failure behavior for cluster license retrieval to match the rest of the CLI so that I can use the command confidently in interactive and scripted workflows.

**Why this priority**: A new command is only dependable if users can discover it easily and reason about its error behavior without reading source code.

**Independent Test**: Review `c8volt get --help`, `c8volt get cluster --help`, and `c8volt get cluster license --help`, then run the command against a failing endpoint and confirm the help text, error messaging, and exit behavior are consistent with nearby cluster commands.

**Acceptance Scenarios**:

1. **Given** a user explores the CLI help tree, **When** they inspect `get` and `get cluster`, **Then** they can discover `license` as a supported cluster subcommand.
2. **Given** the license request fails because the cluster is unavailable or returns an error response, **When** the operator runs `c8volt get cluster license`, **Then** the command reports the failure with the CLI's established exit-code and error-handling behavior for `get` commands.

---

### User Story 3 - Maintain confidence through tests and docs (Priority: P3)

As a contributor, I want automated coverage and user-facing documentation for the new command so that the command can be maintained and reviewed without hidden behavior changes.

**Why this priority**: This repository expects command changes to ship with tests and documentation updates, especially when command discovery and output are user-visible.

**Independent Test**: Review the change set and confirm it includes targeted command tests for help, success, and failure paths plus documentation updates wherever the command becomes user-visible.

**Acceptance Scenarios**:

1. **Given** the new command is added, **When** maintainers run the affected command tests, **Then** they can verify successful output, failing output, and help-path discovery without manual inspection of live services.
2. **Given** users rely on generated CLI docs or repository documentation, **When** they review the updated command references, **Then** they can find `c8volt get cluster license` in the same locations as other supported `get cluster` commands.

### Edge Cases

- The command must succeed even when license responses omit optional fields such as expiration date or commercial-use metadata.
- The command must surface malformed or empty successful responses clearly instead of silently printing incomplete data.
- The command must preserve consistent exit behavior when the cluster endpoint returns transport errors, non-success HTTP responses, or authentication failures.
- Help and documentation must make clear that `c8volt get cluster license` is the supported command path and must not imply support for a legacy direct `c8volt get cluster-license` command.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose cluster license retrieval through the user-visible command path `c8volt get cluster license`.
- **FR-002**: The system MUST retrieve and present the connected cluster's license information, including license validity and license type, and include additional license attributes when they are provided by the connected Camunda version.
- **FR-003**: The system MUST preserve the CLI's established structured output behavior for successful `get` commands when printing cluster license results.
- **FR-004**: The system MUST preserve the CLI's established error reporting and exit semantics when cluster license retrieval fails.
- **FR-005**: The system MUST make the new command discoverable through `get` and `get cluster` help output.
- **FR-006**: The system MUST define the user-visible command behavior for missing optional license fields and malformed success responses.
- **FR-007**: The system MUST keep `c8volt get cluster license` as the only user-visible command path introduced by this feature and MUST NOT add a legacy or alternate direct command such as `c8volt get cluster-license`.
- **FR-008**: The system MUST keep the feature bounded to exposing existing cluster license capability through the CLI without redesigning the underlying cluster service behavior.
- **FR-009**: The system MUST add or update automated tests that cover command discovery/help, successful license retrieval, and representative failure behavior.
- **FR-010**: The system MUST update user-facing documentation in the same change wherever generated CLI references or usage examples are affected by the new command.
- **FR-011**: The system MUST define how maintainers verify the feature through targeted command tests and observable CLI output without reading implementation details.

### Key Entities *(include if feature involves data)*

- **Cluster License Command**: The user-visible CLI entry point for requesting license details from the connected cluster.
- **Cluster License Result**: The structured license information returned to the operator, including validity, license type, and optional metadata when available.
- **License Retrieval Failure**: The observable command outcome for transport, authentication, upstream, or malformed-response problems encountered while requesting license details.

## Assumptions

- The internal cluster service already exposes cluster license retrieval for the supported Camunda versions and does not require a separate service-surface redesign for this issue.
- The new user-visible command should live under the existing `get cluster` hierarchy to match current command organization.
- This feature intentionally adds only the nested command path `c8volt get cluster license` and does not introduce a legacy or alternate direct `c8volt get cluster-license` command.
- Generated CLI reference documentation under `docs/cli/` will be refreshed from Cobra metadata rather than edited by hand if command help changes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can retrieve cluster license information successfully through `c8volt get cluster license` in one command invocation without consulting source code or raw APIs.
- **SC-002**: Maintainers can verify help discovery, successful output, and failure handling for the command through automated tests in a single review session.
- **SC-003**: Users reviewing CLI help or generated command reference pages can discover `c8volt get cluster license` alongside the other supported `get cluster` commands on first read.
- **SC-004**: The feature is implemented without changing the externally observable behavior of existing cluster retrieval commands.
