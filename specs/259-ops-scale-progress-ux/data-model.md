# Data Model: Ops-Scale Preflight And Progress UX

## Preflight Scope

Represents the best available pre-work summary of what a command appears likely to process.

Fields:

- `phase`: operator-facing phase name, normally `preflight`.
- `command`: canonical command path.
- `coreResource`: primary selected resource type, such as `process_instance`, `incident`, `job`, `element`, or `process_definition`.
- `selectorSummary`: compact description of the broad selector, without debug-level filters.
- `total`: optional resource count.
- `totalKind`: `exact`, `lower_bound`, `estimated`, or `unknown`.
- `pageSize`: discovery page size when paging applies.
- `pageCount`: exact or estimated page count when known.
- `pageCountKind`: `exact`, `estimated`, or `unknown`.
- `consequenceSummary`: compact explanation of follow-on discovery, enrichment, planning, or mutation work.
- `requiresConfirmation`: whether interactive human mode must confirm before proceeding.
- `expensivePreflight`: whether obtaining better scope information itself is expected to be expensive.

Validation rules:

- `totalKind=unknown` must not carry a numeric `total`.
- `totalKind=lower_bound` must label the count as at least that value in human wording.
- `pageCount` must be derived from a compatible total and positive page size.
- `consequenceSummary` is required for broad selectors and destructive workflows.

## Total Certainty

Classifies count semantics consistently across resource types.

Values:

- `exact`: backend or frozen scope proves the complete count.
- `lower_bound`: backend reports at least the count shown, but more matches may exist.
- `estimated`: c8volt derives an approximate count from safe progress observations.
- `unknown`: c8volt cannot safely or cheaply know the total.

Validation rules:

- Exact totals can drive exact page counts and done/total progress.
- Lower-bound totals can drive `seen/total+` wording but not final exact completion.
- Estimated totals must be labeled approximate and must not be used for mutation confirmation.
- Unknown totals can show seen count and elapsed time only.

## Page Progress

Represents discovery traversal progress for paged resources.

Fields:

- `phase`: operator-facing phase name, such as `discovering process instances`.
- `currentPage`: 1-based page number already being processed or just completed.
- `pageCount`: known or estimated total pages when available.
- `pageCountKind`: `exact`, `estimated`, or `unknown`.
- `pageSize`: requested page size.
- `currentPageCount`: resources returned on the current page.
- `seen`: resources seen so far before final eligibility filtering.
- `selected`: resources selected or frozen so far after local filtering when applicable.
- `overflowState`: known more matches, no more matches, or unknown.
- `limitReached`: whether user-supplied limit stopped traversal.

Validation rules:

- Page progress must distinguish raw seen resources from selected/frozen resources when local filters exist.
- Page progress must not imply completion when overflow is unknown.
- Page progress must stop when a user limit is reached and label the stop as user-limited.

## Frozen Scope Progress

Represents exact work over a known immutable set.

Fields:

- `phase`: operator-facing phase name, such as `loading runtime elements`.
- `coreResource`: resource type being processed.
- `done`: completed resources in the phase.
- `total`: exact frozen total.
- `elapsed`: elapsed time for the phase once visible.
- `rate`: optional approximate processed resources per time unit.
- `eta`: optional approximate remaining duration.
- `errors`: optional count of failed resources when non-fail-fast workflows continue.

Validation rules:

- `done` must be between 0 and `total`.
- `total` must be exact.
- ETA must not appear until the phase has enough completed work and elapsed time.

## Progress Channel

Defines where progress may be emitted.

Fields:

- `mode`: human, verbose, debug, JSON, keys-only, quiet, or automation.
- `transientAllowed`: whether spinner/activity updates are allowed.
- `durableAllowed`: whether durable progress lines are allowed.
- `stdoutAllowed`: always false for progress, except final command results.
- `stderrAllowed`: true only for allowed human or verbose progress.
- `structuredReportAllowed`: true for automation/report surfaces that already carry structured scope metadata.

Validation rules:

- JSON progress must never write to stdout around the JSON result.
- Keys-only progress must never write to stdout.
- Quiet mode suppresses non-error progress.
- Automation mode remains non-interactive.

## Consequence Summary

Captures the operator-facing explanation of what proceeding will do.

Fields:

- `resourceSummary`: selected core resource count and certainty.
- `workSummary`: follow-on work, such as discovery, timeline loading, dependency checks, planning, repair, or deletion.
- `riskSummary`: read-only, expensive, potentially destructive, or mutation-confirmed.
- `confirmationText`: prompt body when interactive confirmation is required.

Validation rules:

- Destructive workflows must identify mutation consequences before confirmation.
- Read-only workflows should identify expensive enrichment consequences when broad.
- Confirmation text must reference the scope that will be used after proceeding.

## ETA Sample Window

Represents the timing data used to decide whether ETA is safe to display.

Fields:

- `phase`: phase being measured.
- `startedAt`: phase start time.
- `completedSamples`: number of completed work items.
- `total`: exact total when available.
- `elapsed`: time elapsed.
- `minimumSamplesMet`: whether configured sample threshold has been reached.
- `rate`: approximate rate when available.
- `remaining`: approximate remaining time when available.

Validation rules:

- ETA requires exact total, positive elapsed time, and a minimum completed sample count.
- ETA wording must remain approximate.

## Coverage Target

Records how a command family is classified for rollout and testing.

Fields:

- `commandFamily`: basic inspection, process-instance mutation, ops analysis, ops purge, ops repair, or bulk run/smoke.
- `commands`: canonical command paths and aliases.
- `firstSlice`: whether included in the initial proof implementation.
- `preflightRequired`: whether broad selectors need preflight.
- `frozenProgressRequired`: whether exact frozen-scope progress applies.
- `machineOutputModes`: supported output modes that must be protected.
- `documentationRequired`: whether help/generated docs need updates.

Validation rules:

- Every covered high-volume command family must define its preflight and progress expectations before implementation tasks are generated.
- Commands with only explicit small inputs may be marked as no-preflight unless they can fan out into a large affected scope.
