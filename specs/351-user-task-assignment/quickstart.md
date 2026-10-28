# Validation Quickstart: User-Task Assignment and Unassignment

**Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

This guide is for validation after implementation. The commands below were not run during planning, and new test families are planned names, not existing passed tests. Planning itself requires document consistency, whitespace, links and feature-pointer checks only.

## Prerequisites and boundary

- Work from the repository root on `351-user-task-assignment` with Go 1.26.2 and existing dependencies available.
- Read `AGENTS.md`, this feature's specification/design, and `specs/ralph-implementation-rules.md` before Ralph implementation.
- Use only unit tests, generated-client mocks, test-owned HTTP fake servers and isolated command subprocess/terminal fixtures. Every backend address/configuration comes from the test fixture.
- Do not run a built CLI against a saved profile or real endpoint, start Camunda or a cluster, deploy BPMN, discover live targets, or invoke integration/smoke targets or build tags.
- Before each selected test family, inspect its setup for test-owned dependencies and confirm it exists. A successful command saying “no tests to run” is not validation.
- Preserve inherited/default logger destination and terminal coverage within fixtures; do not depend on the operator's own credentials, shell helper flags or configuration.

The optional shell helper below runs a command with inherited application/live-test and helper activation removed. It does not create a service or write any file. Tests still must explicitly construct their own fake endpoint and temporary configuration; environment cleanup alone is not proof of isolation.

```bash
isolated_check() {
  python3 - "$@" <<'PY'
import os, subprocess, sys
blocked_prefixes = ('C8VOLT_', 'CAMUNDA_', 'ZEEBE_')
blocked_names = {'GO_WANT_HELPER_PROCESS', 'KUBECONFIG', 'GOFLAGS', 'GOWORK', 'MAKEFLAGS', 'MFLAGS'}
clean_env = {k: v for k, v in os.environ.items()
             if k not in blocked_names and not k.startswith(blocked_prefixes)}
clean_env.update(GOFLAGS='', GOWORK='off')
raise SystemExit(subprocess.run(sys.argv[1:], env=clean_env).returncode)
PY
}
```

Do not remove fixture-owned values that tests intentionally supply to their child processes. Review any additional repository-specific live-test selector before running broader checks.

## 1. Native adapter and evidence checks

After adding the planned tests, list their names and then run them; align names with the implemented families if needed.

```bash
isolated_check go test ./internal/services/usertask/... -list 'Test(Assign|Unassign)UserTask'
isolated_check go test ./internal/services/usertask/... -run 'Test(Assign|Unassign)UserTask' -count=1
isolated_check go test ./internal/services/common -run 'Test(Assign|Unassign|Complete)UserTask' -count=1
```

Expected evidence:

- Each version maps the correct native method/path and serializes nonnull false by default, true only when requested, and the unchanged identity; unassignment has no body.
- Fake routes reject every task read, completion, variable write and unrelated request.
- Only documented 204 with valid metadata and no call error establishes acceptance; delayed responses are awaited.
- 400/401/403/404/409 and other classified rejections preserve evidence; generic 500/503/504, malformed metadata and transport ambiguity remain uncertain as appropriate.
- Applicable temporary-network/throttling retries preserve earlier ambiguity, last status and cancellation/backoff evidence. Generic 504/409 do not acquire retry behavior.
- Existing completion evidence tests still pass after the private helper extraction.

## 2. Coordinator, facade and option checks

```bash
isolated_check go test ./c8volt/task -run 'Test(Assign|Unassign|Complete)UserTasks' -count=1
isolated_check go test ./c8volt/foptions -run 'TestUserTaskAssignmentOverride' -count=1
isolated_check go test ./internal/services/usertask/... -run 'Test(Assign|Unassign)UserTasks' -race -count=1
```

Expected evidence: all input validated before mutation; immutable intent; stable deduplication/order; public zero-worker automatic behavior; explicit override mapping; no-wait rejection; no backend requests during dry-run; one-worker sequential and bounded concurrent execution; fail-fast retains in-flight evidence and unscheduled slots; cancellation before/after submission; partial results survive facade conversion with original failure classification. Prove both operations use the same collection behavior, including one-key input.

## 3. Command execution and output checks

