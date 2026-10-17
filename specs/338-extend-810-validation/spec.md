# Extend existing integration tests to Camunda 8.10

Issue: #338. Branch: `338-extend-810-validation`.
Revised 2026-10-15 at the operator's request to remove the runner and restore
existing Make/Go test execution. The earlier aggregate-framework design is
superseded; historical progress and validation reports remain preserved.

## Scope

- Keep baseline/volume support for 8.8, 8.9 and 8.10 and real-state support for
  8.9/8.10, with version-matched fixtures and observed gateway checks.
- `make integration-test-all` directly runs existing common, focused,
  baseline, volume and real-state targets serially. Print commands/results
  normally; stop on a failing group. No new testing framework.
- Default this aggregate to c810local and 8.10. Preserve explicit overrides,
  startup settings, confirmation and existing unattended execution controls.
- Keep useful batch/draining, logical/physical tenant, user-task, real-state,
  generation-integrity and shell-suite tests. Retain fixture isolation,
  eventual batch-accounting observations and incident-owner selection fixes.
- Keep existing per-group evidence. Missing optional infrastructure remains
  an explicit skip; required tenant infrastructure must fail when absent.
- Preserve strict failure handling and production compatibility fixes.

## Excluded

Remove the Go runner, aggregate manifests, normalized release-scenario hooks,
coverage/gap catalog reducer and their dedicated tests/assets. Do not replace
those components with another orchestration framework. No aggregate certification
or passed-with-gaps verdict is claimed from normal Make success.

## Acceptance

The existing group targets remain independently usable. The aggregate runs them
serially, shows output, and refuses mismatched requested/configured/server
versions before mutation. Existing useful tests and fixture fixes remain intact.
The removal was explicitly requested with no test runs; runtime revalidation is
pending and must not be represented as passed.
