# Feature Specification: Improve CLI Help Discovery

**Feature Branch**: `077-cli-help-discovery`  
**Created**: 2026-04-19  
**Status**: Draft  
**Input**: User description: "GitHub issue #77: docs(cli): improve ai-oriented help and command discoverability"

## GitHub Issue Traceability

- **Issue Number**: 77
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/77
- **Issue Title**: docs(cli): improve ai-oriented help and command discoverability

## Clarifications

### Session 2026-04-19

- Q: Which representative command set should this feature explicitly target in the spec? → A: Cover all commands.
- Q: When covering all commands, should the feature include parent/group commands as well as executable leaf commands? → A: Cover every command node, including parent/group commands and leaf commands.
- Q: How broad should the example refresh be across the full command tree? → A: Every command should get refreshed examples.
- Q: Should "all commands" include hidden/internal commands too, or only user-visible commands? → A: Cover all user-visible commands, but exclude hidden/internal commands.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Choose The Right Command Path (Priority: P1)

As an operator or AI-assisted caller, I want user-visible `c8volt` commands across the full public command tree, including parent/group commands and executable leaf commands, to explain when they should be used, whether they are read-only or state-changing, and which output mode is recommended for automation so that I can choose the correct invocation without reverse-engineering behavior from source code or trial and error.

**Why this priority**: Clear command selection is the core usability gap called out by the issue. If callers still cannot tell which command or output mode to use, the machine-readable discovery work from `#78` does not fully help day-to-day usage.

**Independent Test**: Review the help output across the public command tree and verify each user-visible parent/group command and executable leaf command tells the caller when to use it, whether it changes state, and whether `--json` is the recommended automation path where supported.

**Acceptance Scenarios**:

1. **Given** a caller is choosing among user-visible `c8volt` commands, **When** they read help for parent/group commands and executable leaf commands, **Then** they can tell what each command is for and whether it reads state or changes it.
2. **Given** an implemented command supports structured output, **When** the caller reads its help text, **Then** the help explicitly recommends `--json` for automation or AI-assisted usage where that is the preferred path.
3. **Given** one command does not offer the same automation affordances as another command, **When** the caller reads the help, **Then** the documented guidance reflects the real behavior instead of implying unsupported automation guarantees.

---

### User Story 2 - Understand Confirmation And Completion Semantics (Priority: P2)

As an automation caller, I want every applicable state-changing command help entry to explain default waiting behavior, the effect of `--no-wait`, and when `--auto-confirm` matters so that I can predict whether work is confirmed complete, merely accepted, or blocked on interaction.

**Why this priority**: Execution semantics are the most important safety detail once a caller has chosen a command. Automation cannot use the CLI safely if prompts, waiting, or follow-up verification steps remain implicit.

**Independent Test**: Inspect help and examples for all applicable state-changing commands and verify the documented invocation explains whether the command waits by default, how `--no-wait` changes the meaning, when `--auto-confirm` avoids prompts, and which follow-up command verifies the outcome.

**Acceptance Scenarios**:

1. **Given** an applicable state-changing command waits for confirmed completion by default, **When** the caller reads its help, **Then** the help says that the default behavior waits and names the condition being confirmed.
2. **Given** an applicable state-changing command supports `--no-wait`, **When** the caller reads its help or examples, **Then** the help explains that `--no-wait` returns before confirmed completion and changes the expected follow-up flow.
3. **Given** an applicable command may prompt for confirmation or paging continuation, **When** the caller reads its help, **Then** the help explains when `--auto-confirm` changes that interactive behavior.
4. **Given** an applicable write command completes or accepts work, **When** the caller reads the provided examples, **Then** the examples include a realistic follow-up inspection or verification command.

---

### User Story 3 - Keep Generated Docs In Sync With Help Metadata (Priority: P3)

As a maintainer, I want the improved command guidance and refreshed examples for every user-visible command to flow into generated CLI reference pages so that user-facing documentation stays aligned with actual command metadata instead of drifting through hand-edited docs.

**Why this priority**: The issue explicitly requires generated docs parity. Updating only in-command help would leave the CLI reference inconsistent and reintroduce documentation drift.

**Independent Test**: Regenerate the CLI docs from command metadata and verify the refreshed pages under `docs/cli/` reflect the improved descriptions and refreshed examples for every user-visible command in the public implemented command set without hand-editing generated output.

**Acceptance Scenarios**:

1. **Given** a maintainer updates command metadata across the user-visible CLI, **When** the CLI docs are regenerated, **Then** the refreshed reference pages include the same guidance and refreshed examples as the command help for every covered command.
2. **Given** a reviewer compares the generated docs to the live help text for representative commands, **When** they inspect both surfaces, **Then** the automation guidance is materially consistent across them.

### Edge Cases

