# Implementation plan: Global compatible refactoring

**Status:** planning only. **Date:** 2026-11-03. **Spec:** [spec.md](spec.md).
**Audit range:** `v4.4.0..33eb3dce`. **Planning checkout:** `370-facade-layering-refactor`.
**Feature directory:** `specs/371-global-refactoring-preparation/` (local sequence; no issue association).
A future implementation branch is not created, renamed or activated by this plan.

## Summary

Cover all twelve audited findings through shared lower-layer mechanics, indexed task/scope
lookup, command-aware failure plumbing, version-neutral raw decoders and focused source/test
organization. Keep CLI/public contracts frozen. Retain public compatibility entry points even
where there are no repository callers; use additive free functions instead of extending
existing public interfaces. The completed 370 refactor remains historical evidence.

Eighteen final work units include implementation, required test fixtures, formatting and
isolated validation. The [task list](tasks.md) is the executable sequence; the [consolidation
review](consolidation-review.md) records why tighter coupling was merged and larger changes
remain separate. No implementation, test execution, auto-commit, Ralph or publishing hook is
part of this planning turn.

## Technical context

- Go 1.26.0, toolchain go1.26.2; existing Cobra/pflag/Viper, slog, testify, yaml.v3,
  oapi-generated adapters and `testx` support. No dependency upgrades or new libraries.
- Platforms: existing CLI platforms; terminal fixtures use existing platform guards.
- Persistence: unchanged config/profiles/artifacts. New types describe internal observations,
  indexes or test contracts, never new persistent or machine-result schemas.
- Validation: units, generated-client doubles, test-owned HTTP, isolated command subprocesses
  and real-terminal stdin. Benchmarks are local pure computation. CI/docs tests stub publishing
  tools and bind no live services. No integration tags, cluster profiles or target designation.
- Performance target: build association indexes once and avoid repeated full target scans;
  normalize shared requested values once per compatible planning scope. No claimed speedup
  without isolated comparative evidence; backend request counts are a compatibility constraint.
- Scope: the 12 findings in spec.md across public facades, services, command support, tests,
  config glue and docs CI. The audit inventoried 1,267 Go files and 47 script/build/workflow
  sources; clone matches were candidates inspected for semantics, not a refactoring quota.

## Constitution check (pre-design and post-design)

| Principle | Result and evidence |
| --- | --- |
| Operational proof | PASS: separate submission/observation facts, frozen scope and confirmation rules retained; contracts forbid acceptance claims for no work. |
| Stable CLI and public APIs | PASS with explicit compatibility dispositions: unchanged exported signatures/types/interfaces; additive adapters only; C05 gates observable legacy diagnostic corrections. |
| Proportionate validation | PASS: planning uses document checks only. Per-unit targeted isolated checks; one justified broad race checkpoint after shared runtime migrations, no live completion gate. |
| Documentation matches behavior | PASS: no planned command grammar/help change. Docs generation only if authorized behavior/metadata actually changes; never regenerate CLI docs for file movement alone. |
| Repository-native changes | PASS: existing pool, JSON, command/logger and test patterns retained. Narrow extra packages justified below; no universal engine, framework or dependency. |

No unresolved technology selection needs external research or research-agent dispatch. Research
is local and recorded in [research.md](research.md). Existing runtime tests from 370 do not
prove this unimplemented plan; current planning ran no application tests.

## Architecture and proposed organization

All names below are proposed internal implementation locations, not application code created
by planning. Preserve packages and public declarations when splitting files.

### Shared JSON values (T001; F01)

Use focused `toolx/json_value.go` and existing `toolx/json_copy.go` for mechanical JSON
normalization, exact comparison and validated snapshot behavior. Keep normalization separate
from representation-preserving snapshots. Migrate service helpers in element/processinstance/
usertask updates and public converter helpers together; their context-specific error wrappers
remain local. Reuse `CopyJSONMap` only where its normalization semantics match the contract.
Do not add JSON workflow policy to facades. Include mutation-after-plan probes and retained
representation tests; a generic JSON round trip must not silently coerce integers or accepted
custom values. New helpers must not import facades/services/config.

### Neutral worker core with original public compatibility (T002; F02)

Move the existing worker implementation into `toolx/pool/execution`; preserve its algorithm,
DEBUG labels, input-index ordering, staggering and cancellation semantics. Keep original
`toolx/pool.ExecuteSlice`, `ExecuteNTimes`, `ExecuteBulkOperation`, `Reporter` and `Reports`
exports and exact signatures; original pool entry points delegate to the neutral core.
Internal production callers import only the neutral core (an import alias may preserve local
`pool.ExecuteSlice` call syntax). Enumerate every current consumer, including incident, ops,
element, job/completion, processdefinition, processinstance, its adapters/waiter, and usertask.
The legacy facade-option bulk helper stays compatible in the original package; it is not a
valid service dependency. Delete unused internal `common.RunBulk` only after a full reference
check; do not change the legacy public helper's logs or option behavior as incidental cleanup.
A transitive import guard verifies the internal core does not reach `c8volt/*` or `cmd`.

