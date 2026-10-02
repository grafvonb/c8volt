# Validation Guide: Supported-Version Cleanup

## Prerequisites

Use the issue branch, Go/toolchain from `go.mod`, existing shell/Python tools, and local test dependencies. Run commands from the repository root. These are implementation validation instructions, not claims that tests ran during planning. No live cluster is needed for the automated checks below.

## Runtime and configuration

```sh
go test ./toolx ./config -count=1
go test ./internal/services/... -run 'Factory|SourceBoundary' -count=1
go test ./cmd -run 'Bootstrap|Config|Root|Version|Tenant|GetProcessInstance|CancelProcessInstance|DeleteProcessInstance|UpdateUserTask|Deploy|Embed' -count=1
go test ./c8volt -run 'New|APIDiagnostics' -count=1
```

Confirm the retained set/default, generic invalid identity rejection, request count zero before bootstrap rejection, and existing error classes. Reuse generic invalid-version fixtures rather than naming the retired release. Verify every selected pattern matches actual tests with `go test -list` if names change; a no-tests warning is not coverage.

## Shared traversal and component contracts

```sh
go test ./internal/services/processinstance -run 'DryRun|PlanProcessInstanceMutationPages' -count=1
go test ./internal/services/processinstance/v88 -run 'DeleteProcessInstance' -count=1
go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 -run 'GetUserTask' -count=1
go test ./cmd -run 'BuildRunProcessInstanceDatasFromDeployments|Deploy|Embed' -count=1
```

Retain tenant evidence, missing ancestors, sparse-page continuation and request counts. Verify Operate deletion, Tasklist primary/fallback routing and rejection of identity/tenant mismatch. Command regressions touching prompts use real terminal stdin with independently captured stdout/stderr; preserve the output contract in [support-policy.md](contracts/support-policy.md).

## Generation

```sh
bash -n api/3-generate-clients-from-fetched-specs.sh api/generate-v810-client.sh
bash api/tests/v810_generation_test.sh
python3 api/tests/v810_provenance_test.py
```

Verify removed output targets cannot be selected/recreated by the full generator, adding focused mocked script coverage if existing isolated-generation tests do not cover its target list. Do not regenerate unrelated retained clients just to prove target exclusion.

After the affected cleanup is committed under the implementation workflow and relevant paths are clean (or from a reviewed temporary snapshot):

```sh
bash api/tests/v810_repository_boundary_test.sh
```

This script intentionally fails on dirty protected trees and integration assets. Record that precondition separately from behavioral failures; do not discard changes or weaken the guard.

## Integration assets without cluster mutation

```sh
go test -tags=integration ./integration/cli ./testx/integration88 -run '^$' -count=1
```

This compiles tagged tests; it is not live-cluster validation. Preserve remaining profile gates and restrictions. Disposable-cluster validation, if performed, must be explicitly recorded with version and scope. Do not expand the harness to a new release as part of this cleanup.

## Final executable validation and docs

Run after the last executable change, after targeted checks pass:

```sh
make test
make build
make docs-content
go test ./docsgen -count=1
git diff --check
```

The full suite is required because shared version selection, factories and generated clients affect many packages. Later prose-only changes do not invalidate that result. If final review fixes code, rerun the impacted checks and broad validation when warranted.

## Tracked-content audit

Use `git ls-files` to inventory tracked paths, including pending deletions/additions, then `rg` with retired identifiers derived from issue #331. Include path names, source, fixtures, tests, scripts, OpenAPI inputs, README, generated docs, `AGENTS.md`, `specs/`, `tasks/`, and these artifacts. Include newly created files awaiting staging. Do not scan only currently supported packages.

Review every match: remove obsolete support references, retain unrelated numeric/dependency data and factual source provenance that makes no support claim. Do not create a repository-wide exclusion for historical plans or API inputs. Verify current docs contain only the retained policy. Record evidence and any unresolved matches in `progress.md` without copying obsolete support strings into permanent documentation.
