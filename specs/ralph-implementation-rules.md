# Ralph Implementation Rules

This file is the repository-specific implementation context for Ralph-driven work in `c8volt`.

Pass it to reusable skills or launchers as:

```text
--implementation-context specs/ralph-implementation-rules.md
```

The purpose of this file is to reduce repeated project lookup cost. Read it before implementation, then inspect only the nearby code and tests needed for the current work unit.

## Mandatory Scope

- Treat these rules as binding for every Ralph implementation iteration that receives this file.
- Apply these rules in addition to `AGENTS.md`, the active feature's `spec.md`, `plan.md`, `tasks.md`, `research.md`, `data-model.md`, and any contracts.
- If a feature artifact conflicts with these rules, stop and surface the conflict before implementing.
- Complete only the current Ralph work unit. Do not start another user story or polish section in the same iteration.
- Preserve externally observable behavior unless the current feature explicitly requires a behavior change.
- Do not stage or commit unless the Ralph workflow explicitly reaches its commit step and validation passes.

## Project Map

Use this map before searching from scratch.

| Concern | Primary Location | Notes |
| --- | --- | --- |
| CLI commands and Cobra wiring | `cmd/*` | Command files define flags, validation, orchestration, command metadata, and output dispatch. |
| CLI output formatting | `cmd/cmd_views_*.go` | Human, JSON, keys-only, envelope, and flat row rendering live here. |
| CLI machine contract | `cmd/command_contract.go`, `cmd/capabilities.go`, `cmd/command_contract_test.go` | Commands must set mutation, contract support, automation support, output modes, and required flag metadata where relevant. |
| CLI bootstrap and config loading | `cmd/root.go`, `cmd/cmd_services.go`, `cmd/cmd_options.go` | Root installs config, logger, HTTP service, authenticator, and facade options into command execution. |
| Public facade API | `c8volt/*.go`, `c8volt/<area>/*.go` | Public types and methods used by commands and consumers. Facades convert to/from internal domain types. |
| Public facade options | `c8volt/foptions/options.go` | Public options map to internal `services.CallOption` values. Do not leak internal options into command code. |
| Public facade errors | `c8volt/ferrors/errors.go` | Converts domain/service/local errors into CLI classes and exit-code behavior. |
| Internal domain model | `internal/domain/*.go` | Shared internal domain types, sentinels, filters, pages, and sorting helpers. |
| Internal service interfaces | `internal/services/<area>/api.go` | Version-neutral service contracts. Facades should depend on these interfaces. |
| Internal service factories | `internal/services/<area>/factory.go` | Select v87, v88, or v89 implementations from `config.App.CamundaVersion`. |
| Versioned Camunda service implementations | `internal/services/<area>/v87`, `v88`, `v89` | Generated-client adapters and version-specific compatibility behavior. |
| Generated Camunda clients | `internal/clients/camunda/v87`, `v88`, `v89` | Generated code. Do not hand-edit unless explicitly required; prefer API regeneration flow. |
| Generated auth clients | `internal/clients/auth/oauth2` | Generated OAuth client code. |
| HTTP service, auth transport, request logging | `internal/services/httpc`, `internal/services/auth` | Root command installs these services into context. |
| Shared service helpers | `internal/services/common` | Payload validation, tenant handling, filter helpers, deployment helpers, defaults, and common service dependency preparation. |
| Shared implementation helpers | `toolx`, `toolx/pool`, `toolx/poller`, `toolx/logging` | Prefer these before creating new utility code. |
| Shared typed aliases | `typex` | `typex.Keys` provides unique key handling and string rendering. |
| Test helpers | `testx`, `testx/activitysink` | Fake servers, command config generation, subprocess command tests, IPv4 httptest servers, safe concurrent collectors. |
| Generated docs | `docs/cli/*`, `docs/index.md` | Generated through `make docs-content`. Do not hand-edit generated CLI docs when command metadata changed. |
| Docs generator | `docsgen/main.go` | Uses Cobra command tree to regenerate CLI markdown. |
| API client generation | `api/*`, `api/mutations/*` | Fetch, mutate, and regenerate OpenAPI clients. See `api/README.md`. |
| Embedded BPMN fixtures | `embedded/processdefinitions/*` | Version-prefixed fixtures use `C87_`, `C88_`, and `C89_`. |
| Feature artifacts | `specs/<feature>/` | Speckit specs, plans, tasks, progress, research, data-models, and quickstarts. |

