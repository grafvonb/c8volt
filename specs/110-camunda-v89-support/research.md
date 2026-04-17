# Research: Add Camunda v8.9 Runtime Support

## Decision 1: Treat repository-wide `v8.8` command-family support as the parity boundary for `v8.9`

- **Decision**: The `v8.9` feature scope is every repository command family already supported on `v8.8`, not only the command areas that currently have the most tests or the easiest factories.
- **Rationale**: The clarified spec explicitly chose repository-wide parity as the acceptance boundary, which keeps the feature honest and prevents hidden exclusions from appearing later in planning.
- **Alternatives considered**:
  - Limit parity to command families already covered by `v8.8` regression suites: rejected because it would let unsupported gaps hide behind test coverage gaps.
  - Defer command-family scope choices to implementation: rejected because it would make the acceptance target unstable.

## Decision 2: Expand supported-version truth from the shared version helpers outward

- **Decision**: Add `v8.9` to `toolx.SupportedCamundaVersions()` and then flow that truth through root/bootstrap messaging, docs, and service factories.
- **Rationale**: `NormalizeCamundaVersion()` already accepts `8.9`, but user-visible support lists and factories still stop at `v8.8`. Updating the shared version helpers first gives one authoritative runtime support boundary.
- **Alternatives considered**:
  - Only update docs and root help text: rejected because factories and version metadata would still contradict the docs.
  - Only update factories: rejected because users would still see stale “supported versions” messaging.

## Decision 3: Keep the existing versioned service/facade structure and add parallel `v89` packages

- **Decision**: Add `internal/services/{cluster,processdefinition,processinstance,resource}/v89` packages that mirror the current `v88` structure, API assertions, constructor shape, and common helper usage.
- **Rationale**: The repository already uses this layout consistently across supported service families, and the constitution favors small, repository-native extensions over new abstraction layers.
- **Alternatives considered**:
  - Build one shared “latest Camunda version” adapter: rejected because it would bypass the established versioned-service contract.
  - Put `v89` logic directly in factory files: rejected because it would duplicate service logic and break the current package organization.

## Decision 4: Final native `v8.9` paths must stay on the generated `v89` Camunda client boundary only

- **Decision**: The final accepted `v89` implementation for cluster, process-definition, process-instance, and resource services must depend only on `internal/clients/camunda/v89/camunda/client.gen.go`; mixed-client internals are allowed only inside documented temporary fallback.
- **Rationale**: The issue text and later clarification both set this boundary explicitly. It gives the feature a clear architectural definition of “native `v8.9` support.”
- **Alternatives considered**:
  - Permit permanent mixing of `v8.8` and `v8.9` generated clients inside final `v89` services: rejected because it weakens the native-runtime claim.
  - Permit permanent use of the `v89` Operate client in final native paths: rejected because the clarified contract chose the `v89` Camunda client as the final client boundary.

## Decision 5: The generated `v89` Camunda client already exposes the process-instance endpoints needed for a native final path

- **Decision**: Plan the `v89` process-instance implementation around the generated Camunda client’s `GetProcessInstance`, `SearchProcessInstances`, `CancelProcessInstance`, `DeleteProcessInstance`, and related endpoints, not around the current `v88` Operate dependency.
- **Rationale**: The generated `internal/clients/camunda/v89/camunda/client.gen.go` includes the critical process-instance operations needed for the final native path, which removes the main technical ambiguity from the process-instance design.
- **Alternatives considered**:
  - Continue relying on Operate for final `v89` deletion or lookup behavior: rejected because it would violate the final client-boundary rule.
  - Assume delete/state flows are impossible without Operate: rejected because the generated `v89` Camunda client now exposes those endpoints directly.

## Decision 6: Use the current factory and top-level client seams as the only version-switching points

- **Decision**: Keep version selection centralized in `internal/services/*/factory.go` and `c8volt/client.go`, with commands continuing to consume facades rather than branching on Camunda version.
- **Rationale**: This preserves the current architecture, minimizes command churn, and keeps version-specific behavior in the service layer where it already lives.
- **Alternatives considered**:
  - Add command-level `if cfg.App.CamundaVersion == v89` branches: rejected because it would fragment version logic and break repository patterns.
  - Add a separate `v89` top-level client constructor: rejected because `c8volt/client.go` already centralizes service wiring.

