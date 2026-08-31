# Ralph Memory

Feature: 282-all-tenants-override
Started: 2026-08-31T06:07:38Z

## Codebase Patterns
- Root persistent flags are declared in `cmd/root.go` inside `init()` from `rootCmd.PersistentFlags()`. Config-backed flags are bound in `cmd/root_config.go:initViper`; `--all-tenants` must stay out of `initViper`, Viper defaults, environment aliases, and durable config structs.
- `--all-tenants` is registered in `cmd/root.go` as a command-line-only root persistent BoolVar backed by `flagAllTenants`; tests in `cmd/root_test.go` cover default false, inherited placement before/after subcommands, explicit true, and explicit false.
- Root bootstrap runs `initViper`, help bypass, log-level overrides, then `retrieveAndNormalizeConfig` before installing activity/logging context and remote services. `tenantOverrideProvenanceFromConfig` now applies active `--all-tenants` after normalization and before `cfg.ToContextWithLogWriter`, validation, and `installRemoteCommandServices`.
- Private tenant provenance lives in `cmd/cmd_tenant_context.go` as `tenantOverrideProvenance` on command context. `AllTenants: true` is private command provenance only; public `tenant.Context` remains effective-state-only and is rendered by `cmd/cmd_views_tenant_context.go`.
- Existing #283 warning rendering is reused: `tenantOverrideHumanLines` emits `configured tenant: <tenant>`, marks named-to-empty explicit `--tenant ""` broadening as warning, and marks named-to-empty `--all-tenants` broadening with exact text `--all-tenants overrides the configured tenant filter; selection is unfiltered`; `tenantContextPrimaryHumanLine` then emits `selection scope: unfiltered across accessible tenants`. Already-empty all-tenants config emits no override chatter. `renderTenantContext` suppresses this in quiet/protected output via `shouldRenderTenantContextHuman`.
- `cmd/config_test.go` now covers active all-tenants output isolation for configuration diagnostics: `config show` YAML serializes effective `app.tenant: ""` plus `tenantContext.filter: none` without provenance; `config test-connection` human logs configured tenant, exact warning, then unfiltered scope; `config test-connection --json` keeps private provenance out of stdout/stderr while serializing effective unfiltered tenant context.
- `cmd/get_processinstance_test.go` now covers active all-tenants request-shape behavior across Camunda 8.7 (`/v1/process-instances/search`) and 8.8/8.9/8.10 (`/v2/process-instances/search`): configured tenant is omitted from search filters while response tenant metadata remains backend-provided. It also covers quiet, total-only, and keys-only protected output keeping `--all-tenants` provenance text out of stdout/stderr.
- `cmd/get_test.go` now covers get-family active all-tenants behavior for `get process-definition --latest --json` and direct `get resource --id --keys-only`: process-definition latest search omits `tenantId`, and direct reads stay on the backend-authorized resource endpoint without warning leakage.
- `cmd/processinstance_mutation_progress_test.go` and `cmd/ops_progress_test.go` now cover active all-tenants durable progress: configured tenant line, exact warning once, then effective unfiltered scope; JSON, quiet, keys-only, and ops automation protected channels suppress private all-tenants provenance.
- `cmd/root.go:validateAllTenantsSelection` now runs after help bypass and before `initViper`, so active `--all-tenants` plus any changed `--tenant` (including explicit empty) fails before config loading, service installation, or command work. It uses the parsed root BoolVar plus `cmd.Flags().Lookup("tenant").Changed`, returns `mutuallyExclusiveFlagsf("--tenant cannot be combined with --all-tenants")`, and is wrapped with `silenceUsageForError` by the root pre-run.
- `cmd/root_test.go` covers named/empty explicit-tenant conflicts, explicit-false/absent/configured-source acceptance, and in-process reset clearing prior all-tenants state. `cmd/bootstrap_errors_test.go` covers the real `Execute()` subprocess path: invalid input, exit 2, silenced usage, and conflict precedence over missing config. `cmd/root_config_test.go` covers inactive all-tenants preserving explicit `--tenant` precedence over environment/base config.
- Command machine-contract metadata is annotation-backed in `cmd/command_contract.go`; all-tenants support now has `AllTenantsSupportAccepted`, `AllTenantsSupportRejectedConcreteDestination`, `setAllTenantsSupport`, and `allTenantsSupportForCommand`, defaulting unannotated/nil commands to `accepted`. `commandCapabilityForCommand` is the later serialization point. Human summary rendering is in `cmd/capabilities.go`.
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
- `go test ./cmd -count=1` currently fails at `TestGetElementHelp_DocumentsSearchAndOutputModes` because the inherited root flag added in Phase 2 contains `--all`; leave this for US4 help/discoverability work rather than editing help expectations during US1.

## Reusable Commands
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks`
- `go test ./cmd -run 'Test.*(AllTenants|TenantOverride|TenantContext)' -count=1`
- `go test ./cmd -run 'Test(CommandCapabilityForCommand_DocumentsTenantContract|CommandCapabilityForCommand_IncludesInheritedAndRequiredFlags|CapabilitiesCommand_JSONOutput|RootHelp_PreservesHumanTaxonomyAndDiscoveryCommand|ProcessInstanceHelp_ExposesCompactGlobalFlags|TenantContext)' -count=1`
- `go test ./cmd -run 'Test(CommandCapabilityForCommand_DocumentsTenantContract|CommandCapabilityForCommand_IncludesInheritedAndRequiredFlags|CapabilitiesCommand_JSONOutput|RootHelp_PreservesHumanTaxonomyAndDiscoveryCommand|ProcessInstanceHelp_ExposesCompactGlobalFlags|AllTenantsRootFlag|AllTenantsSupportForCommand)' -count=1`
- `git diff --check`
- `make docs-content`

## Do Not Repeat
- Do not bind `--all-tenants` through Viper, config files, profiles, or environment variables.
- Do not map the flag to `WithIgnoreTenant`, enumerate tenants client-side, or change backend authorization behavior.
- Do not implement concrete-destination rejection inside the four command runners after they have already initialized clients, inspected inputs, or built reports.

## Current Handoff
- Next iteration starts Phase 5 / US3 at task T025: add deploy rejection tests in `cmd/deploy_test.go`; stay within US3 and prove concrete-destination rejection happens before file, stdin, prompt, activity, report, dry-run plan, or remote work.
