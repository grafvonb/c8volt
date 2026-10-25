# Validation Guide: Element Scope Variables

This guide is for the subsequent implementation. The feature and proposed test groups do not exist yet. Planning validation is document/link/diff review only; a planning command does not require Go tests.

## Preconditions and isolation

Use branch `329-element-scope-variables` with the [specification](spec.md), [plan](plan.md), [models](data-model.md), and [contracts](contracts/cli.md). Use the declared Go toolchain and existing test helpers/dependencies. Review the selected target and helper code before running it, including inherited `GOFLAGS`, test tags, profiles, and test-specific environment. Ensure none can enable live tests indirectly. Do not print credentials while reviewing environment settings.

All service dependencies must be generated-client mocks, test-owned HTTP servers, or in-memory fakes. CLI subprocesses use temporary configs pointing exclusively to those fakes. Reuse `testx` separate-output and real-terminal runners. Do not discover cluster contexts, probe connectivity, deploy BPMN, start real services, run integration/smoke targets, or create/clean up real resources. No disposable environment is required.

The exact names below are intended acceptance-test prefixes to add or map to existing tests during implementation. First list matching tests and verify the expected groups exist; a successful command with no matching tests is not evidence.

## Targeted commands

Run from the repository root after the corresponding implementation work exists:

```sh
go test ./internal/services/variable/... -list 'Test(ScopeVariable|ProcessInstanceVariable|UpdateScopeVariables)'
go test ./internal/services/variable/... -run 'Test(ScopeVariable|ProcessInstanceVariable|UpdateScopeVariables)' -count=1
go test ./internal/services/element/... -list 'Test(ElementScopeVariables|ElementVariableUpdates|ElementSearchTenant)'
go test ./internal/services/element/... -run 'Test(ElementScopeVariables|ElementVariableUpdates|ElementSearchTenant)' -count=1
go test ./c8volt/element -list 'Test(ElementScopeVariables|ElementVariableUpdates)'
go test ./c8volt/element -run 'Test(ElementScopeVariables|ElementVariableUpdates)' -count=1
go test ./cmd -list 'TestElementScopeVariables'
go test ./cmd -run 'TestElementScopeVariables' -count=1
```

Adapt the patterns to verified existing regression names instead of renaming unrelated tests. If an affected helper resides in another package, run its closest focused tests as part of the same work unit. Do not parallelize command tests that share package-global flags. Run `gofmt` on touched Go files.

## Acceptance fixtures and assertions

