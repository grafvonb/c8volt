# Research: Ops Execute Smoke Test

## Decision: Reuse the existing ops workflow/report foundation

**Rationale**: The repository already has ops execute and purge workflows with shared step status vocabulary, report format inference, report overwrite safety, automation confirmation behavior, compact human output patterns, and command contract tests. Reusing those patterns keeps `smoke-test` consistent with related ops commands and avoids a parallel reporting stack.

**Alternatives considered**:

- Build a smoke-test-only report writer. Rejected because #186/#187/#199 already establish shared report semantics and overwrite safety.
- Print only command output without report models. Rejected because the issue requires Markdown/JSON audit reports and stable JSON output.

## Decision: Add smoke-test orchestration to the ops facade/service boundary

**Rationale**: The command should parse flags, validate CLI shape, call the facade, and render output. The ops facade/service can coordinate the predefined playbook and aggregate the report, while lower-level services retain ownership of deployment, process-instance run, traversal, and deletion details.

**Alternatives considered**:

- Shell out to existing CLI commands. Rejected by issue scope and repository layering rules.
- Implement all workflow logic in `cmd`. Rejected because command code must stay thin and the workflow needs reusable service/facade behavior.
- Put deployment/run/walk/delete mechanics inside ops. Rejected because resource-specific primitives belong to their owning service or facade.

## Decision: Use version-matched embedded multiple-subprocess fixtures

**Rationale**: The repository already ships `C87_MultipleSubProcessesParentProcess.bpmn`, `C88_MultipleSubProcessesParentProcess.bpmn`, and `C89_MultipleSubProcessesParentProcess.bpmn`. Selecting from the configured Camunda version proves the same fixture family across supported runtimes and fails early if a fixture is missing.

**Alternatives considered**:

- Always use the C89 fixture. Rejected because the command must support configured 8.7, 8.8, and 8.9 runtimes.
- Allow arbitrary BPMN files. Rejected as out of scope and less deterministic for smoke-test reporting.

## Decision: Prefer the deployed process-definition key for process-instance creation

**Rationale**: Starting from the deployment result prevents accidentally running an older or newer process definition with the same BPMN process ID. If the existing run primitive cannot start by deployed key, the missing primitive should be added to the process-instance service/facade that owns run behavior.

**Alternatives considered**:

- Always run by BPMN process ID. Rejected because the issue explicitly prefers the deployed key when available.
- Re-query latest process definition by BPMN ID in ops. Rejected because lookup semantics belong to process/resource services, not ops orchestration.

## Decision: Keep cleanup safety in existing delete paths

**Rationale**: Process-instance cleanup should reuse existing `delete pi` planning/deletion semantics, including family expansion from created keys, force/wait/no-wait behavior, worker controls, and error aggregation. Process-definition cleanup should reuse the existing process-definition delete path but run only after checking no unrelated instances exist for the deployed definition or BPMN ID.

**Alternatives considered**:

- Directly delete created keys without existing delete planning. Rejected because it would bypass established safety behavior.
- Force delete the process definition regardless of unrelated instances. Rejected by the issue and safety requirements.

## Decision: Treat automation as implicitly confirmed when command support is full

**Rationale**: The issue's later guardrail says `--automation` is the canonical non-interactive mode and supported prompts should use `shouldImplicitlyConfirm(cmd)`. `--auto-confirm` remains useful for humans and scripts, but is not a required companion to `--automation` for a command declaring full automation support.

**Alternatives considered**:

- Require `--automation --auto-confirm` for cleanup. Rejected because the issue's compatibility notes supersede the earlier wording and align with existing ops automation contracts.

## Decision: Use local precondition errors for runtime blockers and invalid-input helpers for static CLI shape

**Rationale**: Existing ops issue guidance distinguishes static invalid flags from runtime blockers discovered after planning or preflight. This protects automation exit codes and keeps reports consistent when an audit file is requested.

**Alternatives considered**:

- Treat all blockers as invalid input. Rejected because report overwrite, unsupported fixture state, aborts, and cleanup blockers may be discovered after planning.
