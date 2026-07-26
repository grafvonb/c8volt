# Architecture Repo Facts: c8volt

**Purpose**: Record observable repository facts that support reverse generation of the project-level 4+1 architecture artifacts.

**Note**: This evidence file is filled in by the `speckit-arch-reverse` workflow before the architecture views are updated. It is a fact source for architecture reasoning, not an implementation audit report.

## Repository Identity

| Fact | Evidence Source | Confidence | Architecture Relevance |
|------|-----------------|------------|--------------------------|
| c8volt is a Camunda 8 command-line tool for operators, developers, support engineers, CI pipelines, and agents. | `README.md` project title, "Why c8volt", and "At A Glance" sections | High | Establishes the primary actor set and CLI-as-boundary architecture. |
| The project is implemented as a Go module named `github.com/grafvonb/c8volt`. | `go.mod`; `main.go` | High | Establishes a single compiled binary as the main runtime unit. |
| The CLI targets Camunda 8.7, 8.8, and 8.9, with version-dependent behavior called out for several operations. | `README.md` "Supported Camunda Versions"; `config/templates/config.example.yaml`; `cmd/root.go` help text | High | Establishes version compatibility as a central architectural force. |
| The license and project governance are GPL-3.0-or-later with separate security, trademark, and contribution documents. | `LICENSE`; `SECURITY.md`; `TRADEMARKS.md`; `README.md` "Project Governance" | Medium | Establishes release and governance context but not runtime behavior. |
| The repo contains Spec Kit feature history and task PRDs for many CLI capabilities. | `specs/`; `tasks/` | Medium | Signals that changes are feature-driven and should preserve command-level contracts. |

## Entry Points

| Entry Point | Type | Evidence Source | Observed Responsibility | Supported Scenario |
|-------------|------|-----------------|--------------------------|--------------------|
| `c8volt` binary | CLI process | `main.go`; `cmd/root.go` | Starts the command tree, loads configuration, installs HTTP/auth services, and dispatches commands. | All user-driven CLI scenarios. |
| Command groups for `get`, `run`, `expect`, `walk`, `ops`, `deploy`, `delete`, `cancel`, `config`, `embed`, `capabilities`, `version`, `completion`, `update`, and `resolve` | User-facing CLI commands | `README.md` "Command Map"; `cmd/root_test.go`; `cmd/*` command files | Expose inspection, mutation, workflow, setup, and discovery operations. | Inspect state, mutate resources, automate workflows, and discover capabilities. |
| `capabilities --json` | Machine discovery command | `README.md` "Automation And Pipelines"; `cmd/capabilities.go`; `cmd/command_contract_test.go` | Emits command contract metadata for scripts, CI, and agents. | Machine-safe discovery before unattended execution. |
| `config` subcommands | Setup and validation gate | `README.md` "Configuration Notes"; `cmd/config*.go`; `config/templates/config.example.yaml` | Validates, shows, templates, and tests effective configuration before remote operations. | Configure and verify Camunda connectivity. |
| `ops` playbooks | High-level operational workflows | `README.md` "New in v4"; `docs/ops/index.md`; `cmd/ops*.go`; `internal/services/ops/*_test.go` | Compose lower-level commands into previewable, auditable operational outcomes. | Execute smoke tests, retention cleanup, purge, and repair workflows. |
| Embedded BPMN fixture commands | Built-in sample/process asset workflow | `README.md` "Fast Start"; `embedded/processdefinitions/`; `cmd/embed*.go` | Lists, deploys, and exports bundled process definitions used for smoke and demo flows. | Fast-start and operational smoke workflows. |

## User-Visible Behaviors