| Group | Isolated fixture and required evidence | Coverage |
|-------|----------------------------------------|----------|
| Scoped reader | Owner `P`, selected scope `E`, parent/sibling/descendant/called-process scopes and overlapping names. Assert separate owner/scope request filters, complete-value flag, raw-count advancement, all variable pages, stable order, truncation evidence, and strict claimed-scope identity/tenant validation in all three adapters. | FR-002–004; SC-001 |
| Pagination failure | Sparse/empty intermediate pages with continuation, cursor cycles, inconsistent/negative totals, canceled and failed later pages. Assert error rather than empty/successful partial scope; retain the existing root pagination matrix unchanged. | FR-003, FR-028; SC-001, SC-008 |
| Discovery/enrichment | Exact-key and filtered/paged selection, limits, user stop, empty results and empty local variables. Assert no extra base GET/search, enriched visitor and aggregate items, independent variable page limits, listeners+variables, omission versus `[]`, keys/total zero variable calls. Verify configured tenant/IgnoreTenant filters and totals in v88/v89/v810. | FR-001, FR-005–006, FR-017; SC-001, SC-008 |
| Inputs and planning | Repeated/comma/stdin keys; malformed/absent/conflicting/non-object payloads; `{}` and null properties; late missing/denied element; tenant conflict; same-name parent values; truncated equality; changed and unchanged locals. Assert stable dedup, local additions, complete classifications and zero writes/prompts on any planning failure. | FR-007–010, FR-017–019; SC-002–003, SC-008 |
| Frozen execution | Tamper with destination/count/classification/maps and mutate caller-owned nested values. Assert validation/deep-copy isolation, exactly one logical local write per changed key, `local=true`, unchanged/untouched exclusion, no replan/redirect/called-root mutation. | FR-011–012, FR-028; SC-002–003 |
| Confirmation | Delayed complete local visibility, parent-only match, truncated matching text, wrong owner, read failure, cancellation, no-wait, and acceptance followed by confirmation never starting after cancellation/fail-fast. Assert all requested names verified; accepted evidence retained on failed/unstarted confirmation with original failure and distinct confirmation status; no-wait zero post-submit reads; no resubmission. | FR-013–014; SC-004–005 |
| Workers and submission | Sequential/bounded workers, fail-fast unscheduled and in-flight items, cancellation during retry backoff, first-attempt pacing rejection, ambiguous dispatch then unsent retry, rejected dispatch then unsent retry, client-timeout error substitution, documented 400 rejection versus 500/503/504 or malformed/response-plus-error uncertainty. Exercise artificial evidence sequences directly on the accumulator when the existing retry policy would not retry that rejection; do not change retry policy to manufacture a scenario. Assert request bounds, ordered complete outcomes, correct not_sent/rejected/unknown/accepted evidence and original error class. | FR-015–016; SC-005 |
| Modes and no-op | Normal and dry-run; human/JSON/keys/quiet; quiet+machine; supported auto-confirm/automation/no-wait. Include empty search, empty locals plus additions, all-unchanged and `{}`. Assert exact human wording, no accepted/pending claim for no-op, one JSON envelope plus EOF, accepted-only executed keys, zero-byte empty keys, and discovery/write/confirmation counts. Retain nonempty/error/abort regressions. | FR-018–022; SC-003, SC-006 |
| Real terminal | Actual terminal stdin with stdout redirected/captured separately; configured and inherited stderr. Exercise approval/decline/EOF, paging stop, keys-only paging, no-op/dry-run prompt absence, auto-confirm and automation. Assert exact prompt routing/text/defaults and no mutation on abort. | FR-023; SC-003, SC-006 |
| Diagnostic failures | Multiple backend errors carrying distinctive URL/method/body/resource-key markers. Execute command with configured plain and structured loggers, separate streams and inherited destinations. Assert one grouped ERROR, severity/destination/count, compact bounded safe reasons, no raw HTTP markers normal/verbose, quiet retains required ERROR; safe verbose resource facts may contain intended keys. | FR-024–025; SC-007 |
| DEBUG and JSON errors | Repeat diagnostic fixtures using flag DEBUG and configured DEBUG independently of verbose, with current quiet/mode precedence. Assert redaction, diagnostic stderr only, no duplicate records or extra requests; JSON exactly one envelope+EOF with machine evidence, partial accepted results, and original nonzero exit. | FR-021, FR-026; SC-005–007 |
| Compatibility/docs | Existing PI root updates, UT effective updates, task completion and listener selection; configured 8.7 bootstrap exit 1 with zero requests; 8.8/8.9/8.10 request contracts; help/alias/metadata/examples and generated docs. | FR-027–029; SC-008 |

Failure assertions must detect the actual violation. Show that changing configured logger routing to direct stderr, logging the raw aggregate at normal level, or suppressing errors under quiet would fail the diagnostic tests. Renderer-only or `stderr contains ...` assertions are insufficient. Do not commit deliberately broken variants; inspect the assertion sensitivity or use a temporary isolated mutation check when needed.

Real-terminal coverage must use the test helper that creates a real terminal, not a pipe or mocked terminal check. When a platform cannot execute that helper, report the unverified interaction coverage rather than replacing it with weaker evidence.

## Integrated regression and documentation

After targeted checks, run the repository full race suite once for the shared traversal/interface/writer-evidence and concurrency changes. Inspect its definition and inherited settings first to confirm only isolated tests execute:

```sh
make test
make docs-content
git diff --check
```

Review generated CLI docs against metadata and README; do not hand-edit generated output. Repeat tests only after relevant changes or unresolved failures, not merely to commit. If a proposed change ends up avoiding shared runtime/concurrency impact, document why narrower checks suffice under the constitution.

Report exact executed checks and outcomes, including unavailable/skipped terminal coverage. Isolated tests establish request/response, scope, workflow, and output contracts against fakes. They do not establish real Camunda consistency timing, deployment configuration, or live authorization behavior; do not label those verified.
