# Quickstart: Validate Observable Run Confirmation

## Targeted Unit Validation

Run service tests around creation confirmation:

```sh
go test ./internal/services/processinstance/v87 ./internal/services/processinstance/v88 ./internal/services/processinstance/v89 -run 'CreateProcessInstance|WaitForProcessInstance' -count=1
```

Run command tests around `run pi`, keys-only output, command contract, and documentation examples:

```sh
go test ./cmd ./docsgen -run 'Run|ProcessInstance|CommandContract|Generated' -count=1
```

## Documentation Validation

After command help or examples change, regenerate generated CLI docs:

```sh
make docs-content
```

## Manual Smoke Validation

Against a configured Camunda 8.9 cluster:

```sh
c8volt run pi -b C89_NoOpCompletion
```

Expected: command succeeds and rendered process instance details show the observed state, likely `COMPLETED`.

```sh
c8volt run pi -b C89_NoOpCompletion --keys-only \
  | c8volt expect pi --state completed -
```

Expected: pipeline succeeds when the process completes quickly.

For a long-running process:

```sh
c8volt run pi -b Some_Long_Running_Process --keys-only \
  | c8volt expect pi --state active -
```

Expected: downstream `expect pi` owns the strict `ACTIVE` assertion.

## Ralph Handoff

Ralph implementation must be launched only with:

```sh
speckit-ralph-run --implementation-context specs/ralph-implementation-rules.md
```
