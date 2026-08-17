# Quickstart: Validate Native Camunda 8.10 Embedded Definitions

## Prerequisites

- Work on branch `275-native-c810-definitions` after the implementation tasks are complete.
- Use the repository Go toolchain.
- Use commit `f2658425` as the protected pre-feature baseline.
- Do not run or modify live integration environments for this feature.

## 1. Validate the fixture inventory and parity

```bash
go test ./embedded -run 'TestC810ProductionDefinitions' -count=1
```

Expected:

- exactly eight C810 files are found;
- all files are well-formed XML and satisfy the planned BPMN invariants;
- identities, platform version, references, and version tags are correct;
- normalized C810 content matches the C89 source;
- no C89 references remain.

## 2. Validate version mapping and embedded commands

```bash
go test ./toolx -run 'TestProductionFixturePrefix' -count=1
go test ./cmd -run 'TestEmbedListCommand_V810' -count=1
```

Expected: V810 maps to `C810_`; stable and unknown mappings retain their prior behavior; the shared embed selector returns exactly the C810 family without a C89 fallback. That selector is used by list and by `--all` export/deploy.

## 3. Validate smoke-test selection and output

```bash
go test ./internal/services/ops -run 'TestExecuteSmokeTestSelectsVersionMatchedFixtures|TestSmokeTestDeploymentUnits' -count=1
go test ./cmd -run 'TestOpsExecuteSmokeTest.*V810' -count=1
```

Expected: V810 selects `C810_MultipleSubProcessesParent` and its two required child definitions, and the chosen CLI smoke path exposes the C810 identity without changing its output schema.

## 4. Verify protected boundaries

```bash
git diff --exit-code f2658425 -- ':(glob)embedded/processdefinitions/C87_*.bpmn' ':(glob)embedded/processdefinitions/C88_*.bpmn' ':(glob)embedded/processdefinitions/C89_*.bpmn'
git diff --exit-code f2658425 -- integration Makefile
```

Expected: both commands produce no diff. The feature adds no stable-fixture changes and no live integration asset or target.

## 5. Verify #273 has one final fixture decision

```bash
rg -n 'V810.*C89|8\.10.*C89|C89.*8\.10' \
  specs/273-camunda-v810-support/spec.md \
  specs/273-camunda-v810-support/plan.md \
  specs/273-camunda-v810-support/research.md \
  specs/273-camunda-v810-support/data-model.md \
  specs/273-camunda-v810-support/quickstart.md \
  specs/273-camunda-v810-support/contracts/service-compatibility.md

rg -ni 'superseded.*#275|#275.*superseded' specs/273-camunda-v810-support/tasks.md
```

Expected: no active C89-reuse requirement remains in the normative #273 design artifacts, and `tasks.md` records that its completed C89-mapping tasks were superseded by #275 without rewriting those historical descriptions.

## 6. Run the repository delivery gate

```bash
make test
git diff --check
```

Expected: all tests pass with the race detector and the diff contains no whitespace errors.
