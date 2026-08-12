# Quickstart: Validate Camunda 8.10 Support

This guide validates the implemented feature locally. It does not add or run a live Camunda 8.10 integration profile.

## Prerequisites

- Go 1.26 / toolchain 1.26.2
- Bash, Python 3 plus mutation dependencies, Redocly CLI, and pinned oapi-codegen
- Network access to public Camunda source for generation
- A clean baseline or recorded pre-feature hashes for protected clients and `integration/`

Review [version selection](contracts/version-selection.md), [generation](contracts/generation-contract.md), [service compatibility](contracts/service-compatibility.md), and the [data model](data-model.md).

## 1. Record Protected Baselines

```bash
git diff -- internal/clients/camunda/v87 internal/clients/camunda/v88 internal/clients/camunda/v89
git diff -- integration
```

Record existing output. Feature work must add no difference to these paths; any pre-existing changes remain byte-for-byte unchanged.

## 2. Generate the Isolated V810 Client

```bash
bash api/refresh-clients.sh --target v810 --camunda-tag 8.10.0-alpha4
go test ./internal/clients/camunda/v810/camunda -count=1
```

Confirm the client exists only under `internal/clients/camunda/v810/camunda`, provenance has the pinned tag/commit and all hashes, no removed component-client directory exists, and protected diffs are unchanged. Run generation again and verify the v810 client/provenance have no diff. Run negative guards under `api/tests/`; invalid target/tag/commit/output/tool/mutation/protected-tree cases must fail before publication.

## 3. Validate Identity, Default, Gateway, and Fixtures

```bash
go test ./toolx ./config -run 'CamundaVersion|CurrentDefault|Capability|Fixture|V810' -count=1
go test ./cmd -run 'Version|RootHelp|SupportMessaging|ConfigTestConnectionCommand_VersionComparison|Embed|V810' -count=1
```

Expected: four aliases select V810; source-tag aliases fail; default stays V88; gateway `8.10.0-alpha4` matches `8.10` while another/unparseable line does not; version output discloses baseline separately; V810 explicitly uses C89 production fixture content while reporting `8.10`.

## 4. Validate All Eleven Service Boundaries

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
go test ./c8volt -run 'TestNew_V810' -count=1
```

Cross-check [service-compatibility.md](contracts/service-compatibility.md). Boundary tests reject older/removed clients. User-task tests prove no Tasklist fallback is constructed or called.

## 5. Validate Capability Gates and Commands

```bash
go test ./internal/services/ops -run 'AllProcessDefinitions|SupportedVersion|UnsupportedVersion|V810' -count=1
go test ./cmd -run 'DeleteProcessDefinition|AllProcessDefinitions|V810|Version|Help' -count=1
```

Direct/bulk deletion share one predicate; V89/V810 pass and V87/V88 retain rejection. Unsupported mutations stop before remote mutation. Human, JSON, keys-only, prompt, activity, and exit behavior remains stable.

## 6. Regenerate and Verify Documentation

```bash
make docs-content
go test ./docsgen ./cmd -run 'Docs|Version|Help' -count=1
```

README, API guidance, root help, version output, docs homepage, and CLI references must consistently show canonical `8.10`, its aliases, V88 default, the `8.10.0-alpha4` prerelease baseline, and in-place update model.

## 7. Recheck Protected Boundaries

```bash
git diff -- internal/clients/camunda/v87 internal/clients/camunda/v88 internal/clients/camunda/v89
git diff -- integration
```

Both outputs match the recorded baseline. No 8.10 integration profile, fixture, script, Make target, or real-state scenario exists.

## 8. Run the Repository Gate

```bash
make test
git diff --check
```

Completion requires deterministic generation, proof for every V810 service/factory, matching docs, unchanged protected paths, focused checks, and the full race-enabled suite.