- A parent/group command may need to explain discovery and routing guidance even when the executable semantics live on its child commands.
- A parent/group command may not naturally require shell examples today, but this feature still expects each command page to carry refreshed examples that help the caller understand intended usage or navigation.
- Hidden or internal commands may exist for shell integration, testing, or framework plumbing, and they should remain out of scope unless they are intentionally exposed as user-facing commands.
- A command may support `--json` but still be unsafe for unattended follow-up unless the help also explains waiting and verification semantics.
- Some commands may be read-only while sibling commands in the same family are state-changing, so the help must describe the behavior of each command rather than relying on family-level assumptions.
- A command may accept `--no-wait` without changing the request itself, so the help must clarify that the semantic change is in completion confirmation rather than in requested work.
- A command may prompt only in certain scenarios, such as destructive actions or paged operations, and the help must describe when `--auto-confirm` matters without implying prompts that never occur.
- Generated CLI docs must remain sourced from command metadata even when the best examples are lengthy or scenario-specific across many commands.
- If a help-text review reveals ambiguous or inconsistent runtime behavior, the issue must surface that gap explicitly in spec or follow-up work instead of hiding it behind polished prose.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The feature MUST improve `Short`, `Long`, and `Example` text across user-visible `c8volt` commands.
- **FR-002**: The feature MUST cover the full user-visible command tree, including parent/group commands and executable leaf commands, rather than only a representative subset.
- **FR-003**: Each command help entry MUST explain when the command should be used.
- **FR-004**: Each command help entry MUST make clear whether the command is read-only or state-changing.
- **FR-004a**: Parent/group command help entries MUST explain the command family purpose and help callers choose the appropriate child command path.
- **FR-004b**: Every covered command, including parent/group commands, MUST include refreshed examples aligned with its role in the command tree.
- **FR-004c**: Hidden or internal commands that are not part of the public user-facing CLI MUST remain out of scope for this feature unless they are deliberately surfaced as user-visible commands.
- **FR-005**: For commands that support the machine-readable contract introduced in `#78`, the help MUST explicitly recommend `--json` for automation where that is the preferred output mode.
- **FR-006**: Applicable state-changing commands MUST document whether they wait for confirmed completion by default.
- **FR-007**: Commands that support `--no-wait` MUST document how that option changes the meaning from confirmed completion to accepted or not-yet-confirmed work.
- **FR-008**: Commands that may prompt or pause for continuation MUST document when `--auto-confirm` matters for non-interactive usage.
- **FR-009**: Command examples MUST be refreshed for every covered command and MUST include realistic, copy-pasteable flows that work for human operators and AI-assisted callers.
- **FR-010**: Applicable state-changing command examples MUST include a realistic follow-up inspection or verification command where relevant.
- **FR-011**: Help text MUST surface important flag relationships that materially affect safe invocation, including mutually exclusive flags or flags that are required together where relevant.
- **FR-012**: The feature MUST align command help with the discovery and machine-result guidance already established in `#78` without redefining that contract.
- **FR-013**: The feature MUST preserve existing CLI terminology and style rather than introducing AI-only branding or a parallel command taxonomy.
- **FR-014**: The feature MUST refresh generated CLI reference pages under `docs/cli/` from command metadata rather than by hand-editing generated files.
- **FR-015**: Documentation updates MUST remain aligned with actual command behavior; if a behavior gap is discovered during help improvement, the work MUST record that inconsistency explicitly rather than masking it in prose.
- **FR-016**: The feature MUST avoid broad command-behavior changes except for small clarifications needed to keep help text truthful.

### Key Entities *(include if feature involves data)*

- **Covered Command**: Any user-visible parent/group command or executable leaf command whose help text and examples are reviewed and improved as part of this feature.
- **Parent/Group Command**: A non-leaf command that organizes a command family and helps callers navigate toward the correct executable subcommand.
- **Hidden/Internal Command**: A non-public command used for shell integration, testing, or framework plumbing that is intentionally excluded from this feature's help-refresh scope.
- **Command Help Metadata**: The Cobra `Short`, `Long`, and `Example` content that defines how the CLI explains command usage.
- **Command Example Set**: The refreshed example block attached to each command that demonstrates intended usage for that command's role in the tree.
- **Automation Guidance**: The command-level explanation of recommended output mode, mutability, waiting behavior, confirmation behavior, and safe follow-up actions for automation and AI-assisted callers.
- **Verification Flow Example**: A realistic example sequence that shows a caller how to inspect the result of a state-changing command after execution or acceptance.
- **Generated CLI Reference Page**: The `docs/cli/` page produced from command metadata and expected to match live help behavior.
- **Behavior Gap Record**: A documented mismatch between intended help guidance and actual command behavior that requires explicit follow-up instead of silent omission.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Across the user-visible command tree, each covered parent/group command and executable leaf command help entry tells the caller when the command should be used, whether it changes state, and whether `--json` is the recommended automation path where supported.
- **SC-002**: For each applicable state-changing command in scope, the help explains default waiting behavior, the effect of `--no-wait` where supported, and when `--auto-confirm` matters where relevant.
- **SC-003**: Every covered command exposes refreshed examples appropriate to that command, and applicable state-changing command examples show at least one realistic verification or follow-up inspection step where relevant.
- **SC-004**: Regenerated pages under `docs/cli/` reflect the same improved guidance and refreshed examples as the underlying command metadata across the covered user-visible command set.
- **SC-005**: Reviewers can compare the updated help text to actual runtime behavior for covered commands without finding undocumented contradictions in the covered guidance areas.

## Assumptions

- The feature is expected to cover all user-visible commands, including parent/group commands and executable leaf commands, even if some command families require lighter-touch edits than others.
- Every covered command is expected to receive refreshed examples, even when the example for a parent/group command is primarily navigational or illustrative rather than operationally complex.
- Hidden or internal commands are intentionally excluded unless they are promoted into the public CLI surface.
- The machine-readable discovery and result-contract work from `#78` remains the canonical automation foundation and should be referenced rather than replaced.
- Existing Cobra metadata fields are sufficient to carry the required help and example improvements for this feature.
- Generated CLI reference pages under `docs/cli/` are still sourced from command metadata via the repository's documented generation flow.
- Small wording or example clarifications may be needed to keep command help truthful, but broad behavioral redesign is out of scope.
- When a covered command already behaves correctly but documents that behavior poorly, improving help text and generated docs is sufficient for this issue.
- If a runtime ambiguity is uncovered during implementation, the correct response is to document or track the gap explicitly rather than infer behavior silently.
- Downstream implementation work for this feature must keep Conventional Commit formatting and append `#77` as the final token of every commit subject.
