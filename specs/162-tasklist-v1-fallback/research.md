# Research: Tasklist V1 Fallback For Task-Key Process-Instance Lookup

## Decision: Keep Camunda v2 user-task search as the primary lookup

**Rationale**: The current `--has-user-tasks` implementation already resolves modern Camunda user tasks through versioned Camunda v2 user-task search, including tenant filter construction and direct extraction of `processInstanceKey`. Keeping that path first preserves existing behavior, avoids unnecessary deprecated API calls, and satisfies the issue's requirement that Tasklist V1 is only fallback compatibility.

**Alternatives considered**: Replacing v2 lookup with Tasklist lookup was rejected because it would make the deprecated API the primary path and could regress modern user-task behavior. Calling both APIs every time was rejected because it increases latency, introduces avoidable failure modes, and violates the "fallback only when v2 misses" requirement.

## Decision: Add Tasklist V1 client dependencies inside v88/v89 user-task services

**Rationale**: `internal/services/usertask/v88` and `internal/services/usertask/v89` already own version-specific task resolution and tests. The repository already has generated Tasklist clients for those versions under `internal/clients/camunda/v88/tasklist` and `internal/clients/camunda/v89/tasklist`, and configuration normalization already exposes `apis.tasklist_api`. Keeping fallback in these services avoids changing the command/facade shape and localizes version differences.

**Alternatives considered**: Adding fallback in `cmd/get_processinstance.go` was rejected because command code should remain thin and would need to understand service-specific error semantics. Adding a new top-level tasklist service package was rejected for this bounded lookup because it would introduce parallel structure before there is a broader Tasklist domain surface.

## Decision: Treat only domain not-found from primary lookup as fallback-eligible

**Rationale**: The issue requires Tasklist fallback only when the v2 client cannot find the task. Existing service behavior maps empty v2 search results to `domain.ErrNotFound`. Using that domain class as the fallback gate keeps authentication, authorization, malformed response, network, and server failures visible to operators and automation.

**Alternatives considered**: Falling back on any primary error was rejected because it could hide broken credentials, outages, tenant mismatches, or malformed upstream responses. Falling back only on HTTP 404 was rejected because the current primary path uses search and represents missing tasks as an empty result, not necessarily a 404.

## Decision: Search Tasklist V1 by task identifier and require exactly one task result

**Rationale**: The generated Tasklist V1 `TaskSearchRequest` response contains `id`, `processInstanceKey`, `tenantId`, and an `implementation` field that distinguishes task styles. The fallback should search narrowly by the supplied task key, require a single matching task, and convert that response into the existing domain `UserTask` shape so downstream process-instance lookup remains unchanged.

**Alternatives considered**: Broad Tasklist paging was rejected because the user supplies an exact key. Accepting multiple fallback matches was rejected because a task key lookup must have lookup semantics, not search semantics.

## Decision: Preserve 8.7 unsupported behavior

**Rationale**: The original task-key lookup contract explicitly rejected Camunda 8.7 and the issue keeps that behavior unchanged. Leaving `internal/services/usertask/v87` unsupported avoids expanding scope and keeps version behavior predictable.

**Alternatives considered**: Adding fallback to 8.7 was rejected because it is outside the issue scope and would change an already documented unsupported-version contract.

## Decision: Update command help, README, and generated CLI docs

**Rationale**: The current user-facing command text says there is no Tasklist or Operate fallback. That statement becomes false once this feature ships. The constitution requires documentation to match user-visible command behavior.

**Alternatives considered**: Updating tests only was rejected because operators primarily discover this behavior through help text and docs.
