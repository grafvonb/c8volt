# Research: Ops Repair Workflows

## Decision: Implement repair as an ops workflow boundary

**Decision**: Add repair workflow methods to `internal/services/ops` and expose them through `c8volt/ops`, while keeping incident, process-instance, and job primitives in their owning services.

**Rationale**: Repair coordinates discovery, target freezing, variable updates, job applicability, incident resolution, confirmation, and report construction across resource types. That orchestration belongs in ops, but resource-specific API behavior already has clear owners.

**Alternatives considered**:

- Put the workflow in `cmd`: rejected because command code should not own backend orchestration, worker behavior, or multi-step resource workflows.
- Put all behavior in one resource service: rejected because repair can start from either incidents or process instances and also uses job and variable services.
- Shell out to existing lower-level commands: rejected because it would break structured output, error classification, and deterministic automation.

## Decision: Freeze targets before mutation

**Decision**: Discovery produces a frozen repair set before any variable, job, or incident mutation is submitted.

**Rationale**: The issue requires bounded repair and explicitly forbids chasing newly created incidents forever. Frozen target data also makes dry-run and reports trustworthy.

**Alternatives considered**:

- Discover and mutate in one streaming pass: rejected because new incidents could change the mutation set and reports would be harder to audit.
- Re-query after every step and expand the set: rejected because that conflicts with the issue's frozen target requirement.

## Decision: Treat job repair as per-incident optional

**Decision**: The repair plan records retry and timeout applicability per incident. Missing job keys produce `not_applicable` job steps, not workflow failure.

**Rationale**: Mixed service-task and assertion-style incidents must be repairable in one operational pass. Job repair is useful but not a prerequisite for resolution.

**Alternatives considered**:

- Reject non-job incidents when job flags are present: rejected because the issue explicitly requires mixed bulk repair.
- Ignore job flags globally when any non-job incident appears: rejected because job-backed incidents should still receive the requested job repair.

## Decision: Use process-instance key as initial variable scope

**Decision**: `--vars` and `--vars-file` update variables by deduped process-instance key scope. Output, help, JSON, and reports state that scope.

**Rationale**: The issue allows initial process-instance-only scope and requires deduped updates before dependent incident resolution. This is the safest first increment and matches existing `update pi` behavior.

**Alternatives considered**:

- Support arbitrary element scopes in this feature: rejected as out of scope and likely to increase service complexity.
- Update variables once per incident: rejected because duplicate scopes would cause repeated mutations and harder confirmation semantics.

## Decision: Reuse existing ops report behavior

**Decision**: Use shared ops report helpers for format inference, path validation, overwrite behavior, and report writing. Add repair-specific structured report models and renderers.

**Rationale**: Other ops workflows already expose report-file/report-format behavior. Reusing those rules keeps operational tooling predictable.

**Alternatives considered**:

- Hand-write report files from command code: rejected because report path validation and structured-first rendering would diverge.
- Emit only stdout JSON: rejected because the issue requires durable Markdown and JSON audit reports.

## Decision: Task generation and Ralph launches must include repository rules

**Decision**: Task generation and Ralph launch instructions must include `--implementation-context specs/ralph-implementation-rules.md`.

**Rationale**: The repository has explicit layering, testing, documentation, comment, and Ralph iteration discipline rules that materially shape implementation correctness.

**Alternatives considered**:

- Rely on AGENTS.md alone: rejected because the user explicitly made the rules file mandatory for planning, task generation, and every Ralph implementation iteration.
