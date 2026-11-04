# Suspended state validation — iteration 6

Validated on 2026-10-17 against disposable profile `c810local`, configured minor
`8.10`, observed gateway **8.10.4**. The focused test checks the full patch before
fixture deployment or mutation. Generated-client baseline remains **8.10.3**.

## Tested source and evidence

Tested HEAD: `ed2b0df45d5264564708b321fc7ddc0b42c4b027` with the following uncommitted
integration work (captured by the test in `suspended-revision.json`):

```text
 M integration/README.md
 M specs/339-suspended-instance-state/quickstart.md
?? integration/cli/get_processinstance_suspended_test.go
?? integration/cli/suspended_fixture_test.go
```

This is a precommit working-tree validation, not a claim that the parent revision
alone contains the new scenario. The coordinated iteration commit adds the tested
sources and this report; later changes are documentation and Ralph records only.

Successful run evidence: `/tmp/c8volt-339-iteration6-live-2/`.
Test stdout/stderr: `/tmp/c8volt-339-iteration6-live-2.log`.
Each command's separate stdout/stderr paths, arguments, timestamps and exit code
are retained in `suspended-commands-c810local.json` and `logs/`. Generated setup
HTTP status/body and independent instance observations are under `data/`.
Profile/version, source revision/provenance, setup proposal and cleanup records
are separate JSON files in the same evidence directory. These are local run
artifacts; the acceptance facts and reproduction commands are retained here.

## Commands and outcomes

- `C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 make integration-cli-common-preflight`
  passed; observed gateway 8.10.4. Evidence:
  `/var/folders/jc/60f5tdds44d2v3b4fc0xs5700000gp/T/c8volt-all-command-it-3721012964/`.
- `go test -tags=integration ./integration/cli -list '^TestGetFamilySuspendedState$'`
  listed exactly the named test and passed. `Makefile` uses `-run TestGetFamily`
  for `integration-cli-get`, so the scenario belongs to the existing slice.
- The focused live invocation below passed, with no skipped scenario, in 36.986s
  including the harness binary build; the test itself took 27.15s.

```sh
C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 \
C8VOLT_IT_WORKDIR=/tmp/c8volt-339-iteration6-live-2 \
go test -tags=integration ./integration/cli \
  -run '^TestGetFamilySuspendedState$' -count=1 -v -timeout=20m
```

Use a new evidence directory for a repeat invocation. Direct Go execution uses
the existing default-local profile and does not ask Make's confirmation prompt.

## Live acceptance

Owned definition: `2251799813692272`. Owned instances: suspended
`2251799813692273`, active `2251799813692280`, terminal control
`2251799813692287`. Comparisons are definition-scoped, independent of unrelated
cluster data. The terminal control was canceled through the CLI.

| Check | Observed result |
| --- | --- |
| Generated suspension setup | HTTP 204; independent read observes SUSPENDED |
| Public lookup | State report and instance preserve SUSPENDED; nonterminal |
| Suspension metadata | `2026-10-17T12:12:04.23Z` preserved in public and CLI results |
| CLI `--state suspended` | Exactly the suspended owned key; exit 0 |
| CLI `--state ACTIVE` | Exactly the active control key; exit 0 |
| CLI `--state all` | All three owned keys, including suspended and terminal; exit 0 |
| State expectation | Mixed-case suspended input succeeds and preserves SUSPENDED; exit 0 |
| Combined expectation | SUSPENDED plus incident=false succeeds with `ok=true`; exit 0 |
| Terminal expectation | Completed/canceled/terminated/absent remains unsatisfied; bounded 1s retry timeout, exit 4, outcome=failed, class=timeout |
| Suspended lookup/search JSON | Exactly one successful envelope and EOF; suspension timestamp present |
| Generated resumption setup | HTTP 204; independent read observes ACTIVE with null suspendedDate |
| Fresh public read | ACTIVE with empty SuspendedDate |
| Resumed lookup/search JSON | Exactly one successful envelope and EOF; suspendedDate omitted, not stale or null |
| Cleanup | Resume before ordinary forced owned-definition deletion; exit 0; all three instance keys confirmed absent |

JSON get commands had empty stderr. The terminal timeout is expected negative
acceptance, not a test failure. Cleanup status is independently recorded as
`cleaned`, `attempted: true` in `suspended-cleanup-c810local.json`.

## Initial failure retained

The first focused run used uppercase `ALL` as a CLI sentinel and failed with
backend HTTP 400/CLI exit 2. Existing CLI help documents lowercase `all`; the
fixture was corrected without changing production behavior. The timeout
assertion was strengthened to require exit 4 and class=timeout. Resume tracking
now clears only after an independent ACTIVE observation. The complete corrected
scenario was rerun successfully. Initial evidence and output remain at
`/tmp/c8volt-339-iteration6-live-1/` and
`/tmp/c8volt-339-iteration6-live-1.log`; cleanup also succeeded in that failed run.

## Unchanged client provenance

- Repository: `https://github.com/camunda/camunda.git`
- Tag: `8.10.3`
- Commit: `3961b79c3fdaf43c48691ed77672d6db42d0abcc`
- Source: `zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml`
- Source SHA-256: `857109dabcb998801f749282836a6d2af1a137284d538161801f0c348ae57837`
- Prepared SHA-256: `461a1cc051a021a3cf937dbfd28a92537cf9b8b4381bcfb9ad6ab6e1d1f91d1b`
- Generated client SHA-256: `589351ac972ab595af721f2e9647a273819e56a9601f7af2aaaf98e6def2f319`
- Tools: Redocly `2.29.2`, oapi-codegen `2.5.0`.

The generated file's actual SHA-256 matches provenance. No generated-client or
provenance diff was introduced. Existing generated get/expect/run references and
README-derived homepage were reviewed against the shipped behavior. This unit
changes integration guidance only, so no CLI documentation regeneration is needed.

## Final local quality gates

`make test` passed in full (exit 0), running `go test ./... -race -count=1`
with the Makefile's race-exit-delay setting. Log:
`/tmp/c8volt-339-iteration6-make-test.log`. This is the final integrated runtime
check for shared state, interface and concurrent-wait changes across T001–T005.
The newly added tagged integration sources passed the focused live run separately.
No full-suite repeat was needed for subsequent prose/record edits.

Touched Go files pass `gofmt` verification; `git diff --check` passes. Generated
get/expect/run documentation remains aligned, and generated clients/provenance
are unchanged. All six feature tasks are validated; the iteration records and
terminal handoff are included in the coordinated work-unit commit.