| Behavior | Evidence Source | Actor / Trigger | Observable Outcome | Supported Use Case |
|----------|-----------------|-----------------|--------------------|--------------------|
| Deploy BPMN and start process instances. | `README.md` "Fast Start", "Core Workflows", and "Everyday Commands"; `docs/cli/c8volt_deploy_process-definition.md`; `cmd/deploy_processdefinition.go`; `cmd/run_processinstance.go` | Operator or pipeline invokes deploy/run commands. | Resource is deployed, a process instance is started, and activation is confirmed by default for run. | Start and confirm workflow execution. |
| Inspect process definitions, process instances, incidents, jobs, tenants, cluster metadata, and resources. | `README.md` "Command Map"; `docs/cli/c8volt_get*.md`; `cmd/get*.go` | User invokes `get` commands. | Human-readable, JSON, keys-only, totals, or detail output depending on flags. | Diagnose workflow state and discover targets. |
| Walk process-instance relationships before destructive actions. | `README.md` "Walk Before You Change"; `cmd/walk_processinstance.go`; `cmd/walk_test.go` | User invokes `walk pi`. | Process family structure and optional incidents/variables are displayed. | Determine safe cancellation or deletion scope. |
| Cancel and delete process instances with dry-run and force controls. | `README.md` "Cancel Safely" and "Delete Thoroughly"; `cmd/cancel_processinstance.go`; `cmd/delete_processinstance.go`; `cmd/cancel_test.go`; `cmd/delete_test.go` | Operator selects instances by key/search/stdin. | Preview, confirmation, execution, and outcome verification paths are available. | Safe destructive workflow cleanup. |
| Wait for process instance state or incident conditions. | `README.md` "Wait For A Known State Or Incident"; `cmd/expect_processinstance.go`; `cmd/expect_test.go` | Script or user invokes `expect pi`. | Command waits for selected state/incident conditions and exits based on result. | Automation-visible readiness or closure checks. |
| Update process-instance variables and jobs with dry-run/confirmation semantics. | `README.md` "Update Runtime Variables" and "Inspect And Update Jobs"; `cmd/update_processinstance.go`; `cmd/update_job.go`; related tests | User selects keys and provides variable or job update input. | Planned change can be previewed and submitted; supported versions are enforced. | Runtime remediation without guessing current state. |
| Resolve incidents directly or by process-instance discovery. | `README.md` "Resolve Incidents"; `cmd/resolve_incident.go`; `cmd/resolve_processinstance.go`; related tests | Operator selects incident or process-instance keys. | Active incident set is resolved with preview and confirmation options. | Incident closure workflow. |
| Run ops playbooks with discover-freeze-plan-validate-execute-verify-report shape. | `docs/ops/index.md`; `README.md` "New in v4"; `cmd/ops_*.go`; `internal/services/ops/*_test.go` | Operator or automation invokes `ops` commands. | Dry-run plan or real execution with audit-oriented reporting. | Repeatable high-level operations. |
| Use JSON, keys-only, automation, auto-confirm, quiet, verbose, profile, config, and tenant controls. | `README.md` "Automation And Pipelines" and "Configuration Notes"; `cmd/root.go`; `cmd/root_test.go`; `cmd/command_contract_test.go` | Scripts, CI, or operators pass global flags. | Output and prompting behavior adapt to the execution context. | Script-safe execution and environment switching. |

## System Boundaries

| Boundary | Evidence Source | Inbound Interaction | Outbound Interaction | Not Proven |
|----------|-----------------|---------------------|----------------------|------------|
| CLI process boundary | `main.go`; `cmd/root.go`; `README.md` | Shell invocation, stdin keys, flags, config, environment variables | Stdout/stderr output, exit status, remote HTTP requests | No daemon, server, or resident worker runtime is shown. |
| Camunda external API boundary | `config/templates/config.example.yaml`; `internal/clients/camunda/v87`, `v88`, `v89`; `internal/services/*/factory.go`; `README.md` | Configured base URLs, tenant, auth mode, version | Camunda v2 and Operate/Tasklist-style HTTP calls | Complete upstream deployment topology is not represented in this repo. |
| Authentication boundary | `config/templates/config.example.yaml`; `internal/services/auth/factory.go`; `internal/services/httpc/service.go` | Auth mode and credentials from config/env/profile | Request editor transport for none, OAuth2, or cookie-backed auth | Identity provider behavior and token policies are external. |
| Public facade boundary | `c8volt/contract.go`; `c8volt/client.go`; `c8volt/*/api.go` | Command layer requests operations through public APIs | Internal services and generated clients | The facade is not documented as a public SDK compatibility guarantee. |
| Generated API-client boundary | `internal/clients/camunda/v86`; `internal/clients/camunda/v87`; `internal/clients/camunda/v88`; `internal/clients/camunda/v89`; `api/refresh-clients.sh`; `go.mod` tool declarations | Service factories choose clients based on config/version | HTTP request/response translation to upstream APIs | The exact upstream OpenAPI provenance is only partially visible from generated output and scripts; v86 client presence alone does not prove supported CLI behavior. |
| Documentation and release boundary | `docs/`; `docsgen/`; `Makefile`; `.github/workflows/docs.yml`; `.github/workflows/release.yaml`; `.goreleaser.yaml` | Generated command docs and release tags | Static site build, GitHub release archives, SFTP docs publication | Hosting runtime details beyond SFTP target secrets are not described. |

