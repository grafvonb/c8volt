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
- Source-boundary tests use AST import scanning rather than package loading so they can catch layering regressions before type checking.
- `internal/services/v810_source_boundary_test.go` is active for `cmd/` and public facade generated-client/versioned-service imports, and conditionally scans V810 adapter packages as they appear.

## Decisions
- V810 adapter boundary checks allow only `github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda` among generated Camunda clients.
- Command and public facade boundary checks reject any generated Camunda client import and any direct versioned service implementation import.

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

## Do Not Repeat

## Current Handoff
- Next iteration should start at T010 in User Story 1: add V810 alias, canonical string, supported/implemented-set staging, source-tag rejection, and unchanged-default tests in `toolx/version_test.go` and `config/app_test.go`.
