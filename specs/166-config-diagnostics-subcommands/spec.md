# Feature Specification: Config Diagnostics Subcommands

**Feature Branch**: `166-config-diagnostics-subcommands`  
**Created**: 2026-05-04  
**Status**: Draft  
**Input**: User description: "GitHub issue #166: feat(config): split config diagnostics into dedicated subcommands"

## GitHub Issue Traceability

- **Issue Number**: 166
- **Issue URL**: https://github.com/grafvonb/c8volt/issues/166
- **Issue Title**: feat(config): split config diagnostics into dedicated subcommands

## Clarifications

### Session 2026-05-04

- No critical ambiguities detected worth formal clarification. The GitHub issue defines the command split, backward-compatible flags, validation/template behavior, test-connection logging, connection failure semantics, version comparison rules, documentation updates, and required test coverage.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Preserve Config Show Compatibility (Priority: P1)

As a CLI user who already relies on `c8volt config show`, I want the command and its compatibility flags to keep working so that existing interactive workflows and scripts do not break while dedicated diagnostics commands are added.

**Why this priority**: Backward compatibility is the foundation for safely splitting diagnostics into clearer subcommands.

**Independent Test**: Run `c8volt config show`, `c8volt config show --validate`, and `c8volt config show --template` with representative valid and invalid configuration inputs, then verify each command keeps its established output and exit behavior.

**Acceptance Scenarios**:

1. **Given** an effective configuration with sensitive values, **When** the user runs `c8volt config show`, **Then** the command prints the effective configuration with sensitive values sanitized.
2. **Given** a valid effective configuration, **When** the user runs `c8volt config show --validate`, **Then** validation succeeds with the same success behavior as before this feature.
3. **Given** the user runs `c8volt config show --template`, **When** template rendering succeeds, **Then** the blank configuration template matches the previous compatibility output.

---

### User Story 2 - Validate Configuration Directly (Priority: P2)

As a CLI user or automation author, I want `c8volt config validate` to validate the loaded effective configuration directly so that validation is discoverable without using a `show` compatibility flag.

**Why this priority**: Validation is a standalone diagnostic workflow and should be available as its own command before deeper connection checks depend on it.

**Independent Test**: Run `c8volt config validate` with valid and invalid configuration sources, then verify valid configuration exits successfully and invalid configuration exits through the standard error handling path.

**Acceptance Scenarios**:

1. **Given** the effective configuration is valid, **When** the user runs `c8volt config validate`, **Then** the command exits with code `0`.
2. **Given** the effective configuration is invalid, **When** the user runs `c8volt config validate`, **Then** the command returns a non-zero exit code through the standard error path.
3. **Given** both `config validate` and `config show --validate` validate the same effective configuration, **When** they are run with equivalent inputs, **Then** they report equivalent validation outcomes.

---

### User Story 3 - Render Configuration Template Directly (Priority: P3)

As a CLI user creating a new configuration file, I want `c8volt config template` to print the blank configuration template so that template generation is discoverable as its own command.

**Why this priority**: Template rendering is a separate workflow from showing the active configuration and can be delivered independently once compatibility is preserved.

**Independent Test**: Run `c8volt config template` and `c8volt config show --template`, then verify both commands produce the same template and the same success or failure semantics.

**Acceptance Scenarios**:

1. **Given** template rendering succeeds, **When** the user runs `c8volt config template`, **Then** the command prints the blank configuration template and exits with code `0`.
2. **Given** template rendering fails, **When** the user runs `c8volt config template`, **Then** the command returns a non-zero exit code through the standard error path.
3. **Given** the same runtime environment, **When** `config template` and `config show --template` are run, **Then** their rendered template output is equivalent.

---

### User Story 4 - Test Configured Camunda Connection (Priority: P4)

As a Camunda operator, I want `c8volt config test-connection` to validate configuration and test the configured Camunda connection so that I can confirm both local configuration and remote reachability in one diagnostic command.

**Why this priority**: This is the highest-value new diagnostic behavior, but it depends on the shared validation behavior being defined first.

**Independent Test**: Run `c8volt config test-connection` against fixtures for valid configuration with successful topology, invalid configuration, remote connection failure, version match, patch-only version difference, and major/minor mismatch.

**Acceptance Scenarios**:

