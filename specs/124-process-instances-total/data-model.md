# Data Model: Add Process-Instance Total-Only Output

## Get-Process-Instance Count Request

- **Purpose**: Represents the existing `get process-instance` search/list invocation when the caller wants only the count of matching process instances.
- **Fields**:
  - existing process-instance search filters (`state`, process-definition selectors, date bounds, parent filters, incident filters)
  - `TotalOnly` command flag intent from `--total`
- **Invariants**:
  - `TotalOnly` is only valid for search/list mode and must not coexist with direct `--key` lookup intent.
  - `TotalOnly` is output-focused and must not change filter semantics or matching rules.
  - `TotalOnly` must reject conflicting detail-output modifiers such as `--json`, `--keys-only`, and `--with-age`.

## Process-Instance Search Result Page

- **Purpose**: Carries one fetched page of process instances plus the metadata the command needs for paging and count-only output.
- **Existing fields**:
  - `Request`
  - `Items`
  - `OverflowState`
- **Planned additions**:
  - `ReportedTotal` as an optional metadata record with `Count` and `Kind`
- **Invariants**:
  - `ReportedTotal` should be version-agnostic at the command boundary even if each service version derives it differently.
  - `ReportedTotal=nil` means no trustworthy backend total is available for the current page.
  - `ReportedTotal.Kind` must be `Exact` or `LowerBound`; the unavailable state is represented by `ReportedTotal=nil`.
  - `OverflowState` remains the paging-control signal; it is not sufficient by itself for `--total`.
  - `ReportedTotal` may be a lower bound and must not be silently promoted to an exact total.

## Reported Total Semantics

- **Purpose**: Normalizes how count-only mode interprets backend totals across `v8.7`, `v8.8`, and `v8.9`.
- **States**:
  - `Exact`: `ReportedTotal.Kind=Exact`, so the reported total is authoritative for the current search
  - `LowerBound`: `ReportedTotal.Kind=LowerBound`, so the reported total is useful but the backend marked it as capped
  - `Unavailable`: `ReportedTotal=nil`, so no trustworthy backend-reported total is available
- **Behavioral rules**:
  - `--total` prints the numeric total for both `Exact` and `LowerBound`.
  - `LowerBound` must preserve the reported numeric value unchanged.
  - `Unavailable` should trigger the version-appropriate fallback defined in implementation tasks, but must not weaken the count-only contract into detail output.

## Count-Only Output

- **Purpose**: The final user-visible result when `--total` is accepted.
- **Fields**:
  - numeric count value only
- **Invariants**:
  - No instance-detail rows may be emitted.
  - Zero matches must print `0`.
  - Count-only output is distinct from JSON envelopes, keys-only streams, and one-line detail rendering.

## Validation Record

- **Purpose**: Captures invalid combinations that must be rejected before command execution continues.
- **Fields**:
  - conflicting flag set
  - rejection reason
- **Required combinations**:
  - `--total` + `--key`
  - `--total` + `--json`
  - `--total` + `--keys-only`
  - `--total` + `--with-age`

## Documentation Contract Input

- **Purpose**: Defines the user-visible command behavior that README and generated CLI docs must describe.
- **Fields**:
  - flag name: `--total`
  - summary: returns only the number of matching process instances
  - caveat: backend-reported capped totals remain lower-bound numeric results
  - incompatibility notes for rejected flag combinations
