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
- For versioned internal services, prefer `internal/services/common.PrepareServiceDeps` plus `common.EnsureLoggerAndClients` in constructors so default config/http client/logger handling and test-time client injection stay consistent.
- For internal service refactors with versioned implementations, follow the established `resource`/`processdefinition` pattern: keep the shared package `api.go` assertions, define a version-local `API` interface in each `v87`/`v88` `contract.go`, and keep generated-client contract interfaces limited to the calls the versioned service actually uses.
- When changing generated or generated-adjacent artifacts, update the source and regenerate rather than editing derived output by hand when the repository already provides a generation path.
- When a service method follows the standard generated-client success path, prefer `internal/services/common.RequirePayload` for the shared HTTP-status plus non-nil JSON payload validation instead of re-implementing the malformed-response check inline.
- For generated single-object endpoints, `RequirePayload` may still return a decoded zero-value struct on malformed `200 OK` bodies; if the endpoint guarantees one object, validate the converted domain model is non-zero before treating the call as success.
- `internal/services/common.RequirePayload` is also the preferred malformed-response guard for generated XML/string success payloads such as `XML200`; reuse it instead of open-coding empty-200 checks in versioned services.
- For generated XML endpoints that expose both `Body` and `XML200`, keep `RequirePayload` for status validation but fall back to the raw `Body` when `XML200` is present yet empty; the generated XML-to-string unmarshal can discard element markup.

## Testing conventions
- Add or update tests alongside refactoring and bug fixes.
- Prefer targeted tests near the changed package, then run the broader repository test suite.
- For refactors, ensure tests verify preserved behavior, not just new internal structure.
- CLI JSON output tests should assert the serialized model JSON keys (for example `tenantId`) rather than exported Go field names, because `toolx.ToJSONString` renders the public model tags directly.
- Versioned service factory tests should assert the concrete v8.7/v8.8 service type returned for each supported version and verify unsupported versions through `services.ErrUnknownAPIVersion`, which may normalize invalid inputs to `"unknown"` in the rendered error.
- Processdefinition v8.8 service tests should preserve the tolerant stats-enrichment behavior by asserting successful search/get results when the follow-up stats endpoint returns `200 OK` with a nil payload, leaving `Statistics` unset instead of treating the response as malformed.
- Integration helpers should use the current facade signatures directly; for process-definition deployment, tenant selection comes from `cfg.App.Tenant` and `DeployProcessDefinition` only accepts `(ctx, units, opts...)`.
- CLI command tests that execute non-help paths should pass an explicit temp `--config` file; repository-local config or env can otherwise leak into test behavior.
- CLI command tests that assert version-specific payloads should set `app.camunda_version` explicitly in that temp config; otherwise the default test version can route to a different generated client shape.
- `cmd` tests that reuse `Root()` across multiple in-process executions should reset Cobra flag state first, because help-oriented executions leave flags set on the shared command tree.
- When command failures go through `ferrors.HandleAndExit`, assert exit codes with a subprocess helper because the handlers terminate via `os.Exit`.

## Documentation conventions
- User-facing documentation and examples should stay in sync with behavior changes.
- When changing user-facing commands, APIs, or workflows, update the relevant documentation in the same change.
- CLI reference pages under `docs/cli/` are generated from Cobra command metadata via `make docs`; update command help text first, then regenerate instead of hand-editing those files.

## Technology baseline
- Follow the repository's current toolchain, dependency, and framework conventions.
- Prefer the libraries and frameworks already established in the project unless the user explicitly asks for a change.

## Issue-specific guidance
- Issue-specific requirements belong in the GitHub issue, the Spec Kit feature artifacts, and the PRD.
- Do not add changing issue-specific details to this file unless they become stable repository rules.

## Active Technologies
- Go 1.25.3 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...` (058-review-and-refactor-internal-service-cluster-api-implementation)
- Go 1.25.3 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common` (71-resource-api-refactor)
- Go 1.25.3 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`, existing facade packages under `c8volt/...` (73-get-resource-id)
- Go 1.25.3 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`, existing helpers in `internal/services/common`, worker utilities in `toolx/pool` (75-processinstance-api-refactor)

## Recent Changes
- 058-review-and-refactor-internal-service-cluster-api-implementation: Added Go 1.25.3 + standard library, `github.com/spf13/cobra`, `github.com/spf13/viper`, `github.com/stretchr/testify`, generated Camunda clients under `internal/clients/camunda/...`
