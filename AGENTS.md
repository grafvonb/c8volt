# AGENTS.md

## Ralph PRD generation
- When generating or updating `scripts/ralph/prd.json`, split work into multiple small user stories.
- Each story must be feasible in one Ralph iteration.
- Prioritize stories by dependency and execution order.
- Keep each story narrowly scoped and independently verifiable.
- Write precise, minimal, testable acceptance criteria.
- Reuse existing project patterns and avoid introducing parallel structures.
- If `scripts/ralph/prd.json` already exists and the task is an update, prefer adding follow-up stories instead of rewriting completed stories, unless they are no longer valid.

## Git and GitHub branch rules
- For issue-based work, first check whether a matching local or remote branch already exists and reuse it when appropriate.
- If the issue already has a linked or existing branch, use that exact branch name.
- Do not invent a different branch name when an issue branch already exists.
- Do not add extra prefixes such as `codex/` unless the user explicitly asks or the repository explicitly requires them.
- Do not create or switch to a different feature branch unless the user explicitly asks.

## Commit rules
- Commit messages must follow Conventional Commits format.
- Add a scope in parentheses after the type when a clear scope exists.
- Reference the GitHub issue in the subject when applicable, for example:
  - `feat(cli): add command #<issue>`
  - `fix(api): handle empty response #<issue>`
  - `refactor(service): simplify implementation #<issue>`
  - `test(module): add coverage for edge cases #<issue>`
  - `docs(readme): update usage examples #<issue>`
- Use lowercase by default, except where capitalization is required for correct names such as `README.md` or product/library names.
- Prefer small commits grouped by purpose.
- Do not use vague commit messages such as `update`, `fix stuff`, or `changes`.

## Validation
- Before committing, run:
  - `make test`

## Project conventions
- Prefer existing project patterns over introducing new structural styles.
- For refactoring work, preserve externally observable behavior unless the issue explicitly asks for behavioral change.
- Favor incremental refactors with verification over broad rewrites.
- CLI root/bootstrap failures should return through the shared `cmd` bootstrap normalization path and `ferrors.HandleAndExit` rather than calling `os.Exit(1)` or open-coding entry-point error rendering.
- When Cobra's generated `completion` command should remain executable but stay out of normal help and interactive suggestions, prefer `rootCmd.CompletionOptions.HiddenDefaultCmd` over custom completion filtering.
- Keep root completion descriptions enabled when interactive shells should show each command's concise `Short` text; Cobra still leaves custom value completions such as flag completions as plain candidate lines when no description is provided.
- CLI command-validation errors that may surface either from Cobra `Execute()` or from in-command `ferrors.HandleAndExit` should be wrapped with the shared `ferrors` class at construction time; bootstrap-only normalization is not enough for paths that fail inside `Run` handlers.
- When a command calls `NewCli(cmd)`, treat the returned config as optional on the error path; bootstrap-only failures can leave `cfg == nil`, so route those failures through the shared command helper/bootstrap context instead of dereferencing `cfg.App.NoErrCodes`.
- Extend shared CLI failure mappings through the rule tables in `cmd/cmd_errors.go` and `cmd/bootstrap_errors.go`; do not duplicate new sentinel-to-class switches in individual commands or ad hoc helpers.
- For versioned internal services, prefer `internal/services/common.PrepareServiceDeps` plus `common.EnsureLoggerAndClients` in constructors so default config/http client/logger handling and test-time client injection stay consistent.
- For internal service refactors with versioned implementations, follow the established `resource`/`processdefinition` pattern: keep the shared package `api.go` assertions, define a version-local `API` interface in each `v87`/`v88` `contract.go`, and keep generated-client contract interfaces limited to the calls the versioned service actually uses.
- When an internal service refactor includes generated-client coverage review, record the kept-versus-deferred candidate endpoints explicitly in the feature `research.md`; if nothing is added, prefer a bounded no-addition rationale over silent omission.
- When changing generated or generated-adjacent artifacts, update the source and regenerate rather than editing derived output by hand when the repository already provides a generation path.
- When a service method follows the standard generated-client success path, prefer `internal/services/common.RequirePayload` for the shared HTTP-status plus non-nil JSON payload validation instead of re-implementing the malformed-response check inline.
- For generated single-object endpoints, `RequirePayload` may still return a decoded zero-value struct on malformed `200 OK` bodies; if the endpoint guarantees one object, validate the converted domain model is non-zero before treating the call as success.
- `internal/services/common.RequirePayload` is also the preferred malformed-response guard for generated XML/string success payloads such as `XML200`; reuse it instead of open-coding empty-200 checks in versioned services.
- For generated XML endpoints that expose both `Body` and `XML200`, keep `RequirePayload` for status validation but fall back to the raw `Body` when `XML200` is present yet empty; the generated XML-to-string unmarshal can discard element markup.
- Processinstance walker helpers treat `Descendants` output as root-inclusive and keep explicit `nil` edge entries for leaf nodes; preserve that shape because delete and tree-rendering flows rely on it.
- Camunda v8.7 processinstance create responses do not include a process-instance key, so service-level wait regression tests should cover wait-dependent delete or explicit state polling paths instead of assuming create-time confirmation can be keyed from the create payload.

