# Research: Preserve High-Level Activity

## Decision: Use Hierarchical Activity Scope Selection

Activity should be represented as multiple active scopes with semantic importance rather than a single active message and reference count.

**Rationale**: The observed flicker happens because the current activity model has only one visible message. A lower-level HTTP or waiter scope can start while a higher-level command workflow is active and take over the spinner text. Hierarchical selection lets nested lower-level work continue without replacing the operator-facing workflow context.

**Alternatives considered**:

- Suppress HTTP activity whenever any activity is active. Rejected because it fixes HTTP flicker but not waiter or service-level flicker, and it loses useful fallback behavior after higher-level work completes.
- Add per-command guards around known flicker paths. Rejected because the problem spans many command families and would drift from the #259 consistency goal.
- Only rename HTTP fallback messages. Rejected because better wording does not prevent lower-level text from replacing high-level progress.

## Decision: Preserve Backward-Compatible Activity Interfaces

Existing `StartActivity`, `UpdateActivity`, and `StopActivity` semantics should continue to work for existing callers while new helpers allow callers to express semantic importance.

**Rationale**: Many packages already use the shared activity sink. A compatibility layer allows incremental migration of central emitters while avoiding a broad mandatory signature change across every command and service in one step.

**Alternatives considered**:

- Replace the existing interface everywhere. Rejected because it increases blast radius and risks unnecessary command churn.
- Use context values only for priority. Rejected because priority belongs to each activity scope, not only to the surrounding command context.

## Decision: Treat HTTP Activity As Fallback Priority

HTTP request activity should have the lowest importance and be visible only when no higher-value activity scope is active.

**Rationale**: HTTP activity is useful for simple commands and connection waits, but it is too technical to replace command-level progress during operations such as delete, analysis, or bulk run.

**Alternatives considered**:

- Remove HTTP activity entirely. Rejected because simple commands would look idle while waiting on Camunda.
- Keep HTTP activity at the same importance as command progress. Rejected because it preserves the current flicker defect.

## Decision: Promote Existing Central Progress Emitters

High-level command progress should be applied at existing shared progress points: ops analysis, process-instance mutation progress, basic search progress, orphan discovery, and total calculation.

**Rationale**: These emitters already represent operator workflow phases and are shared across command families. Promoting them avoids per-command bespoke logic while covering the highest-risk commands.

**Alternatives considered**:

- Add new activity calls in each leaf command. Rejected because it would duplicate routing logic and make future command additions easier to miss.
- Promote all activity updates by default. Rejected because service waiters and HTTP fallback would still compete with command progress.

## Decision: Keep Waiters Below Workflow But Above HTTP

Wait and poller activity should remain visible when it is the best available context, but should not replace broader workflow progress.

**Rationale**: Waiters are more meaningful than individual HTTP requests but still narrower than operations such as deleting process-instance trees or analyzing a frozen work set.

**Alternatives considered**:

- Suppress all waiter activity when nested. Rejected because standalone `expect`, `resolve`, and single-resource operations still benefit from wait feedback.
- Promote waiter activity to workflow level. Rejected because it would continue to hide broader workflow context during nested waits.

## Decision: Resource-Aware Endpoint Labels For Known c8volt Paths

Known Camunda endpoint families used by c8volt should have fallback labels that name the resource and action.

**Rationale**: When fallback activity is visible, operators should see "searching variables" or "deploying resources" rather than generic technical wording. Generic fallback remains appropriate for unknown future paths.

**Alternatives considered**:

- Include HTTP method and URL path in activity. Rejected because this is too technical for default human activity and URLs can be noisy.
- Move all endpoint labels into command code. Rejected because endpoint fallback is a transport concern and should remain centralized.

## Decision: Validate By Representative Command Families

Testing should prove the shared behavior through low-level hierarchy tests plus representative command and service scenarios, rather than exhaustively testing every command leaf.

**Rationale**: The command tree contains many groups and leaf commands, but the activity mechanism is shared. Representative coverage across high-risk families is enough when paired with centralized unit tests.

**Alternatives considered**:

- Add integration tests for every command. Rejected because it would be slow, brittle, and unnecessary for a shared output mechanism.
- Rely only on unit tests for `toolx/logging`. Rejected because command context wiring and service nesting are where the observed defect appears.
