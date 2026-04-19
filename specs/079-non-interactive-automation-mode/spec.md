# Feature Specification: Define Non-Interactive Automation Mode

**Feature Branch**: `079-non-interactive-automation-mode`  
**Created**: 2026-04-19  
**Status**: Draft  
**Input**: User description: "GitHub issue #79: feat(cli): define an explicit non-interactive automation mode for ai agents"

## GitHub Issue Traceability

- **Issue Number**: 79
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/79
- **Issue Title**: feat(cli): define an explicit non-interactive automation mode for ai agents

## Clarifications

### Session 2026-04-19

- Q: How should the non-interactive execution contract be expressed? → A: Add one new dedicated flag for automation mode.
- Q: How should commands behave if they do not define supported automation-mode behavior? → A: Fail explicitly unless the command defines supported automation-mode behavior.
- Q: How should logs and progress output behave in automation mode? → A: Reserve stdout for machine-readable results and send logs or progress to stderr or suppress them.
- Q: How should confirmation prompts behave for supported commands in automation mode? → A: The automation flag implicitly auto-confirms supported prompts.
- Q: How should automation mode interact with `--no-wait`? → A: In automation mode, `--no-wait` should return an explicit accepted or not-yet-complete outcome.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run Commands Safely Without Prompts (Priority: P1)

As an AI agent or CI caller, I want one clear non-interactive execution mode exposed through one dedicated automation flag so that I can run read and write commands without hanging on confirmation prompts, paging continuation, or other interactive waits.

**Why this priority**: Safe non-interactive execution is the core value of the feature. Without it, automation still depends on fragile command-specific knowledge and can stall at runtime.

**Independent Test**: Run representative state-changing and paged commands in the supported automation mode and verify they either proceed without blocking or fail immediately with an explicit actionable message instead of waiting for user input.

**Acceptance Scenarios**:

1. **Given** an automation caller invokes a representative state-changing command in the supported automation mode, **When** the command would normally request confirmation, **Then** it does not block on an interactive prompt.
2. **Given** an automation caller invokes a representative paged or continuation-oriented command in the supported automation mode, **When** additional interactive input would normally be required, **Then** the command follows the documented non-interactive behavior without waiting for terminal input.
3. **Given** a command cannot safely continue in the supported automation mode, **When** the caller runs it non-interactively, **Then** the command fails immediately with an explicit and actionable explanation.
4. **Given** a command has not defined supported automation-mode behavior, **When** the caller uses the automation flag, **Then** the command fails explicitly instead of falling back to interactive behavior or guessing a command-specific non-interactive default.
5. **Given** a supported command would normally ask for confirmation, **When** the caller uses the automation flag, **Then** the command treats the confirmation as accepted without requiring a separate confirmation flag.

---

### User Story 2 - Combine Automation Mode With Machine Output (Priority: P2)

As an automation caller, I want the non-interactive mode to work predictably with the machine-readable CLI contract introduced in `#78` so that I can combine safe execution with structured discovery and structured results.

**Why this priority**: Non-interactive execution is only useful for automation if the resulting output and completion semantics remain predictable enough for machine consumers to interpret safely.

**Independent Test**: Run representative read and write commands using the documented automation invocation pattern together with structured output and asynchronous execution options where relevant, then verify the output remains usable and the completion semantics remain clear.

**Acceptance Scenarios**:

1. **Given** an automation caller requests structured output while using the supported automation mode, **When** a representative command completes, **Then** the structured output remains usable without relying on human-oriented interaction.
2. **Given** a representative command supports explicitly asynchronous execution, **When** the caller combines `--no-wait` with the automation flag, **Then** the command returns an explicit accepted or not-yet-complete outcome rather than reporting confirmed completion.
3. **Given** an automation caller inspects the guidance for the supported automation mode, **When** it reads the documentation, **Then** it can tell how to combine the mode with `--json`, `--no-wait`, and related existing flags where relevant.
4. **Given** an automation caller requests machine-readable output in automation mode, **When** the command emits logs or progress information, **Then** the machine-readable result remains isolated on stdout while logs or progress are redirected away from stdout or omitted.

---

### User Story 3 - Preserve Human CLI Workflows (Priority: P3)

