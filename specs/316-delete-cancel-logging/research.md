# Research: Pragmatic Delete/Cancel Logging Fix

The earlier design was heavier than this issue needs. This revision follows the user's request to simplify while retaining the acceptance contract.

| Decision | Repository evidence and rationale | Alternative rejected |
| --- | --- | --- |
| Waiter owns one observation | Shared waiter and adapter checking/fetching/result logs overlap. Gate only nested lookup diagnostics. | Removing direct-lookup diagnostics or replacing the logger. |
| Keep existing wait mechanics | Family confirmation and later single-key checks are separate real operations; v87 uses tenant-safe search. | Collapsing waits or changing v87 retrieval. |
| Delete routine OAuth cache logs | RetrieveTokenForAPI emits lookup/hit pairs for cached requests. | A polling-specific authentication subsystem. |
| Restore selected existing verbose messages | Adapter transition sites already contain useful explanation text; command compact options suppress them. Existing common.VerboseLog and option gating can expose only required explanations. | New workflow event kind, public DTOs, and callback mapping. |
| Small error wrapper only where needed | Waiter timeout loses last-state evidence; ferrors.wrap stringifies the original cause. Keep necessary evidence and original text/class with focused tests. | Stage ledger, generic evidence hierarchy, or broad normalization rewrite. |
| Existing warning and final error paths | Completion FailureDetail and handleCommandError already own these surfaces. Use concise rendering there; full chain once at DEBUG. | New completion payload and facade projection model. |
| One full-path fixture plus nearby regressions | Existing local HTTP, subprocess, logger and terminal helpers cover the needed surfaces. | A separate test infrastructure phase or rewriting already-covered tests. |

The exact 36-check count is tested with deterministic responses/attempt bounds; the timeout transcript uses short real wait settings. Do not equate a wall-clock duration with exactly 36 checks.

If a shared adapter helper is needed, place it in existing internal/services/common: the processinstance parent factory imports the adapters and cannot be imported back by them. No new package is required.

No unresolved product questions remain. Implementation details should be chosen from the concrete reproduction, with helper extraction driven by actual duplication. The seven tasks in tasks.md supersede the previous 46-task decomposition.