## Layering Rules

Follow the existing direction of dependencies.

1. `cmd` is the CLI layer.
   - Owns Cobra commands, flags, argument validation, user-facing command help, command contract annotations, and rendering.
   - Calls the public facade returned by `NewCli` or helper construction.
   - Must not call generated Camunda clients directly.
   - Must not bypass facade APIs to reach versioned service packages.
   - Should translate CLI flags into facade inputs and `c8volt/foptions` only.

2. `c8volt/<area>` is the public facade layer.
   - Owns exported public models, facade interfaces, and method behavior used by commands and external consumers.
   - Converts public models to internal domain models in `convert.go` files.
   - Converts service/domain errors through `c8volt/ferrors.FromDomain`.
   - Receives public `foptions.FacadeOption` values and maps them to `internal/services.CallOption`.
   - May orchestrate cross-service behavior when the behavior belongs to the user-facing concept, for example process facade resolving incidents through the incident service.
   - Must not expose generated client types, internal service config structs, or internal domain sentinels in public API signatures.

3. `internal/domain` is the version-neutral internal model layer.
   - Owns service-facing models, filters, page metadata, response summaries, and domain error sentinels.
   - Keep domain types free of CLI rendering concerns.
   - Add domain fields here before wiring them through services and facades.

4. `internal/services/<area>` is the version-neutral service contract layer.
   - Owns interfaces consumed by facades and factories.
   - Factories select versioned implementations using `toolx.V87`, `toolx.V88`, and `toolx.V89`.
   - Shared service-level helpers belong in `internal/services/common` or the area package, not in `cmd`.

5. `internal/services/<area>/v87`, `v88`, `v89` are generated-client adapter layers.
   - Own Camunda version-specific request bodies, generated type conversions, endpoint differences, and compatibility workarounds.
   - Use generated client methods such as `...WithResponse`.
   - Convert generated types into `internal/domain` values before returning.
   - Keep version-specific behavior explicit and tested in the matching version package.

6. `toolx` and `testx` are shared support packages.
   - `toolx` is production helper code.
   - `testx` is test-only support.
   - Prefer existing helpers before adding new ones.

## Dependency Boundaries

- `cmd` may import `c8volt/*`, `config`, `consts`, `internal/services/httpc` for bootstrap/context, and test helpers in tests.
- `cmd` should not import `internal/clients/camunda/*` or versioned service implementations.
- Public facade packages may import `internal/domain`, `internal/services/<area>`, `c8volt/ferrors`, `c8volt/foptions`, `toolx`, `toolx/pool`, and `toolx/logging`.
- Internal services may import generated clients and `internal/domain`.
- Generated clients must not import facade or command packages.
- `internal/services/common` can hold cross-version service helpers, but it should not become a dumping ground for facade or CLI behavior.
- If a new dependency feels convenient but crosses these boundaries, first look for the existing path in nearby packages.

## Where New Behavior Usually Belongs

- New CLI command: add a `cmd/<verb>_<noun>.go` file or extend the existing command file under the relevant verb (`get`, `run`, `update`, `resolve`, `delete`, `cancel`, `walk`, `expect`, `config`, `embed`, `deploy`).
- New CLI output row or JSON view: add or extend `cmd/cmd_views_<area>.go`.
- New command contract metadata: update command init code and tests in `cmd/command_contract_test.go`.
- New public operation: add to `c8volt/<area>/api.go`, implement in `c8volt/<area>/client.go`, convert in `c8volt/<area>/convert.go`, and test in `c8volt/<area>/client_test.go`.
- New internal service capability: add to `internal/services/<area>/api.go`, implement in each supported `v87`, `v88`, and `v89` package or return an explicit unsupported domain error for unsupported versions.
- New generated-client mapping: put generated-to-domain conversion in the versioned service's `convert.go`.
- New shared service helper: use `internal/services/common` only when at least two service areas or versions need it.
- New public helper: use `toolx` only when it is general, production-safe, and not tied to one feature.
- New test-only helper: use `testx` only when at least two test files can share it.
- New generated CLI documentation: update command source and run `make docs-content`; do not hand-edit `docs/cli/*`.

