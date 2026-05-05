# Research: Process Instance Variable Output

## Decision: Model variable enrichment separately from base process instances

**Rationale**: Existing incident enrichment uses a dedicated enriched response shape instead of mutating the base process-instance model. A `VariableEnrichedProcessInstances` shape keeps default `get pi` output stable, keeps JSON explicit, and lets command rendering decide how to display variable lines without changing ordinary process-instance search results.

**Alternatives considered**: Adding variables to `ProcessInstance.Variables` was rejected because that field already represents process creation/input-style variables in the facade model and would make enrichment indistinguishable from base process-instance data.

## Decision: Query variables by both `processInstanceKey` and `scopeKey`

**Rationale**: The issue requires process-instance-level variables only. Filtering both fields to the selected process-instance key excludes local variables scoped to tasks, subprocesses, gateways, events, or other elements while still allowing variables owned by the process instance.

**Alternatives considered**: Fetching all variables for `processInstanceKey` and filtering only in the CLI was rejected because it fetches unnecessary element-scoped data and makes the service contract less precise.

## Decision: Sort variables by name in the service/facade boundary and preserve that order in renderers

**Rationale**: Stable ordering is required for human scanability, JSON consumers, and deterministic tests. Sorting after conversion keeps behavior independent from backend ordering and lets renderers stay simple.

**Alternatives considered**: Relying only on API sort was rejected because local filtering/conversion and backend behavior may vary across versions.

## Decision: Use `--var-value-limit <chars>` as an opt-in human display limit

**Rationale**: Clarification selected no default CLI shortening. A numeric limit follows the repository's existing `--incident-message-limit` style, is easy to test, and leaves full returned values visible by default. CLI shortening is human-only and must mark `cli-truncated` when applied.

**Alternatives considered**: A default display limit, a boolean `--full-vars`, or always-unlimited output were considered. The clarified contract keeps unlimited output by default while still giving operators an explicit compacting control.

## Decision: Use focused raw response decoding if generated variable result structs omit value metadata

**Rationale**: v8.8/v8.9 generated clients expose `/variables/search`, filters, sorting, and response envelopes, but the inspected generated `VariableResultBase` exposes identity/scope fields and omits the returned `value` and truncation state required by this feature. A small raw decoder at the service boundary can preserve repository service patterns while capturing the fields needed by the domain model.

**Alternatives considered**: Full generated-client regeneration may be appropriate if the repository already has a trusted generator path ready for this API surface, but it is broader than necessary for this feature. Ignoring value/truncation metadata would violate the issue.

## Decision: Handle API-version support explicitly

**Rationale**: The v8.8/v8.9 Camunda client surface includes `/variables/search`. The v8.7 process-instance service does not expose an equivalent generated Camunda variable search method in its current contract. The implementation must either add a version-appropriate v8.7 lookup using its available client surface or return a clear unsupported error when `--with-vars` is used with v8.7.

**Alternatives considered**: Silently returning no variables for unsupported versions was rejected because it would look like a successful empty process-instance variable list.
