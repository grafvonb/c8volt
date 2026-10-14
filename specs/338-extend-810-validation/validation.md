# Acceptance validation — issue #338

Date: 2026-10-14. T010 remains open: the aggregate failed before the final batch harness repair; the repaired batch slice passes separately. No clean aggregate result is claimed.

## Fixes and offline results

- Fixed v810 resource deletion to accept a valid response without a batch.
  Runtime definitions drain asynchronously; the existing process-definition
  workflow still waits for disappearance unless no-wait is set.
  Missing payloads, wrong/missing resource keys, and batches without keys remain errors.
- Fixed the aggregate runner to stop later slices for a profile when its common
  preflight slice fails; added a fake-subprocess regression.
- Targeted resource, process-definition (all adapters), validation policy and
  runner tests passed with the race detector.
- Full `make test` passed (all packages, race detector, count=1).
- `bash integration/scripts/tests/run-suite_test.sh` passed.

## Authorized live invocation

The user explicitly selected disposable `c810local` and authorized mutation.
Preflight observed Camunda **8.10.3**.

```sh
C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 C8VOLT_IT_WORKDIR=/tmp/c8volt-338-release-evidence make integration-test-all C8VOLT_IT_AUTOMATION=1 C8VOLT_IT_GO_TEST_FLAGS=-v
```

Evidence root:
`/tmp/c8volt-338-release-evidence/release-20261014T155401Z-cfbd7e01133e`.

The manifest records slice progress; final `release.json` and `summary.md`
record the verdict. Slice directories retain separate command streams,
normalized scenario evidence, and existing family reports.

## Review limits and follow-ups

Focused review covered runner gates, evidence registration, tenant assertions,
draining coverage, and the deletion adapter/workflow. This is not an exhaustive
review of every added assertion.

- No tenant roles were supplied. Logical/physical live isolation remains a gap.
- No 8.8/8.9 live environment was selected. Ordinary adapter regressions passed;
  older-version live regression remains unverified.
- Restricted-principal tenant coverage uses dry-run, so it does not prove
  backend mutation rejection. Physical separation compares keys across endpoints;
  endpoint-specific fixture markers would avoid assuming keys are globally unique
  across separate stores. Address these before relying on tenant certification.
- Normalized evidence assigns catalog proof levels to top-level test outcomes.
  Review subtest skips and actual reports before interpreting a passed family
  as complete proof.
- Track repeated blocked Ralph iterations separately in
  [ralph-follow-up.md](ralph-follow-up.md). Ralph was not restarted.

## Live result

The aggregate was intentionally interrupted during common examples after finding
unbounded watch examples. The retained verdict is `incomplete`, with 35 missing
requirements, 8 gaps, and no completed failing slice. This is not a release pass.
No test/watch descendants remained after interruption.

The examples harness now bounds process-definition watch examples with
`--backoff-timeout 3s` and records the substitution. A direct live read-only watch
exited zero and emitted `watch stopped: timeout reached`.

A separate focused `TestRealStateProcessDefinitionDraining` run failed after
132 seconds: definition `2251799813685337` remained visible in `DRAINING`
after its two-minute cleanup deadline. Its deletion submission succeeded with
`outcome: accepted`, HTTP 200, and no batch, verifying the adapter fix live.
Evidence: `/tmp/c8volt-338-draining`. Final disappearance is not proven; investigate
held instances and server draining/export timing before rerunning acceptance.
T010 stays unchecked. The full aggregate was not restarted.

Read-only diagnosis found three pre-existing ACTIVE child instances on definition
2251799813685337, started on 2026-10-13: 2251799813685367,
2251799813685381, and 2251799813685390. They have parent/root instance keys.
The draining scenario reused a shared fixture definition in this dirty cluster;
its batch cancellation did not remove those child instances. This explains why
final disappearance was not established and is a fixture/cleanup follow-up,
not evidence that the null-batch adapter still fails. Prefer a uniquely deployed
held-instance fixture for this test, or deliberately account for parent/root
cancellation before asserting cleanup. Diagnostic snapshot:
`/tmp/c8volt-338-draining-active.json`.

The repaired common examples slice completed in 31 seconds, with all four
process-definition watch examples passing. The slice still fails on independent
harness gaps: empty quoted tenant arguments are lost during tokenization,
profile/resource/user-task placeholders are unresolved, and a hard-coded tenant-a
example cannot find the default-tenant fixture. Evidence:
`/tmp/c8volt-338-examples/examples.json`. These are acceptance blockers to resolve
before another aggregate run; no failing example was waived.

Final `git diff --check` passed. At that checkpoint changes were uncommitted; existing Ralph progress
and memory edits were preserved.

## Follow-up fixes and rerun (2026-10-14)

The previously listed example and draining blockers are now repaired:
- Example arguments preserve empty quoted values; regression covers single/double
  quotes and JSON containing spaces.
- Profile, resource, and user-task placeholders use selected-profile/observed
  fixture identities. The documentation tenant-a is substituted with the selected
  tenant, with the substitution retained in evidence.
- Watch examples use a three-second lifecycle timeout.
- Draining deploys its own uniquely named BPMN definition and starts one held
  instance, avoiding unrelated pre-existing child instances.

The targeted parser/substitution, examples, and draining tests passed together
in 47.450 seconds against c810local (8.10.3). Evidence root:
`/tmp/c8volt-338-fixed-slices`.
Examples: 681 inventoried, 269 executed, zero failed, 104 skipped and 308 blocked
under the existing explicit non-execution policy. Those counts are not a claim
that every example was executed. The draining test observed DRAINING, canceled
the held instance, and confirmed final definition disappearance.

Resource adapter and runner regressions also passed with the race detector.
The full aggregate rerun is retained at:
`/tmp/c8volt-338-aggregate-fixed/release-20261014T162424Z-2112b8a48d1f`.

Commits:
- `9fbf1ed5` — fix(resource): accept Camunda 8.10 draining deletions #338
- `732229f0` — docs(cli): use all-tenants in cross-tenant examples #338
- `4ddaa876` — fix(integration): isolate fixtures and repair example validation #338
- `07403956` — fix(integration): isolate batch fixtures and await accounting #338

## Aggregate outcome

The full rerun completed all 32 slices: 31 finished successfully, one failed
(batch). All baseline, volume, seven real-state, and common slices passed.
The isolated draining scenario also passed within the failed batch slice.
The aggregate verdict remains `failed`, with zero missing catalog requirements,
two failure records referring to the same batch failure, and eight named gaps.
The batch fixture expected four instances but selected fifteen from a reused
shared definition. Its counts were not weakened or waived.

Named gaps retained by the reducer:
- bpmn-error-mutation
- durable-standalone-resolve
- orphan-purge-candidates
- repair-partial-failures
- job-timeout-mutation
- listener-variants
- logical-tenant-infrastructure
- physical-tenant-infrastructure

Follow-up work isolates positive/negative batch fixtures using the same helper
as draining. The first isolated rerun exposed counter-export timing: terminal
COMPLETED was visible before completedCount updated. The harness now observes
exact expected total/completed counts for at most 30 seconds and rejects any
failed operation; resource post-state assertions remain mandatory. Production
WaitBatchOperation semantics were not changed.

Final focused rerun passed in 58.160 seconds: TestBatchFamily and
TestRealStateProcessDefinitionDraining. Evidence:
`/tmp/c8volt-338-batch-accounting`. This confirms the previously failing slice
with isolated fixtures and exact eventual accounting. It does not replace or
rewrite the failed aggregate evidence. A clean aggregate rerun is still needed
before closing T010; tenant and older-version limitations remain explicit.
