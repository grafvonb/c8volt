# Ralph Memory

Feature: 282-all-tenants-override
Started: 2026-08-31T06:07:38Z

## Codebase Patterns
- Root persistent flags are declared in `cmd/root.go` inside `init()` from `rootCmd.PersistentFlags()`. Config-backed flags are bound in `cmd/root_config.go:initViper`; `--all-tenants` must stay out of `initViper`, Viper defaults, environment aliases, and durable config structs.
- Root bootstrap runs `initViper`, help bypass, log-level overrides, then `retrieveAndNormalizeConfig` before installing activity/logging context and remote services. The all-tenants override must run after `retrieveAndNormalizeConfig` and before `cfg.ToContextWithLogWriter`, `tenantOverrideProvenanceFromConfig`, validation, and `installRemoteCommandServices`.
- Private tenant provenance lives in `cmd/cmd_tenant_context.go` as `tenantOverrideProvenance` on command context. Public `tenant.Context` remains effective-state-only and is rendered by `cmd/cmd_views_tenant_context.go`.
- Existing #283 warning rendering is reusable: `tenantOverrideHumanLines` emits `configured tenant: <tenant>`, marks named-to-empty broadening as warning, then `tenantContextPrimaryHumanLine` emits `selection scope: unfiltered across accessible tenants`. `renderTenantContext` suppresses this in quiet/protected output via `shouldRenderTenantContextHuman`.
- Command machine-contract metadata is annotation-backed in `cmd/command_contract.go`; setter/resolver pairs such as `setCommandMutation`/`commandMutationForCommand` and `setContractSupport`/`contractSupportForCommand` are the local pattern for adding all-tenants support. `commandCapabilityForCommand` is the serialization point. Human summary rendering is in `cmd/capabilities.go`.
- Integration example parsing treats inherited root flags in `integration/cli/examples_test.go:isRootFlag`, with value-consuming flags separately listed in `rootFlagConsumesValue`. New boolean inherited flags belong only in `isRootFlag`.
- Generated CLI docs are owned by `docsgen/main.go` and regenerated with `make docs-content`; it calls Cobra markdown generation, `syncCLICommandTree`, and `syncDocsIndexFromReadme`. Do not hand-edit `docs/cli/*` or `docs/index.md`.

## Decisions
- Phase 1 confirmed this feature is CLI-only. Do not touch `c8volt/`, `internal/services/`, `internal/clients/`, generated Camunda clients, or facade options for the all-tenants override.
- The complete concrete-destination rejection inventory remains exactly four executable leaves: `deploy process-definition`, `embed deploy`, `run process-instance`, and `ops execute smoke-test`.

## Gotchas
- `deploy process-definition` currently calls `NewCli`, automation validation, `validateFiles`, and `loadResources` before deployment. All-tenants destination rejection must happen before those side effects.
- `embed deploy` currently calls `NewCli`, `embedded.List`, validates embedded file choices, and reads from `embedded.FS` before deployment. Rejection must happen before embedded inventory/file access.
- `run process-instance` currently calls `NewCli`, parses `--vars`, may validate process-definition selectors remotely for BPMN IDs, and builds creation data with `cfg.App.TargetTenant()`. Rejection must happen before stdin/prompt/activity/request work and before default-tenant targeting can be derived.
- `ops execute smoke-test` validates local flags first, then calls `NewCli`, prepares report/planning state, may validate report paths, prompts, starts activity, and calls `ExecuteSmokeTest`. Rejection must happen before report, prompt, activity, dry-run plan, or remote work.
- Existing untracked `progress.md` and `ralph-memory.md` were created before this iteration's edits; keep them in the coordinated work-unit commit.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd -run 'Test(CommandCapabilityForCommand_DocumentsTenantContract|CommandCapabilityForCommand_IncludesInheritedAndRequiredFlags|CapabilitiesCommand_JSONOutput|RootHelp_PreservesHumanTaxonomyAndDiscoveryCommand|ProcessInstanceHelp_ExposesCompactGlobalFlags|TenantContext)' -count=1`
- `git diff --check`
- `make docs-content`

## Do Not Repeat
- Do not bind `--all-tenants` through Viper, config files, profiles, or environment variables.
- Do not map the flag to `WithIgnoreTenant`, enumerate tenants client-side, or change backend authorization behavior.
- Do not implement concrete-destination rejection inside the four command runners after they have already initialized clients, inspected inputs, or built reports.

## Current Handoff
- Next iteration starts at Phase 2 foundational task T006: add failing inherited `--all-tenants` root/subcommand parsing tests in `cmd/root_test.go`, then stay within Phase 2 until the root flag and support resolver compile and focused tests pass.
