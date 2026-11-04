# Research: Supported-Version Cleanup

## 1. Compilation boundary

**Decision**: Remove the public retired identity, all eleven adapter trees, factories/assertions and affected constant-dependent tests in one runtime cutover.

**Rationale**: `toolx/version.go` feeds `config/app.go`, command capability checks, service factories and shared tests. Retained adapters do not import the retired adapter. Leaving an old adapter test behind after normalization changes would fail construction even if compilation succeeded.

**Alternatives considered**: Keeping a deprecated constant or transitional shim conflicts with complete removal. Splitting references across tasks creates broken intermediate revisions. Unreferenced generated clients can be deleted with generation tooling later.

## 2. Retained component services

**Decision**: Keep Operate configuration/authentication and the 8.8 process-instance deletion client; retain Tasklist fallback in the 8.8 and 8.9 user-task adapters.

**Rationale**: `internal/services/processinstance/v88/contract.go` requires Operate deletion; `internal/services/usertask/v88/service.go` and its v89 counterpart use Tasklist after primary misses. `c8volt/diagnostics_test.go` needs its Operate coverage moved to supported deletion when its old search case disappears.

**Alternatives considered**: Removing all component clients or consolidating adapters would remove supported behavior and exceed scope.

## 3. Shared dry-run and deployment behavior

**Decision**: Remove the legacy traversal marker, dispatch and key-only dry-run implementation from `internal/services/processinstance/dryrun.go`, including its exclusive tenant-evidence helper and test double. Keep normal traversal/walker behavior. Retain generic deployment response-ID and BPMN parsing fallback in `cmd/cmd_deploy_run_data.go`.

**Rationale**: Only the retired production adapter enables the legacy dry-run marker. Deployment fallback, however, has version-independent tests and handles incomplete responses without consulting version identity. Removing it would broaden the behavior change for retained versions.

**Alternatives considered**: Deleting every fallback by name is unsafe. Preserve shared safeguards and delete only exclusive support code and release-specific tests.

## 4. Generation and source inventory

**Decision**: Remove only retired generated output and corresponding commands/file lists in `api/3-generate-clients-from-fetched-specs.sh`, protected-tree entries in `api/generate-v810-client.sh` and `api/tests/v810_repository_boundary_test.sh`, and matching `.mockery.yml` entries. Keep generation checks for retained output and source provenance.

**Rationale**: Deleting generated output alone allows normal refresh to recreate it. Existing boundary tests also require clean protected paths and integration content, so execution timing must account for intended changes.

**Alternatives considered**: Full regeneration of retained clients is unnecessary churn. Other unreferenced generated clients are a separate issue.

## 5. Documentation and metadata audit

**Decision**: Audit all tracked paths/content, including `specs/`, `tasks/`, guidance, generation inputs and this feature. Remove actual obsolete support statements and artifacts; preserve unrelated content. Describe this feature through the retained policy and source issue.

**Rationale**: Historical feature requirements otherwise instruct future agents to rebuild deleted support. Tracked OpenAPI inputs also carry factual endpoint-introduction metadata: distinguish such numeric provenance from a claim of current support. Do not delete supported endpoints, corrupt checksums, or rewrite unrelated numeric data. There is no blanket exclusion for API inputs or historical documents.

**Alternatives considered**: Excluding historical specifications or retaining a migration page contradicts the requested scope. Blind string replacement could damage schemas, numeric keys and dependency versions.

## 6. Work-unit and validation discipline

**Decision**: Three units: runtime plus exclusive shared behavior; generation plus fixtures/integration; remaining support documentation and final audit. Each check has one owner in quickstart. Reuse existing generic validation tests and avoid new broad matrices.

**Rationale**: Ralph permits multiple tasks within a story but never starts a second story in the same iteration. Reducing five checkboxes alone would not lower its three-story minimum. Removing repeated inspection, commands and artificial cleanup boundaries reduces actual work. The old dry-run task revisited deploy/embed coverage already required by runtime deletion. Generation and fixture removal can share one executable completion check. Historical content remains separate: 419 specification files and six PRDs had candidate references during review, so combining that work with runtime changes would be impractical.

**Alternatives considered**: One giant task removes the useful validation boundary and exceeds a practical iteration. Keeping five tasks increases handoff opportunities without guaranteeing smaller logical work. Splitting historical content per document would create hundreds of unnecessary iterations.

## 7. Validation execution traps

**Decision**: Compile tagged integration packages with `go test -c`; run the Git-cleanliness boundary check once in a temporary clean snapshot of reviewed changes before task completion.

**Rationale**: `integration/cli/harness_test.go` defines `TestMain`, which builds a binary and writes reports even with `-run '^$'`. The prior compile-only instructions therefore performed unnecessary work. The repository-boundary shell check rejects intended dirty integration edits; waiting for a commit conflicts with validating a unit before marking it complete. A temporary snapshot resolves that ordering without resetting the checkout or relaxing the guard.

**Alternatives considered**: Repeated post-commit checks and follow-up evidence commits add loop overhead. Disabling the guard would lose useful protection.