As a maintainer, I want the automation mode to layer onto the existing CLI instead of replacing it so that interactive human workflows stay available while automation gets one explicit contract.

**Why this priority**: The issue explicitly avoids a broad CLI redesign. The feature should improve determinism for automation without creating a separate user experience that diverges from established command patterns.

**Independent Test**: Compare representative commands run with and without the supported automation mode and verify the automation guidance becomes explicit while current human-friendly behavior remains available outside that mode.

**Acceptance Scenarios**:

1. **Given** a human operator uses a representative command without the automation mode, **When** the command reaches an interactive step, **Then** the existing human-oriented behavior remains available.
2. **Given** maintainers review the final design, **When** they compare it to the current CLI patterns and the `#78` machine contract, **Then** they can see the automation behavior is an intentional extension rather than a parallel UX.

### Edge Cases

- A representative command may already be automation-friendly through an existing flag combination; the feature must still define one canonical non-interactive contract instead of leaving equivalent patterns undocumented.
- A command may suppress confirmation successfully but still emit progress or informational output that confuses machine consumers; the documented automation mode must make output expectations explicit enough for reliable use.
- A command may support structured output but still require a prompt or continuation step; the feature must define how that conflict is handled in non-interactive mode.
- A command may support asynchronous execution, so the automation contract must distinguish explicit accepted or not-yet-complete outcomes from confirmed completion clearly enough for safe follow-up automation.
- Some commands may not be safe to run non-interactively; those cases must fail explicitly and actionably instead of hanging or silently choosing an unsafe default.
- A command may not yet define automation-mode semantics at all; in that case the automation contract must reject the invocation explicitly rather than inferring behavior.
- Human-oriented logs or progress updates must not pollute stdout when automation callers depend on machine-readable results there.
- Human interactive behavior must remain available outside the automation mode so existing workflows are not degraded.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST define one clear, documented, and testable non-interactive execution contract for `c8volt` automation callers.
- **FR-002**: The non-interactive execution contract MUST build on the machine-readable CLI contract introduced in `#78` rather than defining a separate parallel automation UX.
- **FR-003**: The feature MUST express the non-interactive execution contract as one new dedicated automation flag.
- **FR-003a**: The dedicated automation flag MUST be documented as the canonical way for automation callers to opt into non-interactive execution.
- **FR-004**: Representative state-changing commands MUST not block on confirmation prompts when run in the supported non-interactive mode.
- **FR-004a**: For commands that explicitly support automation mode, the automation flag MUST implicitly auto-confirm prompts that are part of the documented non-interactive flow.
- **FR-005**: Representative commands that normally require paging continuation or similar interactive acknowledgement MUST behave predictably without waiting for terminal input in the supported non-interactive mode.
- **FR-006**: If a command cannot safely proceed in the supported non-interactive mode, the system MUST fail explicitly and provide an actionable explanation to the caller.
- **FR-006a**: If a command has not defined supported automation-mode behavior, the system MUST reject automation-mode execution explicitly instead of falling back to interactive behavior or inferring a best-effort non-interactive path.
- **FR-007**: The feature MUST document how the supported non-interactive mode relates to `--json`, `--no-wait`, `--auto-confirm`, and relevant output-shaping options.
- **FR-008**: Structured output on supported commands MUST remain usable when the supported non-interactive mode is used.
- **FR-008a**: When automation mode is used with machine-readable output, stdout MUST be reserved for the machine-readable result.
- **FR-008b**: Logs, progress updates, and other human-oriented diagnostic output in automation mode MUST be sent to stderr or be suppressed.
- **FR-009**: The feature MUST make the relationship between confirmed completion, accepted work, and explicitly asynchronous execution clear when the supported non-interactive mode is used.
- **FR-009a**: When a supported command is run with the automation flag and `--no-wait`, the command MUST return an explicit accepted or not-yet-complete outcome instead of implying confirmed completion.
- **FR-010**: The feature MUST define the intended non-interactive behavior for representative read commands and representative write commands.
- **FR-011**: The feature MUST include targeted automated coverage for representative prompting and state-changing commands proving they do not block in the supported non-interactive mode.
- **FR-012**: The feature MUST reuse current project patterns around confirmation, waiting, error handling, and result envelopes where practical.
- **FR-013**: Existing human-friendly interactive behavior MUST remain available outside the supported non-interactive mode.
- **FR-014**: Documentation MUST describe the recommended invocation pattern that automation callers should use for safe non-interactive execution.
- **FR-015**: The documented automation contract MUST remain compatible with future MCP-style use that depends on deterministic non-interactive semantics.