```bash
isolated_check go test ./cmd -list 'TestUpdateUserTask(Assignment|Unassignment)'
isolated_check go test ./cmd -run 'TestUpdateUserTask(Assignment|Unassignment)' -count=1
isolated_check go test ./cmd -run 'TestUpdateUserTaskCompletion' -count=1
```

Implement scenarios through command execution, not only parser/view calls. Expected outcomes follow [CLI contract](contracts/cli-user-task-assignment.md):

| Scenario | Required evidence |
| --- | --- |
| Repeated/comma/stdin keys and aliases | Unique flag-first/stdin-next selection, one logical operation per key, ordered results |
| Missing/empty/malformed keys, late invalid input, blank assignee, workers | Failure before any mutation, no discovery, no successful empty no-op |
| Mode conflicts, false flags, empty payload flags, assignment-only override | Exact presence/value rules; existing variable/completion behavior retained |
| Assign default/override and auto-confirm/automation | Override remains explicit false unless enabled; unattended mode grants no extra permission |
| Local dry-run | Zero backend requests, no approval, visible intent without success claim |
| JSON success/error/partial/quiet | Decode exactly one envelope, require a second decode to return EOF, preserve partial payload and original exit |
| Keys-only preview/execution/all-failed | Preview keys or successful keys only; order preserved; zero bytes if none; partial failure exit retained |
| Successful summaries, preview, verbose | Empty human stdout; INFO records at configured/inherited logger destination; correct plain/structured format; quiet/WARN suppress INFO |
| Repeated backend failures | One compact ERROR with bounded grouped reasons; distinctive URL/method/body/key transport detail absent from normal and verbose-only streams; quiet retains ERROR |
| DEBUG flag and configured DEBUG | Redacted existing diagnostics, independent of verbose, stdout remains selected-format results; existing flag precedence respected |
| Render failure after operation failure | No second envelope, compact logger error, original operation class/exit preserved |

Assertions must distinguish logger records from direct stderr prints and catch raw aggregate leakage. Use test-owned distinctive backend evidence, configured logger destinations, severity assertions and separate streams rather than substring-only checks. Machine error evidence remains available under its established schema.

## 4. Real-terminal approval checks

Use `testx.NewCmdTerminalRunner` and existing terminal exchange patterns with process stdin attached to a real terminal. Keep stdout, stderr and configured logger captures separate. Run the relevant new terminal families within the command checks above, adding focused runs while developing them.

Prove one approve prompt naming action/scope and assignment policy; default-no/decline/EOF submit nothing; inherited/configured stderr gets plain text; stdout redirection does not alter existing eligibility; dry-run and invalid input never prompt; auto-confirm/automation bypass approval without granting override. Do not count pipe-only or mocked-terminal tests as proof of this contract. Record any unsupported-platform skip explicitly.

## 5. Compatibility, documentation and final checks

Run affected legacy task/read tests after the shared parser/interface extraction. Select the smallest families first and use package-wide checks where interface/double changes need wider coverage:

```bash
isolated_check go test ./cmd -run 'Test(UpdateUserTask|GetUserTask|GetProcessInstance.*UserTask)' -count=1
isolated_check go test ./c8volt/task ./internal/services/usertask/... -count=1
```

Format touched Go files, update command/parent metadata and README, then regenerate docs from source:

```bash
isolated_check make docs-content
git diff --check
```

Inspect generated changes for all new flags, examples, safe override default, response waiting, output and no-wait explanation. Do not hand-edit generated CLI pages.

Once targeted checks pass, run the repository race-enabled suite once because this design changes shared option mapping, common completion evidence internals and interfaces across packages:

```bash
isolated_check make test
```

Before this broad run, verify no live-test build tags or environment-backed tests can activate and inspect selected package guards. Never substitute `make integration-test*`, `make integration-cli-*`, smoke commands or live endpoint probes. Do not rerun passed runtime suites solely for subsequent document edits or commits.

## Report coverage honestly

Record exact commands, outcomes and any test-family/terminal limitations. Isolated tests establish request mapping, evidence handling, scheduling, command output and compatibility. They do not establish actual task-listener execution or patch-specific engine behavior on real Camunda. Report those as unverified coverage, without making live validation a completion gate.
