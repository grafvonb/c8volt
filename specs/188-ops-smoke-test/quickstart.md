# Quickstart: Ops Execute Smoke Test

## Preview The Workflow

```bash
c8volt ops execute smoke-test --dry-run
```

Expected behavior:

- validates local configuration
- performs read-only connectivity validation where applicable
- selects the version-matched embedded fixture
- shows deployment, run, walk, cleanup, and report steps as planned
- submits no mutation requests

## Preview With An Audit Report

```bash
c8volt ops execute smoke-test --dry-run --report-file smoke-test.md
```

Expected behavior:

- writes a Markdown report only because `--report-file` was supplied
- report clearly marks `dryRun: true`

## Run One Smoke Test With Default Cleanup

```bash
c8volt ops execute smoke-test
```

Expected behavior:

- validates connectivity
- deploys the configured-version multiple-subprocess fixture
- starts one process instance
- walks the created process-instance family
- deletes the created process instance through existing delete behavior
- deletes the deployed process definition only when no unrelated instances block cleanup
- reports final outcome

## Run Multiple Instances

```bash
c8volt ops execute smoke-test --count 5
```

Equivalent shorthand:

```bash
c8volt ops execute smoke-test -n 5
```

Expected behavior:

- starts five instances from the deployed smoke-test definition
- walks every created instance
- reports requested count, created count, and created keys

## Retain Created Resources For Inspection

```bash
c8volt ops execute smoke-test --no-cleanup
```

Expected behavior:

- leaves created process instances and deployed process definition in place
- prints retained keys and definition metadata for later inspection or cleanup

## Automation JSON With Report

```bash
c8volt ops execute smoke-test --count 10 --automation --json --report-file smoke-test.json --report-format json
```

Expected behavior:

- runs non-interactively according to the existing automation confirmation contract
- emits deterministic JSON on stdout
- writes a structured JSON audit report

## Validation

Targeted validation should start with the package closest to the changed area, then broaden:

```bash
go test ./cmd -run 'TestOpsExecuteSmokeTest|TestCommandCapabilityForCommand_OpsExecuteSmokeTest' -count=1
go test ./c8volt/ops ./internal/services/ops -run 'SmokeTest' -count=1
go test ./cmd ./c8volt/ops ./internal/services/ops ./internal/services/processinstance ./internal/services/processdefinition -count=1
make docs-content
make test
```

Ralph implementation iterations must receive:

```text
--implementation-context specs/ralph-implementation-rules.md
```
