# Consolidation review: global compatible refactoring

**Status:** COMPLETED. **Date:** 2026-11-03. **Reviewed:** 20 draft tasks -> 18 final tasks.
All checkboxes remain unchecked. Review changed only tasks.md and this review; it did not run
application tests, implement changes or alter the completed 370 work or saved scan prompt.

## Code-grounded merges applied

| Original IDs | Final ID | Cohesive outcome and eliminated boundary |
| --- | --- | --- |
| T001 + T002 | T001 | Shared JSON helpers plus all three service and facade consumer families. element/update.go, processinstance/update.go and usertask/update.go repeat the same marshal/UseNumber/EOF comparison mechanics; existing converter helpers repeat snapshots. A helper/element-only task would leave policy drift and require another ownership/API test handoff. The common fixture matrix is built once and every concrete consumer moves together. |
| T003 + T004 | T002 | Existing pool engine extraction, original public wrappers, all internal consumers, unused internal runner removal and transitive-import acceptance. pool/bulk.go's FacadeOption signature is already public; migration of only one consumer would leave the unwanted facade dependency in other internal paths. The code is a moved existing engine, not new scheduling implementation, so 30 import sites/11 directories are a practical atomic adaptation rather than separate tasks. |

Every original obligation survives. New helper APIs, implementations, consumers, required
doubles, tests and validation share their resulting checkbox. No test/doc/format task was split
out to make implementation look smaller. IDs and dependencies were rewritten and the final
finding/compatibility mapping checked again.

## Original-to-final mapping

| Original | Final |
| --- | --- |
| T001, T002 | T001 |
| T003, T004 | T002 |
| T005 | T003 |
| T006 | T004 |
| T007 | T005 |
| T008 | T006 |
| T009 | T007 |
| T010 | T008 |
| T011 | T009 |
| T012 | T010 |
| T013 | T011 |
| T014 | T012 |
| T015 | T013 |
| T016 | T014 |
| T017 | T015 |
| T018 | T016 |
| T019 | T017 |
| T020 | T018 |

## Retained boundaries after adjacent and nonadjacent review

Boundaries are justified by the combined implementation/acceptance workload, not directory
names, hypothetical parallelism or the fact that individual tests can run separately.

