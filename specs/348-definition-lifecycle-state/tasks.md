# Tasks: Process-Definition Lifecycle State

**Feature**: `348-definition-lifecycle-state` · [spec.md](spec.md) · [plan.md](plan.md)
**Design inputs**: [research.md](research.md), [data-model.md](data-model.md), [CLI contract](contracts/cli.md), [library contract](contracts/library.md), [quickstart.md](quickstart.md).

Tasks are cohesive work units; implementation, fixtures, meaningful tests, formatting, documentation, and targeted validation belong to the same checkbox. Paths below are repository-relative. Story labels name the acceptance owner, not a separate implementation layer. Tests are required by this feature's acceptance criteria.

## Execution rules

Read `AGENTS.md`, these feature artifacts, and any existing `progress.md` before work; Ralph also reads `specs/ralph-implementation-rules.md`. Reading, branch checks, review, progress recording, and commits are responsibilities within each unit, not extra tasks. New/touched functions and tests need the intent comments required by the repository rules.

Use only generated-client mocks, test-owned HTTP fakes, and isolated CLI/subprocess/real-terminal tests. Audit selected helpers first and use the clean environment commands in `quickstart.md`; inherited flags, profiles, credentials, or configuration must not select real services. Never discover, provision, contact, mutate, or clean up a live Camunda environment during implementation. Do not execute integration targets or their binaries. Integration-source validation is compile-only. Do not add explicit `--no-wait` tests or a new live fixture matrix.

Write or extend the relevant regression assertions before the corresponding behavior change, then implement and make them pass within the same unit. Run `gofmt` on touched Go files and `git diff --check`. Do not repeat checks for later prose-only edits or commits. Broaden to `make test` only for actual shared runtime/concurrency impact or unresolved focused failures, documenting why. Never claim live engine transitions were verified. Issue-associated commits end their subject with ` #348`.

## Phase 1: Setup

No project initialization, dependencies, or infrastructure changes are needed. Reuse the existing Go/Cobra project and fixtures; there is no setup checkbox.

## Phase 2: Foundational prerequisites

No independent foundational work is needed. The model and converter work belongs to the first read outcome, rather than a separate infrastructure phase.

## Phase 3: US1 — See the current definition lifecycle (P1, MVP)

**Goal**: Ordinary single/list reads and public-library results preserve the backend's lifecycle observation, without additional requests or compatibility changes.

**Independent test**: Controlled 8.10 lookup/list responses with ACTIVE, DRAINING, DELETED, unfamiliar, omitted, and null state survive the public path and selected output; 8.8/8.9 remain unchanged. Empty, keys-only, quiet, XML, and not-found behavior stay intact.

- [ ] T001 [US1] Deliver lifecycle through ordinary single/list reads: add `State` and shared mappings in `internal/domain/processdefinition.go`, `internal/services/processdefinition/v810/convert.go`, `c8volt/process/model.go`, and `c8volt/process/convert.go`; extend `cmd/cmd_views_processdefinition.go`, the existing tests named below, `integration/cli/real_state_destructive_test.go`, source help in `cmd/get_processdefinition.go`, `README.md`, and generated docs; validate service/facade/command paths and compile the integration package as specified in `quickstart.md`.

**T001 data-path acceptance and implementation**:

