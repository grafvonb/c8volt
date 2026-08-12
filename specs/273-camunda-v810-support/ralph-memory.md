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
- Root help now lists supported versions through 8.10 and discloses the active 8.10 prerelease baseline; after full US2 factory construction proof, V810 is also included in `ImplementedCamundaVersions()`.
- Source-boundary tests use AST import scanning rather than package loading so they can catch layering regressions before type checking.
- `internal/services/v810_source_boundary_test.go` is active for `cmd/` and public facade generated-client/versioned-service imports, and conditionally scans V810 adapter packages as they appear.
- `toolx.V810` now normalizes only the stable aliases `8.10`, `810`, `v810`, and `v8.10`; prerelease/source tags such as `8.10.0-alpha4` remain rejected configuration identities.
- Config test-connection gateway compatibility now compares explicit release-line states: same major/minor matches (including `8.10.0-alpha4`), different major/minor warns with existing mismatch wording, and empty/malformed gateway versions warn that compatibility cannot be verified.
- Native V810 batch-operation adapters can follow the V89 unified v2 shape with V810 generated types: read access uses `SearchBatchOperationsWithResponse`, cancel uses `CancelProcessInstancesBatchOperationWithResponse`, and completion polls `GetBatchOperationWithResponse`.
- Native V810 cluster adapters reuse `internal/services/cluster/common` for response/error handling; only generated-client wiring and V810-to-domain conversions are package-local.
- Batch-operation and cluster factories now have explicit `toolx.V810` cases and package-level interface assertions.
- Native V810 element and incident adapters can follow v89 behavior with local V810 generated types; V810 element search differs by requiring `ElementIdFilterProperty` for `elementId`, and V810 cursor pagination requires `*EndCursor` for `after`.
- Element and incident factories now have explicit `toolx.V810` cases and package-level interface assertions.
- Native V810 job and process-definition adapters can follow v89 unified-client behavior with local generated types; V810 process-definition cursor pagination uses `*EndCursor` for `after`, and V810 job tests should use `JobKindEnumBPMNELEMENT`.
- Job and process-definition factories now have explicit `toolx.V810` cases and package-level interface assertions.
- Native V810 process-instance adapters can follow v89 unified-client behavior with local V810 generated types and V810 cursor pagination requires `*EndCursor` for `after`.
- V810 process-instance construction owns the nested native V810 variable service; `internal/services/v810_source_boundary_test.go` has a single allowlist for `processinstance/v810/service.go` importing `variable/v810`.
- Native V810 variable adapters preserve raw JSON decoding for value/truncation fields omitted by generated models and use the process-instance key as the element-instance scope for updates.
- Process-instance and variable factories now have explicit `toolx.V810` cases and package-level interface assertions.
- Native V810 resource and tenant adapters can follow v89 unified-client behavior with local V810 generated types; resource deployment visibility polling and resource history deletion confirmation reuse the existing shared poller/payload helpers.
- Native V810 user-task adapters use only `GetUserTaskWithResponse` from the V810 unified client; tenant mismatches are mapped to the existing not-found/visibility wording and 503 responses preserve `domain.ErrUnavailable`.
- User-task factories now have an explicit `toolx.V810` case and package-level interface assertions.
- Resource and tenant factories now have explicit `toolx.V810` cases and package-level interface assertions.
- `internal/services/incidentfilter` now owns version-neutral canonical incident state/error-type validation without generated-client imports; the list includes V810 `SECRET_RESOLUTION_ERROR`.
- `internal/services/v810_source_boundary_test.go` scans `incidentfilter` production and test files to reject generated Camunda client imports, in addition to V810 adapter and cmd/facade boundary scans.
- `toolx.SupportsFullProcessDefinitionHistoryDeletion` is the named capability for full process-definition history deletion and explicitly returns true only for V89 and V810.
- Direct process-definition deletion and all-process-definitions purge both consume the named capability; V87/V88 still fail before remote discovery or mutation, while V89/V810 pass the local gate.
- `c8volt/client_test.go` now proves full V810 top-level construction by calling one blocked-transport method through each facade surface plus process-instance variable lookup for the nested V810 variable service.
- `cmd/bootstrap_errors_test.go` now expects `NewCli` to construct a V810-backed client successfully; V810 is no longer a staged unsupported bootstrap identity.

