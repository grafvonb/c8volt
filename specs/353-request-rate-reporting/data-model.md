# Data Model: Request Rate Control

This feature adds configuration and invocation-local state only. It adds no persisted entities or command-result fields. See [the CLI contract](contracts/request-rate.md).

## Configuration

| Field | Type/default | Validation and meaning |
|-------|--------------|------------------------|
| `http.request_rate` | `float64`, `0` | Finite and nonnegative; positive values are requests/second, including fractions. Zero bypasses pacing. |
| `http.request_rate_report_interval` | Duration string, `5s` | Parseable, nonnegative duration; zero disables reporting. An explicit empty value is invalid. |

`--request-rate` overrides configuration when explicitly supplied, including zero. Existing environment/profile/base precedence applies. Normalization supplies defaults only for absent settings; it does not hide invalid explicit input.

## Invocation request control

One shared HTTP service owns these optional components:

| Component | State | Lifetime |
|-----------|-------|----------|
| Limiter | Positive rate, fixed burst 1 | Created only when rate > 0; shared by all API attempts in the invocation. |
| Dispatch counter | Atomic unsigned cumulative start count | Needed only for enabled reporting; increment once immediately before transport delegation. |
| Reporter | Logger, interval, previous count/time, timer, stop/completion coordination | One when effective DEBUG and interval > 0; stopped on cancellation or service close. |
| CLI cleanup handle | Idempotent close callback for current service | Consumed/cleared by the singleton root finalizer; no rate/count state is global. |

The service transitions from constructed to active to closed. Closing twice is safe. Failure during initialization closes any created service. Reporting cannot survive a completed in-process invocation or delay exit until its next tick.

## Request attempt

```text
ready → waiting (positive rate only) → admitted → dispatched → response/error
                  │                     │
                  └── local cancellation/deadline ──→ not sent
```

Check cancellation after admission and before dispatch. An unsent attempt increments no counter and makes no underlying transport call. A dispatched attempt counts even if it fails. An eligible application retry is a new attempt sharing the same allowance; transport-internal replays are not separate observable attempts.

With positive pacing enabled, each mutation retry callback receives fresh context-local failure evidence: initially empty, set only when request admission fails locally. It carries a typed cause preserving cancellation/deadline identity. The retry helper reads it after the callback and returns the cause without applying network retry rules. It must not carry over to another attempt or resource. The evidence describes only that attempt, not the outcome of earlier attempts or the overall command. The context-carried HTTP service supplies the enabled state; at rate zero the helper adds no observation/context allocation.

## Traffic window

At each report, snapshot count `C` and monotonic time `T`. With previous snapshot `(C0, T0)`, report `count = C - C0`, `elapsed = T - T0`, and `requests_per_second = count / elapsed.Seconds()`, then advance both snapshots. A concurrent increment falls into exactly one adjacent window. Guard nonpositive elapsed durations; never divide by zero. Delayed ticks use actual elapsed time, not the nominal configuration interval.

No final partial-window report is required. Unreported trailing starts are not lost from the counter; the reporting contract simply ends with the invocation.
