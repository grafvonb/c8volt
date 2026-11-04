> Current architecture (2026-10-15): the Go runner and aggregate policy were
> removed at the operator's request. Direct serial Make execution is restored.
> Runner references below are historical, not current instructions. No tests
> were run for the removal; see the revised plan/tasks/quickstart.

# Ralph Memory

Feature: 338-extend-810-validation
Started: 2026-10-14T09:50:16Z

## Codebase Patterns

- Integration release policy is an ordinary, no-build-tag Go package at `integration/internal/validation`; keep it independent of the tagged `integration/cli` TestMain.
- Checked-in policy JSON is embedded by `integration/assets/embed.go`, so `validation.LoadDefaultPolicy` works from arbitrary working directories.
- Stable catalog scenario IDs use the existing integration test entry-point names, including `TestGetFamilyUserTask`, `TestGetFamilyUserTaskVariables`, and `TestVolumeGetFamilyUserTaskVariables` selected by the existing family regexes.
- The sequential aggregate runner lives in the ordinary `integration/runner` package, builds once (or honors an absolute executable with `C8VOLT_IT_BUILD=0`), and invokes exact catalog-owned Go test names without shell parsing.
- Runner child evidence is a strict JSON array at `C8VOLT_IT_EVIDENCE_FILE`; the tagged harness binds every record to `C8VOLT_IT_RUN_ID`, `C8VOLT_IT_SLICE_ID`, and the single child profile while preserving legacy reports additively.
- Tagged top-level scenarios register a cleanup callback before any skip/failure boundary; `TestMain` flushes their final outcomes to the runner-owned normalized evidence file after `m.Run`, while legacy reports remain unchanged.
- The runner passes the version observed by its per-slice readiness recheck through `C8VOLT_IT_OBSERVED_VERSION`; normalized records never infer actual version from configured profile metadata.
- Common release coverage is split into `common-preflight` (inventory, manifest, profile, smoke) and separately confirmed `common-examples`; the runner orders preflight first and Make preserves every legacy family target.
- Real-state fixture policy now lives in ordinary `integration/internal/validation/fixtures.go`: only canonical 8.9/8.10 minors enter real-state work, while shared embedded fixture selection still supports 8.8 baseline/volume use.
- All seven real-state family consumers share `selectedRealStateProfiles`; readiness preserves the configured minor plus the full observed gateway version and rejects mismatched or unparseable observations before mutation.
- Cascade scenarios derive exact C89/C810 parent, mid, and leaf process IDs and require all three deployed definitions before creating evidence; historical 8.9 matrix claims remain distinct from pending 8.10 live proof.
- The focused `batch` slice is 8.10-only and owns `TestBatchFamily` plus held-instance `TestRealStateProcessDefinitionDraining`; batch cancellation/deletion evidence separates submission, terminal accounting, and independent post-state.
- Tagged tests can classify a required failure as `product` through the normalized evidence override without accepting it as a gap; the null deletion-batch rejection was fixed separately in 9fbf1ed5 and verified with a uniquely isolated live draining fixture.
- Direct 8.10 compatibility scenarios reuse the default-local profile through the same HTTP/auth bootstrap while persisting only profile, observed version, keys, counts, states, and proof levels.
- Logical tenant roles are strict credential-free JSON mapped to four distinct default-local profiles; runner validation reads only inherited minor/tenant metadata, preflights all role profiles before mutation, and never persists authentication fields.
- Tenant catalog proof is role-specific: logical and physical absence remain named optional gaps, `physical` requires both groups, and every declared profile passes topology readiness before tenant mutations.
- Physical roles require two distinct `/physical-tenants/<id>/v2` routes, one shared logical tenant ID and declared cluster ID, plus an explicit default `/v2` equivalence control; retained evidence is limited to observable API routing, authorization, and resource separation.
- V810 provenance expectations independently pin 8.10.3 identity, commit, artifact hashes, ordered transformation hashes, tool versions, package identity, and canonical generation command.
- Network reproducibility overlays current API edits into a detached disposable worktree; the repository-boundary guard wraps commands with before/after v88/v89 content fingerprints and intentionally ignores integration-harness edits.
- The separate shell suite now supports C810 through the shared driver; its gate parses the gateway-only version into an exact major/minor before any fixture deployment or mutation, and its report is explicitly not aggregate certification.

## Decisions

- Strict release mode supports canonical minors 8.9 and 8.10 and joins every applicable required catalog row to current-run, per-profile evidence. Camunda 8.8 remains exploratory baseline/volume scope.
- `passed_with_gaps` is reserved for named nonrequired proof gaps and always has `coverageComplete=false`; required rows cannot carry accepted gaps.
- Minimum proof levels are explicit and incomparable where necessary: controlled proof never satisfies live proof, and submission/dry-run proof cannot satisfy live post-state.

## Gotchas

- Child slice and observation paths must stay lexically beneath the absolute invocation root; reducers reject foreign-run, unselected-profile, duplicate, unknown, and foreign-minor records.
- Every persisted request, target, slice, scenario evidence, catalog, manifest, and summary uses schema version 1.
- Runner manifests persist planned/running state before child completion; a hard kill therefore remains incomplete, and a readiness loss leaves later slices planned rather than submitting mutations.
- Summary reasons are redacted; do not persist raw configuration, credentials, or credential-bearing route identities in manifests and child artifacts.
- Iterations 10–100 were blocked without a selected target. The operator subsequently authorized disposable c810local; observed version is 8.10.3. Consult validation.md for retained runs and current acceptance status.

## Reusable Commands

- `go test ./integration/internal/validation -race -count=1`
- `go test ./integration/internal/validation ./integration/runner -race -count=1`
- `go test ./integration/... -race -count=1`
- `go vet ./integration/internal/validation ./integration/runner`
- `python3 api/tests/v810_provenance_test.py`
- `bash api/tests/v810_generation_test.sh`
- `C8VOLT_V810_NETWORK_REPRODUCIBILITY=1 python3 api/tests/v810_provenance_test.py`
- `bash -n integration/scripts/run-suite.sh integration/scripts/run-c*-suite.sh integration/scripts/lib/suite-lib.sh integration/scripts/tests/run-suite_test.sh`
- `bash integration/scripts/tests/run-suite_test.sh`

## Do Not Repeat

- Do not read policy files through repository-relative runtime paths; use `validation.LoadDefaultPolicy`.
- Do not infer completeness from child exit code or discovered files; reduce the frozen catalog against normalized current-run evidence.

## Current Handoff
- T010 live acceptance is now authorized on c810local (8.10.3). Adapter, examples, and isolated draining fixes passed focused validation; the aggregate completed 31/32 slices successfully. The batch failure was repaired and passed separately; a clean aggregate is still pending. Evidence is recorded in validation.md. Do not restart Ralph or treat missing tenant/older-version proof as completed coverage.
