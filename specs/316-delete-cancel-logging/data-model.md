# Internal Data Notes: Delete and Cancel Logging

This fix introduces no storage or result model. Reuse existing progress callbacks and reports. Do not add workflow event DTOs, a stage ledger, or a facade projection hierarchy.

## Polling locals

Keep the current key, one-based attempt, observed state or safe lookup failure, lookup-completion elapsed time, and optional next delay. Preserve the established ABSENT interpretation. No observation is invented for an already canceled context. Retain at most one pending observation until the existing continuation/stop decision, flushing exactly once and omitting next delay on stop.

Keep the last successful state and its observation time/attempt separately from a later lookup error. Unknown remains unknown. Do not change the existing StateUnknown/empty-instance timeout return contract or fetch another state to improve an error.

## Error detail

If required, use a small error wrapper that retains the original cause and Error() text while exposing the concise human summary or the few facts needed to render it. Necessary facts are failed phase, known root/scope, timeout/effective budget, available last observations, accepted cancellation and whether resumed deletion was reached. Capture them at existing failure branches; no new orchestration state machine.

Joined errors retain per-key information. A cancellation-confirmation timeout does not mean submission failed or was rolled back. Preserve selected normalized class, original text, errors.Is/errors.As inspection, and repeated-normalization behavior. Freeze slices only where concurrent transfer requires it.

Use existing completion FailureDetail and command rendering for concise warnings; preserve serialized report/status/error content. Do not add callback fields solely to support a parallel display model. The implementation chooses the smallest necessary wrapper in the existing domain/ferrors ownership, with tests proving compatibility.
