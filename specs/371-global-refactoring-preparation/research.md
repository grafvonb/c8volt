# Research and compatibility decisions

Local source inspection only, at `33eb3dce`; no external research, runtime test or live target.
The earlier audit's twelve findings are captured in spec.md. This document resolves the
implementation choices needed to make tasks executable; new runtime discoveries still follow
C05 and the repository's normal stop/report policy.

| Finding | Inspected evidence | Decision and rejected alternative |
| --- | --- | --- |
| F01 | element/update.go normalizeElementRequestedValue/snapshotTypedElementJSONValue; processinstance/update.go snapshotTypedProcessInstanceJSONValue; usertask/update.go copyVariableValue; corresponding facade convert.go helpers; toolx/json_copy.go | Share mechanical JSON operations, migrate all consumers together and preserve representation/error wrappers. Blind CopyJSONMap replacement changes concrete/number behavior; typed aliasing must be tested independently of ordinary map/slice cases. |
| F02 | toolx/pool/bulk.go imports c8volt/foptions; pool/pool.go owns worker engine; imports in 30 production files across 11 internal directories; common/bulk.go has no production callers | Keep original exports as compatibility entry points; extract neutral core and migrate internal imports. Removing a public helper or moving the canonical FacadeOption type is forbidden. Unused internal RunBulk may be removed after complete reference audit. |
| F03 | cmd/cmd_views_contract.go handleCommandErrorWithMode raw fallback; embed_deploy.go, get_processdefinition.go, update_usertask.go and config commands direct ferrors exits; ferrors/errors.go HandleAndExit | Consolidate routing around command context and exactly-once result handling, retaining bootstrap and output-write paths. A helper substitution alone is insufficient. Conflicting raw legacy wording needs a separately accepted correction, not disguised refactoring. |
| F04 | usertask/update.go resultsFromScopeVariableUpdateOutcomes/associatedNamesForTask; update_validation.go findScopeVariableTarget/targetAssociatesTask and nested coverage loop | Build invocation-local indexes from validated associations; preserve ordered slices and request counts. Do not cache service reads or replace order with map iteration. Bench local fan-out only. |
| F05 | processinstance suspend/resume bulk and plan files; v810 suspension_submission.go/resumption_submission.go; waiter suspension.go/resumption.go | Separate shared discovery, collection dispatch, submission evidence and confirmation loop into practical compatible outcomes. Success/conflict policies, endpoint calls and public result types stay separate. A single configurable state-machine engine would obscure uncertainty and terminal-state differences. |
| F06 | cmd_views_{job_logging,usertask_assignment,usertask_complete,element_update,processinstance_suspend,processinstance_resume}.go | Share only count/order/bound mechanics. Keep six reason classifiers and exact current formatting; do not parse raw backend strings. |
| F07 | usertask/v88,v89,v810 variables.go decodeEffectiveVariablePage; variable/v*/convert.go decodeSearchVariablesResponse; variable/search/search.go | Two area-specific raw decoders avoid generated-type and import-cycle coupling; traversal stays in existing owner. Root/local strictness and effective required metadata are different contracts. |
| F08 | process_definition_selector_validation.go validateProcessDefinitionSelectors/searchProcessDefinitionsForSelectorValidation; processdefinition/search.go shared pager | Move strategy into service and adapt existing page-reader methods through an additive free function. Expanding process.API would break downstream implementers and test fakes. Preserve no-prompt/TTY near-match policy in cmd. |
| F09 | deploy_processdefinition.go, embed_deploy.go and cmd_deploy_run_data.go buildRunProcessInstanceDatasFromDeployments/extractBPMNProcessIDs | Move ID selection and BPMN fallback into resource service and expose additive preparation. Existing ordered creation service is reused. Preserve deployment rendering before creation and partial deployment evidence; do not invent combined response contracts. |
| F10 | repeated service factory tests, usertask native assignment/completion tests and existing job shared coordinator tests; testx HTTP/subprocess/terminal helpers | Reuse fixture/contract harnesses after runtime changes; keep native wire assertions. Decoder/lifecycle tests are updated with implementation, not deferred to a blanket test task. |
| F11 | usertask/update.go (639 lines), processinstance/update.go (548), process model.go (1,108)/convert.go (1,242), ops model.go (1,147)/convert.go (2,048) | Group by real concern after upstream migrations. Same package, declarations and init behavior. Mechanical moves do not justify new abstractions or broad rename churn. |
| F12 | common/envs.go and config/config.go binders; empty c8volt/variable API/New; dormant cmd/get_variable.go; docs-release.yaml/release.yaml publishing blocks; docs runners | Remove only private/internal dead paths, retain public shims. Share docs publishing locally and prove behavior with fake tools. Do not execute publication or alter integration runners/source provenance protections. |

## Exact diagnostic compatibility decision

The audit established a static raw-error exposure path, not the complete supported transcript
of every affected command. Before changing it, an implementation unit must characterize real
command execution using isolated fakes and the configured logger. Existing bounded text,
classes, prompts and machine evidence are fixed acceptance expectations. Where those expectations
conflict with raw legacy output, document the exact proposed exception and await explicit
acceptance for that behavior correction; neither historical disclosure nor speculative new
wording is a compliant baseline. The rest of the dependency graph can proceed independently.

## API compatibility decision

Keep public exports even if unused locally, including toolx/pool's facade-option signature,
c8volt/variable's current package/export shape, public ferrors helpers, constructors and
exported converter signatures. New free functions are additive; no existing interface acquires
methods. Internal-only contracts may change only with all adapters/doubles/callers in the same
compiling task. Validate consumer method sets explicitly, not via embedded interfaces.

## Existing behavior versus ownership defects

Shared snapshot helpers must preserve accepted serialized values and stable concrete values.
The advertised frozen-plan isolation of mutable JSON is a required contract, even where a
local copy function currently aliases typed storage. A new ownership assertion should expose
that discrepancy without using a real engine. Any proposed rejection of previously accepted
input, numeric canonicalization or new public error text is outside automatic refactor scope.