| Boundary considered | Combined outcome and concrete reason retained |
| --- | --- |
| T001 + T002 | A complete JSON ownership/representation migration across six service/facade families plus a worker-core relocation and 30 import adaptations would combine two unrelated invariants: exact serialized/concrete values versus goroutine scheduling/order/admission. Required custom-marshaler/cycle/ownership fixtures and worker race/stagger/context/import-closure tests have no shared implementation or fixture setup. One practical unit must not diagnose both broad primitive changes at once. |
| T003 + T004 (strongest rejected merge) | Drafted combined outcome: replace six bounded reason reducers, migrate all direct command exits, and prove every affected formatter, bootstrap, output-write and staged-result path. Source inventory found 40 direct ferrors.HandleAndExit references in 11 cmd files. The reducer alone is an exact-transcript mechanical extraction; dispatcher work adds result-emission state, JSON EOF/partial exits, inherited/configured logger variants and real-terminal success/abort/empty checks, plus the C05 legacy-wording gate. Even reusing the existing formatter and subprocess fixtures, this combines six classifier regressions with eleven differently bootstrapped command paths and an unresolved-behavior decision surface. Keep the stable reducer prerequisite and the command-context migration as two reviewable outcomes. |
| T005 + T001/T014 | Indexing introduces an association-coverage algorithm and must compare reordered/sparse/shared-scope outputs and measured work against a reference. JSON copying is reused, not rebuilt, and residual cross-version test extraction does not implement association indexes. Bundling would mix local complexity evidence with API ownership or protocol fixture migration without eliminating implementation work. |
| T006 + T007 | Shared descendant planning owns capability/timing preflight, overlap deduplication and traversal completeness; shared bulk execution owns all-keys preview validation, scheduling and skipped/partial slot merge. Combined validation includes overlapping tree membership and fail-fast/in-flight worker matrices through CLI terminal paths. Existing walker/pool reuse lowers implementation effort but not those independent failure surfaces; preserve two bounded lifecycle outcomes. |
| T007 + T008 | Moving the existing worker dispatch is not implementing native evidence retention across retries. Native submission must separately cover request-rate admission before dispatch, empty204 versus malformed success, explicit/ambiguous response and later retry outcomes. The combined worker/native retry matrix would couple partial-slot diagnosis to wire ambiguity; keep the neutral collection algorithm separate. |
| T008 + T009 | Drafted combined outcome: common native submission/evidence classifier and common observation loop with two action policies. Existing retry/httpc helpers are reused. Acceptance still requires all call-error/response/admission/retry permutations plus a separate timed observation matrix (after-deadline reads, attempts, 404/auth, identity and terminal conflicts), across four source paths with different fact lifetimes. The distinction is submission proof versus observation, a key public evidence invariant; split to keep each uncertainty/timeout matrix diagnosable in an iteration. |
| T010 + T011 | Drafted combined outcome: extract both decoder families and adapt six native wrappers. Effective-variable tests require mandatory totals and offset evidence; root/local tests require optional-total presence, strict-scope versus legacy-root behavior, cursor/overflow and scope-tenant contracts. Existing shared traversal is reused, not rewritten. A single unit would migrate three distinct raw-read modes across six adapters while proving absence of accidental strictness changes; two area-specific corpora provide smaller practical protocol migrations without any repeated new decoder algorithm. |
| T012 + T013 | Selector strategy performs backend existence/fallback queries and preserves TTY recovery; deployment preparation is request-free XML/metadata selection embedded in a two-stage mutation command. Combining requires both query-transcript/terminal recovery checks and deployment-success/creation-failure stage evidence, with no shared algorithm or fixture lifecycle. Their common facade-adapter pattern is reused without a universal command-preparation abstraction. |
| T014 + runtime tasks | Each runtime unit already owns its necessary tests and doubles. T014 only consolidates remaining repeated factory/native test setup after protocol shapes settle; it must not absorb unbuilt behavior or become a delayed acceptance gate. Merging it into one earlier runtime task would force that task to migrate unrelated supported-version/capability fixtures before all downstream consumers exist. |
| T015 + T016 / upstream tasks | Service variable cohesion remains after JSON/index/decoder edits; public cohesion moves about five thousand existing model/converter lines across process and ops, protecting export/type/tag inventories. Combining with service regrouping or earlier behavior changes would require simultaneous mechanical API declaration review and algorithm/protocol diagnosis. Schedule each settled ownership move once, after its touching tasks, and keep these mechanical comparisons distinct from behavior edits. |
| T017 + T002/T016 | T002 already removes unused internal bulk code as part of dependency closure. Remaining private scaffolding cannot be deleted safely until last consumers/test moves settle. T017 combines those reference checks/deletions with the final justified runtime race/API/import checkpoint; it is a real compatible cleanup, not a standalone review/test checkbox. |
| T018 + runtime tasks | Shared docs CI and runner behavior is executable tooling with event/permission/secret/publish-path contracts. Its acceptance uses YAML parsing and fake external commands, never application mutation or live CI/SFTP. It has no shared runtime algorithm and must not be coupled to credentials/publication or used to trigger another full Go runtime rerun. |

The final dependency graph was checked after these decisions; no newly exposed adjacent or
nonadjacent merge eliminates another real handoff without combining the concrete workloads
above. There is no target count; 18 is the result of reviewing the actual obligations.

## Coverage, progress and validation

All F01–F12 and C01–C10 retain explicit owners. Every unit includes code and its fixtures,
comments/formatting/isolated checks. No [P] or agent launches were introduced. Existing public
exports and interface method sets remain protected, including apparently unused shims. C05
retains the exact legacy-diagnostic reproduction/correction gate, not an invented automatic
behavior exception. Active plan pointers, branch and pre-existing prompt are unchanged.

Planning checks: task checkbox format and contiguous IDs, all eight story labels, complete
finding ownership, valid/acyclic prerequisites ordered before consumers, local document links,
no checked implementation item, no unresolved template placeholder, whitespace and write-scope
review. The files were read back after writing to verify 18 tasks and the saved review. No
application tests, benchmarks, generators, live checks, commits or implementation were run.
