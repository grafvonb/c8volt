# Validation Quickstart

These commands describe the implemented interfaces. Live commands still require
explicitly selected disposable environments and separate execution
authorization; this feature's offline implementation iterations did not run
them.

## Prerequisites

- Go/toolchain from `go.mod`, Make, and the repository's existing dependencies.
- Existing profiles in the default local c8volt configuration pointing only at disposable clusters. Example names below must be replaced by actual profile names.
- Camunda 8.10.3 for initial release evidence; an 8.9 environment for real-state regression and 8.8/8.9 profiles for baseline/volume regression.
- Operator-provisioned tenant infrastructure and credentials if tenant proof is requested. Role metadata names existing profiles without embedding credentials.
- A separately accepted resolution of the known v810 null-batch process-definition deletion defect for a clean release result. Until then, expect an explicit product failure in affected scenarios.

See [the interface contract](contracts/validation.md) for controls and result policy and [the data model](data-model.md) for evidence interpretation.

## 1. Deterministic validation without live clusters

```sh
go test ./integration/internal/validation ./integration/runner -race -count=1
go test ./internal/services/batchoperation/v810 -count=1
bash integration/scripts/tests/run-suite_test.sh
bash -n integration/scripts/run-suite.sh integration/scripts/run-c*-suite.sh \
  integration/scripts/lib/suite-lib.sh integration/scripts/tests/run-suite_test.sh
```

Expected: missing profiles, mismatched versions, whole-family skips, omitted scenarios, stale/duplicate evidence, interrupted runs, and unaccepted gaps cannot produce passed release results. Fake subprocess tests verify preflight ordering, per-profile evidence isolation, and finalization after failures without invoking Camunda.

Run targeted controlled resource-deletion and URL-prefix tests added by implementation. The deletion regression must not be reported passing until the separate production fix is present. Do not run the tagged harness merely to test the pure reducer: its TestMain currently builds and invokes CLI subprocesses.

## 2. Validate the 8.10 aggregate

```sh
C8VOLT_IT_PROFILES=dev810 \
C8VOLT_IT_RELEASE_VERSION=8.10 \
C8VOLT_IT_WORKDIR=/tmp/c8volt-release-evidence \
make integration-test-all C8VOLT_IT_GO_TEST_FLAGS=-v
```

The existing confirmation gate applies. Expected: nonmutating preflight before
destructive slices, all seven real-state families executed, and the five common
categories—inventory, coverage manifest, profile readiness, read-only smoke,
and examples—accounted for. The runner records the actual server version and
separate evidence per slice/profile. Known additional gaps produce an
explicitly incomplete coverage claim. Product failures, including the current
v810 null deletion-batch adapter dependency, produce a nonzero result.

Inspect the printed invocation directory's `release.json` and `summary.md`. Do
not infer completeness from a zero process exit alone: `passed_with_gaps`
explicitly has `coverageComplete=false`. Evidence levels remain distinct:
`live-post-state`, `submission-only`, `dry-run`, `controlled`, and `read-only`.

## 3. Prove wrong-version rejection

Using only a verified 8.9 profile:

```sh
C8VOLT_IT_PROFILES=dev89 \
C8VOLT_IT_RELEASE_VERSION=8.10 \
make integration-test-all C8VOLT_IT_GO_TEST_FLAGS=-v
```

Expected: unsuccessful preflight naming the absent requested minor and zero mutation commands. Confirm this ordering in fake-subprocess checks first. Do not alter production profiles to manufacture mismatches.

## 4. Validate tenant boundaries

Prepare the credential-free role file described in the contract. Its profiles must already exist in local configuration. Use the logical group to verify A/B discovery, search-scoped mutation, restricted-principal foreign-key rejection, and dual-authorized all-tenant behavior.

```sh
C8VOLT_IT_PROFILES=dev810 \
C8VOLT_IT_RELEASE_VERSION=8.10 \
C8VOLT_IT_TENANT_ROLES=/tmp/c8volt-tenant-roles.json \
C8VOLT_IT_REQUIRE_TENANTS=logical \
make integration-test-all C8VOLT_IT_GO_TEST_FLAGS=-v
```

Repeat with `C8VOLT_IT_REQUIRE_TENANTS=physical` only after provisioning the physical role group. Use the same logical tenant ID in both physical endpoints as an isolation control. Expected: positive own visibility, no foreign fixture visibility, unchanged other-endpoint resources after scoped mutation, and no cross-physical traversal from all-tenants. Default routing equivalence is verified with its explicit control profiles.

With `C8VOLT_IT_REQUIRE_TENANTS=none`, unavailable tenant infrastructure stays
an optional named gap. `logical` requires the complete logical role group;
`physical` requires both logical and physical groups. Missing required roles or
infrastructure fails before tenant mutations. Do not substitute logical tenant
results for physical proof.

## 5. Preserve older-version coverage

```sh
C8VOLT_IT_PROFILES=dev89 C8VOLT_IT_RELEASE_VERSION=8.9 \
make integration-test-all C8VOLT_IT_GO_TEST_FLAGS=-v

C8VOLT_IT_PROFILES=dev88,dev89 \
make integration-test integration-test-volume C8VOLT_IT_GO_TEST_FLAGS=-v
```

Expected: existing applicable families remain selected; no new requirement for 8.8 real-state coverage. Version-specific unsupported capabilities remain visible.

## 6. Supporting guards

After guard updates, run static provenance/hash checks first. Run regeneration reproducibility only through the updated isolated-checkout test path, with network and tooling prerequisites available. Verify both current v810 determinism and unchanged v88/v89 artifacts. Do not invoke the old in-place alpha4 regeneration test.

Verify the separate C810 shell runner independently:

```sh
C8VOLT_IT_PROFILE=dev810 integration/scripts/run-c810-suite.sh
```

It uses C810 fixtures and refuses a non-8.10 gateway before mutation. Its
command-level report is a broad workflow check, not a substitute for the
aggregate catalog and normalized `release.json` evidence.

## Evidence review

- Correlate every required scenario with the current invocation, target, observed version, and declared proof level.
- Check failure accounting and independent post-state for batch-backed mutations; keep immediate no-wait acceptance separate.
- Confirm DRAINING evidence describes process definitions, not batch operations.
- Inspect named gaps, interrupted slices, cleanup results, and unchanged tenant controls.
- Retain failed evidence. Never remove a required scenario or weaken its assertion merely to obtain a clean report.