- Add `State string` with `json:"state,omitempty"` to existing domain/public definitions; copy `string(r.State)` in the v810 converter and the domain string in the facade converter. Reuse existing generated client support. No new enum, pointer/wrapper, custom marshaler, interface, option, filter, request, or generated-code edit.
- Preserve these model constraints verbatim: "Preserve every nonempty supplied value, including an unfamiliar value; do not trim, normalize case, or validate against a closed set." "Omitted or null generated values produce the empty string and are omitted from public JSON." "v88/v89 conversion leaves state empty." "Do not merge an omitted state with an earlier value." "The lifecycle field is neither a selector nor a sorting key."
- Assert JSON property absence, not just a zero Go field. Put search fixture state in serialized response `Body` as well as `JSON200`: `decodeSearchProcessDefinitionsResponse` re-decodes the body. Cover lookup, search, pages/visitor and final traversal, latest selection, and watch snapshots through existing shared conversion; preserve options, canonical order, pagination and request expectations. Returned DELETED is still an item.
- Extend existing tests in `internal/services/processdefinition/v810/service_test.go`, `internal/services/processdefinition/v88/service_test.go`, `internal/services/processdefinition/v89/service_test.go`, `internal/services/processdefinition/search_test.go`, `c8volt/process/convert_test.go`, `c8volt/process/client_test.go`, and `c8volt/process/model_test.go`; run targeted service/facade checks from `quickstart.md`.

**T001 output acceptance and implementation**:

- Extend `cmd/cmd_views_processdefinition_test.go` and `cmd/get_processdefinition_test.go` using existing command fixtures. Run targeted command checks, docs generation, and compile-only integration validation from `quickstart.md`; align relevant `integration/README.md` text with the changed observer.

- Add lifecycle to `processDefinitionDisplay` and its shared projection once for lookup/list, stat/non-stat, and watch rows. Always reserve semantic state cell 4, even when empty, and move the existing optional statistics cell to 5. Adapt `formatProcessDefinitionWatchColumn` indices in the same change so existing watch/stat rendering remains correct. Reuse the flat renderer's all-empty-column omission and single-row compaction; do not conditionally append the state cell. State change classification/highlighting is completed in T003.
- Extend execution-path coverage for known/unfamiliar/unavailable state, mixed/all-omitted rows, human/JSON lookup and listing, keys-only, quiet and quiet+JSON/keys, empty listings, explicit not-found, and unchanged XML. Decode exactly one shared JSON envelope and require EOF; preserve existing payload/null/omission conventions and zero-byte empty keys output. Preserve mode precedence, plain prompts, selected-result stdout, and configured diagnostics. Assert request counts and absence of lifecycle-specific discovery, prompts, and mutations. Reuse existing regression coverage rather than multiply every state by every mode.
- In `TestRealStateProcessDefinitionDraining`, replace the intended lifecycle observation from generated API access with bounded `get pd --key <owned-key> --json` execution using the existing harness. Require a successful envelope, EOF, exact owned key, and supplied DRAINING state. Preserve existing setup/deletion arguments and the independent `requireProcessDefinitionGoneFromAPI` cleanup evidence. Do not add suspended live instances to the existing ACTIVE-only cleanup path. Compile via `go test -tags=integration -c ./integration/cli` with a temporary output binary; do not run it or TestMain.
- Document 8.10 availability, older-version/omitted-state behavior, unfamiliar preservation, ACTIVE/DRAINING/DELETED meanings, and runtime-versus-historical removal. Clearly distinguish backend deletion (does not cancel/resume unfinished instances) from existing `delete pd --force` cancellation. No new command, flag, wait, mutation, filtering, synthetic DELETING, or deletion-completion policy. Regenerate docs with `make docs-content` after source/README changes; never hand-edit generated output.

**Checkpoint**: T001 delivers the ordinary-read MVP and the existing draining scenario's CLI observation source, without running a live scenario.

## Phase 4: US2 — Understand lifecycle alongside instance statistics (P1)

**Goal**: Show lifecycle independently of counts, especially DRAINING with zero active and two suspended instances.

**Independent test**: Fake-backed lookup and listing with `--stat` in human and JSON modes preserve lifecycle, all counts and availability indicators, mixed-row alignment, and the exact existing request behavior.

- [ ] T002 [US2] Complete lifecycle/statistics acceptance using `cmd/get_processdefinition_suspended_test.go`, `cmd/cmd_views_processdefinition_test.go`, `internal/services/processdefinition/v810/service_test.go`, and `c8volt/process/convert_test.go`; reuse the T001 renderer and extend `cmd/get_processdefinition.go`/`README.md` examples and regenerated `docs/cli/`/`docs/index.md` for draining with suspended work; run the affected targeted checks in `quickstart.md`.