## Testing conventions
- Add or update tests alongside refactoring and bug fixes.
- Prefer targeted tests near the changed package, then run the broader repository test suite.
- For refactors, ensure tests verify preserved behavior, not just new internal structure.
- Shared CLI error-model tests should assert `c8volt/ferrors.Normalize`, `Classify`, `ExitCode`, or `ResolveExitCode` directly; reserve subprocess tests for command paths that actually terminate via `ferrors.HandleAndExit` and `os.Exit`.
- CLI JSON output tests should assert the serialized model JSON keys (for example `tenantId`) rather than exported Go field names, because `toolx.ToJSONString` renders the public model tags directly.
- Versioned service factory tests should assert the concrete v8.7/v8.8 service type returned for each supported version and verify unsupported versions through `services.ErrUnknownAPIVersion`, which may normalize invalid inputs to `"unknown"` in the rendered error.
- Processdefinition v8.8 service tests should preserve the tolerant stats-enrichment behavior by asserting successful search/get results when the follow-up stats endpoint returns `200 OK` with a nil payload, leaving `Statistics` unset instead of treating the response as malformed.
- Camunda v8.8 generated search request tests should assert the serialized flat filter values (for example `"tenantId":"tenant"`) rather than `"$eq"` wrapper objects; those generated request unions marshal simple equality filters to scalar JSON.
- Integration helpers should use the current facade signatures directly; for process-definition deployment, tenant selection comes from `cfg.App.Tenant` and `DeployProcessDefinition` only accepts `(ctx, units, opts...)`.
- CLI command tests that execute non-help paths should pass an explicit temp `--config` file; repository-local config or env can otherwise leak into test behavior.
- CLI command tests that assert version-specific payloads should set `app.camunda_version` explicitly in that temp config; otherwise the default test version can route to a different generated client shape.
- `cmd` tests that reuse `Root()` across multiple in-process executions should reset Cobra flag state first, because help-oriented executions leave flags set on the shared command tree.
- `cmd` completion regression tests should drive Cobra through the hidden `__complete`/`__completeNoDesc` root entry points and reset shared flags first; this exercises the real completion seam without requiring shell-specific wrappers.
- When invoking Cobra's hidden `__complete`/`__completeNoDesc` commands directly for smoke checks, pass arguments relative to the root command and omit the `c8volt` binary name; including it makes Cobra treat the request as an unknown command path.
- `cmd` flag-value completion regressions should assert the same forbidden output (`__complete`, hidden `completion`, `Usage:`, or long help text) as command completion regressions; plain candidate checks are not enough to catch completion-format regressions.
- Fresh helper-process `cmd` tests should not call the shared flag reset helper before `SetArgs()`: `StringSlice` defaults round-trip through Cobra as a literal `"[]"`, which can inject phantom values into commands that rely on empty slices.
- When command failures go through `ferrors.HandleAndExit`, assert exit codes with a subprocess helper because the handlers terminate via `os.Exit`.

## Documentation conventions
- User-facing documentation and examples should stay in sync with behavior changes.
- When changing user-facing commands, APIs, or workflows, update the relevant documentation in the same change.
- CLI reference pages under `docs/cli/` are generated from Cobra command metadata via `make docs`; update command help text first, then regenerate instead of hand-editing those files.
- The docs homepage content is synced from the repository `README.md` by `make docs-content`; when README changes should appear in `docs/`, regenerate instead of hand-editing generated docs content.

## Technology baseline
- Follow the repository's current toolchain, dependency, and framework conventions.
- Prefer the libraries and frameworks already established in the project unless the user explicitly asks for a change.

## Issue-specific guidance
- Issue-specific requirements belong in the GitHub issue, the Spec Kit feature artifacts, and the PRD.
- Do not add changing issue-specific details to this file unless they become stable repository rules.

## Active Technologies
- Go 1.26 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...` (058-review-and-refactor-internal-service-cluster-api-implementation)
- Go 1.26 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common` (71-resource-api-refactor)
- Go 1.26 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`, existing facade packages under `c8volt/...` (73-get-resource-id)
- Go 1.26 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`, worker utilities in `toolx/pool` (75-processinstance-api-refactor)
- Go 1.26 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing `c8volt/ferrors`, `internal/exitcode`, `internal/domain`, `internal/services`, and command packages under `cmd/` (19-cli-error-model)
- Go 1.26 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing command helpers under `cmd/` (82-tab-completion-format)
- Go 1.26 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing process-instance command helpers in `cmd/`, shared facade/domain filters in `c8volt/process` and `internal/domain`, generated Camunda clients under `internal/clients/camunda/...`, existing versioned services under `internal/services/processinstance/...` (095-processinstance-day-filters)
- Go 1.26 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, existing process-instance command helpers in `cmd/`, facade types in `c8volt/process`, config model in `config/`, versioned services in `internal/services/processinstance/v87` and `internal/services/processinstance/v88`, generated Camunda clients under `internal/clients/camunda/...` (101-processinstance-paging)

## Recent Changes
- 058-review-and-refactor-internal-service-cluster-api-implementation: Added Go 1.26 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`
