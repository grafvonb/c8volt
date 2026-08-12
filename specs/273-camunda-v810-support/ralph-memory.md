# Ralph Memory

Feature: 273-camunda-v810-support
Started: 2026-08-12T16:38:49Z

## Codebase Patterns
- Shell generation tests live under `api/tests/` and are run explicitly with `bash api/tests/<name>.sh`; keep them POSIX/Bash-focused and self-contained.
- `api/tests/v810_generation_test.sh` now provides reusable helpers for temporary git fixture repos/worktrees, SHA-256 file/tree checksums, fake tool installation/invocation logs, and assertion helpers.
- V810 generation contract tests use detached git worktrees of the current repo so refresh scripts resolve paths from an isolated checkout and cannot publish into the working tree.
- `api/generate-v810-client.sh` fetches the full upstream v2 spec directory with sparse checkout; fetching only `rest-api.yaml` breaks Redocly bundling because the spec contains local `$ref` files.
- The V810 generator compiles generated output in a temporary standalone Go module and runs `go mod tidy` there before `go test`, avoiding repository writes before publication.
- Provenance validation lives in `api/tests/v810_provenance_test.py` and checks exact schema keys, ordered mutation hashes, generated-client hash agreement, nondeterministic-field/path absence, and canonical second-run determinism.
- The checked-in pinned V810 artifacts now live at `internal/clients/camunda/v810/camunda/client.gen.go` and `internal/clients/camunda/v810/camunda/provenance.json`; `internal/clients/camunda/v810/camunda/client_test.go` adds package-local compile/constructor/required-symbol coverage.
- With the checked-in artifacts present, `python3 api/tests/v810_provenance_test.py` passes and performs a second canonical generation run to prove deterministic provenance/client output.
- `api/generate-v810-client.sh` preserves existing package-local files when republishing V810 artifacts, so canonical regeneration does not delete `client_test.go`.
- V810 generation guard negative cases now compare the V810 publication checksum before/after failed runs because detached worktrees already contain the committed pinned V810 artifacts.
- `toolx/camunda_baseline.go` owns the active V810 baseline metadata; `toolx/camunda_baseline_test.go` verifies it matches `internal/clients/camunda/v810/camunda/provenance.json`.
- `cmd/version.go` renders V810 baseline disclosure as an additive human line and two additive JSON payload string fields: `camunda810Baseline` and `camunda810BaselineStatus`.
- Root help now lists supported versions through 8.10 and discloses the active 8.10 prerelease baseline, but V810 remains staged out of `ImplementedCamundaVersions()` until US2 factory wiring is complete.
- Source-boundary tests use AST import scanning rather than package loading so they can catch layering regressions before type checking.
- `internal/services/v810_source_boundary_test.go` is active for `cmd/` and public facade generated-client/versioned-service imports, and conditionally scans V810 adapter packages as they appear.
- `toolx.V810` now normalizes only the stable aliases `8.10`, `810`, `v810`, and `v8.10`; prerelease/source tags such as `8.10.0-alpha4` remain rejected configuration identities.
- Config test-connection gateway compatibility now compares explicit release-line states: same major/minor matches (including `8.10.0-alpha4`), different major/minor warns with existing mismatch wording, and empty/malformed gateway versions warn that compatibility cannot be verified.

## Decisions
- V810 adapter boundary checks allow only `github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda` among generated Camunda clients.
- Command and public facade boundary checks reject any generated Camunda client import and any direct versioned service implementation import.
- V810 is listed in supported versions for operator discovery, but `ImplementedCamundaVersions()` intentionally remains `V87,V88,V89` until the complete native V810 factory/client wiring is done.

## Gotchas
- Git worktrees expose `.git` as a file that points at worktree metadata, not as a directory; shell assertions should check path existence for that case.
- `api/tests/v810_generation_test.sh` creates detached worktrees from `HEAD`; use direct isolated smoke tests for uncommitted refresh-script changes, and rerun the guard after the work-unit commit because it does not see uncommitted changes.

## Reusable Commands
- `bash api/tests/v810_generation_test.sh`
- `python3 api/tests/v810_provenance_test.py`
- `bash -n api/tests/v810_generation_test.sh`
- `bash -n api/generate-v810-client.sh`
- `python3 -m py_compile api/tests/v810_provenance_test.py`
- `go test ./internal/clients/camunda/v810/camunda -count=1`
- `go test ./internal/services -run 'TestV810AdapterSourceBoundary|TestCommandAndFacadeSourceBoundaryForGeneratedClients' -count=1`
- `go test ./toolx -run 'CamundaVersion|CurrentDefault|V810|Baseline' -count=1`
- `go test ./config -run 'AppNormalize|CurrentDefault|CamundaVersion|V810' -count=1`
- `go test ./cmd -run 'ConfigTestConnectionCommand_VersionComparison|ConfigTestConnectionDiagnostics_V810|Version|RootHelp|SupportMessaging|V810Bootstrap|GetHelp|GetClusterHelp|GetProcessDefinitionHelp' -count=1`

## Do Not Repeat

## Current Handoff
- Next iteration should start User Story 2 at T018: add V810 batch-operation and cluster factory/adapter tests in `internal/services/batchoperation/factory_test.go`, `internal/services/batchoperation/v810/`, `internal/services/cluster/factory_test.go`, and `internal/services/cluster/v810/`.
