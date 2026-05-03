# Research: Resolve Process Instance From User Task Key

## Decision: Resolve via tenant-aware native Camunda v2 user-task search for 8.8 and 8.9

**Rationale**: The generated Camunda v2 clients for supported versions expose `SearchUserTasksWithResponse` with both `userTaskKey` and `tenantId` filters, and `UserTaskResult` carries the owning process-instance key. This satisfies the issue's API boundary, preserves tenant scoping during the resolution step, and keeps Tasklist and Operate out of the resolution path.

**Alternatives considered**:

- Search process instances by metadata: rejected because the user task relation is already available directly from the native user-task result.
- Tasklist lookup: rejected because the issue explicitly forbids Tasklist.
- Operate lookup: rejected because the issue explicitly forbids Operate.

## Decision: Return explicit unsupported behavior on Camunda 8.7

**Rationale**: 8.7 must not fall back to Tasklist or Operate, and the repository already has unsupported-error patterns for version-specific CLI behavior.

**Alternatives considered**:

- Hidden compatibility fallback: rejected because it violates the issue and makes behavior harder to reason about.
- Treat `--has-user-tasks` as an unknown flag on 8.7: rejected because the flag should be discoverable while the runtime version explains the unsupported condition.

## Decision: Reuse direct process-instance lookup after resolution

**Rationale**: Existing keyed lookup already owns strict not-found behavior, tenant-aware lookup semantics, output modes, default age output, `--keys-only`, and JSON rendering. Reusing it avoids parallel formatting and preserves user-visible behavior. Multiple user task keys resolve to multiple process-instance keys and then reuse the same multi-key process-instance lookup path.

**Alternatives considered**:

- Render directly from user-task data: rejected because the user requested process-instance output, not task output.
- Add a separate `get task process-instance` command family: rejected because the issue says this should fit cleanly into `get pi`.

## Decision: Validate `--has-user-tasks` as a lookup selector, not a search filter

**Rationale**: The command currently separates keyed lookup from search mode. `--has-user-tasks` resolves one or more user task keys to process-instance keys and must therefore conflict with search filters, stdin key input, `--key`, `--total`, and `--limit`.

**Alternatives considered**:

- Allow `--has-user-tasks` with filters after resolution: rejected because filters would create ambiguous semantics and could make the command return something other than the owning process instance.

## Decision: Update docs from command metadata where generated output exists

**Rationale**: The constitution requires documentation to match command behavior, and the repo has `docsgen` for generated CLI docs.

**Alternatives considered**:

- Hand-edit generated CLI docs only: rejected because generated docs should be refreshed from source metadata.
