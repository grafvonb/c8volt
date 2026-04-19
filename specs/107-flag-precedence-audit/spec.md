# Feature Specification: Audit and Fix CLI Config Precedence

**Feature Branch**: `107-flag-precedence-audit`  
**Created**: 2026-04-16  
**Status**: Draft  
**Input**: User description: "GitHub issue #107: bug(config): audit and fix flag precedence across all commands and config sources"

## GitHub Issue Traceability

- **Issue Number**: 107
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/107
- **Issue Title**: bug(config): audit and fix flag precedence across all commands and config sources

## Clarifications

### Session 2026-04-16

- Q: Should feature completion require exhaustive coverage of every config-backed command path, or only shared logic plus representative commands? → A: Audit and validate every config-backed command and flag path in the CLI before considering the feature complete.
- Q: When precedence inputs remain ambiguous or conflicting after the shared rules are applied, should the CLI preserve legacy behavior, fail explicitly, or choose a best-effort winner? → A: Introduce explicit validation errors for ambiguous or conflicting precedence cases whenever the correct winner cannot be determined safely.
- Q: Should the spec name a shared baseline set of critical settings that every command audit must verify everywhere they appear, or leave cross-command coverage fully open-ended? → A: Name a shared baseline set of critical settings that must be checked everywhere they appear, then add command-specific checks as needed.
- Q: Which settings should belong to the shared critical baseline for cross-command precedence verification? → A: Include tenant, active profile selection, API base URLs, auth mode, and auth credentials/scopes in the shared baseline.
- Q: Should the precedence contract be documented only in internal shared sources, or also in user-facing CLI/config documentation? → A: Document the precedence contract in shared code/tests and in the relevant user-facing CLI/config documentation.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Resolve Effective Values Consistently (Priority: P1)

As a CLI operator, I want every config-backed flag to resolve from the same precedence order so that the value used by a command always matches the source I explicitly intended to win.

**Why this priority**: Incorrect precedence changes command behavior in ways that are hard to predict and can affect tenant selection, authentication, endpoint selection, and other operationally important settings.

**Independent Test**: Run representative commands that combine CLI flags, environment variables, active profile values, base config values, and built-in defaults, then verify that the effective value always follows the documented precedence contract.

**Acceptance Scenarios**:

1. **Given** a command has the same setting defined in a CLI flag, environment variable, active profile, and base config, **When** the command executes, **Then** the explicit CLI flag value is used.
2. **Given** a command has the same setting defined in an environment variable, active profile, and base config but no explicit CLI flag, **When** the command executes, **Then** the environment variable value is used.
3. **Given** a command has the same setting defined in an active profile and base config but no explicit CLI flag or environment variable, **When** the command executes, **Then** the active profile value is used.
4. **Given** a command has no explicit override for a config-backed setting, **When** the setting exists only in base config or built-in defaults, **Then** the command uses the highest remaining available source in the documented order.

---

### User Story 2 - Preserve Correctness Across Command Types and Edge Cases (Priority: P2)

As a CLI maintainer, I want precedence handling to stay consistent for root persistent flags, command-local flags, profile-aware settings, and edge-case values so that no command path bypasses the shared configuration contract.

**Why this priority**: Fixing only one command would leave the broader bug class in place and allow future regressions in commands that read configuration differently.

**Independent Test**: Exercise representative root-level and nested commands with mixed-source inputs, zero values, empty strings, and invalid combinations, then confirm they either resolve correctly or fail through explicit validation instead of ad hoc overrides.

**Acceptance Scenarios**:

1. **Given** a root persistent flag and a command-local flag both participate in effective configuration for one command execution, **When** the command resolves its settings, **Then** each value follows the same precedence contract without one flag type bypassing the other.
2. **Given** a setting is intentionally provided as an empty string, zero value, or false-like value from a higher-precedence source, **When** the command executes, **Then** that explicit higher-precedence value is honored instead of being silently replaced by a lower-precedence source.
3. **Given** a command includes special override behavior or ambiguous combinations of configuration inputs, **When** the command executes, **Then** it either follows the shared precedence model or fails through explicit validation rather than hidden fallback logic.
4. **Given** a precedence conflict cannot be resolved safely through the shared rules, **When** the command executes, **Then** it stops with an explicit validation error instead of preserving legacy ambiguity or choosing a best-effort winner.

