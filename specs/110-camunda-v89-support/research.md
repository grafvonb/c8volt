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

### User Story 2 update

- User Story 1 removed the need for any active fallback in the repository's versioned service families: `cluster`, `processdefinition`, `processinstance`, and `resource` now all execute on native `v89` services.
- The remaining fallback guidance is therefore purely a maintenance guard for future incomplete work, not an active description of the current `v8.9` runtime path.

## Decision 10: Make documentation updates part of the feature’s release gate

- **Decision**: README, docs homepage content, and generated CLI docs must all be updated before the feature is considered complete.
- **Rationale**: The clarification session explicitly made docs part of release readiness. This is especially important here because the current user-facing documentation still says `8.9` is recognized but not fully supported.
- **Alternatives considered**:
  - File follow-up doc work after code lands: rejected because it would leave the runtime truth and operator guidance out of sync.
  - Update only developer-facing docs: rejected because the version-support claim is user-facing.

## Final Runtime Boundary Inventory

### Version source of truth

- [`toolx/version.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/toolx/version.go) now normalizes `8.9` and advertises `8.7`, `8.8`, and `8.9` through `SupportedCamundaVersions()`.
- [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go), [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md), `docs/index.md`, and generated `docs/cli/c8volt.md` now present `v8.9` as supported runtime coverage with the same repository command-family scope already available on `v8.8`.

### Repository-wide `v8.9` parity inventory

- [`README.md`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/README.md) and [`cmd/root.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/cmd/root.go) now advertise the operator-facing command families supported on `v8.9`.
- The repository parity boundary shipped for `v8.9` is:
  - Cluster metadata: `get cluster topology`, `get cluster license`
  - Process-definition discovery: `get process-definition` search/latest/XML/statistics
  - Resource lifecycle: `deploy process-definition`, `delete process-definition`, `get resource`
  - Process-instance lifecycle and traversal: `run`, `get process-instance`, `walk`, `expect`, `cancel`, `delete`
- No implementation or verification evidence revealed a hidden command family outside those four runtime-backed areas; additional commands such as config or embed remain outside the versioned service parity boundary defined by the issue/spec.

### Versioned factory boundary

- [`internal/services/cluster/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/cluster/factory.go), [`internal/services/processdefinition/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processdefinition/factory.go), [`internal/services/processinstance/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/processinstance/factory.go), and [`internal/services/resource/factory.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/services/resource/factory.go) now route `v8.7`, `v8.8`, and `v8.9` to version-local implementations.
- Factory tests remain the primary seam proving `v89` selection plus preserved older-version behavior.

### Service-family client boundary

- `cluster/v89`, `processdefinition/v89`, `processinstance/v89`, and `resource/v89` all now depend only on the generated `internal/clients/camunda/v89/camunda/client.gen.go` surface.
- `processinstance/v89` closed the main design-sensitive gap by replacing the earlier mixed-client `v88` pattern with a version-local Camunda-only implementation for search, lookup, cancel, delete, walker, and waiter behavior.
- Verification for this feature therefore treats any future mixed-client `v8.9` behavior as a regression rather than an accepted transition state.

### Repository command-family boundary

- Cluster metadata: `get cluster topology`, `get cluster license`
- Process-definition discovery: `get process-definition` search/latest/xml flows
- Resource lifecycle: `deploy process-definition`, `delete process-definition`, `get resource`
- Process-instance lifecycle and traversal: `run`, `get process-instance`, `walk`, `expect`, `cancel`, `delete`

These command families are the user-facing proof surface for repository-wide `v8.8` parity on `v8.9`.

## Final Implementation Findings

### T027: Final support boundary and feature records

- The repository now presents one consistent `v8.9` runtime-support contract across shared version helpers, service factories, top-level client wiring, command help, README content, and generated docs.
- All four versioned service families now expose native `v89` implementations through their existing factories, so `v8.9` is no longer a normalization-only or docs-only state anywhere in the shipped runtime path.
- The feature closes without any active temporary fallback inside the repository's versioned service architecture; the documented fallback language remains only as a future-change guardrail.
- Final release readiness for this feature depends on the focused validation set plus the repository gate `make test`, not on any remaining implementation gap.

### Final generated-client boundary record

- The generated [`internal/clients/camunda/v89/camunda/client.gen.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/internal/clients/camunda/v89/camunda/client.gen.go) already exposes the main endpoints needed to build the final native `v89` runtime path:
  - Cluster: `GetTopology`
  - Process definition: `GetProcessDefinition`, `GetProcessDefinitionXML`, `SearchProcessDefinitions`, `GetProcessDefinitionStatistics`
  - Resource: `CreateDeploymentWithBody`, `GetResource`, `GetResourceContent`, `DeleteResourceOp`
  - Process instance: `GetProcessInstance`, `SearchProcessInstances`, `CancelProcessInstance`, `DeleteProcessInstance`, `GetProcessInstanceCallHierarchy`, `GetProcessInstanceStatistics`
- The generated client also includes the batch process-instance operations, which keeps future paging or bulk flows inside the same `v89` Camunda boundary if later iterations need them.
- Final implementation confirmed there was no missing endpoint that required permanent fallback to `v8.8` or permanent dependence on `operate` for accepted `v89` behavior.

### Final verification seam record

- [`c8volt/client.go`](/Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/c8volt/client.go) is already the sole top-level runtime wiring seam: it constructs `cluster`, `processdefinition`, `processinstance`, and `resource` through their factories, then wraps them in the public facades. That is the correct place to preserve centralized version selection.
- Each versioned service family already has a focused factory regression suite:
  - `internal/services/cluster/factory_test.go`
  - `internal/services/processdefinition/factory_test.go`
  - `internal/services/processinstance/factory_test.go`
  - `internal/services/resource/factory_test.go`
- Those tests all assert concrete returned service types for supported versions and `services.ErrUnknownAPIVersion` for unsupported ones, so they remain the right seam for keeping `v8.9` support and preserved `v8.7`/`v8.8` behavior honest over time.
- The command-layer regression seam remains the `cmd/*_test.go` suites identified in `tasks.md`; final implementation kept parity proof there rather than adding version-specific branching to command code.
