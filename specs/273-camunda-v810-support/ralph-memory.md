# Ralph Memory

Feature: 273-camunda-v810-support
Started: 2026-08-12T16:38:49Z

## Codebase Patterns
- Shell generation tests live under `api/tests/` and are run explicitly with `bash api/tests/<name>.sh`; keep them POSIX/Bash-focused and self-contained.
- `api/tests/v810_generation_test.sh` now provides reusable helpers for temporary git fixture repos/worktrees, SHA-256 file/tree checksums, fake tool installation/invocation logs, and assertion helpers.
- Source-boundary tests use AST import scanning rather than package loading so they can catch layering regressions before type checking.
- `internal/services/v810_source_boundary_test.go` is active for `cmd/` and public facade generated-client/versioned-service imports, and conditionally scans V810 adapter packages as they appear.

## Decisions
- V810 adapter boundary checks allow only `github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda` among generated Camunda clients.
- Command and public facade boundary checks reject any generated Camunda client import and any direct versioned service implementation import.

## Gotchas
- Git worktrees expose `.git` as a file that points at worktree metadata, not as a directory; shell assertions should check path existence for that case.

## Reusable Commands
- `bash api/tests/v810_generation_test.sh`
- `go test ./internal/services -run 'TestV810AdapterSourceBoundary|TestCommandAndFacadeSourceBoundaryForGeneratedClients' -count=1`

## Do Not Repeat

## Current Handoff
- Next iteration should start at T003/T004 in Phase 2 Foundational generation tests. Extend `api/tests/v810_generation_test.sh` with failing target/tag/commit/output-escape/missing-tool/mutation-no-op/atomic-publication cases and add `api/tests/v810_provenance_test.py`; do not begin US1 until foundational generation tasks are complete.