## Data and State Clues

| Fact / Entity | Evidence Source | Observed Lifecycle Clue | Fact Source | Not Proven |
|---------------|-----------------|-------------------------|-------------|------------|
| Configuration | `config/templates/config.example.yaml`; `README.md` "Configuration Notes"; `cmd/root.go`; config tests | Loaded by precedence, validated, and injected into command context before remote operations. | Local config files, environment variables, profiles, flags, defaults | No persistent in-application database is shown. |
| Process definition | `README.md` deploy/get/delete sections; `internal/domain/processdefinition.go`; `cmd/deploy_processdefinition.go`; `cmd/get_processdefinition.go`; `cmd/delete_processdefinition.go` | Deployed, inspected, counted/stat-ed, and deleted with safety warnings. | Camunda API | Complete BPMN semantics are owned by Camunda, not c8volt. |
| Process instance | `README.md` run/get/walk/cancel/delete/expect sections; `internal/domain/processinstance*.go`; `cmd/*processinstance*.go` | Started, inspected, enriched, walked, waited on, canceled, and deleted. | Camunda API | Long-term process state storage is external. |
| Incident | `README.md` incident sections; `internal/domain/incident.go`; `cmd/get_incident.go`; `cmd/resolve_incident.go`; `cmd/ops_repair*.go` | Listed, filtered, resolved, repaired, and used to select process instances. | Camunda API | Root cause classification beyond exposed incident facts is not proven. |
| Job | `README.md` "Inspect And Update Jobs"; `internal/domain/job.go`; `cmd/get_job.go`; `cmd/update_job.go` | Inspected and updated for retries or timeout where supported. | Camunda API | Worker execution internals are external. |
| Tenant and profile context | `README.md` "Tenant Handling"; `config/templates/config.example.yaml`; `cmd/get_tenant.go`; tenant tests | Used to scope read/write operations and inspect visible tenants. | Config and Camunda API | Organization-level tenant governance is external. |
| Ops report/audit output | `docs/ops/index.md`; `internal/domain/report.go`; `internal/services/ops/*_test.go`; `README.md` ops command examples | Produced by higher-level workflows after planning and execution. | c8volt command execution facts plus Camunda observations | Long-term report storage policy is not shown. |
| Embedded process assets | `embedded/processdefinitions/`; `processdefinitions/`; `README.md` fast-start examples | Used for deploy/export/list and smoke/demo flows. | Repository assets | They do not prove production process definitions. |

## Runtime and Process Clues

| Runtime Fact | Evidence Source | Trigger / Handoff | Failure or Retry Clue | Not Proven |
|--------------|-----------------|-------------------|-----------------------|------------|
| Root command bootstraps config, logging, HTTP, and auth before remote commands. | `cmd/root.go`; `internal/services/httpc/service.go`; `internal/services/auth/factory.go` | CLI invocation to command execution | Config validation errors and authenticator initialization errors are surfaced before command action. | No separate long-running supervisor is shown. |
| Remote operations share a configured HTTP client with timeout and auth transport. | `config/templates/config.example.yaml`; `internal/services/httpc/service.go`; `cmd/root.go` | Command facade to internal services | Timeout is configurable; HTTP logging transport exists. | Network retry policy details are not fully proven by this fact alone. |
| Many mutation flows support dry-run, confirmation, automation, and auto-confirm controls. | `README.md`; `cmd/cmd_cli.go`; `cmd/ops*.go`; mutation command tests | User preview or automated execution to mutation submission | Local precondition and abort errors are modeled; unsupported automation is rejected. | There is no distributed transaction boundary across Camunda operations. |
| Process-instance deletion refuses unsafe mixed final/non-final scope unless forced through cancellation. | `README.md` "Delete Thoroughly"; `cmd/delete_processinstance.go`; `cmd/delete_test.go` | Selection and dependency-expanded scope to delete planning | All-or-nothing refusal before submitting delete requests is documented. | Camunda-side eventual consistency timing is not fully specified. |
| Ops workflows follow discover, freeze, plan, validate, dry-run or confirm, execute, wait, verify, report. | `docs/ops/index.md`; `README.md` "New in v4"; `internal/services/ops/*_test.go` | High-level playbook to lower-level operations | Dry-run mutates nothing; real execution verifies and reports. | Exact recovery behavior after partial remote failures varies by playbook and is not fully summarized in one architecture source. |
| Version gates prevent unsupported state-changing operations for older Camunda versions. | `README.md` version notes; `cmd/update_job_test.go`; `cmd/update_processinstance_test.go`; `cmd/get_processdefinition.go` | Configured Camunda version to command execution | Unsupported-version errors occur before mutation for named operations. | The full compatibility matrix is spread across code and docs. |

