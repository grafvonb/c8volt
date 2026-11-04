# Validation Guide: Supported-Version Cleanup

## Prerequisites and Ownership

Run from the repository root with the toolchain in `go.mod` and existing shell/Python tools. These are implementation commands, not claims that runtime checks ran during planning. No live cluster is required.

Each group belongs to one task. Run focused tests after related edits; repeat only when code, failures or new evidence invalidate their result. Reuse existing regression coverage, adding tests only for demonstrated gaps. Commands are a starting set: if test names change, confirm patterns select the intended tests; a no-tests warning is not coverage.

## T001 — Runtime, Shared Behavior and Current Documentation

```sh
go test ./toolx ./config -count=1
go test ./internal/services/... -run 'Factory|SourceBoundary|DryRun|PlanProcessInstanceMutationPages' -count=1
go test ./internal/services/processinstance/v88 -run 'DeleteProcessInstance' -count=1
go test ./internal/services/usertask/v88 ./internal/services/usertask/v89 -run 'GetUserTask' -count=1
go test ./c8volt -run 'New|APIDiagnostics' -count=1
go test ./cmd -run 'Bootstrap|Config|Root|Version|Tenant|GetProcessInstance|CancelProcessInstance|DeleteProcessInstance|UpdateUserTask|BuildRunProcessInstanceDatasFromDeployments|Deploy|Embed' -count=1
make docs-content
go test ./docsgen -count=1
git diff --check
```

One command-package pass includes deploy/embed and generic fallback tests; do not rerun them as a separate cleanup step. Adapt patterns to any additional affected paths discovered during edits rather than running every command family repeatedly. Relevant tests compile their packages; no additional all-package compile sweep is required before T002's full validation.

Confirm canonical selection/default, generic unsupported-version errors and zero remote requests/prompts before rejection. Existing flag/environment/file/profile precedence tests should be reused. Keep supported component routing, tenant evidence, partial traversal and sparse-page behavior. Preserve existing real-terminal command tests with separate stdout/stderr capture when their behavior is affected, following [support-policy.md](contracts/support-policy.md). Do not create an exhaustive new output/flag matrix for unchanged behavior.

## T002 — Generation and Final Executable Validation

```sh
bash -n api/3-generate-clients-from-fetched-specs.sh api/generate-v810-client.sh
bash api/tests/v810_generation_test.sh
python3 api/tests/v810_provenance_test.py
```

Inspect the full generator target list and existing deterministic tests together. Add a focused mocked test only if this leaves retired-output non-recreation unproven. No live upstream fetch or regeneration of unrelated retained clients is required. The existing generation harness uses disposable Git fixtures; preserve its normal isolation.

Compile the retained integration packages without executing their entrypoints:

```sh
integration_check_dir="$(mktemp -d "${TMPDIR:-/tmp}/c8volt-integration-compile.XXXXXX")"
go test -c -tags=integration -o "$integration_check_dir/cli.test" ./integration/cli
go test -c -tags=integration -o "$integration_check_dir/facade.test" ./testx/integration88
```

These binaries are temporary validation outputs; remove only this created directory when finished. Do not substitute `-run '^$'`: the CLI harness's `TestMain` would still build a binary and write reports. Compilation is not live-cluster validation. Preserve remaining profile gates and do not expand supported integration coverage in this cleanup.

After targeted checks pass, run exactly one final executable pass:

```sh
make test
make build
git diff --check
```

The full suite covers the combined runtime/client/fixture changes. Do not prepend another broad test sweep already included in T001 unless edits have invalidated that evidence. Record the final results and any remaining live-cluster gap.

### Repository Boundary Check Without a Commit Dependency

The existing boundary script rejects dirty protected/integration paths. Verify retained client trees are unchanged in the actual checkout, then execute the current script against a temporary clean snapshot of the relevant reviewed content. This is a test fixture, not a replacement working checkout.

```sh
git diff --exit-code HEAD -- internal/clients/camunda/v88 internal/clients/camunda/v89
(
  snapshot_dir="$(mktemp -d "${TMPDIR:-/tmp}/c8volt-boundary-check.XXXXXX")"
  trap 'rm -rf "$snapshot_dir"' EXIT
  python3 - "$snapshot_dir" <<'SNAPSHOT'
import pathlib, shutil, subprocess, sys
roots = ['internal/clients/camunda/v88', 'internal/clients/camunda/v89',
         'integration', 'api/tests/v810_repository_boundary_test.sh']
paths = subprocess.check_output(['git', 'ls-files', '--cached', '--others',
                                 '--exclude-standard', '-z', '--', *roots])
for raw in set(paths.decode().split('\0')) - {''}:
    source = pathlib.Path(raw)
    if not source.exists():
        continue  # Pending deletion is part of the reviewed result.
    target = pathlib.Path(sys.argv[1]) / source
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, target, follow_symlinks=False)
SNAPSHOT
  git -C "$snapshot_dir" init -q
  git -C "$snapshot_dir" add .
  git -C "$snapshot_dir" -c user.name='Boundary Check' -c user.email='boundary@example.invalid' commit -qm 'test: snapshot boundary inputs'
  bash "$snapshot_dir/api/tests/v810_repository_boundary_test.sh"
)
```

The copy inventory includes tracked and nonignored new files, skips pending deletions and excludes ignored integration reports. Review nonignored new inputs before running it. The actual-checkout diff proves retained clients were not edited; the snapshot check proves the resulting integration content and current guard pass. Do not claim the snapshot's clean Git status proves the original checkout is clean. No production commit, reset, guard relaxation or post-commit bookkeeping task is needed for this test.

## T003 — Historical Content and Final Audit

Build a single inventory from `git ls-files`, pending deletions/additions and new files awaiting staging. Search candidate paths/content with identifiers derived from the source issue. Include source, tests, fixtures, scripts, OpenAPI inputs, README, docs, `AGENTS.md`, `specs/`, `tasks/`, and this feature's artifacts.

Use targeted replacements and diff review of affected passages. Do not reread every historical document, create per-file tasks or broadly rewrite unrelated prose. Remove support statements/artifacts; classify incidental numeric data, unrelated dependency versions and factual provenance individually. No whole-directory audit exclusions.

Check changed local links, support/default/maturity statements and `git diff --check`. Run `make docs-content` and relevant docsgen checks only if their source changed after T001. Do not repeat the runtime suite, build, generation tests or boundary fixture for prose-only changes. If the audit discovers executable omissions, fix them and rerun the invalidated checks.

Record the final audit and requirement coverage in `progress.md` without copying obsolete support strings into permanent artifacts. Carry forward T001/T002 evidence rather than creating a separate validation iteration.