## File Organization And Naming Rules

- Do not contaminate Cobra command files with unrelated helper functions.
- A command file should primarily contain command definition, flags, validation directly tied to that command, and the orchestration needed to call the facade and render output.
- Before adding a helper to a command file, first look for an existing owning file or package:
  - CLI rendering helpers belong in `cmd/cmd_views_<area>.go`.
  - Command contract helpers belong near `cmd/command_contract.go`.
  - Stdin/key parsing belongs near `cmd/cmd_stdin.go`.
  - CLI construction and confirmation behavior belongs near `cmd/cmd_cli.go`.
  - CLI error construction belongs near `cmd/cmd_errors.go`.
  - Facade behavior belongs in `c8volt/<area>/`.
  - Service behavior belongs in `internal/services/<area>/` or `internal/services/common`.
  - General reusable production helpers belong in `toolx`.
  - Test-only reusable helpers belong in `testx`.
- If no existing file or package owns the helper, create a new focused file with a meaningful domain name.
- Do not create files or functions with generic names such as `helper`, `helpers`, `util`, `utils`, `common`, `misc`, or similar vague labels.
- File names must describe the concrete concern, for example `cmd/get_incident_search.go`, `cmd/cmd_views_processinstance_incidents.go`, `internal/services/incidentfilter/incidentfilter.go`, or `toolx/filter_format.go`.
- Function names must also describe the concrete behavior or rule they implement. Avoid names that only describe that the function is a helper.
- If a new file would be named with a generic word, the concern is probably not clear enough yet; inspect nearby patterns and choose a more cohesive ownership boundary.

## Camunda Version And API Rules

- The repository supports Camunda 8.7, 8.8, and 8.9 through `toolx.V87`, `toolx.V88`, and `toolx.V89`.
- The newest supported runtime path is 8.9. Prefer extending v8.9 first for new Camunda v2 API behavior.
- `toolx.CurrentCamundaVersion` is currently `V88`; do not change the default runtime unless the feature explicitly asks for it.
- Camunda v2 API is the preferred path for new behavior. The normalized Camunda base URL version is managed in `config/api.go` with `CamundaApiVersionConst = "v2"`.
- Do not introduce Operate v1 or Tasklist v1 workarounds when a generated v2 Camunda client supports the operation.
- Legacy component APIs still exist in config (`operate_api` and `tasklist_api`, both v1). Use them only when the existing service area already uses them or the feature explicitly requires that path.
- Every service factory under `internal/services/<area>/factory.go` must keep explicit cases for supported versions and return `services.ErrUnknownAPIVersion` for unsupported versions.
- When adding version-specific behavior:
  - Check the v89 package first.
  - Compare the v88 package for compatibility fallbacks.
  - Check v87 before assuming a method exists.
  - Add tests to the version package that owns the compatibility behavior.
- Existing v8.8 incident search intentionally uses compatibility-oriented local filtering where richer request filters may be rejected by clusters. v8.9 uses richer server-side filters where available. Preserve this kind of version-specific behavior instead of flattening versions together.
- Use API regeneration scripts in `api/` for generated clients. The main entry point is `bash api/refresh-clients.sh`; see `api/README.md`.

## Existing Helpers To Reuse First

Before adding helper code, search these locations.

### General Helpers In `toolx`

