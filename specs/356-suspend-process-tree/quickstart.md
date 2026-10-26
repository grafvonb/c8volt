# Quickstart Validation: Process-Instance Tree Suspension

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Contracts**: [CLI](contracts/cli.md), [facade/service](contracts/facade-service.md)

## Boundary and prerequisites

This guide validates a future implementation with isolated fixtures. It does not instruct an operator to run suspension against an environment. During specification/planning, perform documentation checks only; do not run runtime tests merely to complete this phase.

Before implementation validation:

- Use the repository Go toolchain and inspect the actual test/helper declarations matching each command.
- Verify all backend dependencies are generated-client mocks or test-owned HTTP servers. Commands receive test-owned config files/URLs; terminal cases use isolated subprocesses and real PTY stdin.
- Remove inherited live-test tags, GOFLAGS/GOENV overrides, application profiles, base URLs, tokens, and credentials from the test process. Keep only needed tool/runtime environment and explicitly constructed fixture configuration. `GOENV=off GOFLAGS=` below does not by itself sanitize application configuration.
- Do not run integration tags, `make integration-test*`, `make integration-cli-*`, smoke scripts, real Camunda requests, cluster commands, deployments, or live cleanup. No target designation is needed.
- If loopback/PTY permissions are unavailable, report skips/unavailable coverage explicitly; a skipped fixture is not a passing scenario.

## 1. Service planning and facade mapping

Add planner test names under `TestProcessInstanceSuspensionPlan...`, extending existing service stubs and fixture patterns. Then inspect and run:

```sh
env GOENV=off GOFLAGS= go test ./internal/services/processinstance/... -list 'ProcessInstanceSuspension|Descendants'
env GOENV=off GOFLAGS= go test ./internal/services/processinstance/... -run 'ProcessInstanceSuspension|Descendants' -count=1
env GOENV=off GOFLAGS= go test ./c8volt/process -run 'ProcessInstanceSuspension' -count=1
```

Use the existing suspension collection/factory/waiter tests for execution behavior. New planner cases must prove:

- 8.8/8.9 capability guard before discovery or unsupported I/O, including dry-run and public-library entry points.
- All seed validation before I/O; malformed and empty input fail rather than producing no-op success.
- First-seed dry-run is local, validates relevant timing before discovery, and cannot leak its forced option into actual execution. Include invalid waiting configuration with an eventual empty eligible scope and no-wait's existing timing behavior.
- Root/child/nested traversal, active descendants below suspended/terminal ancestors, overlap and duplicate seeds, child-first and ancestor-first order, and deterministic first-observation classification.
- No ancestry/family/root escalation and no mutation during planning. Failed later seeds, canceled context, missing/incorrect identities, malformed keys, and explicit partial/unresolved traversal yield an error with no executable target set.
- Copies of requested/target/member slices at boundaries; caller changes cannot mutate service state. Normalized TargetKeys exactly project eligible members.
- Explicit-key tenant behavior and planner-local suppression preserve discovery scope and DEBUG diagnostics.

Widening public `process.API` requires every affected concrete implementation/stub to compile in the same work unit. After updating them, use an all-package compile-only check if needed:

```sh
env GOENV=off GOFLAGS= go test ./... -run '^$'
```

This checks the new interface's repository-wide compile impact; it is not a substitute for behavioral tests and does not justify running unrelated full suites repeatedly.

## 2. Command execution matrix

Use a tree-specific extension of the existing suspension fixture serving only expected direct reads, child searches, and suspension endpoints. Keep the original exact-key fixture rejecting discovery, so default-mode regressions remain detectable.

```sh
env GOENV=off GOFLAGS= go test ./cmd -list 'SuspendProcessInstance|AllCommandHelpFocusesOnActions'
env GOENV=off GOFLAGS= go test ./cmd -run '^(TestSuspendProcessInstance|TestAllCommandHelpFocusesOnActions)' -count=1
```