1. **Given** a valid configuration and reachable Camunda cluster, **When** the user runs `c8volt config test-connection`, **Then** the command logs the loaded config source at `INFO` level, logs connection success at `INFO` level, prints the human-readable topology output, and exits with code `0`.
2. **Given** no config file was loaded and defaults or environment values supplied the effective configuration, **When** the user runs `c8volt config test-connection`, **Then** the `INFO` log clearly states that no config file was loaded.
3. **Given** the effective configuration is invalid, **When** the user runs `c8volt config test-connection`, **Then** the command fails before attempting the remote connection.
4. **Given** the remote connection fails, **When** the user runs `c8volt config test-connection`, **Then** the command logs through the standard error path and returns a non-zero exit code.
5. **Given** the gateway version differs from the configured Camunda version only by patch version, **When** the connection succeeds, **Then** no version mismatch warning is logged.
6. **Given** the gateway version and configured Camunda version differ by major or minor version, **When** the connection succeeds, **Then** a `WARNING` is logged and the command still exits with code `0`.

---

### User Story 5 - Discover The Split Commands In Help And Docs (Priority: P5)

As a CLI user browsing help or documentation, I want the new config diagnostics commands to be discoverable while legacy flags remain documented as compatibility shortcuts so that I can choose the clearer command shape without losing the old contract.

**Why this priority**: The new command shape is only useful if users can find it, and the compatibility surface should be explicit rather than accidental.

**Independent Test**: Inspect command help, examples, README references, and generated CLI documentation for the config command surface, then verify new subcommands are documented and compatibility flags remain supported.

**Acceptance Scenarios**:

1. **Given** a user views `c8volt config --help`, **When** the command list is shown, **Then** `show`, `validate`, `template`, and `test-connection` are discoverable.
2. **Given** a user views help for `c8volt config show`, **When** compatibility flags are listed, **Then** `--validate` and `--template` are documented as supported legacy paths or compatibility shortcuts.
3. **Given** generated CLI documentation is refreshed, **When** the config command pages are inspected, **Then** they include the new subcommands and preserve the compatibility flag contract.

### Edge Cases

- `config show` must continue sanitizing sensitive values in the effective configuration.
- `config show --validate` must not print the sanitized configuration when its established behavior is validation-only.
- `config show --template` must not load or print the active effective configuration when its established behavior is template-only.
- `config test-connection` must not attempt a remote connection after configuration validation fails.
- `config test-connection` must log the loaded config source at `INFO` level even when `--debug` is absent.
- If no config file was loaded, `config test-connection` must state that clearly instead of implying a path exists.
- Connection failure must use the standard error path and must not be reported as a validation-only failure.
- Version comparison must normalize configured and gateway versions to major/minor before deciding whether to warn.
- Patch-only version differences must not warn.
- Major/minor version mismatch warnings must not change an otherwise successful exit code.
- JSON or machine-readable output behavior for existing commands must not regress if already supported by the affected command paths.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST preserve the current behavior of `c8volt config show`.
- **FR-002**: `c8volt config show` MUST continue to show the effective configuration with sensitive values sanitized.
- **FR-003**: `c8volt config show --validate` MUST continue to validate the effective configuration through the compatibility flag path.
- **FR-004**: `c8volt config show --template` MUST continue to print the blank configuration template through the compatibility flag path.
- **FR-005**: The system MUST expose `c8volt config validate` as a dedicated subcommand.
- **FR-006**: `c8volt config validate` MUST load the effective c8volt configuration using the existing configuration loading flow.
- **FR-007**: `c8volt config validate` MUST use the same validation behavior as `c8volt config show --validate`.
- **FR-008**: `c8volt config validate` MUST exit with code `0` when the effective configuration is valid.
- **FR-009**: `c8volt config validate` MUST return a non-zero exit code through the standard error path when the effective configuration is invalid.
- **FR-010**: The system MUST expose `c8volt config template` as a dedicated subcommand.
- **FR-011**: `c8volt config template` MUST print the same blank configuration template as `c8volt config show --template`.
- **FR-012**: `c8volt config template` MUST exit with code `0` when template rendering succeeds.
- **FR-013**: `c8volt config template` MUST return a non-zero exit code through the standard error path when template rendering fails.
- **FR-014**: The system MUST expose `c8volt config test-connection` as a dedicated subcommand.
- **FR-015**: `c8volt config test-connection` MUST load the effective c8volt configuration.
- **FR-016**: `c8volt config test-connection` MUST log at `INFO` level which config path was loaded, visible without requiring `--debug`.
- **FR-017**: If no config file was loaded, `c8volt config test-connection` MUST log at `INFO` level that configuration came from defaults, environment, or other non-file sources.
- **FR-018**: `c8volt config test-connection` MUST validate the effective configuration before attempting the remote connection.
- **FR-019**: `c8volt config test-connection` MUST stop before attempting the remote connection when validation fails.
- **FR-020**: `c8volt config test-connection` MUST test the configured Camunda connection using the same cluster topology capability used by `c8volt get cluster topology`.
- **FR-021**: When the connection succeeds, `c8volt config test-connection` MUST log a connection success message at `INFO` level.
- **FR-022**: When the connection succeeds, `c8volt config test-connection` MUST print the same human-readable topology output as `c8volt get cluster topology`.
- **FR-023**: When the connection succeeds, `c8volt config test-connection` MUST exit with code `0`.
- **FR-024**: When the connection fails, `c8volt config test-connection` MUST log through the standard `ERROR` path and return a non-zero exit code.
- **FR-025**: When the connection succeeds, the system MUST compare configured Camunda version and gateway version by major and minor version only.
- **FR-026**: Patch-only version differences MUST NOT log a version mismatch warning.
- **FR-027**: Major or minor version mismatches MUST log a `WARNING` and MUST NOT change an otherwise successful exit code.
- **FR-028**: Command help and examples MUST make `config show`, `config validate`, `config template`, and `config test-connection` discoverable.
- **FR-029**: Help and documentation MUST describe `config show --validate` and `config show --template` as supported legacy paths or compatibility shortcuts.
- **FR-030**: Generated CLI documentation MUST be refreshed when command metadata or help changes.
- **FR-031**: Automated tests MUST cover validation success and failure through the dedicated subcommand and compatibility flag path.
- **FR-032**: Automated tests MUST cover template output compatibility through the dedicated subcommand and compatibility flag path.
- **FR-033**: Automated tests MUST cover `test-connection` invalid configuration, connection success, connection failure, loaded config source logging, version match, patch-only version difference, and major/minor mismatch warning.

