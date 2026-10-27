# Quickstart: Isolated Resumption Validation

This guide applies after implementation. The planning command did not run these tests or validate a real engine. Use [CLI contract](contracts/cli.md), [service contract](contracts/service-api.md), and [data model](data-model.md) as expected behavior.

## Prerequisites and isolation

- Work on `347-resume-process-instances`, using the toolchain declared in `go.mod` (Go 1.26.0 / go1.26.2).
- Inspect new tests, helpers, build tags, and selected targets before running. All service dependencies must be generated-client mocks or test-owned HTTP fake servers; command subprocesses must receive test-owned config, URLs, and credentials.
- Do not run integration/smoke targets, `-tags=integration`, environment-backed helpers, or `integration/scripts/*`. Do not inspect/provision/probe live targets. Source edits to fixtures do not authorize execution.
- `testx` subprocess helpers inherit environment: remove inherited `C8VOLT_*` settings before tests, then let tests add only their own variables. Clear inherited Go flags and persisted Go environment configuration so live build tags cannot be activated indirectly. An explicit fake config alone is insufficient if environment overrides remain.
- Run from the repository root. Check planned test names exist before interpreting results; `go test` with no matching tests is not evidence. PTY-dependent skips must be reported as gaps.

## Targeted execution

After the corresponding tests exist, the following wrapper runs only the selected isolated slices with inherited application settings removed:

```sh
python3 - <<'PY'
import os, subprocess
clean = {k: v for k, v in os.environ.items() if not k.startswith('C8VOLT_')}
clean.update(GOFLAGS='', GOENV='off', GOWORK='off')
checks = [
    ['go', 'test', './internal/services/processinstance/...', '-run', 'Resum', '-count=1'],
    ['go', 'test', './c8volt/process', '-run', 'Resum', '-count=1'],
    ['go', 'test', './cmd', '-run', '^TestResumeProcessInstance', '-count=1'],
]
for command in checks:
    subprocess.run(command, env=clean, check=True)
PY
```

Use the same sanitized environment for follow-up commands. First list matching tests with `go test <package> -list '<pattern>'` and confirm each intended suite appears. Suggested names follow existing suspension tests; they are implementation targets, not tests already present at planning time.

## Scenarios and required evidence

| Slice / proposed tests | Fake setup and expected proof |
| --- | --- |
| Versioned `ResumptionSubmission` / `ResumeProcessInstance` | Assert POST path, empty typed body, valid empty 204, rejection statuses, malformed/nil responses, ambiguous retry chains, request-rate admission, timeout/interruption; only valid success establishes acceptance |
| `WaitForProcessInstanceResumption` | Accepted → ACTIVE and accepted → immediate COMPLETED confirm; CANCELED/TERMINATED stop as conflict; SUSPENDED/ABSENT/UNKNOWN remain bounded; auth/read failure retains evidence; identity mismatch and late reads never confirm; no mutation from polling |
| Internal collection `ResumeProcessInstances` | Stable deduplication, parent/child identities, all-key validation before dispatch, wanted workers/no-worker-limit, fail-fast, unattempted slots, partial outcomes and caller cancellation |
| Planner `PlanProcessInstanceResumption` | Nested and overlapping scopes, child-only boundary, active/terminal ancestors, independently suspended descendants, mixed/unknown states, partial/unresolved/missing/malformed discovery, capability before reads, 1,000-child lookup constraint, no mutation during planning |
| Facade `Resum` | Mapping/copy boundaries, options, converted errors plus partial evidence, singleton routed through collection, factory v88/v89 guards including preview/plan, no discovery for exact keys |
| `TestResumeProcessInstanceSelection`, `Output`, `InvalidInputAndCapability`, `PartialEvidence`, `WriterFailure` | Full command execution against test-owned fakes; key/stdin validation, preview/approval/no-wait, single JSON envelope + EOF, original exit code, accepted-only keys, raw machine evidence retained, result writer failure precedence |
| `TestResumeProcessInstanceTreeSelection`, `TreeOutput`, `TreePlanningFailure` | Expanded plan/counts, no ancestor escalation, overlaps once, frozen execution without rediscovery, post-plan state-race rejection, complete exclusions, zero mutation after any discovery failure |
| `TestResumeProcessInstanceDiagnostics`, `SummaryLogging`, `TreeNoopLogging` | Separately capture stdout/stderr/configured destinations; exact severity/routing/record counts in plain and structured formats; INFO suppression at WARN/quiet; compact aggregate failure and no raw HTTP markers under normal/verbose/quiet |
| `TestResumeProcessInstanceTerminal`, `TreeTerminal`, `TreeEmptyTerminal` | Real terminal stdin with redirected stdout, configured/inherited stderr, prompt wording/default/EOF/decline, one frozen expanded count; empty eligible scope never prompts under normal/dry/no-wait and supported auto-confirm/automation combinations |
| `TestResumeProcessInstanceActivity` | Starts after approval, covers execution/wait, stops before output on success/failure/interruption, suppressed under existing protected modes |
| Fresh-date and regression slices | Resumption observation and subsequent get/expect omit stale suspendedDate when absent; suspension/cancel/delete/traversal behavior unchanged |

For diagnostics, use multiple failed keys with repeated distinctive URL, HTTP method, body, resource-key and secret markers. Assert compact grouped ERROR output and no raw details on either human stream with normal or verbose-only output, while quiet retains the error. Verify both DEBUG entry points and redaction without enabling verbose. JSON preserves existing machine error evidence but contains exactly one envelope; stderr cannot gain a duplicate human error. Successful mutation/preview/verbose information must be INFO records on configured/inherited destinations, not merely stderr substrings. Tests must fail if a direct print replaces logging or a raw aggregate replaces bounded reasons.

For empty tree scopes, cover dry-run and normal execution, no-wait, human/JSON/keys-only/quiet, quiet plus machine output, and supported automation/auto-confirm modes. Assert exact human wording, EOF after the one successful envelope, zero-byte keys, no prompt, no mutation, and fixed discovery request counts. Use real PTY input for prompt-free proof. Retain nonempty, explicit-key, abort, and failure regressions; do not treat sparse intermediate discovery as authoritative empty completion.

## Regression, formatting, and generated documentation

After targeted checks pass, run the closest suspension regressions for any shared normalization or structural traversal extraction. Inventory affected interface consumers and run their isolated package tests; compilation failures in test doubles are part of the work unit. Use `gofmt -w` on touched Go files.

The new feature touches internal contracts and concurrent collection behavior. Once all relevant slices pass, use a sanitized `make test` when this shared/concurrent impact warrants the broader race run; it is `go test ./... -race -count=1`. Ensure no integration build tags or live-environment helpers are selected first. Do not rerun it solely for docs or commit creation.

After source help/metadata and README updates, run `make docs-content` in the sanitized environment (the target generates command docs and syncs README; it does not run live services). Review generated `docs/cli/*` and `docs/index.md`; do not hand-edit them. Check relevant source fixture changes with formatting/static review only. Do not execute the integration fixtures or claim their scenarios passed.

## Evidence to record

Record executed commands, actual matched tests, pass/fail/skip results, stream/exit assertions, generated-doc review, and any remaining gap. For planning-only edits, use whitespace, placeholder, local-link, and requirement-coverage checks instead of runtime tests. Real Camunda resumption timing, buffering, and permission behavior remain unverified until a separate explicit target-specific live workflow is requested.