- `toolx.MapSlice`: map slices without hand-written loops.
- `toolx.MapMap`: map maps when converting domain/facade collections.
- `toolx.MapPtr`, `toolx.CopyPtr`, `toolx.Deref`, `toolx.DerefMap`, `toolx.DerefSlice`, `toolx.DerefSlicePtr`, `toolx.DerefSlicePtrE`: pointer and optional value conversions.
- `toolx.CopyMap`: copy maps when moving variables across domain/facade boundaries.
- `toolx.MapNullable`, `toolx.MapNullableV`, `toolx.MapNullableSlice`, `toolx.MapNullableSliceV`: `oapi-codegen/nullable` conversions.
- `toolx.Int64PtrToStringPtr`, `toolx.Int64PtrToString`, `toolx.StringPtrToInt64`, `toolx.StringToInt64`, `toolx.StringToInt64Ptr`: generated numeric key conversion helpers.
- `toolx.UniqueSlice`: stable deduplication with order preserved.
- `toolx.DetermineNoOfWorkers`: worker count policy using `GOMAXPROCS` unless `NoWorkerLimit` is set.
- `toolx.FormatActiveFields`, `AppendQuotedField`, `AppendInt32Field`, `AppendBoolPtrField`, `AppendTrueBoolField`, `AppendRawField`: stable filter debug string formatting.
- `toolx.JSON` and `toolx.ToJSONString`: pretty JSON with HTML escaping disabled.
- `toolx.NewDurationStringValue`: Cobra/pflag-compatible duration validation.
- `toolx.NormalizeCamundaVersion`, `SupportedCamundaVersionsString`, `ImplementedCamundaVersionsString`: version parsing and user-facing version lists.

### Concurrency Helpers

- `toolx/pool.ExecuteNTimes`: bounded concurrent execution by index, context-aware, ordered results, joined errors, optional fail-fast cancellation.
- `toolx/pool.ExecuteSlice`: bounded concurrent map over inputs.
- `toolx/pool.Reports` and `Reporter`: reusable report totals where report types implement `OK()`.
- `internal/services/common.RunBulk`: generic ordered bulk runner used by internal services.
- `testx.SafeSlice`: synchronized collection in httptest handlers that can be called concurrently.
- Prefer these helpers over custom goroutine/channel/WaitGroup code.

### Logging And Activity Helpers

- `toolx/logging.InfoIfVerbose`: verbose-only logging.
- `toolx/logging.StartActivity`: transient activity indicator for longer facade operations.
- `toolx/logging.NewActivityWriterEnabled`, `ToActivityContext`, and `ActivityFromContext`: root command activity plumbing.
- `internal/services/httpc.LogTransport`: HTTP request logging and activity integration.
- Keep human progress messages in command/facade layers; keep service logs diagnostic.

### Service Helpers

- `internal/services/common.PrepareServiceDeps`: normalize config, HTTP client, and logger dependencies for services.
- `internal/services/common.EnsureLoggerAndClients`: validate service dependencies.
- `internal/services/common.RequirePayload`: apply HTTP status handling and ensure non-empty generated response payloads.
- `internal/services/common.RequireSingleProcessInstance`: enforce search-backed single lookup semantics.
- `internal/services/common.ProcessInstanceNotFound`: consistent process-instance not-found error.
- `internal/services/common.EffectiveTenant` and tenant helpers: use existing tenant semantics rather than inventing new defaults.
- `internal/services/common.NewStringEqFilterPtr`, `NewIntegerEqFilterPtr`, `NewProcessInstanceKeyEqFilterPtr`, `NewProcessDefinitionKeyEqFilterPtr`, `NewScopeKeyEqFilterPtr`, `NewProcessInstanceStateEqFilterPtr`, `NewDateTimeRangeFilterPtr`: reusable v2 filter constructors where type-compatible.
- `internal/services/incidentfilter`: incident error type normalization and error-message matching.

### Test Helpers

- `testx.WriteTestConfig` and `WriteTestConfigForVersion`: create temporary config files for command tests.
- `testx.NewIPv4Server` and `NewIPv4TLSServer`: use when command tests need stable IPv4 loopback URLs.
- `testx.RunCmdSubprocess`, `RunCmdSubprocessWithStdin`, and directory variants: test `Execute()` and process exit codes.
- `testx.NewFakeServer`: reusable fake Camunda server for broad integration-style tests.
- `testx/activitysink`: capture activity output where relevant.

