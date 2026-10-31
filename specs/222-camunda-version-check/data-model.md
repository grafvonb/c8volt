# Data Model: Advisory Camunda Version Check

**Scope**: Transient invocation state and typed internal outcomes only. No persistent storage, public facade model, or command result schema is added.

## Release line

Owner: pure `toolx` helper, reused by checker and safe human diagnostic rendering. The existing explicit connection-test parser/classification remains separate for machine-output compatibility, including historically permissive values.

| Field | Meaning / validation |
| --- | --- |
| Major | Nonnegative decimal integer; reject parse overflow. |
| Minor | Nonnegative decimal integer; numeric equality defines compatibility with Major. |
| Display | Trimmed validated observed release, at most 128 bytes; safe ASCII only, no control characters. |

Accepted syntax: optional `v`/`V`, `major.minor`, optional `.patch`, optional `-prerelease`, optional `+build`. Suffix identifiers are nonempty dot-separated ASCII alphanumeric/hyphen components. Only decimal digits are accepted in numeric components. Reject malformed syntax, negative signs, extra numeric dot components, controls, and overlong values. Examples: `8.9`, `8.9.3`, `v8.10.0-alpha4`, `8.10.1+build.2`. All suffix/patch differences are ignored when comparing major/minor. Existing configuration aliases are normalized before comparison; do not feed `810` directly into the gateway parser.

## Gateway check outcome

Owner: proposed `internal/domain/version_check.go`; internal only.

| Field | Meaning |
| --- | --- |
| Status | `matched`, `mismatched`, or `failed`. Disabled/not-needed is represented by absence of an attempted outcome. |
| ConfiguredVersion | Effective normalized configured release line; immutable for the invocation. |
| ObservedVersion | Safe validated display value for a successful comparison; absent on invalid data. |
| FailureReason | Fixed enum: `access_denied`, `timeout`, `invalid_version`, `unavailable`, or `canceled`; present only for failed checks. |
| BrokerVersions | Sorted unique validated broker release displays; `unknown` represents missing/invalid broker versions. Informational only. |
| BrokerCount | Number of brokers in a successfully decoded topology response; omitted from DEBUG when zero/unavailable. |
| ClusterSize, PartitionsCount, ReplicationFactor | Positive reported topology counts; zero represents unavailable/nonpositive metadata and is omitted. |
| WarningNeeded | Derived from mismatched/failed status, independent of logger visibility. Used for explicit connection-test deduplication. |

Topology facts remain available if gateway parsing fails. The callback and each outcome reader receive independent copies of the broker-version slice.

No raw error, body, URL, resource key, credential, or generated response belongs in a renderable outcome. Failure causes may remain local while classifying; the probe must not return them as operational failures or add them to a JSON envelope.

## Invocation gate

Owner: concrete checker in `internal/services/cluster`; small context contract in `internal/services/httpc`.

State transitions:

```text
no gate (disabled/library) --------------------> direct operation
idle (local path ends) -----------------------> discarded, zero attempt
idle -- first active operational request -----> checking
checking -- response/timeout/failure ----------> complete(outcome)
complete ------------------------------------> direct operation if caller active
```

- Exactly one winner changes idle to checking. Concurrent callers wait on one completion channel and can abandon waiting when their own context is canceled.
- A canceled caller before any check starts must not cause a version request. A cancellation during the winner's attempt finishes that attempt; later active callers reuse its failure outcome, not a retry.
- Publish completed outcome before closing the channel, after the single outcome callback. Do not hold a mutex over callback or network I/O.
- All waiters and later operations see immutable terminal facts. The checker does not serialize operational requests after completion.
- Scope is the invocation context, not a facade instance, global variable, worker, or generated client.

## Probe context and HTTP capabilities

| Object | Relationship and lifetime |
| --- | --- |
| Invocation context | Carries the gate and existing config/auth/HTTP services. |
| Operational request | Retains its original context and original concrete HTTP client. |
| APIClient | Additive internal Do interface in ServiceDeps; consults gate before delegating to HTTPClient. |
| Probe context | Child of winning caller; two-second/shorter timeout, bypass marker, no-read-retry marker, and independent request-rate observation. Canceled after probe only. |
| Probe client | Shallow copy of HTTPClient sharing transport and cookie jar; advisory redirect refusal only. Never changes shared client settings. |

The original request's HTTP timeout starts only after the gate. An inherited earlier workflow deadline still applies to both waiting and dispatch. Do not reuse the probe context or its pacing evidence for operational work.

## Existing entities retained

`domain.Topology.GatewayVersion` remains the observation source. Existing command envelopes, user-task/mutation submission reports, error classes, and explicit connection-test JSON warnings retain their schemas. The gate's outcome never substitutes for a submitted operation's evidence.