**T002 acceptance and implementation**:

- Extend `definitionStatisticsFixture` to supply lifecycle independently of its count population and retain its unexpected-request rejection. Check lookup/list human and `--json --stat`, with and without state, unfamiliar state, mixed rows, missing/unsupported statistics, and established statistics-error behavior. Reuse T001 mode checks instead of duplicating the whole matrix.
- Preserve model constraints: "Statistics enrichment may update only statistics; it must neither replace nor infer lifecycle state." "Separate reads are not asserted to form an atomic snapshot." Keep State=DRAINING, Active=0, Suspended=2 independent; zero counts, omitted stats, and a later count change representing resumed work cannot synthesize ACTIVE/DELETED or completion. Missing state remains omitted even when counts are available.
- Preserve active/completed/canceled/incident/suspended values, support booleans, labels, zero-as-minus formatting, and state/stat alignment. Assert existing per-scenario statistics/paging request counts and options, with no lifecycle fetch or mutation. Extend adapter/facade assertions only where command evidence does not cover the boundary; no new statistics mechanism is expected.
- Explain in help/README that suspended unfinished work can retain DRAINING, and resuming an instance does not itself finish draining or reverse deletion. Regenerate docs for changed source. Suspension, resumption, forced cancellation/deletion, root resolution, and completion checks are not changed by this unit.

**Checkpoint**: The separate lifecycle/count contract is proven with isolated fixtures; no real-engine suspension/draining claim is made.

## Phase 5: US3 — Follow lifecycle changes during watch (P2)

**Goal**: Integrate lifecycle with existing watch identity and highlighting, preserving baseline correctness and configured logger behavior.

**Independent test**: Deterministic successful/failed refreshes and real-PTY output against test-owned fakes prove independent value highlights, update counts, omission clearing, unchanged-marker clearing, DELETED versus absence, configured diagnostic routing, and baseline retention on failed writes.

- [ ] T003 [US3] Complete lifecycle comparison/highlighting and configured watch diagnostics in `cmd/cmd_views_processdefinition.go` and `cmd/get_processdefinition_watch.go`, with regressions in `cmd/cmd_views_processdefinition_test.go`, `cmd/get_processdefinition_watch_test.go`, `cmd/get_processdefinition_watch_timeout_test.go`, and `cmd/get_processdefinition_watch_terminal_test.go`; update watch help in `cmd/get_processdefinition.go` and `README.md`, regenerate `docs/cli/`/`docs/index.md`, and run the targeted command checks in `quickstart.md`.

**T003 acceptance and implementation**:

- Add state to `processDefinitionChangedFields`, `any()`, and `changedFields`; highlight only the changed nonempty state cell using existing capability checks. Reuse the current candidate/last-successful-baseline runner. First success is unmarked; state-only/stat-only/both change independently, with one update per row. Supplied→omitted→supplied clears stale values without empty ANSI spans; unchanged success clears markers. Failed fetch or rendering retains the previous successful baseline. Returned DELETED stays visible; actual absence uses existing removal. Accept skipped transitions and unfamiliar values without a state machine or synthetic DELETING.
- Replace slow-refresh direct stderr output with `renderHumanWarningLine` and verbose timing with `renderHumanLine`, retaining the warning-streak and verbose gates. Replace the private summary handler with a small error-returning helper in the existing view file: retrieve the configured logger via `logging.FromContext`, return a missing-context error instead of using its fallback, check INFO enablement, and use the configured handler to propagate enabled summary write failures. Filtered INFO returns success so rows/baseline proceed. Keep repaint→summary→rows ordering, attributes/destination/activity writer, and no manually embedded timestamp/severity or separate/default logger.
- Extend command execution tests with configured/inherited destinations, separate stdout and diagnostics, plain and structured handlers, INFO summary/timing and WARN slow warnings, WARN-level INFO suppression, no duplicate emission, and summary/row write failures preserving baseline. Configure plain-time explicitly for timestamp assertions. Keep watch quiet rejection and ordinary quiet behavior distinct.
- Trace failures through service, facade conversion, renderer, logger and exit. Reuse/extend fake-backed error coverage with repeated multi-resource causes containing distinctive URL, method, body and key markers: normal and verbose retain bounded aggregate diagnostics on configured destinations and expose no raw HTTP on either stream; ordinary quiet retains compact errors; JSON decodes once with EOF, partial results and original failure exit/class. Verify redacted DEBUG diagnostics both by flag and configured level independently of verbose, in plain/structured modes. Do not add unrelated mutation scenarios. Assertions must detect direct-stderr regressions, unconditional private handlers, raw error disclosure, duplicates, or discarded partial results.
- Extend real-terminal tests with test-owned services for state-only/both/omitted-value highlighting and aligned rows; retain nonterminal fallback and report platform skips. Reuse existing timing, timeout/retry, stop, slow-streak and request-count tests. Confirm watch still rejects JSON/keys/XML/quiet/automation before requests. No new polling or scheduling behavior.
- Document lifecycle-only updates, omitted-state clearing, returned DELETED versus disappearance, and configured watch summary/warning/timing levels in source help/README; regenerate docs. Add no new commands or flags.

**Checkpoint**: All three stories are independently verified after their prerequisites with permitted isolated checks; engine timing remains unverified.

## Phase 6: Polish and cross-cutting completion

No standalone cleanup or validation task. Each unit owns required docs, formatting, tests and recorded evidence. Complete the feature only when all checkboxes pass, every requirement below has evidence, and no unresolved output-routing or baseline failure remains. Read-only planning checks do not require application tests.

## Dependencies and execution order

`T001 → T002 → T003`

US1 owns shared read/projection behavior. US2 depends on that projection and independently validates enrichment. US3 follows US2 because it uses its independent state/count fixtures and changes the same shared display/test surfaces. Do not begin the next story within a Ralph iteration.

## Parallel examples

- **US1**: After wiring is complete, service and facade focused validation may run independently; do not split mapping/CLI implementation among concurrent writers to shared fixtures.
- **US2**: Adapter preservation checks and command fixture assertions can be reviewed independently, but the work unit remains serial with US1/US3.
- **US3**: After edits settle, pure comparison checks and PTY checks may be selected separately; avoid parallel test functions that mutate command globals or environment.

No tasks carry `[P]`: overlapping display, command fixtures and source documentation make parallel implementation unjustified. These examples are validation opportunities, not instructions to launch agents or extra tasks.

## Requirement coverage

| Requirements / success criteria | Owning work |
| --- | --- |
| FR-001–003, FR-005; SC-001–002 | T001 mapping/serialization; T001 command modes |
| FR-004, FR-006–007; SC-002, SC-005 | T001 shared display/mode/request checks; T002 statistics requests; T003 watch requests |
| FR-008–009; SC-003 | T002 independent counts and draining examples, using T001 shared projection |
| FR-010–014; SC-004 | T003 watch transitions, failures, baseline, styling, compatibility |
| FR-015 | T001 ordinary mode separation; T003 configured logging and failure execution |
| FR-016–017; SC-006 | T001 lifecycle meanings/compatibility/deletion distinction; T002 suspended-draining explanation; T003 watch help |
| FR-018 | T001 existing draining observer and compile-only check |

## Implementation strategy

Deliver US1 first as the ordinary-read MVP; its task includes library and CLI evidence. Add US2 acceptance and operator explanation without rebuilding rendering or statistics. Finish US3 with comparison and logger/baseline behavior together. Reuse the existing harnesses and helpers at each step. Checkpoints require isolated evidence only, never deployment, a live demonstration, or a real target designation.