## Development Structure Clues

| Module / Package Area | Evidence Source | Observed Responsibility | Dependency Clue | Boundary Risk |
|-----------------------|-----------------|--------------------------|-----------------|---------------|
| Command layer | `cmd/`; `cmd/root.go`; `cmd/*_test.go` | Defines command taxonomy, flags, rendering, bootstrapping, and user interaction. | Calls public facade and command rendering helpers. | Risk of duplicating domain/service behavior in command handlers. |
| Public operation facade | `c8volt/contract.go`; `c8volt/client.go`; `c8volt/*/api.go` | Presents grouped operations for commands while hiding internal service wiring. | Wraps internal services and domain conversions. | Risk of command layer bypassing facade and coupling directly to internals. |
| Internal domain models | `internal/domain/`; domain tests | Holds architecture-level operation concepts such as process state, incidents, reports, and ops plans. | Used by internal services and facade conversions. | Risk of generated-client shapes leaking into stable command/domain contracts. |
| Internal services | `internal/services/`; service tests | Coordinate versioned clients, filters, lookup, waits, discovery, repair, purge, and other remote operations. | Depend on config, HTTP, generated clients, and domain models. | Risk of service packages crossing capability boundaries without facade mediation. |
| Generated client adapters | `internal/clients/`; `internal/clients/camunda/v86`; `internal/clients/camunda/v87`; `internal/clients/camunda/v88`; `internal/clients/camunda/v89`; `api/refresh-clients.sh`; `go.mod` tool declarations | Provide version-specific Camunda and OAuth API clients. | Consumed by internal services; supported-version conclusions still require command/config/docs evidence. | Risk of version-specific generated types leaking upward or generated-client presence being mistaken for user-facing support. |
| Configuration | `config/`; config tests; `config/templates/config.example.yaml` | Owns config schema, defaults, validation, and precedence. | Used by root command and services. | Risk of command-local flags diverging from config-backed precedence. |
| Documentation generation | `docsgen/`; `Makefile` `docs-content`; `docs/cli/` | Regenerates CLI reference from command metadata. | Depends on command tree and build metadata. | Risk of user-facing docs drifting from command behavior. |
| Test support | `testx/`; `cmd/*_test.go`; `internal/services/*_test.go` | Provides fake servers, subprocess runners, integration env guards, and command fixtures. | Supports command, facade, and service tests. | Test-only helpers should not become runtime dependency points. |

## Repository-First Projection

Record repository-first evidence only when `.specify/memory/repository-first/` exists. Leave explicit gaps when the directory or expected artifacts are absent.

### Build Manifest Detection

| Ecosystem | Manifest Evidence | Detection Status | Runtime Surface Notes |
|-----------|-------------------|------------------|-----------------------|
| Go | `go.mod`; `go.sum`; `Makefile`; `.goreleaser.yaml` | Detected | Single CLI binary built from module root, with release archives for multiple OS/architecture targets. |
| Ruby/Jekyll docs | `docs/Gemfile`; `docs/_config.yml`; `.github/workflows/docs.yml` | Detected | Static documentation site build surface, not core CLI runtime. |
| Node auxiliary dependency | `package.json`; `package-lock.json`; `node_modules/yq` directory present | Detected auxiliary manifest | Root Node manifest declares `yq`; no source evidence shows it as a core CLI runtime dependency. |
| Repository-first artifacts | `.specify/memory/repository-first/` absent | Gap | No generated dependency matrix or module invocation spec was available for this pass. |

### First-Party Module Edges

