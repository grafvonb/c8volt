# Research: Job Ops Workflow Primitives

## Decision: Extend Existing Job Boundary

**Decision**: Build on the current `cmd/get_job.go`, `cmd/update_job.go`, `c8volt/job`, `internal/domain/job.go`, and `internal/services/job` packages.

**Rationale**: Issue #180 already established job keyed lookup, retry/timeout update, dry-run planning, retry confirmation, v8.7 unsupported behavior, v8.8/v8.9 service adapters, command views, and generated docs. Extending that boundary preserves layering and avoids duplicating job behavior in ops workflows.

**Alternatives considered**: Add separate `ops job` or worker-specific verbs. Rejected because the issue explicitly asks for low-level primitives centered on `get job` and `update job`.

## Decision: Treat `get job` As Keyed Lookup Or List/Search

**Decision**: Keep `--key` as exact lookup. When `--key` is omitted, `get job` enters list/search mode with validated filters and explicit limit behavior.

**Rationale**: This preserves current behavior for known keys and adds operator discovery without a new command. It matches existing command patterns where keyed lookup and list/search modes are separated by validation.

**Alternatives considered**: Require a filter for list mode. Rejected because the issue example includes `get job --limit 50` and asks for discovery when the key is not known.

## Decision: Use Generated v8.8/v8.9 Job APIs

**Decision**: Use generated Camunda v8.8 and v8.9 `SearchJobsWithResponse`, `UpdateJobWithResponse`, `CompleteJobWithResponse`, `ThrowJobErrorWithResponse`, and `FailJobWithResponse` through versioned service adapters.

**Rationale**: The generated clients already expose these methods. Keeping request construction in versioned services preserves the generated-client boundary and allows explicit version differences.

**Alternatives considered**: Call generated clients directly from command code. Rejected because it violates the repository layering rules and makes the CLI contract version-fragile.

## Decision: Keep v8.7 Unsupported Before Mutation

**Decision**: v8.7 service paths should fail before unsupported job search or worker outcome mutation calls.

**Rationale**: The existing job service already uses explicit unsupported behavior for unsupported v8.7 operations. The issue acceptance criteria require a clear unsupported capability error before mutation.

**Alternatives considered**: Attempt best-effort legacy Operate/Tasklist fallbacks. Rejected because the source issue requires generated-client support for v8.8/v8.9 and explicit v8.7 failure.

## Decision: Make Worker Outcomes Mutually Exclusive

**Decision**: `--fail`, `--throw-bpmn-error`, and `--complete` are mutually exclusive. `--fail` may use retry count and retry-backoff as technical failure inputs. `--throw-bpmn-error` and `--complete` cannot be combined with retry or timeout update flags.

**Rationale**: These modes represent distinct worker outcomes. Keeping them explicit prevents ambiguous operator intent and makes task slices independently testable.

**Alternatives considered**: Allow arbitrary mixing of retry/timeout updates with outcomes. Rejected because it blurs existing update semantics and increases mutation safety risk.

## Decision: Preserve Existing Mutation Safety Model

**Decision**: Every material state-changing update builds a dry-run/confirmation plan, enforces JSON guardrails, supports automation behavior, and reports stable result states using existing `update job` patterns.

**Rationale**: The constitution and Ralph implementation rules treat mutation safety and script-safe output as strict requirements. Reusing the current `update job` model limits regressions.

**Alternatives considered**: Submit worker outcomes immediately because they are worker-like operations. Rejected because they are still operator-triggered state changes.

## Decision: Document Ralph Context As A Planning Constraint

**Decision**: Carry `--implementation-context specs/ralph-implementation-rules.md` through plan, tasks, and launch instructions.

**Rationale**: The user explicitly required that context for planning, task generation, and every Ralph implementation iteration, and prohibited launching Ralph without it.

**Alternatives considered**: Mention the file only in chat handoff. Rejected because Ralph needs the requirement embedded in durable artifacts.
