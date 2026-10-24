# AGENTS.md

## Purpose
- This file defines repository-specific guidance for agents working in `c8volt`.
- Keep instructions focused on durable project conventions, architecture boundaries, validation, and documentation.
- Do not add temporary issue details here. Put feature-specific requirements in the matching `specs/<feature>/` artifacts.

## Required Project Context
- Before implementation work, read the active feature artifacts under `specs/<feature>/` when they exist: `spec.md`, `plan.md`, `tasks.md`, and supporting files such as `research.md`, `data-model.md`, `quickstart.md`, and `contracts/`.
- Ralph-driven implementation work must also read and follow `specs/ralph-implementation-rules.md`.
- The Ralph rules are mandatory for Ralph iterations and contain the detailed project map, layering rules, CLI UX rules, validation guidance, and iteration discipline.
- If feature artifacts conflict with `specs/ralph-implementation-rules.md`, stop and surface the conflict before implementing.

## Project Architecture
- CLI commands, flags, validation, command metadata, and rendering live under `cmd/`.
- Public facade APIs live under `c8volt/` and `c8volt/<area>/`.
- Public facade options live in `c8volt/foptions`; public facade error conversion lives in `c8volt/ferrors`.
- Version-neutral domain types live in `internal/domain`.
- Version-neutral service contracts and factories live in `internal/services/<area>`.
- Camunda version-specific adapters live in `internal/services/<area>/v88`, `v89`, and `v810`.
- Generated Camunda clients live in `internal/clients/camunda`; avoid hand-editing generated code unless explicitly required.
- Shared production helpers live in `toolx`; shared test helpers live in `testx`.
- Feature planning artifacts live under `specs/<feature>/`.

## Layering Rules
- Prefer existing package ownership and local patterns before introducing new structures.
- `cmd` may call public facades and command support helpers, but must not call generated Camunda clients or versioned service implementations directly.
- Public facades should be thin: map public inputs to internal service inputs, delegate to internal services, map outputs back, and convert errors through `c8volt/ferrors`.
- Backend mechanics such as pagination loops, worker pools, polling, retries, wait loops, dependency expansion, and mutation workflows belong in internal services, not in CLI commands or public facades.
- Keep version-specific API differences explicit in the matching `v88`, `v89`, or `v810` service package.
- Reuse `toolx`, `toolx/pool`, `internal/services/common`, and `testx` helpers before adding new helper code.

## CLI And Operator UX
- Preserve established command output patterns and wording from nearby command renderers and tests.
- Human output should be compact, stable, and scan-friendly.
- JSON and other machine-readable output must use stable structs and the shared command envelope when the command supports the shared contract.
- Keys-only output must print one key per line and nothing else.
- Do not add noisy endpoint, request, cursor, or per-key lifecycle detail to default human output; keep functional workflow details behind `--verbose` and low-level HTTP diagnostics at DEBUG (`--debug` or configured debug logging), independent of verbose.
- When command output changes, update tests for the affected human, JSON, keys-only, error, prompt, and activity behavior where relevant.

### Mandatory Diagnostic Logging
- Human errors, warnings, and diagnostic messages MUST use the configured command-context logger through the established command/error helpers. Writing to the correct stderr stream alone is not compliance. Do not emit diagnostics with `cmd.PrintErr*`, `fmt.Fprint*`, `fmt.Print*`, builtin `print`/`println`, or direct stdout/stderr writes; do not create a separate logger or use a global/default logger to bypass command configuration. Bootstrap failures must use the existing bootstrap/error fallback helpers.
- Keep the distinction explicit: selected-format results use the result renderers on stdout; interactive prompts use plain text on configured stderr; diagnostics use the configured logger. Do not move result payloads into the logger or add logger prefixes to prompts.
- Classify output before implementing a new execution mode, including its failure path. Batch, partial-result, and exit-after-render paths are not exceptions to logger use. Preserve partial results and the original error class/exit code, render a shared JSON envelope exactly once, and do not invoke a second error renderer after that envelope.
- Never print or log raw joined errors, transport errors, or generated-client errors at normal human levels. Replacing a direct print with `log.Error(err.Error())` does not satisfy this rule when the error contains HTTP diagnostics. Use one compact aggregate error for a failed batch, with counts and a useful bounded reason; group repeated causes instead of printing one transport failure per key.
- Full URLs, HTTP methods, raw request/response bodies, cursors, and wrapped transport chains belong only in the existing DEBUG diagnostics path with its redaction rules. `--verbose` alone must not enable them. Keep useful per-resource outcome facts under verbose and preserve established machine-readable error evidence without changing its schema to fix human formatting.
- Before declaring an output change complete, trace a failing execution from service error through facade conversion, rendering, logging and exit. Review every new or changed direct print and every use of `err.Error()`/`%s`/`%v` with an error at an output boundary. Existing violations are not precedent.

