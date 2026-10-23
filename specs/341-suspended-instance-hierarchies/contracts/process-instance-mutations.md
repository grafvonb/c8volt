# Preserved Contract: Suspended Instance Cancellation and Deletion

#341 verifies existing contracts and clarifies state help. It does not add public methods, result fields, error types, flags, or guarantees for new calling paths.

## Cancellation

- SUSPENDED is unfinished and follows the existing root cancellation path.
- Preserve child force restrictions and ancestry escalation. Camunda handles cascading termination; do not resume or separately cancel children.
- The normal CLI planner deduplicates resolved roots. Verify this existing behavior with an overlapping root/child selection and successful non-retried submission.
- Normal confirmation checks the discovered family against existing terminal/absent states. SUSPENDED does not satisfy it.
- NoWait retains accepted-submission behavior without final confirmation. NoStateCheck and combined opt-outs retain existing read/submission boundaries.
- Keep current retry behavior, timeouts, context cancellation, and terminal/absent handling. No new universal direct-library deduplication contract is introduced.

## Deletion

- Preserve CLI refusal of unfinished scope without force and the direct service's established conflict handling.
- Preserve descendant-first attempts and conflict-triggered force cancellation recovery. A deletion attempt before cancellation is expected behavior, not a test failure.
- Preserve the terminal prerequisite before a recovery deletion retry, including with NoWait.
- If cancellation returns an error or the prerequisite wait fails while still suspended, do not submit the dependent retry or claim completed deletion.
- Preserve final deletion confirmation, opt-outs, and existing absence-recovery behavior. No whole-tree pre-cancel phase or new child-absence gate is required.

## Compatibility and documentation

Keep current public results, JSON schemas, error classes, human/keys-only/quiet/verbose behavior, prompts, authorization, retention, and older-version behavior. Existing repository logging/output rules still apply to any changed boundary; this issue does not plan an output overhaul.

Correct cancel/delete state-selector help to include suspended and its Camunda 8.10 applicability. Update relevant README guidance and regenerate affected CLI references from command metadata during implementation. Existing unsupported-state handling in older versions remains unchanged.

A focused failing regression is required before any suspension-specific runtime correction. General hardening or interface expansion belongs outside #341. #340 consumes existing instance operations and owns definition-level discovery and counting.
