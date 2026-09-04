# Data Model: Force-Cleanup Progress

## Additive callback event

Extend the existing progress envelope with `Kind = "stage"` and an optional `Stage` payload. Proposed internal names are `OpsProgressEventKindStage` and `OpsStageProgress`; expose corresponding public `ProgressEventKindStage` and `StageProgress` types through ops and foptions.

| Field | Type | Meaning / validation |
|---|---|---|
| Phase | string | Stable service identifier from the phase table below; never rendered directly as command wording. |
| CoreResource | string | Existing completion resource unit for quantified work; empty for draining. |
| Total | optional integer | Exact work-item total when known; nil for waiting or unavailable totals; zero means known empty, not unknown. Nonnegative. |
| PlannedAffectedCount | optional integer | Trustworthy unique process-instance scope from the cleanup plan; never a cumulative completed-work count. Nil means unavailable. Nonnegative. |

JSON field names, if callback values are serialized by library consumers, are `phase`, `coreResource`, `total`, and `plannedAffectedCount`, with optional fields omitted when nil. Add the envelope field `stage,omitempty`. This event is not added to CLI result JSON, report documents, or persisted domain data.

Every new event carries its matching payload only. Nil callbacks remain no-ops. Conversion preserves nil versus known zero and copies optional pointers. Existing progress variants and completion structures remain unchanged.

## Phase identity and transitions

| Phase | Activity meaning | CoreResource | Completion source |
|---|---|---|---|
| `cancel` | Cancelling process-instance root trees | `process-instance tree(s)` | Existing PI cancel completions |
| `drain process instances` | Waiting for active process instances to drain | empty | None; existing service wait controls exit |
| `delete` | Deleting process-instance histories | `process-instance tree(s)` | Existing PI delete completions |
| `delete process definitions` | Deleting process definitions | `process definition(s)` | Existing definition completions |

Normal forced progression is cancellation → drain → history deletion → definition deletion → close. No-cleanup execution enters definition deletion directly. Empty selection can close without entering a mutation stage. Existing service failure, timeout, fail-fast, and interruption behavior determines whether later stages are entered; rendering does not make those decisions.

Use the deduplicated cleanup roots and affected keys from `processDefinitionDeleteCleanupScopeForPlan`. Do not recompute scope in the command or borrow completion totals from another stage. Existing execution fallbacks to root counts do not establish full affected coverage.

## Command-owned stage state

Store one record for each entered mutation phase, at most three:

| Field | Purpose |
|---|---|
| Scope | Existing `opsSemanticProgressScope` vocabulary and resource unit |
| Aggregate | Existing completed, failed, total, affected, and affected-valid state |
| PlannedAffectedCount | Optional separate planned scope label |
| Dirty | Aggregate includes facts not yet represented by a durable line |

An explicit current-stage field may reference a mutation stage or draining. It is separate from stored historical aggregates. The same root identity can legitimately complete once in cancellation and once in history deletion; those are distinct stage outcomes. The producer already deduplicates roots and emits one completion per attempted item per stage. Do not introduce an unbounded per-item history solely for rendering.

Rules:

- Only matching `Completion` facts advance completed/failed/affected aggregates. FrozenScope updates are not a second completion source.
- Counters are monotonic within their stage and bounded by the known total. Submitted outcomes count as completed attempts but keep submitted wording.
- A missing or negative contributing affected count invalidates the cumulative affected aggregate for that stage. Known zero remains valid.
- Planned affected count can remain visible even when completed affected coverage is unknown; labels must distinguish them.
- A completion for an unentered, unknown, or empty phase does not invent a stage. Late facts for an entered historical stage can update its evidence but cannot roll current activity backward.
- Waiting has no completed/total aggregate. Failure of the wait uses the existing error path rather than fabricating a root completion.

## Workflow-owned state

The coordinator holds output policy, activity stop handle, current stage, stage records in execution order, mutation start time, last informational timestamp, durable activation flag, and closed flag. A mutex serializes callback state changes and output so printed order agrees with aggregate order.

A real execution starts a generic activity; the first mutation stage anchors pacing. The same handle is updated throughout execution. The preview wrapper retains its own separate lifetime and is stopped before a confirmation prompt. No two workflow-priority owners compete during real execution.

Durable policy:

- Default informational clock starts at first stage entry and resets only after an informational milestone, not stage entry, polling, or warning.
- Warning or verbose output accounts for the aggregate of that line's stage only; it does not clear other dirty records.
- Default paced emission prints the stage that produced the eligible completion, clears that stage's dirty flag, and activates durable progress. A historical completion does not repaint current activity.
- Close may emit one historical record covering all dirty entered mutation stages in order when default durable progress has activated. It does not infer completion of missing work or success of the workflow.
- Close then stops activity exactly once; subsequent events and closes do nothing.

All state is transient. Backend request/result models, public operation signatures, persistence, and final report structures are unchanged.