| Scenario | Required evidence |
| --- | --- |
| Mixed/nested subtree | Each unique active discovered key is one logical target; already-suspended/terminal/other states have plan classifications and zero mutations. |
| Overlapping flags/stdin | Stable deduplicated seeds and scope; child/ancestor keys never collapse to roots. |
| Failed/partial/unresolved planning | Error before confirmation or any mutation, including failure after another seed was resolved. |
| Tree preview with/without no-wait | Discovery reads allowed; preview statuses only; zero mutations and prompts; plan and eligible keys retained. |
| Actual empty eligible scope | Successful no-op, not dry-run or acceptance; zero mutation facade calls and prompts; plan classifications retained. |
| Empty/invalid input | Existing input error before discovery; no no-op claim. |
| Freeze after approval | Change the fake hierarchy/state before responding to the prompt; mutation keys/counts remain frozen, new descendants are not targeted, changed targets preserve real failure evidence. |
| No-wait | Discovery still occurs; zero confirmation reads; accepted keys/outcome only when actual submissions were accepted and no failure exists. |
| Partial failure/fail-fast/interruption | Original error class/exit, all known per-key acceptance/observations, unattempted eligible targets, one result envelope. |
| Worker options | Existing barriers/count assertions prove CLI forwarding and unchanged bounded collection scheduling; do not add a second pool. |
| Default exact-key | Existing schema, rejection semantics, preview no-I/O behavior, prompt, counts, and zero descendant discovery remain unchanged. |

## 3. Streams, logger, empty results, and activity

Cover human, verbose, quiet, JSON, keys-only, quiet+JSON, quiet+keys, JSON+keys precedence, rejected JSON+verbose, and supported dry-run/auto-confirm/automation/no-wait combinations.

- Human summaries/previews/verbose facts: empty stdout; exact configured INFO message records; plain-time and structured log formats; configured and inherited destinations; suppression at configured WARN and under quiet. Direct writes to stdout or stderr must fail assertions.
- Human failures: one compact aggregate ERROR through the configured logger; grouped repeated causes; no raw URLs/methods/bodies/cursors/transport chains on either stream at normal or verbose level. Test DEBUG via both flag and configured level, existing quiet precedence, and redaction using distinctive repeated-error markers.
- JSON: decode one envelope and require EOF; assert scope classifications separately from mutation items. Preserve original machine error evidence. Actual no-op and preview are succeeded even with no-wait; actual partial failure beats no-wait acceptance. Never append a second envelope or logger record to stdout.
- Keys: preview only eligible keys; execution only accepted keys, including accepted-but-unconfirmed failures with failure exit retained. An empty eligible scope must be exactly zero bytes.
- Machine-writer failure: no second renderer; preserve an existing operation failure exit.
- Terminal prompts: real PTY stdin, separately captured configured/inherited stdout/stderr, redirected stdout, approval/decline/default/EOF, and no mutations before approval or after abort. A real-terminal all-ineligible scope must finish without a prompt when interactive confirmation would otherwise be eligible.
- Activity sink: no execution activity during discovery/preflight/approval/preview/no-op; one shared workflow activity throughout approved collection execution, including confirmation waits; stop before final logging and during unwinding. Assert existing protected-mode suppression. Shared writer terminal behavior is reused, not reimplemented.

Run the targeted race check after adding worker-backed command fixtures:

```sh
env GOENV=off GOFLAGS= GORACE='atexit_sleep_ms=0' go test ./cmd -run '^(TestSuspendProcessInstance|TestAllCommandHelpFocusesOnActions)' -race -count=1
```

## 4. Help, docs, and neighboring behavior

Extend `TestSuspendProcessInstanceMetadata` for --tree and revise source help to describe actions, permissions, mixed-state handling, scope, and limits. Keep stream/logger implementation vocabulary out of Short/Long help so `TestAllCommandHelpFocusesOnActions` passes.

Update README and regenerate CLI documentation from source:

```sh
env GOENV=off GOFLAGS= make docs-content
env GOENV=off GOFLAGS= go test ./docsgen -count=1
git diff --check
```

Inspect the suspension CLI page, command metadata, README and generated homepage for correct examples, discovered/frozen target counts, extra permissions, retained direct-child limit, non-atomic behavior, and missing future descendants. Do not hand-edit generated docs.

Select nearby existing walk, cancel/delete, and suspended-state regressions only when touched code or new evidence gives them a blast radius. The intended design does not modify their implementation, shared traversal, pool, generated clients, or HTTP transport.

Broaden to `make test` (the full race suite) only for actual shared runtime/interface/concurrency impact not adequately covered by targeted tests or unresolved failures; state the reason. Do not repeat tests solely for documentation edits, commits, or task completion.

## Evidence to retain

Record actual commands, covered cases, request counts, forbidden requests, classified/frozen scope agreement, logger destination/severity/level/redaction, one-envelope/exit evidence, PTY and activity coverage, and any skips or unavailable checks. Real-engine behavior remains unverified; no live pass is implied. Planning validation consists of artifact consistency, links, placeholders, and whitespace only.
