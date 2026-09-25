# Research: User-Task Integration Coverage

## Existing models and versions

**Decision**: Select existing files with `embeddedFixturePrefix` and `selectEmbeddedFixtureBySuffix`. C88, C89 and C810 have both `SimpleUserTaskWithIncident` and `SimpleParentWithIncidentSubprocess`; C87 has the standalone incident model but native user-task features are unsupported.

**Rationale**: All supported incident models map `assert(1, not(hasIncident), ...)` to task-local `incident`; explicit `hasIncident=false` avoids the default incident. Parent models call the corresponding versioned child. Ordinary models lack task input mappings and provide process-only negative cases.

**Alternatives considered**: New BPMN files, dynamic identifier rewriting, arbitrary task-local mutation, and direct API setup are unnecessary and prohibited by the issue. `DoubleUserTask` is sequential, not two initial active tasks. MultipleSubProcessesParent adds optional topology, not required scope coverage.

**Evidence**: `embedded/processdefinitions/C{88,89,810}_SimpleUserTaskWithIncident.bpmn`; corresponding incident-parent and ordinary task models; `integration/cli/deploy_embed_run_test.go`; `integration/cli/volume_seed_test.go`.

## Seed helper compatibility

**Decision**: Add explicit variable support through a small shared start/payload helper or delegation from existing start helpers; existing signatures/default callers may remain wrappers. Merge scenario values with the existing run marker, with the marker authoritative. Preserve count, selector, evidence and retained-data behavior.

**Rationale**: Both `runSeededProcessInstance` and `runVolumeProcessInstances` currently build `--vars` from `runMarkerVars`. The new scenario needs false, numeric values, strings and structured values without changing unrelated incident seeds.

**Alternatives considered**: Changing default incident behavior or broadly refactoring the seed engine would affect unrelated tests. Testing a fixture's constants independently adds little; check actual payload composition and real observable outcomes.

## Scope evidence and readiness

**Decision**: Poll boundedly for the owned task, process variable `incident=99`, effective local `incident=1`, and expected process-only values/scopes. Fail on timeout with the last observations. Then check positive and negative filters. Discover child PI ownership with the existing walk/PI commands, using public output models or existing decoding patterns.

**Rationale**: Waiting only for a task or only for an empty negative search can hide index lag. Called-process variables are copied; parent updates are not live inheritance. Native search and display adapters exist in v88/v89/v810; v87 rejects the feature.

**Alternatives considered**: Hard sleeps, unchecked first-child selection and a C89-only helper dependency are insufficient or unnecessarily restrictive.

## Operators and display

**Decision**: Use numeric local `incident=1` for five operators ($eq, $neq, $exists, $in, $like), basic native wildcard text, and false existence on the known-present local name. Keep missing-variable negative semantics and rich local string/null cases out of the live dataset. Use parent variables for long structured/Unicode display values, not for positive local string matching.

**Rationale**: Public filter values use serialized text; membership arrays contain strings representing serialized values. JSON display uses `task.VariableEnrichedUserTasks` with `items[].item` and `items[].variables`, while ordinary search uses `task.UserTasks`. Assert both correctly without inventing a new schema.

**Alternatives considered**: Arbitrary local string fixtures or exhaustive output/operator cross-products violate scope or duplicate unit coverage. A backend mismatch in a required basic operator is a visible failed check, not grounds to silently skip or change expectations.

## Dirty-cluster volume scope

**Decision**: Use the exact deployed definition key and state; discover seeded task keys independently by PI. Record a bounded pre-seed match snapshot and compare against its union with seeded keys where the definition already has tasks.

**Rationale**: Model IDs and process-only run markers cannot isolate local-filter queries to a run. Exact global counts and assumptions of fresh definition keys are unsafe. The test can prove paging over a stable known scope while distinguishing preexisting from owned tasks.

**Alternatives considered**: Filtering local variables by the process-only run marker is incorrect; deleting unrelated instances or renaming models is unnecessary. Record concurrent scope churn as a validation limitation/failure instead of fabricating stable counts.

## Targets and evidence

**Decision**: Name the baseline entry point `TestGetFamilyUserTaskVariables`; extend `TestVolumeGetFamily` for volume. Existing Make regexes and aggregate then include the new scenarios. Add the five missing manifest flags and validate inventory separately.

**Rationale**: Existing `integration-cli-get` selects `TestGetFamily`, volume selects `TestVolumeGetFamily`. Existing profile configuration, binary build and evidence plumbing already suffice.

**Alternatives considered**: New issue-number targets, new report schemas, aggregate concurrency/build optimizations or general inventory-runner changes are separate work.

## Research boundary

No live cluster requests or runtime tests were made during planning. Repository facts resolve implementation choices; all backend semantic assertions remain subject to actual version-specific live evidence during implementation. No unresolved design clarification remains.

## Superseding user decision

Remove both live `$notIn` cases across all versions, without skip branches. C89 produced HTTP 500; similar behavior on other versions is assumed only for this coverage decision. Retain unit request-encoding coverage; do not claim live `$notIn` verification.