### Key Entities *(include if feature involves data)*

- **Automation Mode Contract**: The single documented non-interactive execution pattern that automation callers use to run `c8volt` safely and predictably.
- **Automation Flag**: The dedicated command-line flag that opts a caller into the canonical non-interactive automation mode.
- **Automation Caller**: An AI agent, CI job, script, or other machine consumer that invokes CLI commands without human supervision.
- **Implicit Confirmation Rule**: The rule that supported automation-mode commands treat confirmation prompts as accepted when the automation flag is present.
- **Interactive Gate**: Any prompt, confirmation step, pager continuation, or terminal input dependency that can block unattended execution.
- **Structured Output Path**: The machine-readable result flow introduced in `#78` that automation callers combine with the non-interactive execution contract.
- **Automation Output Channel Rule**: The requirement that machine-readable results remain on stdout while human-oriented logs and progress are redirected away from stdout or suppressed.
- **Execution Outcome State**: The documented interpretation of confirmed completion, accepted work, and explicitly asynchronous execution when the automation mode is used.
- **Accepted Outcome**: The explicit machine-readable indication that requested work was accepted but is not yet confirmed complete, including automation-mode uses of `--no-wait`.
- **Actionable Failure**: A non-interactive failure response that tells the caller the command could not proceed safely and what adjustment is required.
- **Unsupported Automation Invocation**: An automation-mode request for a command that has not defined supported non-interactive semantics and therefore must be rejected explicitly.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Documentation identifies one dedicated automation flag as the recommended non-interactive invocation pattern for automation callers instead of relying on scattered command-specific conventions.
- **SC-002**: Automated coverage proves representative prompting and state-changing commands do not block on interactive input in the supported non-interactive mode.
- **SC-002a**: Automated coverage proves supported automation-mode commands do not require a separate confirmation flag to bypass documented prompts.
- **SC-003**: Automated coverage proves structured output remains usable when representative supported commands run in the supported non-interactive mode.
- **SC-003a**: Automated coverage proves representative commands keep machine-readable results on stdout without mixing human-oriented logs or progress into stdout in automation mode.
- **SC-004a**: Automated coverage proves supported commands using automation mode plus `--no-wait` return an explicit accepted or not-yet-complete outcome instead of a confirmed-completion result.
- **SC-004**: Documentation makes the relationship between confirmed completion, accepted work, and explicitly asynchronous execution clear enough that automation callers can choose the correct follow-up action for representative commands.
- **SC-005**: Human operators can still use representative commands with their existing interactive behavior when they do not opt into the automation mode.

## Assumptions

- The feature focuses on defining one explicit automation contract on top of existing CLI behavior rather than redesigning the command taxonomy.
- The automation contract will be surfaced through one dedicated flag rather than only through a documented bundle of existing flags.
- Representative read and write commands are sufficient to establish the non-interactive contract as long as the documented rules can extend to additional commands later.
- Existing flags and output conventions should be reused where practical instead of introducing new parallel concepts without clear value.
- Structured output from `#78` remains the preferred machine-readable result surface for commands that already support it.
- Automation callers should be able to parse stdout deterministically without filtering out human-oriented logs or progress lines first.
- Automation mode does not make supported commands asynchronous by default; explicit asynchronous behavior is still selected through `--no-wait`.
- The feature may document a shared invocation pattern even if some individual commands still require explicit failure behavior when non-interactive execution would be unsafe.
- Commands that have not opted into supported automation-mode semantics should fail explicitly under the automation flag rather than inheriting interactive defaults.
- Supported automation-mode commands should treat the automation flag itself as sufficient confirmation for documented prompt flows.
- Downstream implementation work for this feature must keep Conventional Commit formatting and append `#79` as the final token of every commit subject.
