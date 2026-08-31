# Research: Semantic Progress Milestones

## Decision 1: Extend the existing typed callback with a completion fact

**Decision**: Add one canonical completion event/fact to `internal/domain/ops_progress.go`, then mirror it mechanically through `c8volt/foptions` and `c8volt/ops`. The fact carries a phase/scope, stable item identity, exact total when known, disposition (`submitted`, `confirmed`, or `failed`), optional failure detail, and an optional trustworthy affected delta.

**Rationale**: Existing preflight, page, frozen-scope, and ETA events cannot identify an item or its lifecycle outcome, so they cannot support immediate failures or verbose per-item lines. The callback already crosses every required layer and is the smallest repository-native extension.

**Alternatives considered**:

- Infer completion by differencing `FrozenScopeProgress.Done`: rejected because concurrent snapshots can arrive out of order and contain no identity, outcome, or affected delta.
- Read final result slices in commands: rejected because warnings would not be immediate, fail-fast unscheduled work is ambiguous, and commands would infer backend lifecycle facts.
- Add a second progress interface: rejected because it would duplicate callback propagation, output gating, and tests.

## Decision 2: Aggregate and render in one command-owned semantic reporter

**Decision**: Add a focused, mutex-protected reporter in `cmd/ops_semantic_progress.go`. It consumes completion facts, maintains exact monotonic counters, owns a real workflow-priority activity scope, checks an injected clock, emits durable lines, and finishes idempotently.

**Rationale**: Services own facts but must not own operator wording. A single reporter prevents divergent implementations across command families and safely serializes concurrent callbacks. The existing activity writer already clears a spinner before a durable write and redraws the surviving scope.

**Alternatives considered**:

- Render directly inside worker callbacks: rejected because it duplicates mode gates and mixes service facts with human wording.
- Add locks to every existing command callback independently: rejected because pacing, final flush, affected validity, and quiet behavior would drift.
- Put aggregation in a public facade: rejected by repository layering rules; it is command presentation state.

## Decision 3: Use completion-driven 10-second pacing without a ticker

**Decision**: Change semantic informational pacing to 10 seconds and evaluate it only when a real completion arrives. The clock begins when the post-confirmation work scope starts and resets after each informational milestone.

**Rationale**: The repository's current pacer is already clock-injectable and event-driven, but it is 30 seconds and only used by slow analysis. Extending that pattern avoids background lifecycle complexity, emits nothing while idle, and makes deterministic tests fast.

**Alternatives considered**:

- Keep 30 seconds: rejected by the clarified operator requirement; it is too sparse for useful feedback.
- Emit on a 10-second ticker: rejected because idle workflows would chatter and shutdown/final-flush logic would need another goroutine.
- Emit every N items: rejected because item duration varies widely between command families.

## Decision 4: Make durable activation and final flushing explicit

**Decision**: Durable progress activates only when a paced informational milestone or immediate failure warning is emitted. The reporter tracks changes since the last durable line. `Finish` emits one last aggregate only when durable progress is active and unreported progress remains.

**Rationale**: This preserves silence for clean sub-10-second operations while retaining the final state of long or failing operations. An explicit idempotent finish also covers partial/fail-fast exits without falsely claiming success.

**Alternatives considered**:

- Always print a final line: rejected because it duplicates normal final output for fast clean commands.
- Flush whenever `completed == total`: rejected because fail-fast and cancellation may end with fewer completions than the frozen total.

## Decision 5: Treat affected counts as an all-or-nothing capability

**Decision**: A scope renders cumulative affected counts only when its producer declares complete coverage and every completion provides a trustworthy value, including pointer-to-zero. Any unavailable contributor disables affected rendering for the entire scope.

**Rationale**: Some workflows know only aggregate cleanup scope, some per-root scopes overlap, and some versions expose incomplete metadata. Showing a partial sum and later retracting it would mislead operators.

**Alternatives considered**:

- Treat missing as zero: rejected because unknown is not zero.
- Show a partial `known affected` sum: rejected by the accepted clarification.
- Estimate affected work: rejected because the feature requires exact semantic facts.

## Decision 6: Derive output policy from existing command modes

**Decision**: Extend the existing progress channel policy with explicit failure-warning permission. Default human mode gets transient activity, paced aggregate information, and immediate failures. Verbose/debug gets transient activity and one per-item durable line instead of paced aggregate information. Quiet gets immediate failures only. Automation, JSON, and keys-only get no human progress.

**Rationale**: Current mode gating is centralized and stdout-safe. Quiet needs a direct activity-aware stderr failure path because root logging filters warnings at error level. Automation remains fully silent and uses structured results/reports.

**Alternatives considered**:

- Use `slog.Warn` for quiet: rejected because the quiet log level filters it.
- Allow warnings in JSON/keys-only: rejected because the spec requires no human progress text in machine-oriented modes.

## Decision 7: Emit facts at real service completion boundaries

**Decision**: Emit exactly one completion fact for every actually executed worker item or explicit workflow stage, immediately after its result/error is known. Unscheduled fail-fast items emit nothing. Existing ordered result assembly stays unchanged.

**Rationale**: Worker return points have identity, error, acceptance/confirmation state, and scope context. They are the earliest truthful and concurrency-safe semantic boundary.

**Command-family boundaries**:

| Family | Completion boundary | Notes |
| --- | --- | --- |
| Process-instance cancel/delete | One root-tree worker returns after acceptance or configured wait | Direct, stdin, and search use the same reporter; force cleanup remains nested/lower priority. |
| Basic process-definition delete | Each definition deletion worker returns; include the serial first capability probe | Do not sum overlapping cleanup impact unless per-item values are proven. |
| All-process-definition purge | Each frozen candidate definition finishes deletion | Discovery pages remain separate; confirmed mutation starts a fresh scope. |
| Deployment | A returned process definition becomes visible, or acceptance in no-wait mode | Multipart file upload is one request, not per-file completion; v8.7 uses only facts its response can prove. |
| Retention/orphan/incident purge | Each planned process-instance root-tree deletion returns | Orphan parent checks remain discovery progress until an exact frozen total exists. |
| Incident/process-instance repair | Each repair worker returns, not a loop after the entire pool | Preserve step details and report schemas. |
| Smoke test | Deploy, start, walk, and cleanup stage/item boundaries | Remove or gate legacy human stage logging that bypasses automation policy. |

## Decision 8: Assess secondary long-running commands conservatively

**Decision**:

- Include `run process-instance --count` and ordered multi-definition starts because totals and service completion boundaries are exact.
- Reuse the shared 10-second behavior for slow-process analysis frozen work while keeping discovery pages distinct.
- Include multi-key `expect process-instance` because each key has a meaningful satisfied/failed terminal result and a known total.
- Keep plain searches and enrichment as transient/verbose discovery progress; their pages are not mutation or lifecycle completions.
- Exclude process-definition watch, single-target polling, and ordinary walk output from semantic completion aggregation because they are repeating or do not expose a useful frozen multi-item lifecycle.

**Rationale**: This gives operators durable evidence where it communicates real completed work without turning every wait or page into noisy pseudo-progress.

## Decision 9: Preserve legacy diagnostics only for non-structured callers

**Decision**: When a structured progress callback is installed, suppress legacy timer-driven bulk progress and duplicate smoke-test informational logs. Retain legacy behavior for callers that did not opt into structured progress.

**Rationale**: This avoids two competing progress streams while preserving public/facade callers that rely on existing diagnostics.

## Validation Decision

Use fake-clock reporter tests instead of real 10-second sleeps; concurrency tests under `-race`; service tests at worker boundaries; command integration tests for direct/stdin/search parity and every output mode; activity writer tests for clear/redraw; targeted package tests first; then `make docs-content`, `git diff --check`, and `make test`.
