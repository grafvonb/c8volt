# Ralph Memory

Feature: 331-simplify-version-support
Started: 2026-10-05T06:30:28Z

## Codebase Patterns

- Runtime support identity is centralized in `toolx/version.go`; service factories and compile-time API assertions mirror that set across all eleven service families.
- Current CLI reference pages are generated with `make docs-content`; `docs/cli/index.md` is maintained separately and must be updated directly when its support matrix changes.

## Decisions

- The retained runtime set is exactly 8.8, 8.9, and 8.10, with 8.9 as the default. Generic unsupported-version tests use non-release values rather than retaining retirement-specific suites.
- The version-independent deployment identifier/BPMN fallback remains covered; only release-specific deploy/embed cases and helpers were removed.

## Gotchas

- The v8.8 facade diagnostics deletion path reads the root and descendants through Camunda before issuing the Operate deletion, so its fixture must return a valid keyed process instance and empty search page.
- `go test ./cmd -count=1` is comparatively slow. T001's required command regex is the primary repeatable gate; the broader package run found one v8.8 license-rendering expectation, which was corrected and rechecked directly.

## Reusable Commands

- `go test ./toolx ./config -count=1`
- `go test ./internal/services/... -run 'Factory|SourceBoundary|DryRun|PlanProcessInstanceMutationPages' -count=1`
- `go test ./cmd -run 'Bootstrap|Config|Root|Version|Tenant|GetProcessInstance|CancelProcessInstance|DeleteProcessInstance|UpdateUserTask|BuildRunProcessInstanceDatasFromDeployments|Deploy|Embed' -count=1`
- `make docs-content && go test ./docsgen -count=1 && git diff --check`

## Do Not Repeat

- Do not restore retired adapter-specific rejection or compatibility tests; generic invalid-version bootstrap/config coverage already proves early rejection.
- Do not remove the generic deployment identifier/BPMN parsing fallback in `cmd/cmd_deploy_run_data.go`; retained versions use it independently of the removed adapter.

## Current Handoff
- Continue with T002 / US2: remove the retired generated client, generation targets, mockery/protection entries, fixtures, and integration assets. Follow `integration/AGENTS.md`, run the T002 generation and compile-only checks, then the one full executable validation and temporary-snapshot boundary check from `quickstart.md`.
