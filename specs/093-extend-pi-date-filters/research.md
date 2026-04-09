# Research: Extend Process-Instance Management Date Filters

## Decision 1: Reuse the issue `#90` shared filter and service support

- **Decision**: Keep the new work scoped to the management command surfaces and reuse the existing shared process-instance filter fields plus the versioned v8.8/v8.7 service behavior already added for issue `#90`.
- **Rationale**: The repository already contains `StartDateAfter`, `StartDateBefore`, `EndDateAfter`, and `EndDateBefore` in the shared facade/domain models, along with v8.8 date-filter mapping and v8.7 unsupported handling. Reusing that path keeps the follow-up change small and repository-native.
- **Alternatives considered**:
  - Re-implement date-filter mapping separately for cancel/delete: rejected because it would duplicate working issue `#90` behavior.
  - Add management-only filter structures in `cmd/`: rejected because it would create a parallel model path for the same process-instance search semantics.

## Decision 2: Add the missing date flags directly to cancel/delete command surfaces

- **Decision**: Register the same four date flags on `c8volt cancel process-instance` and `c8volt delete process-instance` using the existing shared flag variables and help text already defined in `cmd/get_processinstance.go`.
- **Rationale**: The management commands already call `hasPISearchFilterFlags()` and `populatePISearchFilterOpts()` when no explicit keys are given, but they do not currently expose the date flags themselves. Extending the existing command surfaces closes the gap with minimal code movement.
- **Alternatives considered**:
  - Create new command-local date flag variables for cancel/delete: rejected because it would duplicate shared flag state and increase drift risk.
  - Add a new helper layer just for management commands: rejected because the existing `get` command helpers already define the canonical process-instance search filter surface.

## Decision 3: Reuse shared command validation instead of relying on backend failures

- **Decision**: Route cancel/delete through the same date validation helpers used by `get process-instance`, including format validation and independent range validation for start and end date bounds.
- **Rationale**: The spec requires failures before search execution, and the repository already has tested validation helpers in the command layer. Reusing them keeps user-facing error style consistent and avoids backend-only surprises.
- **Alternatives considered**:
  - Let services reject malformed or unsupported filters: rejected because user input validation already belongs in the Cobra command layer.
  - Copy validation logic into each management command: rejected because it increases maintenance burden and risks inconsistent messages.

## Decision 4: Reject `--key` combined with date filters for management commands

- **Decision**: Treat any explicit `--key` usage combined with one or more `--start-date-*` or `--end-date-*` flags as an invalid command for cancel/delete.
- **Rationale**: This matches the clarification captured in the feature spec and preserves script safety by avoiding silent or ambiguous narrowing semantics on keyed operations.
- **Alternatives considered**:
  - Ignore date filters when `--key` is present: rejected because silent drops are unsafe and misleading.
  - Revalidate keyed instances against the date bounds: rejected because it complicates direct key workflows without adding meaningful operational value.

## Decision 5: Keep search-based selection logic centralized in existing helpers

- **Decision**: Continue to use `hasPISearchFilterFlags()` and `populatePISearchFilterOpts()` as the central composition points for search-driven process-instance management commands.
- **Rationale**: Cancel/delete already rely on these helpers to decide whether they may search for target instances. Centralizing the filter shape there keeps `get`, `cancel`, and `delete` aligned as the search surface evolves.
- **Alternatives considered**:
  - Inline bespoke search-filter assembly in each management command: rejected because drift between commands would become likely.
  - Move all search-flag registration into a new shared command factory: rejected for this feature because it is broader than needed and not required to deliver the user-visible outcome.

## Decision 6: Expand tests at the command seam and rely on existing versioned service coverage

- **Decision**: Add targeted cancel/delete command tests for new flags and invalid combinations, while keeping the deeper versioned date-filter semantics covered by the existing issue `#90` service tests unless a gap is discovered.
- **Rationale**: The new behavior introduced by issue `#93` is primarily command-surface exposure and validation. Existing v8.7/v8.8 service behavior is already covered for the shared filter path, so the highest-value new tests sit at the command seam.
- **Alternatives considered**:
  - Rebuild all date-filter coverage in new service tests: rejected because it duplicates existing coverage rather than validating the new integration point.
  - Rely only on manual smoke checks: rejected because the constitution requires automated validation for changed behavior.

## Decision 7: Update both README examples and generated CLI docs

- **Decision**: Update hand-written user-facing examples in `README.md` and regenerate CLI reference pages after adjusting Cobra metadata for cancel/delete.
- **Rationale**: The constitution and repository conventions require user-visible command behavior to be documented in the same unit of work, and `docs/cli/` is generated from command metadata.
- **Alternatives considered**:
  - Update only Cobra help text: rejected because repository guidance explicitly keeps README and generated docs in sync with command behavior.
  - Hand-edit `docs/cli/` pages: rejected because those files are generated and should be refreshed via the repo’s documentation commands.
