# Isolated implementation validation guide

This is a future validation guide. Planning has run no application tests and authorized no
implementation. Read spec.md, plan.md, tasks.md, the compatibility contract, repository AGENTS
and Ralph rules before executing a unit after implementation is separately authorized.

## Scope and environment

Use only fake services, test-owned HTTP/subprocess/terminal fixtures and pure local data.
Inspect each selected test/script first. Clear inherited GOFLAGS/build tags; do not select
integration tags, live-test helpers, credentials, profiles or cluster endpoints. Do not inspect
Kubernetes, start Camunda or ask for a disposable target. Ordinary facade API calls may occur
only against test-owned fakes. Cache/sandbox/loopback issues use normal tool permissions, not a
live substitute. Proposed TestRefactor* names must be introduced by their owning tasks before
these commands are meaningful; zero matched tests is not a pass.

## Targeted commands and evidence

Run the task's first relevant command before broader coverage. These commands include future
package names/tests and are not claimed runnable or passed today.

| Unit(s) | First relevant isolated command | Required evidence |
| --- | --- | --- |
| T001 | `GOFLAGS= go test ./toolx ./c8volt/element ./c8volt/process ./c8volt/task ./internal/services/element ./internal/services/processinstance ./internal/services/usertask -run 'TestRefactorJSON' -count=1` | Exact values, representation, invalid input and mutation-after-plan isolation; then affected existing variable suites |
| T002 | `GOFLAGS= go test ./toolx/pool/... -run 'TestRefactorPool|TestExecute|TestStagger' -race -count=1` | Original public API probe, core limits/order/fail-fast/cancellation/DEBUG; then neutral consumers and transitive import guard |
| T003 | `GOFLAGS= go test ./cmd -run 'TestRefactorFailureSummary' -count=1` | Byte-identical six-formatter transcript table and command mode routing |
| T004 | `GOFLAGS= go test ./cmd -run 'TestRefactorCommandFailure' -count=1` | Separate streams/configured records; complete diagnostics/JSON/terminal matrix; C05 disposition for legacy conflicts |
| T005 | `GOFLAGS= go test ./internal/services/usertask -run 'TestRefactorScopeIndex' -race -count=1` | Exact ordered equivalence, associations, counts and rejection semantics |
| T005 benchmark | `GOFLAGS= go test ./internal/services/usertask -run '^$' -bench '^BenchmarkRefactorScopeIndex$' -benchmem` | Pure local baseline/indexed measurements across sparse/shared shapes, no timing pass threshold |
| T006 | `GOFLAGS= go test ./internal/services/processinstance ./c8volt/process -run 'TestRefactorTreePlan|Test.*(SuspensionPlan|ResumptionPlan|PlanProcessInstance)' -count=1` | Capability, frozen scope, state classification, ordering and traversal transcript |
| T007 | `GOFLAGS= go test ./internal/services/processinstance ./c8volt/process -run 'TestRefactorLifecycleBulk|Test.*(Suspend|Resume|Suspension|Resumption)' -race -count=1` | Ordered previews/worker outcomes and partial error/no-wait/empty behavior |
| T008 | `GOFLAGS= go test ./internal/services/processinstance/v810 -run 'TestRefactorLifecycleSubmission|TestProcessInstance.*Submission' -race -count=1` | Native endpoint bodies, admission/retry/ambiguous evidence, original errors and independent DEBUG |
| T009 | `GOFLAGS= go test ./internal/services/processinstance/waiter -run 'TestRefactorLifecycleWait|Test.*(Suspension|Resumption)' -count=1` | Read-after-deadline, state-policy differences, identity, attempts and cancellation |
| T010 | `GOFLAGS= go test ./internal/services/usertask/... -run 'TestRefactorEffectiveVariablePage|TestService_SearchUserTaskEffectiveVariablesPage' -count=1` | Same corpus across adapters, required metadata and existing effective-variable traversal |
| T011 | `GOFLAGS= go test ./internal/services/variable/... -run 'TestRefactorScopeVariablePage|Test.*(ScopeVariable|ProcessInstanceVariable)' -count=1` | Separate root/local corpus, optional totals/strict scopes/cursors and native payloads |
| T012 | `GOFLAGS= go test ./internal/services/processdefinition ./c8volt/process -run 'TestRefactorSelectorPreparation' -count=1` | Service strategy and unchanged public reader implementers; then cmd selector/TTY/machine cases |
| T013 | `GOFLAGS= go test ./internal/services/resource ./c8volt/resource -run 'TestRefactorDeploymentRun' -count=1` | Pure ID/XML preparation; then existing cmd deployment/embedded/creation-stage cases |
| T014 | `GOFLAGS= go test ./internal/services/... -run 'TestRefactorVersionContract|TestFactory_|TestAssignUserTask|TestCompleteUserTask|TestBulkCompletion' -count=1` | Version/assertion ownership mapping and retained negative/wire cases; only isolated fakes |
| T015 | `GOFLAGS= go test ./internal/services/element ./internal/services/processinstance ./internal/services/usertask -run 'TestRefactor|Test.*Variable' -count=1` | No lost/moved declarations or execution behavior after service file split |
| T016 | `GOFLAGS= go test ./c8volt/... -count=1` | Public consumer compile probes, unchanged tags and model/conversion wire fixtures |
| T017 | `GOFLAGS= go test ./config ./internal/services/common ./cmd -run 'TestRefactorScaffolding|TestConfig|TestCommandContract|TestCapabilities|TestCompletion' -count=1` | Reference audit, public compatibility and dormant registration retained |
| T018 | `GOFLAGS= go test ./testx -run 'TestRefactorDocumentationWorkflow' -count=1` | Parsed workflows and shell executed only with fake tools/dummy credentials |
| T018 syntax | `bash -n scripts/docs-serve.sh scripts/docs-site.sh` | Syntax only, no server/build/publish |

All units format touched Go files and use `git diff --check`. Race validation is mandatory
for changed worker/shared fixtures and used as shown; select broader existing families when a
targeted result exposes an uncovered path. Do not blanket-run unrelated packages after every task.

## Cross-cutting checkpoint

T017 includes one broad isolated race checkpoint because shared snapshots, worker mechanics,
lifecycle orchestration and command reporting changed across packages. Inspect Makefile and
ordinary test build tags first, then use `GOFLAGS= make test` if it still runs only
`go test ./... -race -count=1`, or equivalently the verified isolated command. Environment-
backed suites remain excluded. Record the actual command/results; prior 370 test evidence is
historical and cannot substitute. T018 static/fake tooling checks run afterward and do not
invalidate unchanged runtime evidence. Public export/interface/transitive-import checks and
the finding/task map are final acceptance gates, not additional setup/polish tasks.

## Recording results and limitations

Record relevant task evidence and any C05 proposed diagnostic exception in this feature's
future progress.md during implementation. No progress completion is recorded by planning.
Differentiate a no-tests match, sandbox failure, failed assertion and successful run. Report
any real-engine behavior as unverified; it is not a blocker requiring a target. CLI help,
metadata and schemas are frozen; no docs-content run is needed for pure internal refactoring.