### Command failure aggregation and routing (T003–T004; F06/F03)

A focused `cmd/cmd_failure_summary.go` owns first-seen reason counting, three-reason bounding
and the existing remainder bucket. Six current formatter families supply allowlisted reasons
and retain their labels, punctuation, fallback wording and mode policies. The helper does not
inspect raw error strings or absorb command-specific status logic.

Then consolidate command-aware handling around `cmd/cmd_views_contract.go`, the configured
logger, bootstrap fallback helpers and `exitAfterRenderedResult`. Inventory all direct
`ferrors.HandleAndExit` uses, especially embed deploy/export, get-pd, user-task variable writes
and local config commands. Preserve bootstrap failures as a distinct local path and preserve
already-attempted-output handling. The public ferrors API remains unchanged. C05 explicitly
prevents treating removal of legacy raw diagnostics as an invisible refactor: reproduce any
conflict in an isolated command fixture, record the exact proposed wording in feature evidence,
and require acceptance for that correction. This is not permission to implement or bless a
normal-level HTTP disclosure. An unresolved required output case blocks that unit, not unrelated
safe units. No new command, output schema, error class or second envelope is a remedy.

### Task/scope indexes (T005; F04)

In usertask variable validation and result fan-out, build `scope -> target`, `task -> ordered
scope references` and `(scope, task) -> ordered names` once from the validated frozen plan.
Reuse existing domain identities and preserve first-seen/sorted sequences; maps are lookup
structures, not output iteration sources. Keep indexes invocation-local and out of public
plans/results. No verification requests are cached or removed. Retain unknown/conflicting
identity validation. Test sparse independent scopes, highly shared scopes and shuffled input;
measure the helper's local complexity without wall-clock pass/fail thresholds.

### Suspend/resume mechanics (T006–T009; F05)

Four outcomes keep validation practical:

1. Shared selected-descendant discovery/deduplication under processinstance; per-action
   eligibility/capability and public plan shapes remain explicit.
2. Shared ordered bulk preview/dispatch/merge mechanics using the neutral pool. Preserve
   preflight-before-mutation, empty requests, fail-fast placeholders and original error evidence.
3. Shared v810 submission-response/evidence mechanics. Preserve native calls, empty-204 rules,
   explicit rejection versus ambiguous response, retained acceptance/unknown evidence across
   retries, local admission/pacing failures and endpoint labels.
4. Shared waiter observation loop under processinstance/waiter. Keep per-action success/conflict
   predicates and diagnostic phrases explicit. Resume may confirm ACTIVE or COMPLETED;
   suspension confirms SUSPENDED and treats terminal outcomes differently. Deadline checks
   after a read, attempts, backoff, identity checks and no-wait exemptions remain unchanged.

Use narrowly named service files/subpackages only where required to avoid parent/adapter import
cycles. Prefer a small typed callback over a universal lifecycle policy object; do not introduce
an option-heavy state-machine framework. These shared mechanics serve two concrete operations.

### Variable raw decoding (T010–T011; F07)

Use `internal/services/usertask/variablepage` for effective-variable raw decoding and
`internal/services/variable/variablepage` for root/local raw decoding. Parent factories already
import adapters, so placing reusable decoder code directly in a parent that adapters import
would create a cycle. Each decoder uses raw bytes and domain facts, never generated clients.
Retain each version's request construction and converters that depend on generated types.
Effective variables require their established metadata; root/local reads retain their own
optional-total, strict-scope, tenant, truncation and cursor conventions. Do not merge these two
contracts into boolean-heavy generic pagination. Existing traversal owners remain unchanged.

### Service-owned selector and deployment preparation (T012–T013; F08/F09)

- Move selector normalization, per-ID existence/near-match query strategy and result facts into
  `internal/services/processdefinition/selectors.go`. Add an optional public free-function
  adapter in `c8volt/process/selectors.go` using the existing page-search capability of current
  interfaces. Its narrow reader adapter maps existing public page results to domain results;
  the service owns query strategy. Do not add a required method to `process.API` or root API.
  Command code keeps flags, prompts, formatted diagnostics and listing/rendering policy.
- Move deployment-to-run ID precedence and fallback BPMN parsing into
  `internal/services/resource/deployment_run.go`. Expose an additive mechanical free function
  from `c8volt/resource/deployment_run.go` mapping current public deployment/unit types into
  domain inputs and public creation inputs back out. The existing CreateProcessInstances
  service already owns ordered creation. Commands retain visible stage dispatch: deploy,
  render established deployment output, then invoke prepared creation. Do not collapse that
  sequence or invent a combined result envelope. File/embedded acquisition remains command-owned.