### Diagnostic Output Validation
- For changes to error/output paths, exercise failures through command execution with a configured logger and capture stdout and stderr separately. Assert logger routing, severity, configured/inherited destination, and no duplicate emission. A substring assertion such as `stderr contains "operation:"` is insufficient.
- Include a multi-resource failure with repeated backend errors containing a distinctive URL, method, body and resource key. At normal level and with verbose alone, assert compact aggregate diagnostics and absence of raw HTTP details on **both** streams. Under quiet, retain the compact error while suppressing informational output. Under JSON, decode exactly one envelope with EOF and retain partial results and the original failure exit code.
- Verify DEBUG visibility through both `--debug` and configured debug level, independently of verbose, using only isolated fakes. Verify diagnostic redaction and keep stdout free of diagnostic records. Check the active plain and structured logger formats; do not assume structured attributes appear in every handler. Account for existing flag precedence when combining modes.
- Required checks must detect the violation: reverting logger routing to a direct stderr print or exposing the raw aggregate at normal level must fail the assertions. Passing happy-path, result-stdout-only, or renderer-only tests does not establish compliance.

### Empty Results And Successful No-Ops
- Treat empty results and early returns as part of the command's output contract.
  Route them through a mode-aware view helper; do not print human summaries
  directly from discovery or execution branches before checking the output mode.
- For commands using the shared JSON contract, a successful empty result must
  emit exactly one successful envelope with the existing command-appropriate
  empty payload. Preserve established omission and null-versus-empty-array rules;
  do not invent report entries or change shared schemas to represent no work.
- Keys-only output with no keys must contain zero bytes, including no blank line.
  Preserve the established human empty-result wording, and suppress informational
  summaries in quiet mode without suppressing explicitly requested JSON results.
  Preserve existing output-mode precedence and automation behavior.
- Derive a successful no-op from authoritative completed discovery or planning,
  not from an empty intermediate page, absent reports, a user abort, or an error.
  Preserve continuation through sparse pages and existing failure handling.
- When no mutation was submitted, do not report an accepted or pending mutation,
  even with `--no-wait`. Dry-run output must remain a preview and must not claim
  mutation submission. Empty scopes must not trigger mutation confirmation or
  mutation requests, and rendering must not add discovery requests.
- Test affected commands through their execution paths, not only view helpers:
  cover normal execution and dry-run, human/JSON/keys-only/quiet output, quiet
  combined with machine output, and supported auto-confirm/automation/no-wait
  combinations. Capture stdout and stderr separately; decode one JSON envelope
  and require EOF afterward, assert exact human output and zero-byte keys output,
  and verify request counts and absence of confirmation and mutation calls.
- Use real terminal stdin to verify prompt-free empty-scope completion when
  interactive confirmation is otherwise eligible. Retain regression coverage
  for nonempty results, sparse pages, explicit-key execution, aborts, and errors.

### Output Streams And Interactive Prompts
- Reserve stdout for command results in the selected output format.
- Write confirmation questions, paging continuation prompts, and other interactive
  control text to the command's configured stderr writer (`cmd.ErrOrStderr()`).
- Shared prompt helpers must accept an explicit writer; do not write directly to
  process stdout or use a mutable global prompt destination.
- Keep prompts as plain interactive text without logger prefixes.
- Terminal stdin does not imply terminal stdout. Preserve each command's existing
  prompt eligibility checks when stdout is redirected or piped.
