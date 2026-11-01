# Validation Guide: Business-ID Filters and Process Selection

This guide is for validating the implementation after it is written. It does not claim that proposed tests exist or pass today. Planning uses document checks only.

## Prerequisites and isolation

- Use branch `361-business-id-selection` at the repository root and the Go toolchain declared in `go.mod`.
- Read `AGENTS.md`, `specs/ralph-implementation-rules.md`, [plan.md](plan.md), [data-model.md](data-model.md) and the four contracts.
- Tests must own every backend dependency: generated-client mocks, test-owned HTTP servers, temporary configuration, and isolated CLI/PTY subprocesses. Never point a fixture at a real configured endpoint.
- Inspect selected targets and build constraints before execution. Do not run integration/demo/smoke targets, enable live-test tags, load a real profile, inspect clusters, provision services, deploy BPMN, or clean up live resources.
- Strip inherited application/helper environment overrides and build flags. Existing subprocess fixtures must generate their own temporary config and verify that requests reach only their owned fake endpoints. Do not print environment values or credentials.

The following temporary wrapper preserves ordinary toolchain/cache access while removing application/helper overrides and disabling inherited Go flags and make flags. It does not make an unsafe test safe; reviewing fixture ownership remains required.

```bash
validation_runner=$(mktemp /tmp/c8volt-361-tests.XXXXXX)
cat > "$validation_runner" <<'PY'
import os
import subprocess
import sys

blocked_prefixes = ("C8VOLT_", "CAMUNDA_", "ZEEBE_", "GO_WANT_")
blocked_names = {"KUBECONFIG", "MAKEFLAGS", "MAKEFILES", "MFLAGS", "GOPACKAGESDRIVER"}
clean = {
    key: value for key, value in os.environ.items()
    if not key.startswith(blocked_prefixes) and key not in blocked_names
}
clean.update({"GOFLAGS": "", "GOENV": "off"})
raise SystemExit(subprocess.call(sys.argv[1:], env=clean))
PY
```

Use `python3 "$validation_runner" ...` for commands below. Run from the repository root. Do not execute the commands as a planning-only check.

## Targeted checks after each behavior unit

Start with the closest package; run only the groups affected by the current work. These patterns include existing regression areas and proposed `BusinessID` test families. Use `-list` first and require the intended new tests to be present; a “no tests to run” result is not coverage.

```bash
python3 "$validation_runner" go test ./internal/services/processinstance/... -list 'Test.*(BusinessID|Search|Selection|Traversal|Children|DryRun|Discover|SelectedMutation)'
python3 "$validation_runner" go test ./internal/services/processinstance/... -run 'Test.*(BusinessID|Search|Selection|Traversal|Children|DryRun|Discover|SelectedMutation)' -count=1

python3 "$validation_runner" go test ./internal/services/usertask/... -run 'Test.*(BusinessID|Search)' -count=1
python3 "$validation_runner" go test ./c8volt/process ./c8volt/task -run 'Test.*(BusinessID|Search|Selection|Convert|SelectedMutation)' -count=1

python3 "$validation_runner" go test ./internal/services/ops -run 'Test.*(BusinessID|RepairProcessInstances|Retention|Orphan|SlowProcessAnalysis)' -count=1
python3 "$validation_runner" go test ./c8volt/ops -run 'Test.*(BusinessID|Convert|Repair|Retention|Orphan|Slow)' -count=1

python3 "$validation_runner" go test ./cmd -list 'Test.*(BusinessIDSelection|BusinessIDCancel|BusinessIDOperations)'
python3 "$validation_runner" go test ./cmd -run 'Test.*(BusinessIDSelection|BusinessIDCancel|BusinessIDOperations)' -count=1
python3 "$validation_runner" go test ./cmd -run 'Test.*(GetProcessInstance|GetUserTask|WalkProcessInstance|CancelProcessInstance|DeleteProcessInstance|UpdateProcessInstance|ResolveProcessInstance|SuspendProcessInstance|ResumeProcessInstance|OpsRepairProcessInstance|OpsExecuteRetentionPolicy|OpsPurgeOrphanProcessInstances|OpsAnalyseSlowProcessInstances|ConfirmationEmptySelectorResults)' -count=1
```

For new common/domain validators or incident resolution planning, run targeted tests in those owner packages and at least one process/ops consumer. Do not omit tests solely because their names fall outside the illustrative patterns; update the focused pattern to cover the actual changed tests.

## Required scenario outcomes

### Native filters and compatibility

Use [search contract](contracts/search.md). Own fake requests must prove exact scalar BID in the custom PI body and native task body, including BID-only, combined predicates, tenant selection, paging, totals and enrichment. Verify explicit empty versus absent, literal commas/whitespace/Unicode/wildcard text, copied pointer isolation and description/report round trips. Older-version supplied-BID cases make zero native search/mutation requests; nil BID retains old behavior. Pass a v810-prepared or caller-supplied BID plan directly to v88/v89 process execution and require unsupported failure with zero mutation requests, including an empty prepared plan.

