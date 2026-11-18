# Quickstart: Isolated Validation of Connection Failure Explanations

**Plan**: [plan.md](plan.md) | **Contract**: [contracts/human-errors.md](contracts/human-errors.md) | **Evidence model**: [data-model.md](data-model.md)

This guide is for the later implementation. Planning has not run these tests or changed runtime behavior.

## Prerequisites and isolation

- Work from repository root on `381-explain-connection-failures`, based on `10bd90ae`.
- Use the toolchain recorded in `go.mod` (`go1.26.10`) and existing dependencies. No dependency or engine upgrade is required.
- Inspect the chosen test paths and build flags before execution. Use no integration/live-test tags, environment-backed helpers, real credentials/profiles, or running Camunda service.
- New/extended parent fixtures must clear inherited `C8VOLT_*`, `GO_WANT_HELPER_PROCESS`, and Go flags before setting explicit temporary configuration and helper state; reuse `isolateRequestRateEnvironment`. Child-process fake transports must reject every unexpected target without delegating to real network transports. Other fixture requests go only to test-owned HTTP servers.
- Existing test helpers execute subprocesses because error handlers exit. Preserve their separate stdout/stderr capture and configured/inherited logger destinations. Use the appropriate record parser for each logger format.

```sh
cd /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt
git branch --show-current
git merge-base --is-ancestor 10bd90ae HEAD
```

Do not run live CLI examples, provision services, designate test targets, or invoke integration/demo/smoke targets. This feature needs no live-environment validation.

## 1. Formatter causes and connection-test compatibility

After implementation, add `TestBoundedCommandFailureReasons` near the shared formatter tests. Confirm it actually exists before running the selector; Go can succeed with no matching tests.

```sh
GOENV=off GOFLAGS= go test ./cmd -list '^TestBoundedCommandFailureReasons$'
GOENV=off GOFLAGS= go test ./cmd -run '^(TestBoundedCommandFailureReasons|TestConfigConnectionFailureReasons)$' -count=1
```

Expected: all supported typed causes produce the contract's bounded reason, misleading text and class-only cases retain fallback, cancellation remains compatible, and request-rate-plus-connection evidence retains exact request-rate precedence. Existing connection-test strings, especially topology 404 and cancellation, remain exact. Cover raw/wrapped/joined evidence and retained normalized authentication/status evidence separately; do not assume normalization retains every bare transport type.

## 2. Original command failure and logger routing

Extend `TestGetProcessDefinitionLifecycleSearchDiagnostics` with the original authentication refusal, reusing its subprocess helper. Install a test-only fake default transport inside that child to return typed connection refusal for its configured OAuth token request, reject unknown requests, and count attempts. Disable version advisory in this specific fixture and set zero retries. Do not alter production transport injection or add a real refused-service dependency.

```sh
GOENV=off GOFLAGS= go test ./cmd -run '^(TestGetProcessDefinitionLifecycleSearchDiagnostics|TestCommandErrorEnvelopeProcessDefinition|TestConfigConnectionDiagnosticLogging)$' -count=1
```

Expected for the original fixture: exact human message from the contract, one failed authentication attempt, zero gateway requests, unchanged failure class/exit, and successful exit suppression where requested. For human modes, assert zero stdout bytes and exactly one correctly formatted ERROR at the selected logger destination. Check normal, verbose, quiet, quiet+DEBUG precedence, configured/inherited writers, plain/structured formats, and both DEBUG activation paths. Normal/verbose output excludes seeded credentials and raw HTTP markers on both streams and any separate logger sink. JSON retains original detail with one envelope and EOF; quiet+JSON preserves established precedence.

Retain successful fake-response checks and existing connection-test request/retry counts. Do not change machine evidence merely to make human redaction assertions pass.

## 3. Repeated failures, partial results, and local admission

Extend existing fixtures rather than creating a new batch execution path:

```sh
GOENV=off GOFLAGS= go test ./cmd -run '^(TestDeleteProcessDefinitionUnfinishedDiagnosticLogging|TestRequestRateMutationLocalFailure|TestRequestRateMutationInterruption|TestRequestRateCancellationConfirmation|TestRequestRateReportingPartialFailure)$' -count=1
```

Expected: selected/accepted counts and aggregate prefix remain exact; only recognized bounded cause suffixes change. Repeated seeded URL/method/body/key failures yield one compact final ERROR without leaking raw text at normal/verbose levels. Quiet retains the error. Partial JSON retains successful and failed items, original error evidence, class, exit, and exactly one envelope with EOF. Mutation and request counts remain unchanged. Local request-rate admission and cancellation preserve unsent/possibly-submitted distinctions and existing confirmation behavior.

Review assertions to ensure a direct stderr print or raw aggregate would fail: severity and whole-record validation, independent destinations, zero human stdout bytes, no duplicate ERROR, exact aggregate counts, and forbidden markers are required. A substring assertion alone is insufficient.

### Shared formatter consumer coverage

The initial focused selectors missed existing callers, and CI exposed stale class-only human expectations. Inventory all `boundedCommandFailureReason` consumers, including business-ID selection/cancel/delete/assignment/slow analysis, element reads, definition lookup, purge, process creation, and suspension/resumption planning. Update only the human expectations whose typed evidence now produces recognized guidance; preserve machine classes/detail, command-specific overrides, unknown fallback, logger/redaction checks, exits, partial results, and request counts. Do not calculate expected text with the production formatter.

Use the whole command package to find remaining consumers, then run the CI-equivalent race/coverage suite after the affected selectors pass. This broader check is justified by the demonstrated shared-output regression:

```sh
GOENV=off GOFLAGS= go test ./cmd -count=1
GOENV=off GOFLAGS= make cover
```

Use the same isolated environment and test-owned dependencies described above. Inspect test events for skips as well as failures: a sandbox-denied fake listener does not validate a command. Record the broader coverage once it passes; do not repeat it for subsequent documentation-only edits.

## 4. Formatting and documentation

Format only touched Go files using `gofmt -w` with explicit paths. Update README, configuration diagnostics guidance, and concise root help metadata as described in the plan, then regenerate:

```sh
GOENV=off GOFLAGS= make docs-content
git diff --check
git diff --stat
git diff -- cmd/cmd_views_contract.go cmd/cmd_views_config_test_connection.go cmd/root.go README.md docs/configuration.md docs/index.md docs/cli
```

The docs-content target reads command metadata without executing Camunda requests. Do not hand-edit generated CLI files or run the full site build for this correction.

## Completion evidence

Record commands actually executed, outcomes, exact original-failure wording, logger routing/redaction coverage, one-envelope partial JSON coverage, and unchanged class/exit/request counts. Review changed output boundaries for direct prints and raw error interpolation. The targeted selectors above are the initial implementation validation; broaden only for a changed blast radius or unresolved regression. The full race suite and live-engine testing are not automatic gates. Report any unverified engine behavior accurately.