## Go Concurrency Rules

- Prefer `toolx/pool.ExecuteSlice` or `ExecuteNTimes` for facade bulk operations.
- Always pass and respect `context.Context`.
- Use `options.ApplyFacadeOptions` or `services.ApplyCallOptions` to honor `FailFast`, `NoWorkerLimit`, `NoWait`, `DryRun`, `Verbose`, and related flags.
- Deduplicate key lists before bulk work with `typex.Keys.Unique()` or `toolx.UniqueSlice`.
- Keep output order deterministic; existing pool helpers preserve input order.
- For fail-fast behavior, stop scheduling new work while preserving already produced results and aggregated errors.
- Do not create goroutines that can outlive the caller's context.
- Do not share mutable slices or maps across worker goroutines without synchronization.
- In tests for concurrent handlers, use `testx.SafeSlice` or a local mutex.
- Avoid adding worker pools in `cmd`; command code should pass worker flags into facade methods.

## Error Handling Rules

- Service and domain code should return domain errors from `internal/domain/errors.go` or transport errors normalized by `internal/services/httpc`.
- Public facades should convert service/domain errors with `ferrors.FromDomain`.
- CLI code should use `handleCommandError`, `handleNewCliError`, `localPreconditionError`, `invalidFlagValuef`, `mutuallyExclusiveFlagsf`, and related command helpers rather than printing and exiting directly.
- Machine-readable error output must flow through `renderResultEnvelope` and `resultEnvelopeForError` when the command supports the shared contract.
- Preserve wrapped error detail exactly once. Do not build nested duplicate messages.
- For unsupported version or missing API support, use explicit unsupported errors rather than silent fallbacks.
- If a partial result is meaningful, return the partial report plus an error, following existing facade patterns for resolution, variable updates, and bulk operations.

## CLI Command Rules

- Commands should validate local inputs before creating or calling remote clients when possible.
- Use Cobra `Args` for argument shape and flag value validation that should fail before execution.
- Use `silenceUsageForError` for validation failures where usage should not be reprinted.
- Use `useInvalidInputFlagErrors` on commands with custom flag validation.
- For read-only commands, set `setCommandMutation(cmd, CommandMutationReadOnly)`.
- For state-changing commands, set `CommandMutationStateChanging`.
- If the command supports stable machine output, set `setContractSupport(cmd, ContractSupportFull)` and render JSON through the shared envelope path.
- If the command supports unattended operation, call `setAutomationSupport` with a concrete note.
- Mark required flags for discovery with `setFlagContractRequired`; command validation still owns enforcement.
- Keep aliases short and compatible with existing command naming patterns (`pi`, `pd`, `inc`, etc.).
- For stdin key pipelines, reuse existing dash helpers such as `validateOptionalDashArg`, `readKeysIfDash`, `mergeAndValidateKeys`, and `validateKeys`.
- Do not add prompts to automation-supported paths unless `--automation` and `--auto-confirm` behavior is explicitly handled.
- Keep command help and flag descriptions directly useful. Avoid filler phrases such as "human output", "human incident output", "human incident messages", "for terminal diagnosis", or similar mode labels when they do not add actionable meaning. Describe the actual behavior instead, for example "omit error messages from incident rows" or "maximum characters to show for incident messages".

## Output And Rendering Rules

- Human output should be compact, scan-friendly, and use existing flat row helpers.
- JSON output should use stable structs and shared envelopes when the command has full contract support.
- Keys-only output must print one key per line and nothing else.
- `--total` style output should print only the number and a newline.
- Do not let human-only formatting options affect JSON or keys-only output.
- Use `toolx.ToJSONString` for command JSON payloads.
- Use `renderJSONPayload`, `renderCommandResult`, `renderSucceededResult`, or `renderAcceptedResult` rather than hand-encoding JSON in commands.
- Existing flat row helpers:
  - Process instance rows: `cmd/cmd_views_get.go`
  - Incident rows: `cmd/cmd_views_processinstance_incidents.go`
  - Resolve views: `cmd/cmd_views_resolve.go`
  - Job views: `cmd/cmd_views_job.go`
  - Cluster views: `cmd/cmd_views_cluster.go`