Force the direct-incident shortcut through both CLI eligibility and direct public/service input. The BID predicate must survive by selecting the normal PI route, including a match found only on a later page. Task fixtures deliberately use a stored BID different from the parent PI's current BID and must not trigger identity-substitution lookups.

### Complete selection and unique walk

Use [selection contract](contracts/selection.md). Fixtures cover duplicate keys before a limit, sparse pages, exact versus lower-bound totals, cursor/offset continuation, cycles/nonprogress, cancellation and later-page failures. A bounded collection is distinguishable from exhaustive discovery. Walk with one distinct key requires exhaustion; two distinct matches fail even in one family; duplicate observations of one key do not fail. Empty exhausted discovery never calls walk or mutation.

### Cross-family cancellation

Use the exact graph and expected sets in [cancellation contract](contracts/cancellation.md). Require three selected seeds, two roots, seven affected instances and four additional members, with duplicate seeds and a sparse intermediate page. Require one planning expansion per root and no duplicate submissions caused by overlap. A limit of two selects `A1` and `A2` despite duplicates and still expands all four members of family A.

Add 1001 direct children plus a deeper descendant. Capture requests to prove later child pages were read and every observed family member appears in full JSON/verbose evidence. A failure on the last search or descendant page must leave mutation counts at zero. Confirmation occurs only after the full scope is frozen; a new matching independent root introduced afterward is never selected or submitted.

Separate the fake's backend cascade behavior from client-target assertions: existing root cancellation may affect concurrent descendants, and read-only verification may observe them. The test must not assert an unsupported atomic snapshot guarantee.

### Other mutation scopes

Exercise update-variable plans without family expansion, resolution with its existing family/incident rules and reviewed work preserved, exact suspend/resume, and explicit descendant-only tree plans. Check ordinary wait, no-wait, state/precondition failures, partial action results, and actual explicit-key authorization/preview regressions. Assert tenant scope on BID discovery before any explicit-key administrative option transition.

For delete, retention and orphan plans, create a new child in the fake after approval. Require frozen-member execution in planned children-before-parent order, no delete request for the new child, and preserved failure evidence if the backend rejects the planned action. This distinguishes a frozen executor from the current recursive live-child discovery.

### Operations and audit evidence

Use [operations/output contract](contracts/operations-output.md). Cover all four operations through actual command and public entry paths. Slow analysis supports BID-only and definition+BID, with the additive mode and no key/search mixing. Verify every duplicated conversion, both report directions, safe Markdown, and criterion-versus-observation identity.

Repair/retention/orphan confirmation consumes the same typed reviewed plan; mutate the fake discovery responses during confirmation and verify no new scope is admitted. Capability checks run before prefilled-key/plan shortcuts. Replace the contradictory orphan-count regression with inspected-distinct-candidate semantics: a non-orphan child consumes the documented candidate limit, and duplicates do not.

Use real terminal stdin for BID slow analysis whose sparse/unknown-total pages ultimately exhaust empty: require zero preflight prompts and zero enrichment calls. For a later nonempty page, require the existing prompt only after an actual match is selected and before enrichment, preserving decline/EOF behavior.

### Empty scopes, diagnostics and terminals

Use owned backend responses with real terminal stdin via existing `testx` terminal helpers. Capture stdout and stderr separately, including configured and inherited destinations. Verify normal and dry-run empty selection across human, JSON, keys-only and quiet, plus supported auto-confirm/automation/no-wait combinations. Quiet plus JSON still yields one success envelope; keys-only is zero bytes; no mutation or confirmation occurs. A user abort or discovery error must never pass as empty success.

For successful mutation/dry-run/verbose output require empty human stdout, INFO logger records, plain and structured formats, and suppression under quiet/WARN. For repeated resource failures embed distinctive URL, method, body and resource-key strings; require compact aggregate errors at normal/verbose/quiet, no raw HTTP details on either stream, original exit classification and retained partial outcomes. Exercise DEBUG independently via flag and configured level, including redaction; stdout remains free of diagnostics. JSON decodes once and then returns EOF, with no second error emission. Tests must fail if logging is replaced by a direct print or the raw joined error.

Retain nonempty approve/decline/EOF and redirected-output prompt tests. Report unsupported PTY platforms as skipped coverage rather than passing prompt validation.

## Formatting, docs and final breadth

Run `gofmt` only on changed Go files. When command metadata/help changes, regenerate documentation locally:

```bash
python3 "$validation_runner" make docs-content
```

After targeted runtime checks pass across the assembled feature, run the full race suite once. This is justified by shared discovery, traversal, frozen-plan and cross-package conversion changes; do not repeat it for subsequent prose-only edits.

```bash
python3 "$validation_runner" make test PKG=./...
rm "$validation_runner"
```

Record commands, actual tests matched, outcomes, request-count evidence and any skips. No live-engine check is required or implied by these tests. Real-engine concurrent cascade behavior remains an explicitly unverified coverage boundary unless separately tested in a user-requested target-specific workflow.
