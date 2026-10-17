# Integration validation quickstart

Use a disposable environment: these tests mutate cluster state.

```sh
# Defaults: c810local, requested Camunda 8.10; shows settings and confirms.
make integration-test-all

# Override the selected profile/version.
make integration-test-all C8VOLT_IT_PROFILES=dev89 C8VOLT_IT_RELEASE_VERSION=8.9

# Scenario-level logs and unattended execution.
make integration-test-all C8VOLT_IT_GO_TEST_FLAGS=-v C8VOLT_IT_AUTOMATION=1

# Rerun just the affected repair group, with its own retained reports.
C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 \
C8VOLT_IT_WORKDIR=/tmp/c8volt-repair-check \
make integration-cli-ops-repair-volume C8VOLT_IT_GO_TEST_FLAGS=-v
```

Make runs existing targets serially and prints their commands/results directly.
Common preflight verifies the selected environment before mutation. A failing
group stops the invocation; rerun that existing target after fixing the failure.
The old Go runner and release.json contract are removed.

Use integration/assets/tenant-roles.example.json with existing default-local
profiles when tenant infrastructure is available. Set C8VOLT_IT_TENANT_ROLES to
its absolute path and C8VOLT_IT_REQUIRE_TENANTS=logical or physical to require it.
Without prerequisites, optional tenant checks skip; skips are not proof of support.

Existing per-group reports and logs remain in separate temporary directories by
default. An explicit workdir is an exact directory, best used for a single-group
rerun. The harness builds per group unless C8VOLT_IT_BUILD=0 and C8VOLT_IT_BIN
select an existing binary.

These are usage instructions, not evidence of execution. No tests were run for
the 2026-10-15 runner removal at the user's explicit request.
