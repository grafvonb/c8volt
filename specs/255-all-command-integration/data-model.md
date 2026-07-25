# Data Model: All-Command Integration Suite

## Command Inventory

Represents the live command paths returned by `c8volt capabilities --json`.

Fields:

- `command`: root command that produced the inventory
- `version`: command contract version
- `commands`: recursive command capability tree
- `flattenedPaths`: flattened command paths used for coverage comparison

Validation rules:

- The flattened inventory must contain exactly the current expected count for this feature: 55 command nodes.
- Every flattened path must either have a coverage entry or be reported as missing.
- Stale coverage entries that no longer exist in the live inventory must be reported.

## Coverage Entry

Represents expected integration coverage for one command path.

Fields:

- `path`: command path from the live inventory
- `family`: command-family group that owns the scenario
- `aliasesCovered`: whether aliases are exercised
- `flagsCovered`: command-local and relevant persistent flags covered by scenarios
- `outputModesCovered`: output modes covered by scenarios
- `versionMatrix`: expected behavior per configured Camunda version
- `destructive`: whether the scenario may mutate cluster state
- `scenarioNames`: named scenarios that provide evidence

Validation rules:

- Leaf commands must cover every command-local flag reported by capabilities.
- Parent/grouping commands must cover help/discovery and no-argument behavior.
- Destructive entries must identify preview and confirmed mutation coverage when the command supports both.

## Integration Profile

Represents a local c8volt profile selected from the operator's default configuration.

Fields:

- `name`: profile name
- `expectedVersion`: intended Camunda minor version
- `actualVersion`: version observed during profile gate
- `reachable`: whether connectivity succeeded
- `tenant`: effective tenant context when applicable

Validation rules:

- Profiles must come from the default local c8volt configuration.
- Required profiles must be reachable before destructive scenarios run.
- Actual version must match the expected version for version-specific scenarios.

## Run Marker

Represents a unique identifier for data created during one suite run.

Fields:

- `value`: unique marker value
- `startedAt`: run start time
- `profiles`: profiles that used the marker

Validation rules:

- Every command-created process instance should receive the run marker when the command supports variables.
- Evidence should use the marker to distinguish seeded data from pre-existing data where possible.

## Evidence Record

Represents the reusable proof captured for one command scenario.

Fields:

- `commandPath`
- `scenarioName`
- `profile`
- `camundaVersion`
- `arguments`
- `stdin`
- `stdoutPath`
- `stderrPath`
- `exitCode`
- `startedAt`
- `finishedAt`
- `dataOwnership`: one of `seeded`, `preexisting`, `mutated`, `retained`, `cleanup_failed`
- `resourceKeys`
- `outcome`: pass, fail, skipped, or blocked
- `failureClass`: product, harness setup, missing fixture support, missing command support, or environment availability

Validation rules:

- Every required scenario must produce one evidence record.
- Failures must identify a failure class.
- Mutating scenarios must record known affected keys when available.

## Proposal Record

Represents a missing command capability or embedded fixture needed by the suite.

Fields:

- `kind`: command proposal or embedded BPMN proposal
- `requiredState`
- `coverageNeed`
- `fallbackUsed`
- `affectedCommands`
- `affectedVersions`
- `operatorValue`

Validation rules:

- Every direct Camunda setup fallback must produce a command proposal record.
- Every unavailable embedded model need must produce an embedded BPMN proposal record.
- Proposal records must not automatically authorize product implementation.