- When changing command output, update or add command tests for one-line, JSON, keys-only, and error modes as applicable.

## Public Facade Rules

- Public interfaces live in `c8volt/<area>/api.go`.
- Public models live in `c8volt/<area>/model.go`.
- Public behavior normally lives in `c8volt/<area>/client.go`, with bulk behavior often split into `bulk.go` and specialized behavior into focused files such as `resolve.go`, `dryrun.go`, or `walker.go`.
- Keep public types stable and JSON tags intentional.
- Convert all internal domain values at the facade boundary using `convert.go`.
- Copy maps and slices when crossing boundaries if mutation by callers could leak internal state.
- Use `foptions.FacadeOption` in public signatures and convert to `services.CallOption` internally.
- Public facade methods should not expose generated client types or `internal/domain` types.
- For bulk facade operations, deduplicate keys and determine worker counts before scheduling work.

## Internal Service Rules

- Service interfaces live in `internal/services/<area>/api.go`.
- Factories live in `internal/services/<area>/factory.go` and choose versions by `cfg.App.CamundaVersion`.
- Versioned service constructors should call shared dependency preparation helpers and validate generated clients.
- Versioned service methods should use generated `WithResponse` methods and `common.RequirePayload` for JSON payloads.
- Convert generated response objects to `internal/domain` types before returning.
- Keep version-specific filter shape, pagination, response total semantics, and compatibility behavior in the version package.
- If v87 lacks a v2 endpoint that v88/v89 have, return a clear unsupported domain error from v87. Do not fake success.
- Avoid introducing new service interfaces until an existing area cannot own the operation.

## Domain And Conversion Rules

- Add shared internal fields to `internal/domain` before adding facade fields.
- Add facade fields to public models only when they are part of the user-facing API or command output contract.
- Keep converter functions mechanical and close to the models they convert.
- Use `toolx.MapSlice`, `MapPtr`, `CopyMap`, and related helpers in converters.
- Do not put business logic in generated-client converters except for unavoidable version-specific representation normalization.
- For filters, implement `String()` with `toolx.Append*Field` helpers so debug logging stays consistent.

## Testing Rules

- Add tests beside the changed package.
- For CLI behavior, prefer command tests in `cmd/*_test.go`.
- For command exit codes or `os.Exit` paths, use `testx.RunCmdSubprocess`.
- For command tests needing HTTP, use `httptest` or `testx.NewIPv4Server`.
- For facade behavior, add tests in `c8volt/<area>/client_test.go` using stub service interfaces.
- For versioned service behavior, add tests under the matching version package, for example `internal/services/incident/v89/*_test.go`.
- For shared helpers, add tests in the helper package (`toolx`, `toolx/pool`, `internal/services/common`, `testx` where applicable).
- Use `require` from `testify` consistently with existing tests.
- Use `t.Parallel()` when the test has no shared global state, no package-level flag mutation, no command tree mutation, and no shared environment mutation.
- Do not use `t.Parallel()` for tests that mutate package-level command flags, Cobra command tree state, global env vars, or shared fake servers unless the existing helper guarantees isolation.
- Include tests for invalid flags, mutually exclusive flags, JSON output, keys-only output, human output, pagination/limit behavior, version behavior, and error classification when relevant.
- Every created or materially modified test function must have a concise comment explaining the behavior, regression, or contract being verified.
- Complex test setup should include comments explaining why the setup matters.

## Validation Rules

