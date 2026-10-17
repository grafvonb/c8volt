# Validation Guide: Suspended Process-Instance State

This guide is for implementation validation. The planning phase has not changed runtime behavior or run these tests. See [contracts/process-instance-state.md](contracts/process-instance-state.md) for expected behavior and [data-model.md](data-model.md) for timestamp semantics.

## Prerequisites

- Use branch `339-suspended-instance-state`, the repository Go toolchain and existing dependencies.
- Complete the feature implementation and its named acceptance tests before treating any command below as proof; a passing run that selected no tests is not evidence.
- For live checks, use an explicitly designated disposable profile backed by observed Camunda **8.10.4**, with permissions for owned fixture deployment, instance creation, suspension, resumption and cleanup. Do not use production data.
- Keep the existing generated client and provenance unchanged at baseline **8.10.3**. Record the exact provenance from [planning-context.md](planning-context.md).
- Reuse the existing integration harness and evidence outputs. Read `integration/README.md` and applicable `integration/assets/all-command-go-integration-rules.md` before adding fixtures.

## Targeted local checks

Run from the repository root after the relevant slice is implemented:

```sh
go test ./internal/domain -run 'State|ObservableProcessInstanceCreation' -count=1
go test ./c8volt/process -run 'State|ProcessInstance|Expectation|Suspended' -count=1
go test ./internal/services/processinstance/... -run 'State|Expectation|CreateProcessInstance|Search|Suspended' -count=1
go test ./cmd -run 'GetPI|GetProcessInstance|Expect|RunProcessInstance|Completion|Suspended' -count=1
```

Select the exact new/regression test names once written and verify that the intended tests ran. Keep deterministic first-observation and sequential timestamp tests free of sleeps; use controlled responses, bounded attempts or canceled contexts with existing fixtures.

Required checks:

1. Parse upper/lower/mixed-case suspension and whitespace; preserve invalid/unknown handling and cancellation aliases. Assert suspension is nonterminal in both models without changing ABSENT semantics.
2. Verify v810 serialized search requests contain SUSPENDED equality, ACTIVE remains exact, and ALL/unfiltered reads include suspended matches without filtering them out.
3. On both older versions, count requests and require zero for explicit suspended page/total searches, direct-incident searches (including an empty incident scope), and single/bulk waits with and without incident conditions. Include mixed desired states and empty keys.
4. Observe suspension as the first creation lookup and confirm successful visibility. Preserve older-version default creation, no-wait, timeout, error and terminal-observation behavior.
5. Read/convert/render the same instance first with a timestamp and then with nil/absent timestamp. Require UTC RFC3339Nano when present and omitted `suspendedDate` when absent. Test both facade conversion directions and nil/zero source dates. No instance watch mode currently needs modification.
6. Exercise real command paths for nonempty and empty suspended selection: human, JSON, keys-only, quiet, quiet+JSON and quiet+keys where supported. Capture streams separately, decode one envelope and require EOF, assert exact human output and zero-byte empty keys output, and check request counts and lack of added prompts. Preserve existing paging/interactive regression coverage; use real terminal stdin for any prompt-eligibility test required by AGENTS.md.
7. Verify completion without config/bootstrap and help/validation text. Do not advertise new mutation eligibility or broaden existing command-specific state restrictions.

After relevant implementation slices pass:

```sh
make docs-content
make test
```

Review generated docs with the source metadata and README. The final race suite is justified by shared domain/service-contract and concurrent-wait consumers; do not rerun it solely for later prose edits.

## Live validation on 8.10.4

Existing nonmutating preflight:

```sh
C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10 make integration-cli-common-preflight
```

Use the actual designated profile if its name differs. The release variable accepts the minor line, not `8.10.4`. The focused scenario must additionally verify the full observed server version is 8.10.4 before setup mutations; the minor gate alone is insufficient.

Implement a focused `TestGetFamilySuspendedState` in the existing `integration/cli` structure so `integration-cli-get` includes it. The following command is runnable **after that new test exists**; first list the test and require its presence to prevent a false pass:

```sh
go test -tags=integration ./integration/cli -list '^TestGetFamilySuspendedState$'
C8VOLT_IT_PROFILES=c810local C8VOLT_IT_RELEASE_VERSION=8.10   go test -tags=integration ./integration/cli -run '^TestGetFamilySuspendedState$' -count=1 -v -timeout=20m
```

The test uses isolated held instances and the existing generated v810 client only for setup suspension/resumption. It validates public lookup and CLI search/expectation/metadata behavior against independent observed state, including active-only exclusion and all-state inclusion. On resumption, verify a fresh response no longer carries the suspension timestamp. Resume owned instances before normal cleanup so the scenario does not depend on pending suspended-hierarchy cancellation/deletion work from #341. Report cleanup failure separately rather than hiding it.

For operator inspection using the fixture's recorded key:

```sh
./bin/c8volt --profile c810local get process-instance --key '<owned-instance-key>' --json
./bin/c8volt --profile c810local get process-instance --state suspended --json
./bin/c8volt --profile c810local expect process-instance --key '<owned-instance-key>' --state suspended --json
```

Replace the quoted key placeholder with an owned suspended fixture key. Restrict list comparisons to the fixture's process definition/tenant using existing selectors so unrelated cluster instances do not distort counts. Terminal expectation failure is verified with bounded timing rather than an unbounded manual wait.

## Evidence and completion

Record the tested Git revision and dirty status, actual server version, configured profile/minor, generated-client provenance (including tag/commit/hash), scenario results, command streams/exit codes, and fixture cleanup outcome in the existing evidence structure. Exclude credentials. Make helpers/fixtures reusable by #342.

No live endpoint was used during planning. If 8.10.4 or required permissions are unavailable during implementation, report live validation as unavailable with the concrete reason. Do not relabel a skip, different server patch, no-tests run, or partial coverage as a passing live result.
