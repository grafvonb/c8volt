# Data Model: Push Supported Get Filters Into Search Requests

## Process-Instance Search Filter

- **Purpose**: Represents the shared request intent that flows from CLI flags through the public process facade into versioned process-instance services.
- **Existing attributes**:
  - `Key`
  - `BpmnProcessId`
  - `ProcessVersion`
  - `ProcessVersionTag`
  - `ProcessDefinitionKey`
  - `StartDateAfter`
  - `StartDateBefore`
  - `EndDateAfter`
  - `EndDateBefore`
  - `State`
  - `ParentKey`
- **Planned additions**:
  - `HasParent` as an optional `*bool` presence semantic distinct from explicit `ParentKey`
  - `HasIncident` as an optional `*bool` presence semantic for incident filtering
- **Invariants**:
  - `ParentKey` means exact parent equality and must not be overloaded to mean parent presence or absence.
  - `HasParent` and `HasIncident` are optional so unchanged callers preserve current request shapes.
  - `HasParent` and explicit `ParentKey` may coexist only when the combination is semantically consistent; the CLI should continue rejecting contradictory flag combinations before the filter reaches the service layer.

## Filter Capability Record

- **Purpose**: Captures whether a filter semantic is request-capable on the active version or must remain client-side.
- **Fields**:
  - Filter semantic name
  - Active runtime version
  - Capability state: `RequestSide` or `ClientFallback`
  - Request encoding note
- **Current mapping**:
  - `roots-only`: `ClientFallback` on `v8.7`, `RequestSide` on `v8.8` and `v8.9`
  - `children-only`: `ClientFallback` on `v8.7`, `RequestSide` on `v8.8` and `v8.9`
  - `incidents-only`: `ClientFallback` on `v8.7`, `RequestSide` on `v8.8` and `v8.9`
  - `no-incidents-only`: `ClientFallback` on `v8.7`, `RequestSide` on `v8.8` and `v8.9`
  - `orphan-children-only`: `ClientFallback` on all versions

## Process-Instance Search Request

- **Purpose**: The versioned JSON request body sent before a page is fetched.
- **Relevant fields**:
  - Tenant filter from config/bootstrap
  - Process-definition selectors
  - Date bounds
  - State
  - Parent-key equality
  - Parent-key presence/absence when supported
  - Incident presence/absence when supported
  - Pagination metadata
- **Invariants**:
  - Supported versions must encode request-side capability fields directly into this request.
  - Unsupported versions must omit unsupported request-side capability fields rather than approximating them incorrectly.

## Search Result Page

- **Purpose**: Represents the current fetched page plus overflow state used by interactive and automation paging flows.
- **Fields**:
  - `Request`
  - `Items`
  - `OverflowState`
- **Behavioral rules**:
  - On supported versions, page counts should already reflect roots/children/incidents narrowing before local rendering.
  - On fallback versions, local filtering may still reduce visible items after fetch, and overflow state remains bounded by current version limitations.

## Client-Side Fallback Filter

- **Purpose**: Represents a post-fetch narrowing step retained only when the active version cannot express a filter server-side.
- **Members**:
  - root-only fallback
  - children-only fallback
  - incidents-only fallback
  - no-incidents-only fallback
  - orphan-children-only lookup
- **Invariants**:
  - `orphan-children-only` always stays in this category.
  - Fallback is version-specific and should disappear only where the request builder can prove semantic equivalence.

## Audit Finding

- **Purpose**: Records the outcome of scanning other `get` command families for the same late-filtering pattern.
- **Fields**:
  - Command family
  - Qualifies for same pattern: `Yes` or `No`
  - Request-side equivalent available: `Yes` or `No`
  - Follow-up action
- **Behavioral rules**:
  - A `No` outcome should still be recorded with bounded rationale so the broader audit requirement is explicitly satisfied.
