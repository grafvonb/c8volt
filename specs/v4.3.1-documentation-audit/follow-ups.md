# Follow-ups from the v4.3.1 documentation audit

Status: proposed; no runtime changes are included in the documentation correction.

Milestone: https://github.com/grafvonb/c8volt/milestone/26

## Consistent tenant visibility for discovery commands

`get process-definition`, `get process-instance`, and selector-based `ops analyse slow-process-instances` apply tenant filtering and `--all-tenants` without the tenant-context messages available on mutation workflows. The documentation correction describes that existing behavior.

A separate feature should decide which discovery surfaces need selection and override messages, when those messages should appear, and how they interact with preflight output. Preserve filtering and backend authorization. Reuse existing context and evidence rather than adding discovery solely for display. Specify stdout/stderr placement, logger formatting and filtering, and human, quiet, JSON, keys-only, automation, empty-result, and error behavior before implementation. Cover named filters, cleared filters, explicit overrides, and direct-key semantics through command execution tests.

Related issues: #282 and #283.

## Cancellation and partial-failure reporting

Issue https://github.com/grafvonb/c8volt/issues/293 was closed as not planned; its broader workflow redesign is not delivered by #294. Terminal-state cancellation acceptance is implemented, but process-instance force deletion still traverses children and cancels reactively after a deletion state rejection.

Reassess the remaining work in a separate specification: cancellation ordering, preservation of partial results, clear submitted versus confirmed versus unfinished outcomes, and audit evidence when cleanup fails. Explicitly decide whether force deletion should cancel the entire planned scope before deleting any history. Preserve or deliberately specify compatibility for no-wait, no-state-check, retries, fail-fast, worker limits, machine envelopes, and exit codes. Keep workflow mechanics in internal services and test all supported Camunda adapters plus dependent process-definition and ops cleanup paths. Do not treat the closed issue as an implemented guarantee.
