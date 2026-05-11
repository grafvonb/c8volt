# Research: Ops Command Foundation

## Decision: Implement ops as Cobra grouping commands under `cmd`

**Rationale**: Existing command families such as `get` and `resolve` define root/grouping command files under `cmd/`, attach them to `rootCmd`, set mutation metadata, and return `cmd.Help()` for grouping behavior. Reusing this pattern keeps help, generated docs, and command discovery aligned.

**Alternatives considered**:

- Put command foundation in a new package: rejected because existing command grouping structure lives in `cmd` and no reusable runtime behavior exists yet.
- Implement concrete workflow scaffolding now: rejected because issue #197 explicitly excludes concrete playbooks.

## Decision: Mark ops grouping commands as state-changing metadata with limited/unsupported automation until concrete leaves exist

**Rationale**: Ops workflows are intended to be operational and state-changing, but the grouping commands do not perform behavior directly. Existing contract metadata supports explicit mutation classification and inherited/limited contract support from children. The implementation should avoid claiming full automation support for grouping-only commands unless a concrete machine contract is actually present.

**Alternatives considered**:

- Mark ops root read-only because help is read-only: rejected because command discovery should describe the command family's intended operational nature.
- Mark grouping commands full automation support: rejected until concrete leaf commands define deterministic payload behavior.

## Decision: Define workflow/report contract types only as lightweight shared scaffolding

**Rationale**: The issue asks for conventions around dry-run, report files, report format inference, structured reports, automation JSON stdout, and step statuses. A small status/report contract can help upcoming workflow issues without implementing resource-specific logic.

**Alternatives considered**:

- Create a full ops facade package immediately: rejected as speculative because no concrete workflow orchestration is implemented in this issue.
- Defer all shared contracts: rejected because upcoming ops workflows would otherwise redefine statuses and report semantics.

## Decision: Validate through focused command tests and generated docs

**Rationale**: The observable behavior is command discovery/help and contract metadata. Existing tests in `cmd/root_test.go`, `cmd/capabilities_test.go`, and command-specific test files are the closest useful validation layer. User-facing docs are generated with `make docs-content`.

**Alternatives considered**:

- Add service/facade tests: rejected because this feature should not add resource service behavior.
- Manually edit generated docs: rejected by repository documentation conventions.

## Decision: Carry Ralph implementation rules through all downstream instructions

**Rationale**: The user explicitly required `specs/ralph-implementation-rules.md` for planning, task generation, and every Ralph implementation iteration. The feature spec and plan record this as a hard launch condition.

**Alternatives considered**:

- Mention the rules only in chat: rejected because Ralph iterations need persistent artifact context.
