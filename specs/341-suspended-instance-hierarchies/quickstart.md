# Quickstart: Focused Isolated Suspension Regressions

This guide describes future implementation validation. No implementation or runtime test was performed during the narrowed planning revision.

## Prerequisites

- Read [spec.md](spec.md), [plan.md](plan.md), [contract](contracts/process-instance-mutations.md), repository AGENTS.md, and `specs/ralph-implementation-rules.md` when using Ralph.
- Inspect selected tests and helpers before running them. Use generated-client mocks, test-owned HTTP servers, and isolated command configuration only; no real endpoint fallback.
- Do not run integration/smoke targets, live build tags, environment discovery, deployments, or live cleanup. Isolate inherited profiles/credentials in any subprocess fixture.
- Commands below clear inherited Go build settings. Select exact new test names if the shown pattern does not include them; a no-tests result is not evidence.

## Focused cases

| Case | Required observation |
| --- | --- |
| Suspended root | Ordinary root cancellation; zero resume |
| Active root / suspended child | Engine cascade modeled by observations; no separate child cancellation |
| Suspended root / active child | Same existing root flow and force rules |
| CLI root plus child selection | One planned root and one successful submission without retries |
| Family remains suspended | Normal confirmation fails within budget; no completion claim |
| Delete without force | CLI refuses before mutation; direct-service conflict semantics retained |
| Force deletion recovery | Existing descendant-first attempts, conflict, cancellation, terminal wait, retry |
| Failed recovery / still suspended | No dependent deletion retry, including with NoWait |
| Opt-outs and existing states | Existing read/submission counts, terminal/absent/unknown behavior retained |
| Dry-run and selector help | No mutation; suspended/8.10 guidance accurate |

Reuse current cancellation/deletion fixtures and generic waiter coverage. Do not create new machinery to simulate the engine or reproduce listener internals. Delayed observations establish client waiting behavior only.

## Targeted checks during implementation

Start with the new cases in their changed package, for example:

```sh
env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/v810 -run 'TestService_CancelAndDeleteProcessInstance' -count=1
env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/... -run 'DryRunCancelOrDeletePlan|SuspendedStateWaits' -count=1
env GOENV=off GOFLAGS= GOWORK=off go test ./cmd -run 'CancelProcessInstance|DeleteProcessInstance|Suspended' -count=1
```

Use actual matching names for new planner/dispatch/help tests and record which cases ran. Reuse older-version controls when relevant:

```sh
env GOENV=off GOFLAGS= GOWORK=off go test ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'TestService_(CancelAndDeleteProcessInstance|CancelProcessInstance|DeleteProcessInstance)' -count=1
```

If a production fix touches shared code or output boundaries, expand checks according to the actual diff and mandatory AGENTS.md requirements. Do not automatically add a full-suite, facade conversion, retry, retention, or logging test project to a tests/help-only change. Run `make test` only when shared runtime impact or unresolved failures justify it.

## Help and documentation

During implementation, format touched Go files, update command source help and relevant README guidance, inspect the generation target, then regenerate CLI documentation:

```sh
env -u MAKEFLAGS -u MFLAGS -u MAKEFILES GOENV=off GOFLAGS= GOWORK=off make docs-content
git diff --check
```

Do not hand-edit generated CLI references. Check that help still describes descendant-first deletion and conflict recovery rather than a new pre-cancel phase.

## Planning-only validation and reporting

For this revision, validate Markdown links, requirement/acceptance mapping, removal of superseded requirements, whitespace, and changed-file scope. Do not run runtime tests solely for planning documents.

During implementation, record actual commands/results and any minimal fix's reproducing regression. Fakes do not prove real Camunda listener/exporter behavior. Earlier #339 live evidence and the earlier 8.10.4 validation proposal remain historical context; no live test is required to complete this implementation.