- Run the narrowest relevant tests first.
- For command-only changes, prefer targeted `go test ./cmd -run '<TestNameOrPattern>' -count=1`.
- For facade changes, prefer `go test ./c8volt/<area> -run '<TestNameOrPattern>' -count=1`.
- For internal service changes, prefer `go test ./internal/services/<area>/... -run '<TestNameOrPattern>' -count=1`.
- For shared helper changes, run that helper package and at least one consumer package.
- Before committing a completed Ralph work unit, run broader validation appropriate to the blast radius.
- Repository full test target is `make test`, which runs `go test ./... -race -count=1`.
- Formatting is `go fmt ./...` or targeted `gofmt -w <files>`.
- Vet target is `make vet`.
- Generated docs refresh is `make docs-content`.
- Full local quality pipeline is `make all`, but prefer targeted checks first during Ralph iterations.
- Do not mark tasks complete while targeted validation is failing.

## Comment Rules

- Every created or materially modified exported function, type, method, constant, or variable must have a Go doc comment.
- Every created or materially modified non-exported function must have a comment explaining purpose, invariant, edge case, or reason for existence.
- Every created or materially modified test function must have a comment describing the behavior, regression, or contract being verified.
- Comments must explain intent, behavior, constraints, or edge cases. Do not merely repeat the function name.
- Existing nearby code may not fully satisfy this stricter rule; apply it to new or touched code without doing broad comment-only churn.
- Prefer short comments for obvious glue and more complete comments for concurrency, version compatibility, API fallback, pagination, error classification, and test fixtures.

## Generated And Generated-Adjacent Artifacts

- Do not hand-edit generated Camunda clients under `internal/clients/camunda/*` unless the task explicitly requires a surgical generated-code patch.
- Prefer updating OpenAPI mutation scripts under `api/mutations/*` and regenerating clients.
- Use `api/refresh-clients.sh` for the full client refresh flow.
- CLI docs under `docs/cli/*` and `docs/index.md` are generated by `docsgen`; update command metadata/help text and run `make docs-content`.
- Speckit artifacts under `.specify` are ignored in this repo; active feature artifacts under `specs/<feature>/` are the persistent source of planning context.
- `scripts/ralph` is ignored runtime state. Do not put persistent rules there.

## Documentation Rules

- If command behavior, flags, aliases, examples, or output contracts change, update command help text and regenerate docs with `make docs-content`.
- Keep README-facing behavior and docs-generated behavior aligned.
- Do not update generated CLI docs by hand when the command source can generate them.
- If changing API client generation, update `api/README.md` only when the workflow itself changes.

## Ralph Iteration Discipline

- Start by reading the active feature's `tasks.md`, `plan.md`, `spec.md`, and `progress.md`.
- If present, also read `research.md`, `data-model.md`, `quickstart.md`, and `contracts/`.
- Identify the first incomplete work unit and implement only that unit.
- Before adding code, inspect nearby implementation and tests in the package that should own the behavior.
- Before adding helpers, search `toolx`, `internal/services/common`, the current area package, and `testx`.
- Record reusable discoveries in the feature `progress.md` under codebase patterns.
- Mark tasks complete only after implementation and relevant validation pass.
- When the work unit is complete, include task/progress updates in the same work-unit commit.
- Use Conventional Commits and append the issue suffix when feature artifacts persist an issue number.

## Quick Search Checklist

Use these searches before introducing new structures:

```sh
rg -n "func .*<name>|type .*<name>|<concept>" cmd c8volt internal toolx testx
rg -n "With<Flag>|ApplyFacadeOptions|MapFacadeOptionsToCallOptions" c8volt internal cmd
rg -n "Search.*Page|OverflowState|ReportedTotal|EndCursor" c8volt internal cmd
rg -n "ExecuteSlice|ExecuteNTimes|DetermineNoOfWorkers|FailFast|NoWorkerLimit" .
rg -n "Incident|ProcessInstance|ProcessDefinition|Tenant|Job|Resource" c8volt internal/services cmd
rg -n "setCommandMutation|setContractSupport|setAutomationSupport|setOutputModes|setFlagContractRequired" cmd
```

Prefer `rg` over broad manual browsing, then read the nearest implementation and test files fully enough to follow local patterns.
