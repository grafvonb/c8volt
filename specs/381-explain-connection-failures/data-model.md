# Data Model: Bounded Human Failure Explanations

**Spec**: [spec.md](spec.md) | **Contract**: [contracts/human-errors.md](contracts/human-errors.md)

This feature adds no stored entity, public model, JSON field, configuration, or workflow state. The entities below describe existing transient evidence and its human presentation.

## Existing failure evidence

| Fact | Existing representation | Meaning / constraint |
| --- | --- | --- |
| Failure category | `failure` from `commandConnectionFailureDetails` | Existing bounded categories such as `TIMEOUT`, `DNS_ERROR`, `TLS_ERROR`, `INVALID_RESPONSE`, `CANCELED`; preserve source precedence |
| Connection reason | `reason` from the same adapter | Existing typed refusal, reset, or unexpected-EOF evidence; never derived from arbitrary text |
| HTTP status | `status` integer | Existing typed status annotation; zero means absent, not a fabricated response |
| Authentication attribution | `authentication` boolean | Existing authentication-stage annotation; distinguishes token/login failures from gateway failures |
| Request-rate admission failure | `commandHasRequestRateWaitFailure(err)` | Higher-priority local unsent evidence; earlier work may have been submitted |
| Error class | Existing `ferrors.Classify(err)` | Authoritative for original exit policy and unrecognized human fallback; explanation does not change class |

Status/auth facts may survive normalization when other bare transport facts do not. Absence of a fact means preserve fallback, not recover it from the error string. Wrapped and joined evidence follows existing `errors.As`/`errors.Is` traversal; no new cross-branch correlation model is introduced.

## Recognized connection reason

A private function returns a bounded message and a recognition boolean. The message contains only allowlisted category/status labels and fixed next-check wording. It contains no raw error text, endpoint values, resource keys, body content, or credentials. A small caller context selects the existing topology endpoint label for the connection-test renderer; ordinary callers receive neutral endpoint advice.

| Recognition result | Consumer behavior |
| --- | --- |
| Recognized supported cause | Shared formatter returns that reason; caller supplies its existing operation prefix or aggregate summary |
| Cancellation | Generic recognition declines; shared class fallback and existing cancellation exit semantics remain unchanged |
| No supported evidence | Recognition declines; existing class fallback remains unchanged |

The connection-test wrapper separately retains its cancellation wording and class-only timeout, malformed-response, setup, and unknown fallbacks. These are not generic recognized causes.

## Existing command failure presentation

- Human message: existing operation prefix/counts plus selected bounded reason, sent through the configured command-context logger.
- Machine result: existing envelope, detail, class, outcome, and partial payload; never rebuilt from the human reason.
- Exit status: derived from the original error and existing exit-suppression policy.

Relationship: original error → existing facts → selected bounded human reason. The original error independently continues to machine rendering and exit resolution. Human formatting does not mutate it.

## Validation rules and transitions

1. Request-rate evidence takes precedence; retain the exact existing unsent/uncertainty suffix.
2. A specific cause requires existing typed evidence. A plain error containing diagnostic keywords is unrecognized.
3. The existing cause precedence is preserved, including malformed-response and joined-error behavior.
4. The generic formatter never uses connection-test-specific cancellation or topology advice.
5. Output routing and JSON/exit selection remain at existing command boundaries.
6. No state transitions, network operations, retries, or persistent mutations occur while selecting a reason.