- Stream-routing changes must preserve prompt wording, default answers, input
  handling, EOF behavior, auto-confirm, automation, and caller-specific abort or
  paging-stop behavior.
- Tests for interactive output must exercise real terminal stdin with stdout and
  stderr captured separately. Pipe-only input and mocked terminal checks are
  insufficient to prove prompt routing.
- Verify configured and inherited stderr destinations, uncontaminated command
  results, and keys-only paging with exactly one key per stdout line.

## Command File Cohesion
- Keep `cmd/<verb>_<noun>.go` focused on Cobra construction, flags, validation, top-level dispatch, and the command's ordinary execution path.
- Put a distinct execution mode or lifecycle such as watch, polling, streaming, follow, batch, or interactive operation in `cmd/<verb>_<noun>_<mode>.go` once it has its own runner, state, timing, retry, status, or request-building behavior.
- A mode that adds three or more mode-specific declarations across functions, types, constants, or package variables must use a dedicated mode file before the change is complete.
- Keep final output formatting in `cmd/cmd_views_<area>.go`; keep mode lifecycle and mode-only status/control behavior in the focused mode file.
- Existing large command files are not precedent for adding another concern. Split newly added cohesive behavior without broad unrelated refactoring.

## Testing And Validation
- Follow the implementation testing boundary below before selecting or running any validation command.
- Add or update tests close to the changed package.
- Prefer targeted validation first, then broader validation when the change has wider blast radius.
- For command changes, start with targeted `go test ./cmd -run '<TestNameOrPattern>' -count=1`.
- For facade changes, start with targeted `go test ./c8volt/<area> -run '<TestNameOrPattern>' -count=1`.
- For internal service changes, start with targeted `go test ./internal/services/<area>/... -run '<TestNameOrPattern>' -count=1`.
- The repository full test target is `make test`, which runs `go test ./... -race -count=1`.
- Run `gofmt` on touched Go files.
- If validation cannot be run, call that out explicitly.

### Implementation Testing Boundary
- During implementation, including Ralph iterations, never run integration tests, smoke tests, or manual validation against a real running environment. This includes local Camunda installations, Docker/Kind/Kubernetes clusters, development, shared test, staging, and production environments; a local or disposable label does not make an environment an isolated test fixture.
- Use unit tests, generated-client mocks, test-owned HTTP fake servers, and isolated CLI/subprocess/terminal tests. Cross-layer tests are allowed only when their service dependencies are test-owned fakes, not real Camunda services.
- Do not discover candidate live test targets, inspect Kubernetes contexts or workloads, probe cluster versions/connectivity, start or provision real services, deploy BPMN, or create/mutate/clean up live resources to validate implementation.
- Before running a test target or script, verify that it stays within this boundary. Do not invoke `make integration-test*`, `make integration-cli-*`, live-test build tags, or environment-backed test helpers. Ensure inherited flags, profiles, credentials, or environment variables cannot enable live tests indirectly.
- Planning and task generation must not add live-environment validation, target designation, or cleanup as an implementation completion gate. A general request to implement, test, validate, or run Ralph does not authorize live-environment testing.
- This boundary overrides live-test requirements in existing specifications, plans, tasks, quickstarts, and agent handoffs. Correct those requirements to isolated validation, preserve historical evidence, and report actual coverage accurately. Never label an unperformed live check as passed.
- Do not ask for a disposable target or keep implementation blocked waiting for one. Complete the permitted implementation validation and report any real-engine behavior that remains unverified as a coverage limitation.
- Live-environment testing belongs to a separate workflow explicitly requested by the user for a specific target; it must not be initiated or folded into implementation automatically.

## Documentation And Generated Artifacts
- Keep user-facing documentation and examples aligned with behavior changes.
- When command behavior, flags, aliases, examples, or output contracts change, update command source metadata and regenerate CLI docs with `make docs-content`.
- Do not hand-edit generated CLI docs under `docs/cli/*` when the command source can regenerate them.
- For generated Camunda clients, prefer updating the OpenAPI mutation or refresh workflow under `api/` and regenerating clients.
- Speckit memory under `.speckit/memory` and feature artifacts under `specs/` are project context and should be preserved.

## Launching Ralph