## Decision 7: Treat four versioned service families as the implementation core of repository-wide parity

- **Decision**: The implementation core is `cluster`, `processdefinition`, `processinstance`, and `resource`; repository command-family parity follows from wiring those services correctly through the existing facades.
- **Rationale**: Those are the current versioned services selected by factories and wired in `c8volt/client.go`. Other command behavior in scope is built on top of them rather than on separate versioned runtime families.
- **Alternatives considered**:
  - Treat docs and config as the main implementation surface: rejected because support claims are empty without the service/runtime layer.
  - Treat every command file as an independent implementation target: rejected because the commands mostly depend on shared facades and services.

## Decision 8: Require one explicit `v8.9` execution test per repository command family, plus factory coverage

- **Decision**: The minimum regression bar is at least one explicit `v8.9` execution test for each repository command family already supported on `v8.8`, plus factory selection tests for the four versioned service families and preserved behavior coverage for `v8.7`/`v8.8`.
- **Rationale**: The clarification session chose this as the test floor because it is concrete enough to prove parity without requiring exhaustive combinatorial command coverage.
- **Alternatives considered**:
  - Only test factories and assume commands follow: rejected because command wiring and output contracts are part of the feature.
  - Only spot-check a few high-traffic commands: rejected because repository-wide parity is the acceptance boundary.

## Decision 9: Keep temporary fallback explicit, narrow, and non-final

- **Decision**: Temporary fallback from `v8.9` to older behavior is allowed only as a documented bridge during implementation and must not remain in the final accepted native path for service families that already follow the versioned-service pattern.
- **Rationale**: The clarified spec allows fallback during rollout but also says final acceptance still requires native `v8.9` services. Planning should therefore treat fallback as a short-lived transition tool, not as an equal design target.
- **Alternatives considered**:
  - Ban fallback entirely: rejected because the clarification session explicitly allowed it as a bridge.
  - Allow indefinite fallback if users cannot tell: rejected because it contradicts the native `v8.9` end-state requirement.

## Decision 10: Make documentation updates part of the feature’s release gate

- **Decision**: README, docs homepage content, and generated CLI docs must all be updated before the feature is considered complete.
- **Rationale**: The clarification session explicitly made docs part of release readiness. This is especially important here because the current user-facing documentation still says `8.9` is recognized but not fully supported.
- **Alternatives considered**:
  - File follow-up doc work after code lands: rejected because it would leave the runtime truth and operator guidance out of sync.
  - Update only developer-facing docs: rejected because the version-support claim is user-facing.

## Current Runtime Boundary Inventory

### Version source of truth