### Contract tests and file cohesion (T014–T016; F10/F11)

Consolidate remaining repeated factory and native assignment/completion test contracts after
the runtime migrations. Decoder/lifecycle units already own their required test changes;
T014 reuses them rather than rebuilding those suites. Prefer existing `testx` and area-level
external-test contract harnesses. Retain native-version request assertions; never expose a
production test-hook API or weaken rejection/evidence tests to share fixtures.

Regroup surviving variable planning/execution code after JSON/index/decoder migrations settle.
Group process facade models/converters by variables, lifecycle, selection and business identity;
group ops models/converters by repair, cleanup, smoke and analysis. All remain in current
packages. Mechanical movement must preserve exported declarations, tags, methods and init
ordering; no generic helpers.go or incidental field cleanup. Large-file length is not the
acceptance gate; coherent ownership and unchanged contracts are.

### Compatible scaffolding and documentation tooling (T017–T018; F12)

Delete only unreferenced internal/private scaffolding (common environment binder, dormant
unregistered command and redundant private wrappers where proven). Keep `c8volt/variable`
public package name, API/New exports and other public legacy entry points; record their
compatibility disposition instead of removing them. Keep the maintained config binder owner
in config. No config precedence/default change is included.

Share docs build/publish steps through a local composite action, proposed
`.github/actions/publish-docs/action.yml`, preserving each workflow's triggers, version inputs,
permissions, setup versions, concurrency, SFTP configuration and publish path. Reuse docs-site.sh
for any obsolete docs-serve wrapper while preserving its invocation behavior. Validation parses
workflow YAML and runs extracted shell only with test-owned fake tools; it never builds/publishes
real docs, installs tools, contacts SFTP or launches a server. Generation/provenance scripts,
version-specific source pins and integration runners remain unchanged unless a concrete same-
contract duplication is proven; this plan does not relax their protection boundaries.

## Dependencies and execution policy

Default order is T001–T018, serial. The explicit graph is in tasks.md. Independent units are
not a request for agents or parallel execution. The order settles shared runtime and test
contracts before regrouping files; it is not a requirement to revalidate unrelated units.
The eight stories have acceptance coverage, not permission to start multiple stories in one
Ralph iteration. A future Ralph launcher must obey its normal per-work-unit/story discipline.
No launch is authorized now.

## Compatibility, validation and documentation gates

Follow [contracts/compatibility.md](contracts/compatibility.md), [data-model.md](data-model.md)
and [quickstart.md](quickstart.md). Targeted validation belongs in every task; no separate
checkbox exists for setup, context reading, routine formatting or review. Public consumer
probes must use explicit implementations of interfaces they protect: embedding the current
interface would silently satisfy newly added methods and would not detect a breaking change.

One final broad isolated race checkpoint belongs to T017 because JSON ownership, worker
execution and lifecycle behavior cross many packages. Clear inherited Go test flags and
inspect test-owned dependencies first. Run `make test` only after verifying its target remains
`go test ./... -race -count=1` without live additions. T018 tooling-only checks do not invalidate
prior runtime passes. No generated CLI refresh is planned; if implementation requires a changed
CLI contract, stop that change under C05 and obtain explicit scope acceptance first.

## Complexity and compatibility decisions

| Choice | Why needed | Simpler approach rejected |
| --- | --- | --- |
| Neutral worker subpackage plus old exported wrappers | Real transitive dependency problem and public signatures must both be addressed | Deleting ExecuteBulkOperation breaks public API; moving/redefining facade option types changes public type identity |
| Two raw decoder subpackages | Three adapter consumers per area, factory/adapter cycle avoidance and distinct page contracts | Importing parent from adapter cycles; one generic decoder hides endpoint requirements |
| Additive selector/run-preparation functions | Commands need service ownership without changing existing implementer method sets | Adding facade interface methods is source-breaking; leaving loops/parse strategy in cmd keeps finding unresolved |
| Explicit compatibility dispositions | Public caller absence cannot be proven outside repository | Removing apparently unused exports violates C01 |
| Conditional diagnostic correction gate | User requested CLI preservation; repository forbids raw normal-level HTTP diagnostics | Silently changing wording or calling a raw fallback compliant would misstate behavior |

## Artifact set and delivery status

`spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/compatibility.md`,
`quickstart.md`, `tasks.md`, and `consolidation-review.md` are planning artifacts only.
All implementation checkboxes remain unchecked. The saved scan prompt is a preserved,
separate untracked file. Git state and active implementation context are not changed.