## Decisions
- V810 adapter boundary checks allow only `github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda` among generated Camunda clients.
- Command and public facade boundary checks reject any generated Camunda client import and any direct versioned service implementation import.
- V810 is listed in supported and implemented versions after complete native factory/client construction proof.
- Full process-definition history deletion capability is an explicit V89/V810 set, not a version-order comparison.

## Gotchas
- Git worktrees expose `.git` as a file that points at worktree metadata, not as a directory; shell assertions should check path existence for that case.
- `api/tests/v810_generation_test.sh` creates detached worktrees from `HEAD`; use direct isolated smoke tests for uncommitted refresh-script changes, and rerun the guard after the work-unit commit because it does not see uncommitted changes.
- `go test ./internal/services/... -count=1` currently reaches an unrelated `internal/services/ops` smoke-test fixture wording failure (`unsupported smoke-test fixture version` expected vs `embedded smoke-test fixture not found` actual); keep T024/T032 validation scoped to incidentfilter, incident consumers, cmd incident validation, and source boundaries.

## Reusable Commands
- `bash api/tests/v810_generation_test.sh`
- `python3 api/tests/v810_provenance_test.py`
- `bash -n api/tests/v810_generation_test.sh`
- `bash -n api/generate-v810-client.sh`
- `python3 -m py_compile api/tests/v810_provenance_test.py`
- `go test ./internal/clients/camunda/v810/camunda -count=1`
- `go test ./internal/services -run 'TestV810AdapterSourceBoundary|TestCommandAndFacadeSourceBoundaryForGeneratedClients' -count=1`
- `go test ./internal/services/incidentfilter -count=1`
- `go test ./internal/services -run 'TestIncidentFilterSourceBoundaryForGeneratedClients|TestV810AdapterSourceBoundary|TestCommandAndFacadeSourceBoundaryForGeneratedClients' -count=1`
- `go test ./internal/services/incident/... -count=1`
- `go test ./toolx -run 'CamundaVersion|CurrentDefault|V810|Baseline' -count=1`
- `go test ./toolx -count=1`
- `go test ./config -run 'AppNormalize|CurrentDefault|CamundaVersion|V810' -count=1`
- `go test ./cmd -run 'ConfigTestConnectionCommand_VersionComparison|ConfigTestConnectionDiagnostics_V810|Version|RootHelp|SupportMessaging|V810Bootstrap|GetHelp|GetClusterHelp|GetProcessDefinitionHelp' -count=1`
- `go test ./cmd -run 'DeleteProcessDefinition|OpsPurgeAllProcessDefinitions' -count=1`
- `go test ./internal/services/batchoperation/... -count=1`
- `go test ./internal/services/cluster/... -count=1`
- `go test ./internal/services/job/... -count=1`
- `go test ./internal/services/processdefinition/... -count=1`
- `go test ./internal/services/processinstance/... ./internal/services/variable/... -count=1`
- `go test ./internal/services/resource/... ./internal/services/tenant/... -count=1`
- `go test ./internal/services/usertask/... -count=1`
- `go test ./internal/services/ops -run 'PurgeAllProcessDefinitions' -count=1`

## Do Not Repeat

## Current Handoff
- Next iteration should continue User Story 2 at T036: add representative command fake-server coverage for supported reads, confirmed mutations, unsupported-before-mutation errors, and stable human/JSON/keys-only/prompt/activity behavior in `cmd/get_test.go`, `cmd/delete_test.go`, `cmd/update_test.go`, and `cmd/run_test.go`.
