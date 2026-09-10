# Cancellation Confirmation Contract

## Exposed interfaces

This contract applies to existing `cancel process-instance`, cancellation inside forced `delete process-instance`, and process-definition cleanup, through the existing public facade and internal service calls. No signatures, flags, aliases, public types, or output formats are added.

## Required behavior

1. Each cancellation-specific confirmation accepts COMPLETED, CANCELED, TERMINATED, and ABSENT. Every member in the existing family confirmation scope must qualify.
2. With state checks enabled, an already-terminal root is a successful no-op: no cancellation request, existing HTTP 200/status text, and `CancelResponse.Ok=true`. A precheck returning wrapped domain not-found is treated locally as absent. Other errors propagate.
3. With state checks disabled, do not add a precheck or early no-op. Preserve submission behavior.
4. With no-wait enabled, preserve cancellation's skipped family confirmation. Preserve the existing forced-delete recovery wait even when no-wait is enabled, and retain the existing final-deletion wait guard.
5. Family discovery, retries, mutation error handling, draining, history deletion, and definition deletion remain unchanged. Do not turn a failed lookup during family discovery into overall success.
6. Explicit `expect process-instance --state canceled` continues matching only canceled/terminated under existing rules. Completed and absent do not satisfy it. Incident expectations remain unchanged.
7. Runtime output wording and shape remain stable. The corrected no-op success value and elimination of false timeout failures are intentional behavior corrections. README/help may clarify their meaning; no runtime renderer redesign is allowed.

## Acceptance matrix

Apply rows A–I to Camunda 8.7, 8.8, 8.9, and 8.10 using real versioned service methods and controlled client responses.

| Case | Setup/action | Required observation |
| --- | --- | --- |
| A | Active root, completed descendant, cancel root | Confirmation succeeds once remaining members qualify |
| B | Member changes active to completed during confirmation | Success without waiting for cancellation-specific state |
| C | Known member disappears during confirmation | Existing absence handling satisfies that member |
| D | Terminal or absent root with state checks enabled | Zero cancellation requests; Ok true and status 200; bulk reports success |
| E | At least one active/unknown member remains | No premature success; existing bounded timeout/interruption |
| F | Explicit canceled expectation against each terminal state | Only canceled/terminated match |
| G | No-wait and no-state-check, separately and combined | Existing reads, mutation submission, and wait-stage guards retained |
| H | Submission/read errors other than established absence | Existing retry/error outcomes, no fabricated success |
| I | Forced-delete recovery encounters completed/absent during cancellation wait | Continue existing delete retry/final verification; do not accept cancellation as deletion proof |

Add one focused process-definition cleanup run with an active root and completed descendant using real cancellation confirmation. Assert progress into draining and deletion, correct final result, and preservation of existing later-failure behavior. Keep cleanup orchestration production code unchanged unless a directly demonstrated issue within this contract requires a separately justified minimal fix.
