# Feature Specification: Define Machine-Readable CLI Contracts

**Feature Branch**: `078-machine-cli-contracts`  
**Created**: 2026-04-17  
**Status**: Draft  
**Input**: User description: "GitHub issue #78: feat(cli): add machine-readable command contracts for ai and automation"

## GitHub Issue Traceability

- **Issue Number**: 78
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/78
- **Issue Title**: feat(cli): add machine-readable command contracts for ai and automation

## Clarifications

### Session 2026-04-17

- Q: Which discovery surface should define the machine-readable CLI contract? → A: Add one dedicated top-level discovery command such as `c8volt capabilities --json`.
- Q: How should machine-readable command results be shaped across command families? → A: Add one shared top-level result envelope with common outcome fields plus a command-specific payload field.
- Q: Which outcome categories should the shared result envelope standardize? → A: Use four explicit outcome categories: `succeeded`, `accepted`, `invalid`, and `failed`.
- Q: How should commands without the shared machine result contract appear in discovery? → A: List them in `capabilities` and mark their machine-readable result contract as unsupported or limited.
- Q: How should machine-readable results relate to existing exit-code behavior? → A: Preserve existing exit-code semantics as the primary process-level signal, and use the shared JSON envelope for detailed machine-readable outcome data.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Discover Safe Command Contracts (Priority: P1)

As an AI or automation caller, I want one dedicated top-level machine-readable discovery command for the CLI so that I can learn which commands exist, which flags they accept, which output modes they support, and whether they read or change state without scraping prose help.

**Why this priority**: Reliable discovery is the entry point for every machine consumer. Without it, downstream automation stays brittle even if some individual commands already support structured output.

**Independent Test**: Call the dedicated top-level discovery command alone and verify it returns machine-readable command metadata for representative top-level and nested commands, including supported flags, output modes, and mutation/read-only classification.

**Acceptance Scenarios**:

1. **Given** a machine consumer needs to choose a `c8volt` command, **When** it calls the dedicated top-level discovery command, **Then** it receives a machine-readable description of the available commands and nested command paths without parsing human help text.
2. **Given** a machine consumer inspects a representative command family such as `get`, `run`, or `delete`, **When** it reads the discovery output, **Then** it can identify supported flags, output modes, whether the command is read-only or state-changing, and whether the shared machine result contract is supported, limited, or unsupported.

---

### User Story 2 - Receive Stable Machine Results (Priority: P2)

As an automation caller, I want supported commands to return a stable machine-readable contract with one shared top-level result envelope, four explicit outcome categories, and a command-specific payload so that I can make safe follow-up decisions without guessing from mixed human output.

**Why this priority**: Discovery only helps if command execution results are equally reliable. Stable result shapes are required for safe retries, waits, and error handling.

**Independent Test**: Execute representative commands from each in-scope family under successful, accepted-but-not-yet-confirmed, validation-failure, and remote-failure conditions, then verify the caller can distinguish the `succeeded`, `accepted`, `invalid`, and `failed` outcome categories through the shared top-level result envelope while still receiving the command-specific payload.

**Acceptance Scenarios**:

1. **Given** a supported command completes successfully, **When** a machine consumer requests the structured contract, **Then** the command returns the shared top-level result envelope with a success outcome and the command-specific payload rather than requiring the caller to parse human-oriented text.
2. **Given** a state-changing command accepts work before completion can be confirmed, **When** the command returns control to the caller, **Then** the shared result envelope uses the `accepted` outcome instead of `succeeded`.
3. **Given** a caller submits invalid input, **When** the command rejects the request, **Then** the shared result envelope uses the `invalid` outcome and tells the caller what to correct.
4. **Given** a command fails because of a remote or infrastructure problem, **When** the machine-readable result is returned, **Then** the shared result envelope uses the `failed` outcome so the caller can distinguish that failure class from `invalid` and `succeeded`.
5. **Given** a machine consumer runs a command in structured mode, **When** the process exits, **Then** the existing exit-code contract remains the primary process-level signal and the shared result envelope provides the detailed machine-readable outcome.

---

### User Story 3 - Keep Human CLI Behavior Intact (Priority: P3)

As a maintainer, I want the machine contract to reuse the current CLI structure and preserve human-oriented behavior so that automation improves without forcing a redesign of the existing command model.

**Why this priority**: The issue explicitly avoids a CLI taxonomy rewrite. The new contract must layer onto the established command surface rather than creating a parallel operator experience.

**Independent Test**: Review representative commands with and without machine-oriented output, then verify the existing human-facing command structure and documented usage remain supported while automation guidance becomes explicit.

**Acceptance Scenarios**:

1. **Given** an existing human operator uses the current command structure, **When** the machine contract is introduced, **Then** the same command families and human-friendly usage patterns remain available.
2. **Given** maintainers review the updated documentation and tests, **When** they inspect representative command families, **Then** they can see one recommended automation contract that builds on the current CLI structure instead of replacing it.

### Edge Cases

