# Contract: All-Process-Definitions Force-Cleanup Progress

## User interface and compatibility

Applies to `c8volt ops purge all-process-definitions --force` and existing aliases `all-pds` and `apd`. Adds no flags or supported output modes. Discovery, selection, confirmation, mutation permissions, tenant handling, workers, fail-fast, no-wait, final results, audit reports, and exit behavior remain unchanged. `--batch-size` describes discovery pages only.

APD currently advertises one-line and JSON output. Keys-only remains protected by shared progress policy and existing unsupported-mode behavior; this feature does not enable it for APD. Full-history capability checks retain the existing 8.9/8.10 supported paths and older-version rejection.

## Callback interface

See [data-model.md](../data-model.md) for fields and phase identifiers. Service stage entry is an additive callback observation, not a command result or a new remote operation.

1. `cleanupProcessDefinitionDeletePlanForceScope` emits cancellation entry before calling the existing PI cancellation routine, with exact unique roots and trustworthy planned affected scope.
2. Existing `cancel` completion facts update root counters. Ignore nested FrozenScope/ETA events as completion sources.
3. After successful cancellation, emit drain entry immediately before the existing active-instance wait. Emit no synthetic completion or periodic stage event while polling.
4. After successful draining, emit history-deletion entry before PI deletion. Existing `delete` completion facts update a fresh root counter set.
5. For the force/preplanned path, `DeleteProcessDefinitionResources` emits definition-deletion entry before its first mutation request, once per resource-deletion invocation. For the ordinary non-force path, `DeleteProcessDefinitions` supplies a private once-only entry hook reached in `deleteProcessDefinition` immediately before the first validated resource deletion. Its total is the unique bulk key count; concurrent items do not repeat entry. Preserve both execution paths and their different validation/scheduling. Existing `delete process definitions` facts count definitions, including the force path's serial first-item probe and subsequent pool outcomes.
6. Errors preserve existing returns and results; no later entry may be emitted for a stage never reached. Empty work must not manufacture successful stages.

Map stage events without semantic reinterpretation through both ops and foptions. Old consumers may ignore the new event kind. The command routes only recognized, entered phases; it must not parse service message text.

## Transient activity

A preview activity stops before confirmation. A real execution has one generic workflow activity during discovery/revalidation; first actual stage entry changes its label and starts mutation pacing. Frozen candidate keys alone do not justify saying definition deletion has begun.

At stage entry, show zero completed with the exact total for quantified work. Update after each matching completion. Root units remain distinct from definitions. Waiting shows a waiting label without a denominator. Nested lower-priority request/polling activity cannot replace the current stage.

Representative stage snapshots, with punctuation aligned to nearby renderers during implementation:

```text
cancelling process-instance root trees, 0/2 process-instance tree(s), affected scope: 7 process instance(s)
cancelling process-instance root trees, 1/2 process-instance tree(s), affected scope: 7 process instance(s)
waiting for active process instances to drain
deleting process-instance histories, 0/2 process-instance tree(s)
deleting process-instance histories, 2/2 process-instance tree(s)
deleting process definitions, 0/3 process definition(s)
```

The planned `affected scope` label is not a claim of completed work. Cumulative affected values from outcomes may additionally be shown only when every contributing value is trustworthy. Unknown values are omitted, never estimated or displayed as zero.

## Output policy

| Mode | Transient stage activity | Informational progress | Failed item outcome |
|---|---|---|---|
| Default human | Current stage and aggregates | Paced aggregate on stderr | Immediate warning on stderr |
| Verbose/debug | Current stage and aggregates | One item outcome per applicable stage, replacing paced lines | One warning outcome, not duplicated |
| Quiet | None | None | Immediate existing warning, including when logger filters warn records |
| JSON | None | None | Existing structured error/result behavior only |
| Automation, including verbose combinations | None | None | Existing error/result contract only, no human progress warnings |
| Keys-only policy | None | None | Preserve existing policy/unsupported-mode behavior |

No human progress is written to stdout. Existing progress writers clear and restore transient activity around durable lines. Final output and errors retain established rendering and ordering. Existing verbose diagnostics are not expanded or removed by this feature; exactly-one outcome assertions refer to semantic completion lines.

## Pacing and final record

- One 10-second informational clock spans the entire mutation workflow, beginning with its first stage entry.
- Each completion updates its own stage before output decisions. If the interval has elapsed, default mode emits that stage's aggregate and resets the informational clock.
- Time passing, stage entry, FrozenScope counters, and drain polling alone never emit a durable informational line.
- A failure warning is immediate, activates durable progress, and does not reset the informational clock. Its line accounts for that stage's aggregate only.
- Stage changes preserve historical dirty aggregates and do not flush.
- Default close emits at most one final record for dirty stages when durable output was activated. A multi-stage record joins labeled aggregate summaries in execution order; a single dirty stage uses the ordinary aggregate form. No record for clean/unentered stages or waiting. No activity updates occur during this historical flush.
- Close is idempotent. Verbose and quiet/machine modes do not add an aggregate final record. A clean workflow ending before the first pacing threshold emits no durable progress.

Example deterministic timeline:

| Time | Fact | Expected default durable behavior |
|---|---|---|
| 0s | Cancellation entry | No line; activity shows 0/2 roots |
| 4s | First root completes | No line; activity shows 1/2 |
| 10s | Second root completes | Cancellation 2/2 milestone |
| 11s | Drain entry | No line; waiting activity |
| 30s | Still draining | No line |
| 31s | History entry then first completion | History 1/2 milestone; entry itself prints nothing |
| 32s | Second history completion | No line; history 2/2 remains dirty |
| 33s | Definition entry and first completion | No line; definitions 1/3 remains dirty |
| 34s | Remaining definitions complete, close | One final record includes history 2/2 and definitions 3/3 |

Failures may produce warnings at any timestamp. Failed counts remain stage-local and a final record never relabels failed work as successful or announces a later stage.

## Acceptance evidence

A command-level fake-backend test must execute the actual Cobra → public facade → ops service → process-definition service → PI cancellation/drain/history → definition deletion path. Capture stage entries and completion-driven activity through synchronized barriers. Standalone injected callbacks are useful pacing unit tests but are insufficient as the only command proof.

Required cases: full nested order; activity before first root completion; delayed draining; shared roots; non-force ordinary-path entry; unknown versus zero affected counts; concurrent/out-of-order results; no cleanup; empty selection; cancellation/history failure; drain error/interruption; no-wait wording; serial deletion probe failure; prompt decline/dry-run/auto-confirm; mode precedence; cross-stage timing; final dirty-stage retention; cleanup on error and repeated close.

Compare request payloads, request counts and required ordering (allowing existing concurrency), worker options, frozen scope, results, reports, and exit behavior against existing fixtures. No additional mutation or discovery request is allowed solely for progress.
