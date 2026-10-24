# CLI Contract: Request Pacing and DEBUG Reporting

## Operator controls

| Interface | Default | Contract |
|-----------|---------|----------|
| `--request-rate <number>` | `0` | Root persistent flag; explicit value overrides resolved configuration, including zero. |
| `http.request_rate` | `0` | Invocation-wide requests/second; zero disables pacing, positive fractions are accepted. |
| `http.request_rate_report_interval` | `5s` | Configuration-only reporting cadence; `0s` disables reporting independently of pacing. |

Existing environment mapping also exposes `C8VOLT_HTTP_REQUEST_RATE` and `C8VOLT_HTTP_REQUEST_RATE_REPORT_INTERVAL`. Preserve normal profile and environment precedence. Negative, non-finite, malformed rate values and invalid/negative report durations fail through existing configuration/flag errors before remote work. Explicit empty configured durations are invalid.

Example configuration fragment:

```yaml
http:
  request_rate: 10
  request_rate_report_interval: 10s
```

This example rate is a choice, not a guaranteed safe Camunda capacity. Explicit `--request-rate 0` disables its pacing while leaving DEBUG reporting eligible. Setting the interval to `0s` disables only the new rate summaries, not existing DEBUG diagnostics.

## Request behavior

- Positive rate shares one burst-one allowance across the command's discovery, preflight, mutation, confirmation, and application-level retry requests. No independent worker or phase allowance exists.
- Rate controls request starts, not concurrent in-flight requests, selected resources, or response completion. Idle time cannot accumulate a larger token burst. Timing has normal timer/scheduler precision; this is not a socket-write scheduler.
- Rate zero introduces no pacing waits, regardless of reporting settings. Reporting generates no remote calls and no per-request log records.
- Existing cancellation and request/workflow deadlines remain authoritative, including time spent waiting. A failed local wait reports the affected attempt as not sent, retains its cancellation/timeout classification, and introduces no retry. This says nothing about prior attempts or other resources.
- Existing retry eligibility and budgets for dispatched attempts remain unchanged. Each dispatched application retry counts again; queued or cancelled-before-dispatch work does not count.
- The boundary is the invocation's shared HTTP client, including bootstrap traffic that uses it. Separate identity-provider clients and transport-internal replays are excluded. Separate invocations do not share budgets.

## DEBUG output

Reporting requires both an effective DEBUG logger and a positive report interval. It works when rate is zero. Configured DEBUG and `--debug` both enable it; verbose alone does not. Existing quiet precedence applies.

Each periodic message contains count of dispatch starts, actual elapsed duration, and measured requests/second. Illustrative message, excluding the configured logger prefix:

```text
HTTP request rate: 42 requests in 10s (4.20 requests/s)
```

Message wording/precision should remain consistent in implementation; the three values must be visible in plain and structured logging. This is a diagnostic message, not a new result schema. Use the command-context logger and its inherited/configured destination. Never emit rate diagnostics on stdout. Include no keys, URLs, bodies, credentials, or raw transport errors.

Empty windows report zero. A short command may finish without a report. Completion, failure, or interruption stops reporting without waiting for the next interval or adding a mandatory final report.

## Unchanged command contract

Preserve scope selection, workers, dry-run previews, preflight, prompts, cleanup ordering, no-wait, human/JSON/keys-only output, quiet behavior, partial results, and exit classes. JSON emits exactly one envelope where supported. Empty scopes do not prompt or mutate; keys-only empty results contain zero bytes. No new resource limit, batching stage, confirmation, or mutation mode is added.

Acceptance coverage and isolated validation commands are in [quickstart.md](../quickstart.md).
