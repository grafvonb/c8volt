# Research: Add Resource Get Command By Id

## Decision 1: Expose the feature as a new `get resource` subcommand

- **Decision**: Implement the feature as `c8volt get resource --id <id>` under the existing `get` command tree.
- **Rationale**: The repository already groups retrieval workflows under `cmd/get*.go`, and user-facing consistency is a core requirement in both the spec and constitution. A dedicated subcommand keeps help, aliases, and generated docs aligned with existing Cobra patterns.
- **Alternatives considered**:
  - Add a top-level `resource` root command: rejected because it would create a parallel command hierarchy inconsistent with current CLI organization.
  - Extend an existing non-resource `get` command: rejected because it would blur resource boundaries and make help output harder to discover.

## Decision 2: Add single-resource retrieval to the `c8volt/resource` facade

- **Decision**: Extend `c8volt/resource.API` and `c8volt/resource/client.go` with one single-resource retrieval method that maps the internal domain `Resource` into a public `c8volt/resource` model.
- **Rationale**: The internal service capability already exists in `internal/services/resource`, but the public facade exposed through `c8volt.API` does not yet surface it. Adding the method to the existing facade preserves current layering and lets the command depend on the same high-level interface as other CLI flows.
- **Alternatives considered**:
  - Call `internal/services/resource` directly from `cmd/`: rejected because it would bypass the existing facade boundary used by other commands.
  - Reuse an unrelated process facade: rejected because resource lookup is a resource concern and already has an internal service area.

## Decision 3: Treat `200 OK` with no resource payload as a malformed-response error

- **Decision**: Preserve current resource-service semantics and treat a success status without a JSON payload as an error, not as empty success and not as synthetic not-found.
- **Rationale**: Both supported versioned resource services already use `internal/services/common.RequirePayload`, which normalizes nil success payloads into malformed-response errors. Keeping that behavior avoids silent false positives and preserves service-layer consistency.
- **Alternatives considered**:
  - Treat empty success as not found: rejected because transport-level `200 OK` is not equivalent to an authoritative not-found response.
  - Allow empty success with no output: rejected because it would violate the constitution’s operational-proof standard and make the CLI harder to validate.

## Decision 4: Return the normal single-resource object/details view

- **Decision**: Successful lookups should render the normal single-resource details/object output rather than raw resource content.
- **Rationale**: The clarification session already bounded the feature to metadata/details lookup and explicitly excluded raw content retrieval. This keeps the command aligned with the issue’s scope and with other single-item `get` commands.
- **Alternatives considered**:
  - Return raw resource content: rejected because raw-content export is explicitly out of scope for this feature.
  - Support both details and raw content: rejected because it would widen the command contract, tests, and docs without issue support.

## Decision 5: Document the public behavior with a CLI contract and generated docs

- **Decision**: Create a spec-side CLI contract and treat `make docs` regeneration as part of the implementation path.
- **Rationale**: This feature introduces a new public CLI workflow, so stable flag semantics, output behavior, and discoverability need to be captured before task breakdown. The repository already keeps `docs/cli/` derived from Cobra metadata.
- **Alternatives considered**:
  - Skip a contract because the feature is small: rejected because the command is user-visible and public behavior is the core of the feature.
  - Hand-edit `docs/cli/`: rejected because repository guidance requires updating help text first and regenerating docs.