- On macOS, use Terminal.app via `osascript` and Terminal's `do script` command for Ralph launches unless the user requests another terminal. A request to launch Ralph authorizes this normal launch mechanism; computer-use UI automation is not a prerequisite.
- Use the shell execution tool and its normal approval/escalation mechanism when required. A computer-use tool's inability to control Terminal.app does not establish that shell-based AppleScript is unavailable. Report the specific restriction; do not bypass an actual shell approval denial or a higher-priority prohibition.
- Follow the `speckit-ralph-run` skill for prerequisite checks and configuration resolution. Launch the orchestrator only; do not implement tasks inline. Set the working directory to the repository root and quote every argument across shell and AppleScript boundaries.
- Distinguish the launcher's arguments, the child agent's sandbox mode, and this chat's tool permissions. Inspect the orchestrator before making claims about `--sandbox danger-full-access`; when it already supplies that Codex option, no launcher flag or change to this chat's permissions is needed.
- Keep the launch visible and interruptible with Ctrl+C. Capture output in a fresh temporary log outside the repository while displaying it in Terminal.app, and preserve the orchestrator's exit status. An opened window, echoed command, or hidden PTY alone does not prove launch success.
- Verify fresh orchestrator startup output within the skill's bounded check: the selected feature header and iteration line, or its documented immediate completion/failure result. Then report the observed status, configuration, and terminal location and stop monitoring.
- Once a command has been dispatched, never start another loop merely because verification failed. Inspect the existing launch output and report uncertainty or the concrete error; avoid duplicate runs. Do not ask the user to paste commands manually until the permitted launch mechanism has been attempted or a specific applicable restriction prevents it.

## Task Planning For Ralph

- Apply these rules when creating or revising feature plans and `tasks.md`, before Ralph starts.
- Treat each implementation task as a cohesive work unit that leaves the project compiling and passes relevant checks.
- Include implementation, necessary fixtures, tests, formatting, and targeted validation in the same task.
- Treat context reading, branch checks, routine review, and progress tracking as work-unit responsibilities, not standalone tasks.
- Keep interface changes, affected implementations, and required test-double updates together when separating them would break compilation.
- Build shared behavior once. Do not implement a single-item workflow and later rebuild it for collection input when both are already required.
- Use user stories for acceptance coverage; avoid story boundaries that require repeated changes to the same underlying behavior. Respect Ralph's permitted story scope when grouping work for execution.
- Split tasks only when each part has a distinct, independently verifiable result or when the combined work would exceed a practical iteration. Do not consolidate tasks into oversized units merely to reduce the count.
- Do not create tasks solely to enable hypothetical parallel execution.
- Before finalizing `tasks.md`, review adjacent tasks for consolidation and explain any deliberate separation of tightly coupled work.
- Preserve all requirements and meaningful tests when consolidating. Fewer tasks must not mean reduced coverage.
- For a feature extending existing command patterns, more than ten tasks triggers a consolidation review, not an automatic rejection or request for approval.

## Git And Commit Rules
- Never use the `codex/` prefix for new branches.
- Name new issue-backed branches `<issue-number>-<slug>`.
- Reuse existing issue or feature branches when they already exist; do not rename them without explicit instruction.
- For GitHub issue-backed Spec Kit work, the GitHub issue number is authoritative for the `specs/<number>-<slug>/` prefix and feature branch label. This overrides `.specify/extensions/git/git-config.yml` `branch_numbering: sequential`; pass the issue number explicitly with `--number <issue>` or correct the generated folder and references before planning or implementation continues.
- Commit messages must follow Conventional Commits format.
- Add a scope in parentheses when a clear scope exists.
- Every issue-associated commit MUST end its subject with a space followed by the bare issue number, such as `#331`. Do not wrap the issue reference in parentheses or brackets. Example: `docs(specs): refine implementation tasks #331`.
- Before committing issue-associated work, verify the subject ends with ` #<issue-number>`.
- Prefer small commits grouped by purpose.
- Run the closest relevant checks before committing when the repository provides them.

## Active Speckit Plan
<!-- SPECKIT START -->
- Active Speckit implementation plan: `specs/311-complete-user-tasks/plan.md`
<!-- SPECKIT END -->
