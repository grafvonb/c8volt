# Quickstart: Validate Definition Lifecycle Observation

**Feature**: [spec.md](spec.md)
**Contracts**: [CLI](contracts/cli.md), [library](contracts/library.md)

This guide is for implementation validation after the feature is built. Commands below exercise isolated tests or compile integration code. They do not require a Camunda installation, credentials, a selected live profile, or deployment of BPMN. Planning itself needs only document checks; do not run runtime tests just to complete this plan.

## Prerequisites and boundaries

- Work on `348-definition-lifecycle-state` from the repository root, with the declared Go 1.26 toolchain and existing dependencies available.
- Read `AGENTS.md`, `spec.md`, `plan.md`, and, for Ralph work, `specs/ralph-implementation-rules.md` before implementation.
- Audit selected test helpers before execution: generated-client mocks, test-owned HTTP servers, and isolated subprocess/PTY fixtures are allowed; real services, cluster discovery, environment-backed helpers, and live-test build tags are not.
- The commands use a minimal environment with inherited build flags disabled. Keep all command-test service URLs and configuration explicitly test-owned. Do not inherit profiles, tokens, endpoint overrides, or test-enabling environment settings.
- Preserve existing PTY platform availability handling and report any skipped terminal coverage. Do not substitute a real environment when a test-owned fixture fails.

## 1. Data-path checks

After implementing the additive field and shared mappings, run the affected service and facade cases:

```sh
env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= \
  go test ./internal/services/processdefinition/... -run 'ProcessDefinition' -count=1

env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= \
  go test ./c8volt/process -run 'ProcessDefinition' -count=1
```

Extend existing tests or name new focused cases with `ProcessDefinition` so these commands select them. Confirm the selected tests include lifecycle assertions; a no-tests-to-run result is not validation.

Expected outcomes:

- ACTIVE, DRAINING, DELETED, and an unfamiliar nonempty value survive lookup, serialized search-body decoding, facade conversion, paging/latest selection, and watch snapshots.
- Missing/null state serializes without the property; later omitted responses do not retain an old value.
- v88/v89 reads keep state empty and use their current requests.
- DRAINING with active=0 and suspended=2 retains independent values; existing count/page requests and options are unchanged.
- Returned DELETED items are preserved and do not become not-found results.

## 2. Ordinary output and watch checks

After updating the shared display and watch behavior, run the relevant command tests:

```sh
env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= \
  go test ./cmd -run 'ProcessDefinition' -count=1
```

Use the existing fake-backed statistics fixture, watch harness, and PTY helpers. This pattern includes current definition read, view, watch, suspended-statistics, timeout, terminal, and error-envelope tests; add new lifecycle cases under matching names.

| Controlled case | Required evidence |
| --- | --- |
| Key lookup and listing; human/JSON; with/without stat | Supplied state is visible in the proper field/cell, counts stay independent, original read counts remain. |
| Mixed known/unknown/omitted state rows | Stable lifecycle/stat columns, unknown values preserved, no placeholders for missing values. |
| All state omitted, including v88/v89 | Existing row bytes and JSON omission remain compatible. |
| Empty listing, keys-only, quiet plus requested JSON, XML, explicit not-found | Existing payload/exit conventions, one envelope followed by EOF, zero empty key bytes, no extra requests/prompts. |
| First/unchanged/state-only/stats-only/simultaneous watch changes | Unmarked baseline; independent highlighting; one updated count per changed row; cleared old markers. |
| Supplied → omitted → supplied and unfamiliar values | No stale state or empty bold spans; correct change markers. |
| Failed refresh, failed result/summary write, later recovery | Last successful baseline retained until a complete successful rendering. |
| ACTIVE → DELETED without DRAINING; later disappearance | No invented transition; returned row kept until actual key absence. |
| Real terminal output with test-owned backend | Only changed value is bold, alignment intact, supported plain fallback, uncontaminated result stream. |
| Rejected watch combinations and existing timing/timeout/retry paths | No compatibility or scheduling change. |

### Logger regression checks within the same command tests

Install configured command-context loggers in fixtures; do not rely on an unconfigured default logger or a private summary handler. Capture stdout, configured diagnostic destination, and inherited command stderr independently.

- Summary is INFO; slow warning is WARN; timing detail requires verbose and INFO. Plain and structured handlers honor configured/inherited destinations with no duplicate records.
- WARN level suppresses INFO summaries/timing while allowing warnings and normal result rows. Suppression is not a failed render and must not block baseline commit. Existing quiet non-watch behavior and rejection of quiet watch remain intact.
- Configure plain-time explicitly for timestamp-specific assertions. Preserve repaint → summary → rows ordering.
- A configured summary handler that returns a write error must leave the baseline unchanged. A void logging helper alone cannot prove that guarantee.
- Retain/extend fake-backed failure execution with distinctive URL, method, body, and resource-key markers; normal/verbose human output must not expose raw HTTP details. Where multiple definition reads fail, retain a bounded aggregate/class rather than repeated raw transport text. Under JSON require one envelope, EOF, and the original failure class/exit behavior.
- Verify existing redacted DEBUG diagnostics through both `--debug` and configured debug level, independently of verbose, using isolated fakes. Keep diagnostics off stdout and test the supported plain and structured formats.
- Assertions must fail if the corrected paths return to direct stderr printing or a new unconditional plain handler. Preserve existing retry-warning streak and activity-writer behavior.

## 3. Documentation and integration-source checks

Update command help/metadata and README before regenerating documentation:

```sh
env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= make docs-content
```

Review generated CLI references and the docs homepage for 8.10 availability, omission on older versions, ACTIVE/DRAINING/DELETED meanings, separate suspended counts, runtime-versus-history deletion, configured watch logging, and the explicit `delete pd --force` distinction. Do not hand-edit generated docs or add new flags.

After changing the existing draining scenario to observe via the CLI, compile it only:

```sh
lifecycle_check_dir=$(mktemp -d)
env -i PATH="$PATH" HOME="$HOME" GOENV=off GOFLAGS= \
  go test -tags=integration -c ./integration/cli -o "$lifecycle_check_dir/cli.test"
```

Do not execute that binary, the draining test, or an integration Make target during implementation. Review the observer for bounded command execution, exact definition identity, supplied state, one-envelope decoding, unchanged setup/deletion semantics, and preserved independent cleanup evidence. Do not add suspended live work to an ACTIVE-only cleanup path or add explicit no-wait tests.

## 4. Completion evidence

Run `gofmt` on touched Go files, review `git diff --check`, and record the selected checks and actual outcomes in the implementation progress record. Do not repeat checks merely because a documentation-only edit or commit follows them. Use `make test` only when actual shared-helper/mechanics changes or unresolved focused failures justify the broader race-enabled suite; keep its environment free of integration/live tags and record why it was selected.

Passing isolated checks establishes conversion, rendering, comparison, configuration, and request-contract behavior. It does not establish live engine transition timing, patch-specific observations, or production performance. Live testing remains a separate workflow explicitly requested for a specific target.