---

### User Story 3 - Trust the Contract Through Shared Coverage and Documentation (Priority: P3)

As a contributor, I want the precedence contract centralized in tests and supporting documentation so that future changes can be reviewed against a stable, shared rule instead of rediscovering precedence behavior command by command.

**Why this priority**: A configuration contract is only durable if maintainers can verify it quickly and understand where the authoritative rules live.

**Independent Test**: Review the updated specification, targeted regression tests, and relevant documentation or code comments, then confirm the precedence order is stated once, applied broadly, and covered by mixed-source test cases across representative commands.

**Acceptance Scenarios**:

1. **Given** a maintainer reviews the precedence implementation, **When** they look for the intended resolution order, **Then** they can find one authoritative shared contract for `flag > env > profile > base config > default`.
2. **Given** representative commands across the CLI are covered by regression tests, **When** those tests run with mixed-source combinations, **Then** the tests demonstrate that precedence behavior remains consistent beyond the original failing command path.
3. **Given** a config-backed setting belongs to the shared critical baseline and appears in multiple commands, **When** the audit and regression coverage are reviewed, **Then** that setting is checked consistently everywhere it appears in addition to any command-specific checks.
4. **Given** a command reads tenant, active profile selection, API base URLs, auth mode, or auth credentials/scopes, **When** the audit and regression coverage are executed, **Then** each of those settings is verified against the shared precedence contract wherever it appears.
5. **Given** an operator or maintainer needs to understand precedence behavior, **When** they consult the shared implementation guidance or the relevant CLI/config documentation, **Then** both sources describe the same precedence contract and override expectations.

### Edge Cases

- An explicit higher-precedence zero value, empty string, or false-like value must not be treated as "unset" and replaced by a lower-precedence source.
- Commands that use both root persistent flags and command-local flags must not resolve the same setting through competing precedence rules.
- Profile-derived values must not override explicit CLI flags or environment variables, even when profiles are applied late in command construction.
- Commands that construct service or helper configuration indirectly must still respect the same effective value as direct command reads.
- Invalid or ambiguous combinations of flags and overrides must fail clearly when required instead of silently picking an unexpected source.
- Ambiguous or conflicting precedence cases that cannot be resolved safely must fail explicitly instead of preserving legacy ambiguous behavior or silently choosing a winner.
- The audit must cover the whole CLI's config-backed surface, not only the originally observed `--tenant` behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST apply one consistent precedence contract for all config-backed flags across the CLI.
- **FR-002**: The precedence contract MUST be `explicit CLI flag > environment variable > active profile value > base config value > built-in default`.
- **FR-003**: The system MUST apply the same precedence contract to root persistent flags and command-local flags.
- **FR-004**: The system MUST apply the same precedence contract to values that flow through shared command helpers, service construction, and other indirect command setup paths.
- **FR-005**: The system MUST ensure active profile values never override explicit CLI flags or environment variables.
- **FR-006**: The system MUST preserve explicit higher-precedence zero values, empty strings, and false-like values instead of treating them as missing input.
- **FR-007**: The system MUST identify and remove or normalize command-specific override behavior that bypasses the shared precedence contract.
- **FR-008**: The system MUST define and enforce explicit validation for invalid or ambiguous override combinations where silent resolution would be misleading.
- **FR-008a**: When precedence inputs remain ambiguous or conflicting after the shared rules are applied, the system MUST fail with an explicit validation error instead of preserving legacy behavior or choosing a best-effort winner.
- **FR-009**: The system MUST audit every config-backed command and flag path across the CLI, not only the originally reported failing command.
- **FR-009a**: The system MUST define a shared baseline set of critical config-backed settings whose precedence behavior is verified everywhere those settings appear across the CLI.
- **FR-009b**: The shared critical baseline MUST include tenant, active profile selection, API base URLs, auth mode, and auth credentials/scopes.
- **FR-010**: Regression coverage MUST include mixed-source combinations for flag plus env plus profile plus base config, env plus profile plus base config, profile plus base config, and explicit empty or zero-value edge cases.
- **FR-011**: Regression coverage MUST include representative commands that combine root persistent flags and command-local flags in one execution path.
- **FR-012**: The system MUST centralize or clearly document the intended precedence behavior in one shared repository location so future changes can be checked against it.
- **FR-012a**: The system MUST document the precedence contract in the relevant user-facing CLI and configuration documentation wherever operators need to understand flag, environment, profile, and config override behavior.
- **FR-013**: The feature MUST preserve existing externally observable command behavior except where the current behavior violates the defined precedence contract.

