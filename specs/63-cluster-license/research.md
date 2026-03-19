# Research: Add Cluster License Command

## Decision 1: Add `license` as a real child of `get cluster`

- **Decision**: Implement the new command as `c8volt get cluster license` beneath the existing `get cluster` parent command.
- **Rationale**: The clarified spec explicitly chooses the nested command only, and a real child command preserves the repository's current `get <resource>` Cobra hierarchy and help discovery behavior.
- **Alternatives considered**:
  - Add a direct `c8volt get cluster-license` command: rejected because the clarification session explicitly ruled out a legacy or alternate direct path.
  - Hang `license` directly under `get`: rejected because cluster-specific read operations are already grouped under `get cluster`.

## Decision 2: Reuse the existing cluster service and domain payload

- **Decision**: Route the new command directly to `cli.GetClusterLicense(ctx)` and print the returned `domain.License` value through the same JSON helper pattern used by neighboring commands.
- **Rationale**: The cluster service interface and versioned implementations already expose cluster license retrieval, so command-level reuse is the safest way to preserve supported-version behavior and avoid duplicate translation logic.
- **Alternatives considered**:
  - Add a command-specific service wrapper: rejected because it would introduce unnecessary abstraction around an already-available API.
  - Create a new output model just for the CLI command: rejected because the existing domain model already represents the required payload shape.

## Decision 3: Validate primarily at the command layer

- **Decision**: Add focused tests in `cmd/get_test.go` for command discovery, successful execution, and failure exit behavior while keeping existing service-level license tests as regression coverage for supported versions.
- **Rationale**: The behavioral change is the addition of a public CLI command, so the highest-value new coverage is at the Cobra wiring layer where help text, inherited flags, output handling, and exit semantics are exercised.
- **Alternatives considered**:
  - Change only service tests: rejected because service tests do not prove the public CLI surface was added correctly.
  - Rely on manual invocation checks alone: rejected because repository policy requires automated validation and `make test`.

## Decision 4: Treat docs as part of the feature, not follow-up work

- **Decision**: Update `README.md` and `docs/index.md` where cluster read examples are documented, and regenerate CLI reference pages with `make docs`.
- **Rationale**: The constitution requires user-facing documentation to match shipped command behavior, and this feature adds a new discoverable command in the public CLI surface.
- **Alternatives considered**:
  - Update only generated CLI docs: rejected because repository-level usage guidance in README and `docs/index.md` would otherwise lag behind the shipped command set.
  - Skip documentation because the service capability already exists: rejected because the CLI entry point itself is new and user-visible.