- [`toolx/version.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/toolx/version.go) already normalizes `8.9`, but `SupportedCamundaVersions()` still returns only `8.7` and `8.8`.
- [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go), [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md), `docs/index.md`, and generated `docs/cli/c8volt.md` still describe runtime support as stopping at `v8.8`.

### Repository-wide `v8.8` command-family parity inventory

- [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md) advertises the current operator-facing command families that already work on `v8.8`: cluster metadata inspection, process-definition discovery, resource lifecycle, and process-instance lifecycle/traversal.
- [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) still states that process-instance runtime support is implemented only for `8.7` and `8.8`, so the current root help text is an explicit release-gate surface for this feature.
- The repository parity boundary for `v8.9` therefore remains:
  - Cluster metadata: `get cluster topology`, `get cluster license`
  - Process-definition discovery: `get process-definition` search/latest/XML/statistics
  - Resource lifecycle: `deploy process-definition`, `delete process-definition`, `get resource`
  - Process-instance lifecycle and traversal: `run`, `get process-instance`, `walk`, `expect`, `cancel`, `delete`
- No Phase 1 evidence suggests a hidden command family outside those four runtime-backed areas; additional commands such as config or embed are not part of the versioned service parity boundary defined by the issue/spec.

### Versioned factory boundary

- [`internal/services/cluster/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/factory.go), [`internal/services/processdefinition/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory.go), [`internal/services/processinstance/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go), and [`internal/services/resource/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory.go) all stop at `v87`/`v88`.
- Factory tests already exist for all four service families and are the primary seam for proving `v89` selection plus preserved older-version behavior.

### Service-family client boundary

- `cluster`, `processdefinition`, and `resource` `v88` services already depend only on the generated `v88` Camunda client and therefore map cleanly to the final `v89` rule.
- `processinstance/v88` still mixes `internal/clients/camunda/v88/camunda` and `internal/clients/camunda/v88/operate`, making it the main design-sensitive family for `v89`.
- The generated `internal/clients/camunda/v89/camunda/client.gen.go` surface includes the endpoints needed to move process-instance final behavior back onto a single client boundary.

### Repository command-family boundary

- Cluster metadata: `get cluster topology`, `get cluster license`
- Process-definition discovery: `get process-definition` search/latest/xml flows
- Resource lifecycle: `deploy process-definition`, `delete process-definition`, `get resource`
- Process-instance lifecycle and traversal: `run`, `get process-instance`, `walk`, `expect`, `cancel`, `delete`

These command families are the user-facing proof surface for repository-wide `v8.8` parity on `v8.9`.

## Phase 1 Setup Findings

### T001: Current support boundary and user-facing gap

- The runtime support claim is currently inconsistent by design:
  - `toolx` already accepts `8.9` as a normalized version input.
  - `README.md` and `cmd/root.go` still tell users that runtime support stops at `v8.8`.
  - The four versioned service factories still reject `v8.9`, with `processinstance/factory_test.go` explicitly asserting that `v8.9` is normalized but unsupported at runtime.
- This means the current repository state is not a partial native `v8.9` rollout; it is still a documentation-plus-factory boundary that deliberately caps real runtime support at `v8.8`.

### T002: Generated `v89` Camunda client endpoints required for parity

- The generated [`internal/clients/camunda/v89/camunda/client.gen.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v89/camunda/client.gen.go) already exposes the main endpoints needed to build the final native `v89` runtime path:
  - Cluster: `GetTopology`
  - Process definition: `GetProcessDefinition`, `GetProcessDefinitionXML`, `SearchProcessDefinitions`, `GetProcessDefinitionStatistics`
  - Resource: `CreateDeploymentWithBody`, `GetResource`, `GetResourceContent`, `DeleteResourceOp`
  - Process instance: `GetProcessInstance`, `SearchProcessInstances`, `CancelProcessInstance`, `DeleteProcessInstance`, `GetProcessInstanceCallHierarchy`, `GetProcessInstanceStatistics`
- The generated client also includes the batch process-instance operations, which keeps future paging or bulk flows inside the same `v89` Camunda boundary if later iterations need them.
- Phase 1 did not reveal a missing endpoint that would force permanent fallback to `v8.8` or permanent dependence on `operate` for the final accepted `v89` path.

### T003: Existing factory, client-wiring, and regression-test seams

- [`c8volt/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/client.go) is already the sole top-level runtime wiring seam: it constructs `cluster`, `processdefinition`, `processinstance`, and `resource` through their factories, then wraps them in the public facades. That is the correct place to preserve centralized version selection.
- Each versioned service family already has a focused factory regression suite:
  - `internal/services/cluster/factory_test.go`
  - `internal/services/processdefinition/factory_test.go`
  - `internal/services/processinstance/factory_test.go`
  - `internal/services/resource/factory_test.go`
- Those tests all assert concrete returned service types for supported versions and `services.ErrUnknownAPIVersion` for unsupported ones, so they are the right seam for adding `v89` support while explicitly proving `v8.7`/`v8.8` behavior is preserved.
- The command-layer regression seam remains the `cmd/*_test.go` suites identified in `tasks.md`; Phase 1 confirms that later explicit `v8.9` execution coverage should stay there rather than adding version-specific branching to command code.