### Key Entities *(include if feature involves data)*

- **Effective Configuration**: The resolved c8volt configuration assembled from supported configuration sources and sanitized before display.
- **Config Source Description**: The user-visible description of where the effective configuration was loaded from, including the explicit case where no config file was loaded.
- **Configuration Validation Result**: The success or failure outcome produced by validating the effective configuration.
- **Configuration Template**: The blank configuration document printed for users who need to create a configuration file.
- **Connection Test Result**: The outcome of validating local configuration, reaching the configured Camunda cluster, rendering topology output, and checking version compatibility.
- **Gateway Version**: The Camunda version returned by the cluster topology response.
- **Configured Camunda Version**: The Camunda version selected by the effective configuration and compared against the gateway version by major and minor parts.
- **Version Compatibility Warning**: A warning emitted after successful connection when configured and gateway versions differ by major or minor version.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Automated command tests show `c8volt config show` still prints sanitized effective configuration.
- **SC-002**: Automated command tests show `c8volt config show --validate` and `c8volt config validate` produce equivalent validation outcomes for valid and invalid configuration.
- **SC-003**: Automated command tests show `c8volt config show --template` and `c8volt config template` produce equivalent template output.
- **SC-004**: Automated command tests show `c8volt config test-connection` fails before remote connection when configuration validation fails.
- **SC-005**: Automated command tests show `c8volt config test-connection` logs the loaded config source at `INFO` level without requiring `--debug`.
- **SC-006**: Automated command tests show successful `config test-connection` logs success, prints human-readable topology output, and exits with code `0`.
- **SC-007**: Automated command tests show failed remote connection uses the standard error path and returns a non-zero exit code.
- **SC-008**: Automated command tests show matching major/minor versions do not warn when patch versions differ.
- **SC-009**: Automated command tests show major/minor version mismatches log a warning while preserving exit code `0` after successful connection.
- **SC-010**: Help output and generated docs list the new dedicated config subcommands and document `config show --validate` and `config show --template` as compatibility shortcuts.
- **SC-011**: Repository validation passes with targeted config command tests and the broader relevant test suite required by the project.

## Assumptions

- The affected command hierarchy remains under `c8volt config`.
- Existing configuration loading, validation, template rendering, command logging, and standard error handling paths remain the source of truth for the split commands.
- Existing cluster topology behavior is the source of truth for testing the remote Camunda connection and rendering successful topology output.
- The configured Camunda version and gateway version can be normalized to major/minor values without requiring patch-level comparison.
- The compatibility flags `config show --validate` and `config show --template` remain supported for this feature and are not deprecated out of the command surface.
- Documentation generated from command metadata should be regenerated rather than hand-edited when command help changes.
