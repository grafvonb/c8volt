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
- Tagged tests can classify a required failure as `product` through the normalized evidence override without accepting it as a gap; the current null deletion-batch rejection remains intentionally unsuccessful until a separate adapter fix lands.
- Direct 8.10 compatibility scenarios reuse the default-local profile through the same HTTP/auth bootstrap while persisting only profile, observed version, keys, counts, states, and proof levels.

## Decisions

- Strict release mode supports canonical minors 8.9 and 8.10 and joins every applicable required catalog row to current-run, per-profile evidence. Camunda 8.8 remains exploratory baseline/volume scope.
- `passed_with_gaps` is reserved for named nonrequired proof gaps and always has `coverageComplete=false`; required rows cannot carry accepted gaps.
- Minimum proof levels are explicit and incomparable where necessary: controlled proof never satisfies live proof, and submission/dry-run proof cannot satisfy live post-state.

## Gotchas

- Child slice and observation paths must stay lexically beneath the absolute invocation root; reducers reject foreign-run, unselected-profile, duplicate, unknown, and foreign-minor records.
- Every persisted request, target, slice, scenario evidence, catalog, manifest, and summary uses schema version 1.
- Runner manifests persist planned/running state before child completion; a hard kill therefore remains incomplete, and a readiness loss leaves later slices planned rather than submitting mutations.
- Summary reasons are redacted; do not persist raw configuration, credentials, or credential-bearing route identities in manifests and child artifacts.

## Reusable Commands

- `go test ./integration/internal/validation -race -count=1`
- `go test ./integration/internal/validation ./integration/runner -race -count=1`
- `go test ./integration/... -race -count=1`
- `go vet ./integration/internal/validation ./integration/runner`

## Do Not Repeat

- Do not read policy files through repository-relative runtime paths; use `validation.LoadDefaultPolicy`.
- Do not infer completeness from child exit code or discovered files; reduce the frozen catalog against normalized current-run evidence.

## Current Handoff
- Start US4 with T006: implement credential-free logical tenant role validation and focused tenant scenarios, keeping required missing infrastructure unsuccessful and optional absence as a named gap.
