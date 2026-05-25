# Research: Element Terminology Standardization

## Decision: Rename Public Incident Context To Element Terms

**Decision**: Public incident models, filters, JSON, command metadata, and human rendering use `elementId` and `elementInstanceKey`.

**Rationale**: The GitHub issue explicitly defines these as the canonical public names and aligns incident behavior with job terminology and Camunda v2 API language.

**Alternatives considered**: Keep flow-node aliases for compatibility. Rejected because the issue declares the cleanup intentionally breaking and forbids transitional aliases.

## Decision: Rename Public Process Parent Context

**Decision**: Public process-instance parent context uses `parentElementInstanceKey`.

**Rationale**: Process-instance JSON currently exposes parent execution context through old flow-node wording. Renaming completes the public contract cleanup beyond incident-only views.

**Alternatives considered**: Limit the change to incidents. Rejected because issue #233 explicitly includes process-instance surfaces and `parentFlowNodeInstanceKey`.

## Decision: Keep Legacy Generated Names Adapter-Only

**Decision**: Generated `FlowNode*` names may remain in generated clients and version-specific adapter conversions, but must not appear in public c8volt contracts.

**Rationale**: Older generated Camunda/Operate clients still expose legacy wire names. The repository architecture requires generated-client churn to stay below service/facade boundaries.

**Alternatives considered**: Hand-edit generated clients. Rejected because generated clients are derived artifacts and the issue explicitly says not to manually edit them.

## Decision: Update Ops Workflows Through Shared Incident Filters

**Decision**: `ops repair incident` and `ops purge process-instances-with-incidents` should use the same canonical incident filter names as `get incident`.

**Rationale**: Ops workflows compose incident discovery; maintaining separate filter names would violate the issue and the architecture rule that playbooks reuse lower-level resource semantics.

**Alternatives considered**: Rename only the primitive command first. Rejected because it would leave high-level workflows with divergent public contracts.

## Decision: Validate By Absence And Presence

**Decision**: Tests must assert canonical names are present and legacy public names are absent.

**Rationale**: A pure positive test could leave deprecated aliases or stale JSON fields in place. The acceptance criteria require old public flags and fields to be gone.

**Alternatives considered**: Rely on manual search before merge. Rejected because contract cleanup needs automated regression coverage.

## Decision: Document Ralph Context As A Durable Constraint

**Decision**: Carry `--implementation-context specs/ralph-implementation-rules.md` through plan, tasks, and launch instructions.

**Rationale**: The user explicitly required this context for planning, task generation, and every Ralph implementation iteration, and prohibited launching Ralph without it.

**Alternatives considered**: Mention the file only in chat handoff. Rejected because Ralph needs the requirement embedded in durable artifacts.
