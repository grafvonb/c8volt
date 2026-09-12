# Data Model: API Request Diagnostics

All entities are in-memory and invocation-scoped. No persisted schema or public facade result type changes.

## Invocation collector

| Field | Type / rule |
| --- | --- |
| writer | Explicit `io.Writer`, bound to effective command stderr; never a package-global destination |
| sequence | Atomic unsigned integer, allocated once per observed exchange, starting at 1 |
| write lock | Serializes complete record writes from this collector |
| profile | Optional resolved active profile identity; absent when unset |
| tenant | Optional resolved configured tenant selection; absent when unset; not inferred resource ownership |
| redaction context | Private known credential values from configuration and sensitive request/response metadata, plus sanitization policy; never serialized |

One collector serves the invocation's API, cookie-auth and separate OAuth token traffic. Absence of a collector means disabled diagnostics, with no observer attached. Two invocations never share a collector, sequence or writer binding.

## Exchange observation

| Field | Type / rule |
| --- | --- |
| sequence | Unique within collector, allocated at delegate entry |
| started | Wall timestamp with monotonic component for elapsed timing |
| request identity | Method and URL components sanitized before serialization |
| trace state | Optional first-response timestamp, connection reuse and matched phase intervals |
| request bytes | Count of bytes actually returned by observed request reads; includes bytes accompanying an error |
| request completion | Known no-body or observed EOF; false if not established at snapshot time |
| response status | Optional HTTP status observed from returned response |
| response bytes | Optional count of observed response body bytes |
| response completion | True only for observed EOF or a known bodyless response; false after early close/read failure |
| error category | Optional safe enum; never arbitrary error text |
| timeout / canceled | Optional observed classifications, not guessed from status codes |
| correlation metadata | Only validated allowed header fields |
| state lock / once guard | Protect counters, trace callbacks and a single terminal snapshot |

A request-body close must preserve Close behavior but is not proof the whole body was uploaded. Snapshot counts describe observations at finalization, even if transport-owned request-body work completes later. Content-Length is never substituted for measured bytes. Counts exclude headers/framing; decompressed response bytes may differ from wire transfer size.

## Phase observation

A phase sample contains phase kind (DNS/connect/TLS), a paired start/end and nonnegative duration. Multiple samples are retained; connection attempts are paired by network/address internally. End callbacks may represent failed attempts, which still performed measurable work. Unpaired/in-progress samples are omitted. Network addresses are not serialized as extra connection detail. Samples are ordered by their start time, not callback completion order.

Absent hooks or unperformed phases are absent fields. A genuine observed zero-duration sample is distinct from absence. Samples may overlap and must not be summed into a supposed breakdown of total duration. Reuse is optional and comes from connection evidence, not the absence of a handshake alone.

## Immutable diagnostic record

One terminal snapshot links to its collector by sequence and contains only sanitized identities, optional observations and completeness evidence. The exact external field names, value grammar and omission rules are in [contracts/api-diagnostics.md](contracts/api-diagnostics.md).

Validation rules:

- All durations and counts are nonnegative. Sequence is positive and unique per invocation.
- Unknown fields are omitted, not filled with zero, empty strings or fabricated defaults.
- HTTP error statuses retain their real status; transport error categories do not manufacture HTTP statuses.
- First-response can precede final headers because informational responses are possible.
- A non-EOF body error can coexist with a real HTTP status and incomplete response evidence.
- Operational profile/tenant context is selection context; no body parsing or extra tenant lookup occurs.
- Secret matching/redaction applies before any string is serialized. Bodies and raw errors never enter a record.
- Escape controls/newlines so a record occupies exactly one physical line.

## Lifecycle

```text
allocated → delegated
  ├─ transport error ───────────────────────→ terminal snapshot → write attempted
  ├─ known no response body ────────────────→ terminal snapshot → write attempted
  └─ response available → caller reads/closes
       ├─ EOF ─────────────────────────────→ complete snapshot → write attempted
       ├─ non-EOF read error ───────────────→ incomplete snapshot → write attempted
       └─ early Close ─────────────────────→ incomplete snapshot → write attempted
```

Once frozen, later Close calls and trace callbacks cannot emit another record or change the snapshot. Underlying Read and Close return values and ownership remain unchanged. A diagnostic write error does not alter the HTTP result and is not retried on stdout. An abandoned body without an observed terminal event remains unfinalized; diagnostics neither drain it nor invent a completion timer. State is released with the ordinary request/body lifecycle.