| From Module | To Module | Evidence Source | Observed Direction | Architecture Boundary Meaning |
|-------------|-----------|-----------------|--------------------|-------------------------------|
| Command layer | Public operation facade | `cmd/cmd_cli.go`; `c8volt/client.go` | CLI commands create/use facade operations. | User interaction stays separate from remote operation implementation. |
| Command bootstrap | Configuration and transport/auth services | `cmd/root.go`; `cmd/bootstrap_errors.go`; `config/`; `internal/services/auth/factory.go`; `internal/services/httpc/service.go` | Root command setup installs effective config, auth, and HTTP transport before remote operations. | Bootstrap wiring is an allowed command-layer exception distinct from operation implementation. |
| Public operation facade | Internal services | `c8volt/client.go`; `internal/services/*/factory.go` | Facade wires capability services and presents grouped APIs. | Internal services remain hidden behind operation families. |
| Internal services | Generated clients | `internal/services/*/factory.go`; `internal/clients/camunda/v87`; `v88`; `v89` | Services select versioned generated clients. | Version-specific upstream API differences are isolated below services. |
| Internal services | Domain models | `internal/domain/`; `internal/services/*` | Services translate remote observations into domain outcomes. | Command-visible behavior should depend on domain concepts, not generated shapes. |
| Documentation generator | Command layer | `docsgen/`; `Makefile` `docs-content`; `cmd/root_test.go` | Generated docs derive from command tree/help metadata. | Command contract changes require docs regeneration. |

### Module Invocation Governance

| Rule Source | Allowed Direction | Forbidden Direction | Architecture Constraint | Risk If Violated |
|-------------|-------------------|---------------------|-------------------------|------------------|
| Observed layering from `cmd/cmd_cli.go`, `cmd/root.go`, and `c8volt/client.go` | Command layer to facade for operations; command bootstrap to config/auth/HTTP setup; facade to internal services | Command operation handlers to generated clients | CLI interaction must remain decoupled from upstream generated API shapes while root bootstrap may assemble execution context. | User-facing contracts would become version-fragile or bootstrap behavior would be mixed with operation implementation. |
| Observed factory layout in `internal/services/*/factory.go` | Services to versioned generated clients | Version-specific clients above service boundary | Version compatibility must be handled below command/facade level. | Camunda version support would scatter across commands. |
| Observed docs generation in `Makefile` and `docsgen/` | Command metadata to generated docs | Hand-edited generated docs as source of truth | Command behavior is the authority for generated CLI reference. | Docs drift and automation contract mismatches. |
| Repository-first artifacts absent | No repository-first governance rules available | No additional forbidden edge proven | Architecture rules in this pass are inferred from observable module direction only. | A future dependency matrix may reveal stricter constraints. |

### Dependency Governance Signals

| Signal Source | Dependency / Concern | Signal Type | Affected Boundary | Architecture Review Trigger |
|---------------|----------------------|-------------|-------------------|-----------------------------|
| `go.mod` | Cobra, Viper, pflag, terminal handling, YAML, OpenAPI codegen/runtime, test assertions | CLI/config/build dependency surface | Command, config, generated-client boundaries | Dependency major upgrades affecting command parsing, config precedence, or generated clients. |
| `go.mod` and `internal/clients/` | Multiple Camunda client versions | Version divergence by design | Internal service and generated-client boundary | Adding/removing supported Camunda versions or changing version selection behavior. |
| `package.json`; `package-lock.json` | Auxiliary `yq` Node dependency | Repository tooling dependency signal | Development tooling boundary, not core CLI runtime | Review before treating Node dependencies as required runtime or release dependencies. |
| `docs/Gemfile`; `.github/workflows/docs.yml`; `docs/_config.yml` | Jekyll docs toolchain | Documentation dependency surface | Documentation/release boundary | Docs build, site generation, or hosting workflow changes. |
| `.specify/memory/repository-first/` absent | No dependency matrix projection | Evidence gap | Development view | Run repository-first analysis before making stricter dependency-governance claims. |

## Physical / Deployment Clues

