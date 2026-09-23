# Quickstart: User-Task Variable Integration Coverage

## Prerequisites

- Implement this plan before expecting the new scenarios to execute.
- Use disposable local clusters and existing profiles in the default c8volt configuration. Select 8.8, 8.9 and/or 8.10 profiles; select 8.7 separately or alongside them for unsupported checks.
- Existing matching embedded models are sufficient. The suite deploys them, supplies variables, and discovers runtime keys; no BPMN editing or manual task-local setup.
- Use the existing Make confirmation. These suites create real instances and may run existing destructive family scenarios.

## Focused baseline

From the repository root, replace the profile names with your configured names:

```sh
C8VOLT_IT_PROFILES=your-c88-profile,your-c89-profile,your-c810-profile \
C8VOLT_IT_VERBOSE=1 \
go test -tags=integration ./integration/cli \
  -run '^TestGetFamilyUserTaskVariables$' -count=1 -v -timeout=60m
```

The direct Go invocation does not provide Make's confirmation prompt. It must select the new test after implementation; a zero-test run is not validation.

Expected: the standalone and called-process cases in [the contract](contracts/integration.md) pass for each ready supported profile. Selecting an 8.7 profile instead exercises explicit unsupported behavior. Review skips as well as failures.

## Existing family targets

```sh
C8VOLT_IT_PROFILES=your-c89-profile make integration-cli-get C8VOLT_IT_GO_TEST_FLAGS=-v
C8VOLT_IT_PROFILES=your-c89-profile C8VOLT_IT_VOLUME_COUNT=3 make integration-cli-get-volume C8VOLT_IT_GO_TEST_FLAGS=-v
```

Repeat/select profiles for other versions as available. At least three matching seeds are required. These runs include existing get-family regressions and volume scenarios, not solely #324. Reuse passing relevant results rather than rerunning overlapping commands just for completion bookkeeping.

## Manifest check

```sh
go test -tags=integration ./integration/cli -run '^TestCommandInventory$' -count=1 -v -timeout=10m
```

This verifies the updated five user-task flags. Unrelated existing drift must be reported separately rather than hidden or broadly fixed here.

## Full suite entry point

```sh
make integration-test-all C8VOLT_IT_GO_TEST_FLAGS=-v
```

The existing aggregate selects the added scenarios through baseline and volume get targets. It is optional broader release validation, not a required extra run after focused evidence passes. Run serially; do not use `make -j` for shared destructive targets. Existing C89-only real-state skips do not establish coverage of other versions.

## Evidence review

Use existing emitted work directories and command logs. If setting `C8VOLT_IT_WORKDIR`, give separate Go invocations separate directories to avoid overwriting summary metadata. Confirm actual test execution, per-profile versions, seeded/child task keys, local versus process scope, complete duplicate-free paging, limit two, consistent total, and zero-byte unmatched keys output. Distinguish preexisting volume matches, failed setup, skipped prerequisites, and verified unsupported behavior.

Record a concise final validation summary with commands and versions actually run. Do not duplicate iteration history here. Planning performed document checks only; no live results are claimed.
