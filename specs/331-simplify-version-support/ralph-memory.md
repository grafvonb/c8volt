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
- Review restored the original v8.10 integration content/path guard. It fails on coverage that predates this feature, confirmed in a clean snapshot of `c9949f11`. Do not disable its checks or remove existing coverage to claim a passing cleanup.
- Historical support cleanup must account for Markdown structure: remove obsolete table columns and their cells together, and remove version-specific file-tree branches with their children rather than leaving orphaned content.
- Final tracked-content audits preserve upstream `x-added-in-version` metadata as provenance and classify apparent matches inside binary assets as incidental bytes.

## Reusable Commands

- `go test ./toolx ./config -count=1`
- `go test ./internal/services/... -run 'Factory|SourceBoundary|DryRun|PlanProcessInstanceMutationPages' -count=1`
- `go test ./cmd -run 'Bootstrap|Config|Root|Version|Tenant|GetProcessInstance|CancelProcessInstance|DeleteProcessInstance|UpdateUserTask|BuildRunProcessInstanceDatasFromDeployments|Deploy|Embed' -count=1`
- `make docs-content && go test ./docsgen -count=1 && git diff --check`
- `bash api/tests/v810_generation_test.sh && python3 api/tests/v810_provenance_test.py`
- `git diff --check`

## Do Not Repeat

- Do not restore retired adapter-specific rejection or compatibility tests; generic invalid-version bootstrap/config coverage already proves early rejection.
- Do not remove the generic deployment identifier/BPMN parsing fallback in `cmd/cmd_deploy_run_data.go`; retained versions use it independently of the removed adapter.

## Current Handoff
- #331 is complete with the user-approved deferral of the pre-existing boundary-check conflict to the stable 8.10 upgrade. Keep the restored guard and existing integration coverage. No further Ralph iteration is needed for this feature. Original implementation commit subjects were not rewritten; the closeout commit uses the required bare issue suffix.