| Deployment Fact | Evidence Source | Environment / External System | Operational Constraint | Not Proven |
|-----------------|-----------------|-------------------------------|------------------------|------------|
| Release archives are built for Linux, Windows, and Darwin on amd64 and arm64. | `.goreleaser.yaml`; `.github/workflows/release.yaml` | GitHub Actions and GoReleaser | Single binary artifact packaged with README, license, and example config. | Installation channel beyond GitHub releases is not fully described. |
| CI builds and runs coverage on Ubuntu using Go version from `go.mod`. | `.github/workflows/go.yml`; `Makefile` | GitHub Actions | Build and tests are expected before integration. | Runtime production hosting is not applicable to the CLI binary. |
| Documentation site builds with Go and Ruby/Jekyll and publishes to c8volt.info through SFTP. | `.github/workflows/docs.yml`; `.github/workflows/docs-release.yaml`; `.github/workflows/release.yaml`; `docs/_config.yml` | GitHub Actions, Jekyll, IONOS SFTP, c8volt.info | Docs release requires hosting secrets and generated docs content. | Web server topology after SFTP upload is not visible. |
| CLI connects to configured Camunda APIs and optional OAuth/cookie identity endpoints. | `config/templates/config.example.yaml`; `README.md` "OAuth"; `internal/services/auth/factory.go` | Camunda, OAuth2 token endpoint, cookie identity base URL | Base URLs and credentials are externally configured. | No bundled Camunda deployment is provided. |
| Local demo and smoke assets assume a reachable Camunda environment. | `README.md` fast-start; `demos/vhs/config/recording.example.yml`; `demos/vhs/scripts/check-vhs.sh` | Local or shared Camunda clusters | Demos and live smoke flows require environment-specific config. | Exact cluster provisioning is outside the repo. |

## Git History Signals

| Signal | Evidence Source | Architecture Meaning | Confidence | Review Trigger |
|--------|-----------------|----------------------|------------|----------------|
| Recent commits include AI tooling installs. | `git log --oneline -n 12` entries `chore(ai): install ai-tooling...` | Auxiliary signal that repo tooling changes may be frequent. | Low | Review generated/tooling changes for architecture-memory drift. |
| Recent commits include docs and command map updates. | `git log --oneline -n 12` entry `docs(readme): update command map` and `docs(cli): clarify reference landing page` | Auxiliary signal that command contract and docs are active change axes. | Low | Regenerate docs when command taxonomy changes. |
| Recent commits include ops and process-definition behavior fixes. | `git log --oneline -n 12` entries for ops scope and process-definition deletion visibility | Auxiliary signal that operational workflows and wait/verification behavior are active risk areas. | Low | Re-check process and scenario views when ops or deletion verification behavior changes. |

## Evidence Gaps

| Gap | Affected View | Why It Blocks Architecture Conclusion |
|-----|---------------|----------------------------------------|
| No `.specify/memory/repository-first/` dependency matrix or module invocation spec exists. | Development View, Architecture Synthesis | This pass can infer layering from source layout, but cannot claim a formally generated dependency-governance model. |
| No Docker, Compose, Kubernetes, Terraform, or service-hosting manifest for c8volt itself was observed. | Physical View | Deployment topology cannot extend beyond released CLI binaries, docs site publication, and external Camunda endpoints. |
| Upstream Camunda cluster topology and identity-provider policies are externally configured. | Physical View, Process View | The repo proves outbound collaboration and auth modes, not ownership or runtime topology of those systems. |
| Long-term audit/report storage policy is not represented. | Process View, Physical View | Ops reports are produced, but retention, archival, and governance of reports cannot be concluded. |
| Complete failure compensation semantics for partial remote mutation failures are distributed across individual command/playbook implementations. | Process View | This pass can state preview/confirm/verify architecture, but not a universal rollback model. |
| Business actors beyond operator, developer, support engineer, CI pipeline, and agent are not evidenced. | Scenario View | README names the actor set; no end-customer or business approver role can be invented. |
| The current logical and process view artifacts are present but did not pass the readiness validator because required headings or source-traceability sections are missing. | Development View, Architecture Synthesis | Development conclusions in this refresh must rely on repository facts rather than treating those supporting views as ready architecture evidence. |

## Evidence Rules

- Every non-placeholder fact must name an evidence source such as a file, directory, configuration, test, script, manifest, command output, or commit signal.
- Confidence values are `High`, `Medium`, or `Low`.
- `High` means multiple independent sources agree, or docs/tests and code entry points agree.
- `Medium` means one strong source is present, such as clear configuration, entry point, route declaration, or behavior test.
- `Low` means naming, directory structure, isolated code, or Git history suggests a fact but lacks behavior evidence.
- Git history is an auxiliary signal for change axes and boundary risks. It cannot independently prove architecture conclusions.
- Repository-first dependency matrices are fact inputs. Do not copy full dependency inventories into architecture views; summarize only architecture constraints, governance signals, gaps, or review triggers.
- Repository-first module invocation specs may support development-view dependency rules when each rule maps to a concrete module edge or dependency signal.
- Concrete classes, functions, fields, endpoints, database tables, and implementation data structures may appear only as evidence sources, not architecture conclusions.
