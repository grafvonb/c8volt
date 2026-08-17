# Research: Native Camunda 8.10 Embedded Process Definitions

## Native fixture source and allowed differences

**Decision**: Use the eight checked-in C89 production definitions as the exact behavioral source. Each C810 counterpart changes only the filename, process ID and name, owning diagram reference, cross-process references, and `modeler:executionPlatformVersion` from `8.9.0` to `8.10.0`.

**Rationale**: The requested feature is version ownership, not workflow redesign. A controlled copy keeps element IDs, task settings, mappings, incident behavior, call propagation, version tag, and diagram coordinates visibly stable.

**Alternatives considered**: Creating new models from scratch risks behavioral and layout drift. Adding a BPMN generator is disproportionate for eight fixed assets and belongs to the separately scoped generator-modernization work.

## Modeler exporter metadata

**Decision**: Preserve each source file's `exporterVersion`. The implementation does not perform a Camunda Modeler round-trip.

**Rationale**: The existing value records the Modeler version that produced the source XML. Controlled identity changes do not justify fabricating a newer export or allowing a tool round-trip to rewrite unrelated XML.

**Alternatives considered**: Setting every file to an assumed current Modeler version would be inaccurate. Re-saving solely to force a common version could rewrite unrelated XML and weaken the parity guarantee.

## Production fixture selection ownership

**Decision**: Keep `toolx.ProductionFixturePrefix` as the single mapping used by embedded commands and smoke-test selection, and change only its V810 result from `C89_` to `C810_`.

**Rationale**: `cmd/embeddedFilesForCamundaVersion` and `internal/services/ops/smokeTestFixtureForVersion` already consume this function. One mapping correction updates list, `--all` export/deploy, and smoke workflows without parallel selection logic.

**Alternatives considered**: Special-casing V810 in each consumer duplicates policy and can drift. Extending integration selection through `CamundaVersion.FilePrefix` would broaden scope into the explicitly excluded live integration harness.

## Fixture verification

**Decision**: Add a repository test that enumerates exactly the eight expected C810 files, parses them as well-formed XML, rejects C89 references, and compares each with its C89 counterpart after normalizing the explicitly allowed identity and platform-version differences.

**Rationale**: Normalized whole-file equivalence is stronger and simpler than separately asserting every element, mapping, task setting, and diagram coordinate. Exact inventory assertions also catch missing or accidental extra C810 production files.

**Alternatives considered**: Snapshotting only hashes cannot explain allowed version differences. A large semantic BPMN comparison framework adds unnecessary machinery and can miss serialization details already protected by normalized equality.

## CLI and smoke-test contract

**Decision**: Preserve command, flag, output-envelope, and workflow semantics. Prove the shared embed selector returns the exact C810 family, prove smoke selection and its dependency closure, and assert the C810 identity on one CLI smoke output path while retaining dedicated V89 coverage for C89.

**Rationale**: The user-visible change is the selected fixture identity, not a new interface. Existing behavior already routes through the shared mapping.

**Alternatives considered**: Changing help text or output schemas is unnecessary because current documentation describes version-matched resources generically.

## Stable and integration boundaries

**Decision**: Do not edit C87, C88, C89, `integration/`, or integration Make targets. Validate those boundaries with diffs against the #273 feature baseline and run repository tests without starting a live integration suite.

**Rationale**: Native C810 production resources are additive. The issue explicitly excludes live 8.10 certification and stable-fixture changes.

**Alternatives considered**: Adding an 8.10 profile or copying C810 files into integration would turn this small product correction into release-certification work.

## Issue #273 artifact correction

**Decision**: Update the normative #273 specification, plan, research, data model, quickstart, and service-compatibility contract to replace active C89-reuse requirements with native C810 selection. Add a clear supersession note to #273 `tasks.md`, but preserve its checked task descriptions, progress, and Ralph-memory files as historical execution records.

**Rationale**: Future planning and review must see one final decision, while historical logs should not be rewritten as though the original work happened differently.

**Alternatives considered**: Leaving the old normative decision creates contradictory requirements. Rewriting checked task descriptions or historical logs would falsely describe what #273 originally implemented and destroy useful provenance.
