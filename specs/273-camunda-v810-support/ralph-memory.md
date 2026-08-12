# Ralph Memory

Feature: 273-camunda-v810-support
Started: 2026-08-12T16:38:49Z

## Codebase Patterns
- Shell generation tests live under `api/tests/` and are run explicitly with `bash api/tests/<name>.sh`; keep them POSIX/Bash-focused and self-contained.
- `api/tests/v810_generation_test.sh` now provides reusable helpers for temporary git fixture repos/worktrees, SHA-256 file/tree checksums, fake tool installation/invocation logs, and assertion helpers.
- V810 generation contract tests use detached git worktrees of the current repo so refresh scripts resolve paths from an isolated checkout and cannot publish into the working tree.
- Provenance validation lives in `api/tests/v810_provenance_test.py` and checks exact schema keys, ordered mutation hashes, generated-client hash agreement, nondeterministic-field/path absence, and canonical second-run determinism.
- Source-boundary tests use AST import scanning rather than package loading so they can catch layering regressions before type checking.
- `internal/services/v810_source_boundary_test.go` is active for `cmd/` and public facade generated-client/versioned-service imports, and conditionally scans V810 adapter packages as they appear.

## Decisions
- V810 adapter boundary checks allow only `github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda` among generated Camunda clients.
- Command and public facade boundary checks reject any generated Camunda client import and any direct versioned service implementation import.

## Gotchas
- Git worktrees expose `.git` as a file that points at worktree metadata, not as a directory; shell assertions should check path existence for that case.
- `api/tests/v810_generation_test.sh` creates detached worktrees from `HEAD`; use direct isolated smoke tests for uncommitted refresh-script changes, then rerun the guard after the work-unit commit.
- T003/T004 remain red TDD guards until `api/generate-v810-client.sh` and the checked-in V810 provenance/client exist; validate syntax with `bash -n` and `python3 -m py_compile`, then confirm remaining red failures point at missing generator/provenance behavior.

## Reusable Commands
- `bash api/tests/v810_generation_test.sh`
- `python3 api/tests/v810_provenance_test.py`
- `bash -n api/tests/v810_generation_test.sh`
- `python3 -m py_compile api/tests/v810_provenance_test.py`
- `go test ./internal/services -run 'TestV810AdapterSourceBoundary|TestCommandAndFacadeSourceBoundaryForGeneratedClients' -count=1`

## Do Not Repeat

## Current Handoff
- Next iteration should start at T006 in Phase 2: implement `api/generate-v810-client.sh` with temporary fetch, peeled-commit verification, bundling, ordered mutations, symbol checks, protected-tree fingerprints, deterministic provenance, and atomic publication; do not begin US1 until T006-T009 complete.
