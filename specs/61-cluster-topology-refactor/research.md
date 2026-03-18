# Research: Refactor Cluster Topology Command

## Decision 1: Introduce a real `get cluster` parent command

- **Decision**: Add a dedicated `cluster` Cobra command under `get` and attach `topology` beneath it instead of trying to simulate the nested hierarchy through aliases alone.
- **Rationale**: The issue explicitly asks for `c8volt get cluster topology`, and a real parent command matches existing repository-native Cobra patterns where nouns hang off action roots such as `get`, `run`, and `delete`.
- **Alternatives considered**:
  - Keep only `cluster-topology` and add aliases that resemble nested words: rejected because aliases do not create the requested help hierarchy.
  - Create a separate top-level `cluster` root: rejected because it breaks the established `get <resource>` command organization.

## Decision 2: Reuse one execution path for both commands

- **Decision**: Keep the current topology retrieval logic in a single handler and have both the nested and legacy command entries route to that same behavior.
- **Rationale**: Shared execution is the safest way to preserve output, error handling, inherited flags, and exit semantics across both command paths.
- **Alternatives considered**:
  - Copy the command implementation into two command definitions: rejected because duplicate logic creates drift risk.
  - Move logic into a new abstraction layer just for command reuse: rejected because a small shared handler or command-construction helper is enough.

## Decision 3: Restrict deprecation messaging to help and documentation

- **Decision**: Mark `cluster-topology` as deprecated in help text and generated documentation only, without printing a runtime warning during successful execution.
- **Rationale**: The clarification session resolved that existing scripts should remain quiet at runtime while still allowing humans to discover the preferred replacement through help and docs.
- **Alternatives considered**:
  - Print a warning on every invocation: rejected because it changes script output and contradicts the accepted clarification.
  - Omit deprecation messaging entirely: rejected because the new preferred path would be harder to discover.

## Decision 4: Validate at command level and preserve service-level regression coverage

- **Decision**: Add or update focused tests in `cmd/` for command wiring and help behavior, while relying on existing cluster service tests to guard the topology retrieval implementation itself.
- **Rationale**: The behavioral change is in the CLI surface, not the cluster service internals, so the most useful new coverage is where Cobra wiring and inherited command behavior are exercised.
- **Alternatives considered**:
  - Change only service tests: rejected because they do not prove the public command tree changed correctly.
  - Rely on manual CLI checks alone: rejected because repository policy requires automated validation and `make test`.

## Decision 5: Regenerate CLI docs and conditionally update README examples

- **Decision**: Treat generated CLI docs as mandatory outputs for this feature and update `README.md` only if it contains cluster-topology usage that would otherwise become stale.
- **Rationale**: This is a user-visible command-hierarchy change, so generated command docs must match shipped behavior. README changes should stay scoped to actual affected examples.
- **Alternatives considered**:
  - Skip docs because runtime behavior is unchanged: rejected because the public command hierarchy is changing.
  - Rewrite broad documentation unrelated to topology: rejected because it adds unnecessary churn.
