# Quickstart: Validate Experimental Camunda 8.10 Alpha Support

## Prerequisites

- Go 1.26.2
- Python 3 with PyYAML
- Redocly CLI
- oapi-codegen 2.5.0
- Access to the `camunda/camunda` repository
- For live smoke only: a Camunda `8.10.0-alpha4` environment and a c8volt config/profile with valid credentials

## 1. Verify the pinned source

```bash
git ls-remote https://github.com/camunda/camunda.git refs/tags/8.10.0-alpha4 refs/tags/8.10.0-alpha4^{}
```

Expected resolved commit: `4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6`.

## 2. Regenerate only the alpha client

Capture the stable generated-client diff before generation:

```bash
git diff -- internal/clients/camunda/v87 internal/clients/camunda/v88 internal/clients/camunda/v89
bash api/refresh-clients.sh --target v810alpha --camunda-tag 8.10.0-alpha4
git diff -- internal/clients/camunda/v87 internal/clients/camunda/v88 internal/clients/camunda/v89
```

Expected outcomes:

- both stable-tree diffs are identical
- only `internal/clients/camunda/v810alpha/camunda/client.gen.go` and its provenance may change
- provenance names the required tag, commit, source, mutation chain, generator, and command
- a second run produces no generated difference

Run negative guard scenarios from [generation-contract.md](contracts/generation-contract.md). Wrong target, wrong tag, and mismatched destination must fail before writes.

## 3. Validate runtime identity

```bash
go test ./toolx ./config ./cmd -run 'V810|Version|CamundaVersion|Embed' -count=1
```

Confirm:

- all documented alpha aliases normalize to `8.10-alpha`
- plain `8.10` and `8.10.0-alpha4` are rejected
- the default remains `8.8`
- alpha uses `C89_` fixtures explicitly
- human and JSON output distinguish stable and experimental support

## 4. Validate every service boundary

```bash
go test ./internal/services/batchoperation/... -count=1
go test ./internal/services/cluster/... -count=1
go test ./internal/services/element/... -count=1
go test ./internal/services/incident/... -count=1
go test ./internal/services/job/... -count=1
go test ./internal/services/processdefinition/... -count=1
go test ./internal/services/processinstance/... -count=1
go test ./internal/services/resource/... -count=1
go test ./internal/services/tenant/... -count=1
go test ./internal/services/usertask/... -count=1
go test ./internal/services/variable/... -count=1
```

Cross-check the outcomes against [service-compatibility.md](contracts/service-compatibility.md). The alpha user-task tests must prove that no Tasklist fallback is constructed or called.

## 5. Validate capability gates and command behavior

```bash
go test ./internal/services/ops ./cmd -run 'DeleteProcessDefinition|AllProcessDefinitions|V810|Version|Help' -count=1
```

Expected outcomes:

- history-safe process-definition deletion accepts v8.9 and alpha
- v8.7 and v8.8 remain unsupported
- unsupported mutations fail before requests
- existing human, JSON, keys-only, prompt, and exit behavior stays unchanged

## 6. Regenerate and verify documentation

```bash
make docs-content
go test ./docsgen ./cmd -run 'Docs|Version|Help' -count=1
```

Inspect README, root help, version output, docs index, and generated CLI references. Each must present 8.7-8.9 as stable support and `8.10-alpha` as experimental support pinned to `8.10.0-alpha4`.

## 7. Run the live pinned-alpha smoke

```bash
C8VOLT_IT_CONFIG=./config.yaml \
C8VOLT_IT_PROFILE=<alpha-profile> \
integration/scripts/run-c810alpha-smoke.sh
```

The report must record:

1. authenticated connection/configuration check
2. topology showing major/minor 8.10
3. one read workflow against an embedded C89-compatible fixture
4. one mutation followed by its existing observable confirmation
5. one deterministic unsupported-target or unsupported-capability result with no mutation

Do not report the smoke as passed when the environment is unavailable. Record it as not run and keep release readiness open.

## 8. Run the repository gate

```bash
make test
git diff --check
```

The feature is ready for task completion only when focused checks, generated documentation, live smoke evidence, and the full race-enabled test suite all pass, with no changes to stable generated-client artifacts.
