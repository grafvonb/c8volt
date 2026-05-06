# Research: Validate Process Definition Selectors for Process-Instance Commands

## Decision: Add a shared command-level visibility preflight for BPMN process ID selectors

**Rationale**: `get pi`, `cancel pi`, and `delete pi` already share process-instance search flags and build `ProcessInstanceFilter` in `cmd/get_processinstance.go`. A command-level helper can translate the BPMN selector portion into `process.ProcessDefinitionFilter`, call the existing process facade, and run before search/paging/mutation starts. This keeps the validation close to prompt eligibility and output mode decisions.

**Alternatives considered**:

- Add validation inside every process-instance service search call. Rejected because keyed process-instance lookups and non-BPMN searches should not pay this cost, and service code does not own CLI prompt/listing policy.
- Add a separate validation command. Rejected because users need automatic safety on existing commands.

## Decision: Use existing process-definition search APIs instead of direct generated clients

**Rationale**: `process.API` already exposes `SearchProcessDefinitions` and `SearchProcessDefinitionsLatest`, with facade conversion and versioned service implementations underneath. Reusing the facade preserves tenant and call-option behavior already used by `get pd` and avoids duplicating generated client handling in command code.

**Alternatives considered**:

- Call versioned process-definition services directly from `cmd/`. Rejected because command code should stay behind the facade.
- Add a new process-instance service dependency on process-definition services. Rejected as a larger boundary change than needed.

## Decision: Match validation mode to command semantics

**Rationale**: Search/mutation commands using `--bpmn-process-id` should validate that at least one visible process definition matches the full BPMN/version/version-tag/tenant context. `run pi` starts the latest definition when no `--pd-version` is supplied, so latest-definition validation should be used for that path; when `--pd-version` is supplied, exact version validation should be used. Multi-ID `run pi` should validate all BPMN IDs before constructing any create requests.

**Alternatives considered**:

- Always use latest-definition search. Rejected because `get/cancel/delete` without a version may match instances across visible versions and should not reject older visible definitions.
- Always use broad search. Rejected for `run pi` latest semantics, where the command promises to start latest by BPMN ID.

## Decision: Treat interactive listing as a recovery prompt, not part of validation

**Rationale**: The issue asks human output to offer listing visible process definitions. The validation failure should be complete even when the user declines or the list request fails. The prompt must be disabled in JSON, automation, keys-only, and non-TTY contexts so scripts never block.

**Alternatives considered**:

- Always print visible definitions after failure. Rejected because it is noisy and unsafe for machine-oriented modes.
- Never offer the list. Rejected because the issue explicitly asks for interactive recovery guidance.

## Decision: Preserve empty-result behavior after successful visibility validation

**Rationale**: The bug is the ambiguity between a missing process definition and a valid empty process-instance result. Once validation confirms the process definition is visible, existing paging and `found: 0` behavior is correct and should remain untouched.

**Alternatives considered**:

- Change all `found: 0` output for BPMN searches. Rejected because that would break a valid and useful operator signal.
