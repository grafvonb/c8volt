# Validation — recovery of #324

Date: 2026-09-25. Ralph remains stopped. No production or BPMN changes.

| Check | Result |
| --- | --- |
| Strict result-envelope and duplicate-key regression checks | Passed |
| Typed seed payload and embedded prefix checks | Passed, including 8.7/8.8/8.9/8.10 prefix cases |
| C89 standalone baseline | Initially failed on `$notIn`; passed after the user-approved removal of those two live cases |
| C89 effective display | Ordinary, filtered and called-process selection, JSON fidelity and human shortening passed |
| C89 called process | Ownership/readiness, local equality and process-value exclusion passed |
| C89 existing basic user-task regression | Passed |
| C89 filtered volume (3 seeds, batch 1) | JSON, keys, total, limit 2 and empty human/JSON/keys passed |
| Profile gate | C89 ready; c88local and c810local both report 8.9.16 and fail expected-version checks |
| C87 | Not selected; unsupported checks implemented but not live-validated |
| Command inventory | User-task drift resolved; unrelated drift remains (listed below) |

## Commands and evidence

Executed with `GOCACHE=/tmp/c8volt-gocache`, integration build tag, `-count=1`, and existing configured profiles:

- `go test -tags=integration ./integration/cli -run '^(TestSeededVariablePayload|TestEmbeddedFixturePrefix)$' -count=1 -v -timeout=10m`
- `C8VOLT_IT_PROFILES=c89local C8VOLT_IT_WORKDIR=/tmp/c8volt-324-recovery-baseline go test -tags=integration ./integration/cli -run '^TestGetFamilyUserTaskVariables$' -count=1 -v -timeout=10m`
- `C8VOLT_IT_PROFILES=c89local C8VOLT_IT_VOLUME_COUNT=3 C8VOLT_IT_WORKDIR=/tmp/c8volt-324-recovery-volume go test -tags=integration ./integration/cli -run '^(TestVolumeGetFamilyUserTaskVariables|TestGetFamilyUserTask)$' -count=1 -v -timeout=10m`
- `C8VOLT_IT_PROFILES=c88local,c89local,c810local C8VOLT_IT_WORKDIR=/tmp/c8volt-324-recovery-profiles go test -tags=integration ./integration/cli -run '^TestProfiles$' -count=1 -v -timeout=2m`
- `go test -tags=integration ./integration/cli -run '^(TestSeededVariablePayload|TestEmbeddedFixturePrefix|TestCommandInventory)$' -count=1 -timeout=10m`

The first baseline run reproduced HTTP 500 from `/v2/user-tasks/search` for `incident.$notIn=["99"]` and `incident.$notIn=["1"]`: "The search server was unable to process the request". Full command envelopes are retained under the baseline evidence directory. This is an observed response, not a proven diagnosis of whether the fault is in c8volt or Camunda. Those observations preceded the user-approved removal of both live cases across versions.

Remaining unrelated manifest drift: `capabilities --all-tenants`; process-definition batch-size/watch/watch-interval and output modes; retention-policy JSON; orphan-process-instance purge JSON/keys-only. These are not repaired in this feature.

Implementation is present; the approved C89 get coverage is now green. Other-version validation and unrelated inventory gaps remain. Do not treat repeated unchanged profile checks as progress. No full repository or aggregate destructive suite was run because the focused checks expose the actual remaining limitations.

Final checks: `TestUserTaskVariableResultContract` and `TestSeededVariablePayload` passed after diagnostic/readiness changes; gofmt and `git diff --check` passed. Existing Make regexes were inspected and select both new entry points. Changes remain uncommitted.

## Approved removal and rerun

Removed both live `$notIn` cases across all versions at user request. Similar behavior on other versions is an assumption only; live `$notIn` is no longer claimed as covered. Unit request-encoding tests are unchanged. No runtime skip or HTTP-500 suppression was added.

`C8VOLT_IT_PROFILES=c89local C8VOLT_IT_WORKDIR=/tmp/c8volt-324-get-without-notin make integration-cli-get C8VOLT_IT_AUTOMATION=1` passed (37.889s), exercising the complete previously failing get slice. The full `integration-test-all` aggregate was not rerun; later slices are not claimed to pass from this result. Formatting and diff checks passed. Changes remain uncommitted.

## Normal test suite follow-up

The full `make test` (`go test ./... -race -count=1`) was run. Its first sandboxed attempt could not bind local test-server ports and was interrupted. With those permissions granted, the suite reached the default ten-minute command-package timeout without reporting an assertion failure; all other packages passed.

`GORACE=atexit_sleep_ms=0 make test` then passed across all packages with race detection enabled. The normal Make recipe now applies this setting while preserving other GORACE options, removing the default one-second exit delay paid by each CLI test subprocess. No tests or race checks were disabled. The validated equivalent command, recipe dry-run, and whitespace checks were used instead of repeating the full suite solely for the recipe/documentation change.
