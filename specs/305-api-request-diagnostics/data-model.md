# Data Model: API Request Diagnostics

All entities are in-memory and invocation-scoped. No persisted schema or public facade result type changes.

## Invocation collector

| Field | Type / rule |
| --- | --- |
| logger / verbose | Existing invocation `*slog.Logger` and resolved verbose setting; no collector-owned output sink or global logger fallback |
| sequence | Atomic unsigned integer, allocated once per observed exchange, starting at 1 |
| emission | Existing verbose INFO helper and synchronized logging writer; honor configured levels and format |
| profile | Optional resolved active profile identity; absent when unset |
| tenant | Optional resolved configured tenant selection; absent when unset; not inferred resource ownership |
| redaction context | Private known credential values from configuration and sensitive request/response metadata, plus sanitization policy; never serialized |

One collector serves the invocation's API, cookie-auth and separate OAuth token traffic. Absence of a collector means disabled diagnostics, with no observer attached. Two invocations never share a collector, sequence or logger binding.

## Exchange observation

| Field | Type / rule |
| --- | --- |
| sequence | Unique within collector, allocated at delegate entry |
| started | Monotonic start for elapsed timing; message carries no duplicate timestamp, logger owns emission time |
| request identity | Method and URL components sanitized before serialization |
| trace state | Connection reuse, matched phase intervals and observed failure-phase evidence |
| final headers | Optional monotonic timestamp immediately after delegate returns final response headers; informational callbacks do not set it |
| terminated | Monotonic timestamp at observed terminal event; equals final headers for a known bodyless response |
| failure phase / reason | Optional technical phase and bounded safe reason enum from the contract; omit unsupported attribution |
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

A phase sample contains phase kind (DNS/connect/TLS), a paired start/end and nonnegative duration. Multiple samples are retained; connection attempts are paired by network/address internally. End callbacks may represent failed attempts, which still performed measurable work. Unpaired/in-progress samples are omitted. Network addresses are not serialized as extra connection detail. Samples are ordered by their start time, not callback completion order. Emit connect samples as `tcp` only for TCP networks. Single samples are scalars; multiple samples use the contract list grammar.

Absent hooks or unperformed phases are absent fields. A genuine observed zero-duration sample is distinct from absence. Samples may overlap and must not be summed into a supposed breakdown of total duration. Reuse is optional and comes from connection evidence, not the absence of a handshake alone.

## Immutable diagnostic record

One terminal snapshot links to its collector by sequence and contains only sanitized identities, optional observations and completeness evidence. The exact external field names, value grammar and omission rules are in [contracts/api-diagnostics.md](contracts/api-diagnostics.md).

Validation rules:

- All durations and counts are nonnegative. Sequence is positive and unique per invocation.
- Unknown fields are omitted, not filled with zero, empty strings or fabricated defaults.
- HTTP error statuses retain their real status; transport error categories do not manufacture HTTP statuses.
- `headers = final headers - started`; `body = terminated - final headers`; `total = terminated - started`. Before formatting, total equals headers plus body when a response exists. Informational responses do not set final headers. Before-header failures omit headers/body. Known bodyless responses have body=0s.
- A non-EOF body error can coexist with a real HTTP status and incomplete response evidence.
- Operational profile/tenant context is selection context; no body parsing or extra tenant lookup occurs.
- Secret matching/redaction applies before any string is serialized. Bodies and raw errors never enter a record.
- Escape controls/newlines so a record occupies exactly one physical line.

## Lifecycle

```text
allocated → delegated
  ├─ transport error ───────────────────────→ terminal snapshot → logger emission attempted
  ├─ known no response body ────────────────→ terminal snapshot → logger emission attempted
  └─ response available → caller reads/closes
       ├─ EOF ─────────────────────────────→ complete snapshot → logger emission attempted
       ├─ non-EOF read error ───────────────→ incomplete snapshot → logger emission attempted
       └─ early Close ─────────────────────→ incomplete snapshot → logger emission attempted
```

Once frozen, later Close calls and trace callbacks cannot emit another record or change the snapshot. Underlying Read and Close return values and ownership remain unchanged. A diagnostic write error does not alter the HTTP result and is not retried on stdout. An abandoned body without an observed terminal event remains unfinalized; diagnostics neither drain it nor invent a completion timer. State is released with the ordinary request/body lifecycle.