- Commands that do not yet support the shared machine result contract must still be reported by the dedicated top-level discovery command as unsupported or limited rather than silently appearing contract-complete.
- A state-changing command that waits by default must still make its machine-readable outcome clear when confirmation is delayed, partial, or interrupted, including when it must return `accepted` instead of `succeeded`.
- Validation failures must remain precise even when the invalid input involves nested commands, mutually dependent flags, or missing required context.
- Remote failures and local validation failures must not share an indistinguishable machine-readable outcome within the shared top-level result envelope.
- Discovery output must cover nested command families consistently so automation does not have to infer behavior for subcommands from top-level metadata.
- The contract must remain usable by machine consumers even when human-friendly logging or prose help is still present for operators.
- Structured output mode must not redefine process success or failure in a way that conflicts with the existing exit-code contract.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide one dedicated top-level machine-readable discovery command for the CLI that can be consumed without scraping human help text.
- **FR-002**: The dedicated top-level discovery command MUST describe top-level and nested command paths for the representative command families in scope.
- **FR-003**: The dedicated top-level discovery command MUST identify the supported flags and output modes for the commands it describes.
- **FR-004**: The dedicated top-level discovery command MUST identify whether a described command is read-only or state-changing.
- **FR-004a**: The dedicated top-level discovery command MUST identify whether each described command supports the shared machine result contract, supports it only partially, or does not support it yet.
- **FR-005**: The system MUST define one recommended machine-readable execution contract for supported commands in scope.
- **FR-005a**: The recommended machine-readable execution contract MUST use one shared top-level result envelope across the in-scope command families.
- **FR-005b**: The shared top-level result envelope MUST allow each command to include a command-specific payload without redefining the top-level outcome structure.
- **FR-006**: The shared top-level result envelope MUST standardize the explicit outcome categories `succeeded`, `accepted`, `invalid`, and `failed`.
- **FR-006a**: The execution contract MUST use `succeeded` for confirmed successful completion.
- **FR-006b**: The execution contract MUST use `accepted` for state-changing work that has been requested successfully but is not yet confirmed complete.
- **FR-006c**: The execution contract MUST use `invalid` for caller-correctable input or validation problems.
- **FR-006d**: The execution contract MUST use `failed` for remote, infrastructure, or other non-validation execution failures.
- **FR-007**: The execution contract MUST let machine consumers distinguish `invalid` outcomes from `failed` outcomes.
- **FR-008**: Validation failures MUST tell the caller what was wrong with the request and what must be corrected.
- **FR-008a**: Structured output mode MUST preserve the existing exit-code semantics as the primary process-level success or failure signal.
- **FR-008b**: The shared result envelope MUST provide detailed machine-readable outcome data that aligns with, and does not contradict, the process exit code.
- **FR-009**: The machine-readable contract MUST be documented as the canonical automation surface where structured output is supported.
- **FR-010**: The feature MUST document which representative command families support the recommended machine-readable contract and which output modes they expose.
- **FR-010a**: The feature MUST document which listed commands remain limited or unsupported for the shared machine result contract instead of excluding them from discovery.
- **FR-011**: The feature MUST cover representative commands from `get`, `run`, `expect`, `walk`, `deploy`, `delete`, and `cancel`.
- **FR-012**: The feature MUST preserve the existing human-oriented CLI taxonomy and command structure.
- **FR-013**: Existing human-friendly usage MUST remain supported after the machine contract is introduced.
- **FR-014**: The feature MUST provide automated coverage proving the machine-readable contract for at least one representative command from each in-scope command family.
- **FR-015**: The machine contract MUST align with existing project patterns so it can later serve as a stable foundation for an MCP adapter instead of a one-off parallel model.

### Key Entities *(include if feature involves data)*

- **Discovery Surface**: A dedicated top-level machine-readable CLI command that describes commands, nested paths, supported flags, output modes, and command mutability.
- **Command Capability Record**: The machine-readable description of one command or subcommand, including how it can be called and what output behavior it supports.
- **Contract Support Status**: The machine-readable indication that a command fully supports, only partially supports, or does not yet support the shared machine result contract.
- **Machine Result Contract**: A shared top-level result envelope that lets automation distinguish `succeeded`, `accepted`, `invalid`, and `failed` outcomes while carrying a command-specific payload.
- **Result Envelope**: The common machine-readable wrapper that exposes the outcome category consistently across command families.
- **Outcome Category**: One of `succeeded`, `accepted`, `invalid`, or `failed`, used by a machine consumer to decide whether to proceed, retry, wait, or correct input.
- **Process-Level Signal**: The existing command exit code that remains the authoritative coarse-grained success or failure signal for scripts and automation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A machine consumer can discover representative commands from every in-scope family through one dedicated top-level machine-readable command without scraping human help text.
- **SC-001a**: A machine consumer can tell from discovery output whether each listed command fully supports, partially supports, or does not yet support the shared machine result contract.
- **SC-002**: Automated coverage proves that at least one representative command from each in-scope family exposes the documented machine-readable contract.
- **SC-003**: Automated coverage proves machine consumers can distinguish `succeeded`, `accepted`, `invalid`, and `failed` outcomes for the covered commands through the shared top-level result envelope.
- **SC-003a**: Automated coverage proves the structured result envelope stays aligned with the existing exit-code contract for the covered success, accepted, invalid, and failed scenarios.
- **SC-004**: Documentation identifies the recommended automation contract and the supported output modes for the representative command families in scope.
- **SC-005**: Human operators can continue using the existing CLI structure and human-oriented help flows after the feature is introduced.

## Assumptions

- The feature is limited to strengthening machine-consumable command contracts and does not add new Camunda business operations.
- The current CLI command taxonomy remains the operator-facing structure and should be reused rather than redesigned.
- Representative command coverage is sufficient for the initial contract as long as the contract is defined consistently enough to extend to additional commands later.
- Commands that already support structured output can be aligned behind one documented automation contract without removing their existing human-friendly behavior.
- A shared top-level result envelope is the preferred contract shape because it keeps parsing logic stable across command families while preserving command-specific payloads.
- The four explicit outcome categories are sufficient for the initial contract because they separate confirmed completion, accepted work, caller-correctable input problems, and execution failures without introducing extra low-value state variants.
- The existing exit-code contract remains part of the automation surface and should stay authoritative at the process level even when detailed JSON output is requested.
- The contract should be designed so a future MCP adapter can reuse it directly rather than translating a separate automation-only model.
- Downstream implementation work for this feature must keep Conventional Commit formatting and append `#78` as the final token of every commit subject.
