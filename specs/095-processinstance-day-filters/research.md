# Research: Relative Day-Based Process-Instance Date Shortcuts

## Decision 1: Reuse the existing absolute date-filter model as the canonical backend path

- **Decision**: Treat the new `*-days` flags as command-layer convenience inputs that are converted into the existing absolute date filter fields already introduced by issues `#90` and `#93`.
- **Rationale**: The repository already has typed `StartDateAfter`, `StartDateBefore`, `EndDateAfter`, and `EndDateBefore` fields in shared process-instance filter models plus existing v8.8/v8.7 handling in the versioned services. Reusing that model avoids duplicating filter semantics in a second pipeline.
- **Alternatives considered**:
  - Add parallel relative-day fields all the way through the facade and service layers: rejected because it would duplicate logic and diverge from the already-shipped date-filter model.
  - Convert relative-day values only inside versioned services: rejected because command-layer validation already owns user input constraints and cross-command consistency.

## Decision 2: Perform relative-day parsing and conversion inside the shared command search-helper seam

- **Decision**: Extend the existing `cmd/get_processinstance.go` search helper path to parse non-negative day values, derive absolute date bounds, and feed the existing `populatePISearchFilterOpts()` output used by `get`, `cancel`, and `delete`.
- **Rationale**: `validatePISearchFlags()`, `hasPISearchFilterFlags()`, and `populatePISearchFilterOpts()` are already the shared seam for process-instance search inputs across the three affected commands. Adding relative-day conversion there keeps validation uniform and prevents service layers from receiving invalid combinations.
- **Alternatives considered**:
  - Parse each command’s relative-day flags separately: rejected because it would create three behavior paths for the same feature.
  - Convert relative days in the facade layer: rejected because command-level validation and `--key` exclusivity belong closer to the Cobra flag surface.

## Decision 3: Use the configured Camunda environment’s local calendar day for relative-day derivation

- **Decision**: Derive relative-day boundaries using the configured Camunda environment’s local calendar day, matching the clarification and existing absolute date-filter interpretation.
- **Rationale**: The spec explicitly resolves this ambiguity, and issue `#90` already established the environment-local-day rule for absolute date filters. Reusing the same interpretation keeps results consistent across operators and commands.
- **Alternatives considered**:
  - Use the local timezone of the machine running `c8volt`: rejected because identical commands could produce different results for different operators.
  - Use UTC day boundaries: rejected because it can shift day membership relative to the environment being queried.

## Decision 4: Reject mixed absolute and relative filters for the same field at validation time

- **Decision**: Treat any combination of `--start-*-days` with `--start-date-*`, or `--end-*-days` with `--end-date-*`, as a command validation error.
- **Rationale**: The spec requires a clear failure for conflicting filter sources instead of silently choosing one. Command-layer rejection prevents ambiguous request construction and keeps the CLI script-safe.
- **Alternatives considered**:
  - Prefer the absolute filter and ignore the relative shortcut: rejected because silent precedence is unsafe in automation.
  - Allow both if they resolve to the same derived day: rejected because the CLI should not require users to reason about hidden precedence or equivalence.

## Decision 5: Preserve the existing search-only rule for direct `--key` workflows

- **Decision**: Reject any `cancel process-instance` or `delete process-instance` invocation that combines explicit `--key` values with relative day-based flags.
- **Rationale**: The same rule already exists for the earlier absolute date filters, and the clarification for this feature confirmed that relative-day shortcuts should not alter or validate keyed operations. This preserves the current mental model for direct key-based workflows.
- **Alternatives considered**:
  - Ignore relative-day flags when `--key` is present: rejected because silent drops are unsafe and misleading.
  - Apply relative-day checks to fetched keyed instances: rejected because it adds complexity without improving search-based selection behavior.

## Decision 6: Keep missing-`endDate` handling identical to the existing absolute end-date filters

- **Decision**: Exclude process instances with no `endDate` whenever `--end-before-days` or `--end-after-days` is used.
- **Rationale**: The spec clarification explicitly carries forward the existing absolute end-date behavior from issue `#90`, and the v8.8 service already has logic for end-date existence when end-date filters are present.
- **Alternatives considered**:
  - Treat missing `endDate` as matching `before` filters: rejected because it conflates incomplete data with a concrete completed date.
  - Fail the whole request when any candidate lacks `endDate`: rejected because normal filter semantics should narrow results, not make them unusable.

## Decision 7: Cover relative-day behavior at the closest useful test seams

- **Decision**: Add targeted command tests for day parsing, derived-bound composition, invalid combinations, and shared behavior across `get`, `cancel`, and `delete`, while extending lower-level tests only where derived inputs must prove they land on the canonical absolute date path.
- **Rationale**: Most observable behavior changes at the CLI seam. Lower-level services already cover absolute date-filter semantics, so extra tests should focus on the relative-day conversion path and any changed request expectations.
- **Alternatives considered**:
  - Add only command tests: rejected because a small amount of lower-level coverage may still be needed if new conversion helpers or request-shape changes are introduced.
  - Rebuild full service coverage from scratch: rejected because it would duplicate issue `#90` coverage instead of extending the current regression net.

## Decision 8: Update both README guidance and regenerated CLI reference pages

- **Decision**: Update `README.md` examples and command guidance as needed, then regenerate `docs/cli/` with `make docs-content` and `make docs`.
- **Rationale**: The constitution and repository guidance require user-visible command changes to keep hand-written and generated docs in sync.
- **Alternatives considered**:
  - Update only Cobra help text: rejected because repository documentation policy also covers README and generated CLI pages.
  - Hand-edit files in `docs/cli/`: rejected because those pages are generated artifacts.