### Key Entities *(include if feature involves data)*

- **Config-Backed Setting**: Any user-facing command setting whose effective value may come from a CLI flag, environment variable, active profile, base config, or built-in default.
- **Critical Baseline Setting**: A high-risk config-backed setting that must be verified everywhere it appears across commands, specifically tenant, active profile selection, API base URLs, auth mode, and auth credentials/scopes.
- **Precedence Source**: One of the ordered origins that may supply a setting value during command execution.
- **Effective Command Configuration**: The final resolved values a command uses after applying the shared precedence contract and any required validation.
- **Override Validation Rule**: A defined rule that rejects ambiguous or invalid combinations instead of allowing silent mis-resolution.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every config-backed command and flag path in scope for the CLI audit is reviewed against the shared precedence contract, and automated regression coverage demonstrates `flag > env > profile > base config > default` behavior for the affected command paths.
- **SC-002**: At least one representative command path each for root persistent flags, command-local flags, and mixed root-plus-local resolution is covered by regression tests that confirm the same precedence order.
- **SC-003**: Mixed-source regression tests cover explicit CLI flag plus env plus profile plus base config, env plus profile plus base config, profile plus base config, and empty or zero-value edge cases without contradictory outcomes.
- **SC-004**: Commands that previously bypassed the shared precedence model either resolve correctly through shared logic or fail through explicit validation instead of silent fallback behavior.
- **SC-004a**: Ambiguous or conflicting precedence cases that cannot be resolved safely produce explicit validation failures in covered command paths instead of silently choosing a winner.
- **SC-004b**: Each setting in the shared critical baseline is explicitly verified everywhere it appears across the audited command surface, with command-specific checks added where behavior differs.
- **SC-004c**: Tenant, active profile selection, API base URLs, auth mode, and auth credentials/scopes are all verified everywhere they appear across the audited command surface.
- **SC-005**: Maintainers can identify the precedence contract from one shared authoritative location during review without needing to infer it from multiple command-specific implementations.
- **SC-005a**: Operators can identify the same precedence and override contract from the relevant user-facing CLI/config documentation without relying on source-code inspection.

## Assumptions

- The repository already has enough representative commands and test seams to validate precedence behavior across the broader CLI without redesigning the full command surface.
- The feature is meant to correct incorrect precedence behavior, not to redefine the intended precedence order.
- User-facing documentation updates are only needed where the precedence contract or override behavior is visible or likely to affect CLI users.
- Relevant user-facing CLI/config documentation should be updated alongside the shared internal contract so operator guidance stays aligned with implementation behavior.
- The audit should stay bounded to config-backed settings and related validation behavior rather than introducing unrelated command UX changes.
- Exhaustive feature completion means reviewing the entire config-backed command surface rather than stopping after fixing a representative subset.
- When the shared precedence contract cannot determine a safe winner, explicit validation failure is preferable to preserving legacy ambiguous behavior.
- A shared baseline of critical settings should anchor cross-command audit coverage, with additional checks layered on where individual commands introduce extra precedence behavior.
- The highest-risk shared settings for this feature are tenant, active profile selection, API base URLs, auth mode, and auth credentials/scopes because precedence bugs there can misroute commands or break authentication.
