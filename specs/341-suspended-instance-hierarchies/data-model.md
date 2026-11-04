# Data Model: Existing Process Instance Contracts

The narrowed #341 introduces **no new data model, public field, schema, or persistence**. Earlier proposed mutation details, stages, deletion gates, and execution ledgers are superseded.

## Existing concepts used by regressions

| Concept | Existing role | Regression expectation |
| --- | --- | --- |
| Process instance key and parent | Identify the selected instance and ancestry | Existing force escalation resolves the root |
| State | ACTIVE, SUSPENDED, terminal aliases, ABSENT, UNKNOWN | SUSPENDED remains unfinished; existing terminal/absent/unknown rules remain unchanged |
| DryRunPIKeyExpansion | Existing CLI roots and affected scope | Overlapping selected root/child inputs yield one planned root |
| CancelResponse / DeleteResponse / reports | Existing success and status information | Preserve fields and existing failure classification |
| Cancellation failure annotation | Records accepted cancellation whose follow-up fails | Preserve its accepted-only meaning; no new uncertainty model |

## Existing transitions exercised

- Cancellation: optional precheck and existing escalation → root submission → existing family confirmation unless no-wait.
- Forced deletion: descendant-first attempts → conflict on unfinished instance → existing cancellation recovery → terminal prerequisite wait → retry deletion → existing optional final absence wait.
- Persistent SUSPENDED observations: confirmation remains unsatisfied and ends under existing timeout/context limits.

These describe the current workflow, not a new scheduler. Do not move cancellation ahead of all deletion attempts, redefine global terminal classification, change opt-out read counts, or change existing absence recovery. See [contract](contracts/process-instance-mutations.md) for preserved interface behavior.
