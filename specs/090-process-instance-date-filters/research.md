# Research: Day-Based Process Instance Date Filters

## Decision 1: Use native v8.8 search request date filters

- **Decision**: Map the new CLI date flags to native v8.8 process-instance search filter fields using inclusive datetime comparisons.
- **Rationale**: The generated v8.8 Camunda client exposes `startDate` and `endDate` filter fields as `DateTimeFilterProperty`, including `$gte` and `$lte` operators. Using the server-side search API preserves current result-size behavior and avoids client-side overfetch/filter logic.
- **Alternatives considered**:
  - Fetch unfiltered results and post-filter in `c8volt`: rejected because it would be less efficient, could miss matches beyond the current server-side page size, and would diverge from existing search behavior.
  - Add date filtering only in the facade layer: rejected because the v8.8 backend already supports the needed semantics directly.

## Decision 2: Reject date filters on v8.7 instead of emulating them

- **Decision**: When any new date flag is present, return a clear not-implemented error from the v8.7 process-instance search path.
- **Rationale**: The feature spec explicitly limits support to v8.8 and requires a clear not-implemented response on v8.7. Preserving that boundary avoids hidden behavior differences and prevents partial emulation across older Operate search requests.
- **Alternatives considered**:
  - Emulate filtering by fetching and narrowing results locally on v8.7: rejected because it violates the requested version behavior and risks inconsistent result completeness.
  - Silently ignore date filters on v8.7: rejected because it would be misleading and unsafe for scripted use.

## Decision 3: Keep validation in the command-layer search path

- **Decision**: Parse and validate the new date flags in `cmd/get_processinstance.go` before the search request is dispatched.
- **Rationale**: Existing flag validation for `get process-instance` already lives in `validatePISearchFlags()`, and the command currently enforces `--key` exclusivity before invoking list/search behavior. Extending that same validation seam keeps user-facing failures consistent and prevents invalid requests from reaching services.
- **Alternatives considered**:
  - Defer parsing and range validation to the service layer: rejected because command-level validation already owns user input constraints and mutual exclusions.
  - Validate only in tests and rely on backend errors: rejected because the spec requires clear validation failures before executing search.

## Decision 4: Represent date filters as day-range bounds in shared filter models

- **Decision**: Extend the shared process-instance filter models with explicit start-date and end-date lower/upper bound fields rather than overloading existing string fields.
- **Rationale**: The command, facade, domain, and versioned services all already pass a typed `ProcessInstanceFilter`. Adding explicit date-bound fields keeps the existing model flow intact and makes version-specific handling straightforward.
- **Alternatives considered**:
  - Store raw flag strings only in the command package: rejected because services need structured access to know whether filters are present and how to map them.
  - Use a generic map for extra filters: rejected because it would be inconsistent with current repository-native typed filter models.

## Decision 5: Treat CLI dates as configured-environment local-day boundaries

- **Decision**: Convert date-only user input into inclusive lower/upper datetime boundaries using the configured Camunda environment’s local calendar day.
- **Rationale**: Clarification for this feature established that date comparisons must follow the environment being queried rather than the workstation running `c8volt`. This avoids off-by-one-day surprises across different operator machines.
- **Alternatives considered**:
  - Always use UTC boundaries: rejected because it can shift day membership relative to the target environment.
  - Use the local timezone of the machine running `c8volt`: rejected because it makes identical commands produce different results across operators.

## Decision 6: Exclude missing `endDate` values whenever end-date filters are present

- **Decision**: Exclude process instances without an `endDate` from any result set narrowed by `--end-date-after` or `--end-date-before`.
- **Rationale**: This matches the clarification decision for the feature and keeps end-date filtering predictable for completed-instance searches.
- **Alternatives considered**:
  - Treat missing `endDate` as matching `before` filters: rejected because it conflates “missing” with a real completed date.
  - Fail the entire request when any candidate lacks `endDate`: rejected because it is unnecessarily disruptive and not aligned with normal filter semantics.

## Decision 7: Limit the new flags to list/search behavior, not direct ID lookup

- **Decision**: Keep the new date filters mutually exclusive with `--key`-based direct lookup.
- **Rationale**: The command already treats `--key` as an alternate fetch mode and rejects mixing key lookup with other filters. Applying the same rule to the new date flags preserves the current command mental model and avoids ambiguous single-instance semantics.
- **Alternatives considered**:
  - Allow date filters with `--key` and revalidate the fetched item: rejected because it complicates a direct fetch path without meaningful user value.
  - Silently ignore date filters with `--key`: rejected because silent drops are unsafe in scripted workflows.

## Decision 8: Update both hand-written and generated CLI documentation

- **Decision**: Update `README.md` examples/help references as needed and regenerate CLI reference pages with `make docs-content` and `make docs`.
- **Rationale**: The constitution and repository guidance require user-visible command changes to be documented in the same unit of work, and CLI reference files are generated from Cobra metadata.
- **Alternatives considered**:
  - Update only Cobra help text: rejected because the repository also treats README and generated docs as user-facing documentation.
  - Hand-edit `docs/cli/` directly: rejected because repository guidance requires regeneration rather than editing generated docs by hand.
