# Physical View

**Input**: `.specify/memory/architecture-process-view.md`, `.specify/memory/architecture-development-view.md`

**Purpose**: Derive deployment, hosting, external system, fact-source, observability, and operational boundaries from process and development views.

## Architecture Intent

c8volt's physical architecture is a distributed CLI artifact and documentation site that run against externally managed Camunda and identity systems. The repository proves binary release, CI, docs publication, local configuration, and outbound API collaboration, but not ownership of the remote runtime topology.

## Core Tensions

| Tension | Current Tradeoff Direction | Physical Consequence |
|---------|----------------------------|----------------------|
| Local binary runtime vs. remote workflow authority | Package c8volt as a single CLI artifact and configure external service endpoints at runtime. | Deployment concerns are binary distribution and config, not hosting an application service. |
| Multi-platform installability vs. single source build | Release archives target common operating systems and architectures. | Runtime unit remains the same command-line executable across platforms. |
| Public docs availability vs. executable source of truth | Build and publish a static docs site from generated content. | Docs hosting is separate from CLI runtime. |
| Local/dev and protected environments | Support no-auth, OAuth2, and cookie auth with environment-specific base URLs. | Identity and cluster security policies remain external. |

## Stable Boundaries

| Boundary | Must Remain Stable Because | Explicitly Does Not Carry |
|----------|----------------------------|---------------------------|
| CLI binary boundary | Users install and run c8volt locally or in CI. | Server-side workflow execution or local persistent database. |
| External Camunda boundary | All workflow facts and mutations go through configured Camunda APIs. | Camunda cluster hosting, brokers, workers, or storage. |
| External identity boundary | Auth modes configure how requests are authorized. | Identity-provider lifecycle or policy management. |
| Documentation site boundary | Users consume generated reference and playbook docs separately from runtime command execution. | Runtime command state or user credentials. |
| CI/release boundary | Build/test/release workflows produce validated artifacts and docs. | Production operation of user Camunda environments. |

## Change Axes

| Expected Change | Isolated By | Physical Impact |
|-----------------|-------------|-----------------|
| New operating system or architecture release target | Release artifact boundary | Adjust release packaging without changing runtime architecture. |
| New hosted documentation workflow | Documentation site boundary | Update docs build/publish path without changing CLI runtime. |
| New Camunda environment shape | Configuration and external-system boundary | Users provide different base URLs/auth settings; c8volt remains a client. |
| New auth mode | External identity boundary | Extend bootstrap/auth collaboration while keeping remote ownership external. |
| New CI validation requirements | CI/release boundary | Update build/test/docs gates before release. |

## Invariants

| Invariant | Source Deployment / External / Fact Boundary | Risk If Violated |
|-----------|----------------------------------------------|------------------|
| The CLI binary must not require a bundled server-side c8volt deployment. | Repo Facts: Physical / Deployment Clues; Development View: Release archive | Users would need unproven infrastructure. |
| Remote workflow facts are authoritative only through the configured Camunda environment. | Logical View: Workflow resource; Process View: Remote authority boundary | Local output could be treated as durable runtime truth. |
| Docs publication must not be required for command execution. | Repo Facts: Documentation and release boundary | Runtime availability would be coupled to website hosting. |
| Release artifacts should include enough local setup material for first configuration. | Repo Facts: Physical / Deployment Clues | New users would lack a starter config path. |

## Non-goals / Anti-patterns

| Non-goal / Anti-pattern | Why It Is Out of Scope or Harmful |
|-------------------------|-----------------------------------|
| Inventing Kubernetes, Docker, Terraform, or hosted service topology | No such deployment manifests for c8volt runtime are evidenced. |
| Treating c8volt.info as a runtime dependency | It is a documentation site, not part of command execution. |
| Bundling Camunda as part of c8volt | The repo configures external Camunda endpoints and sample assets only. |
| Storing credentials in release artifacts | Config examples show credential placeholders and environment-variable guidance. |

## Deployment and Hosting Boundaries

| Runtime / Hosting Unit | Carries | Boundary | Depends On | Release / Migration Impact |
|------------------------|---------|----------|------------|----------------------------|
| c8volt CLI binary | Command interface, operation facade, services, generated clients, embedded assets. | Local workstation, CI job, or script runner. | Go-built release artifact and effective user configuration. | Release packaging changes affect install/update, not remote service hosting. |
| Release archive | Binary, README, license, example config. | GitHub release distribution. | Tag-triggered release workflow and GoReleaser. | Archive content changes affect onboarding and compliance artifacts. |
| Documentation site | Static reference, use cases, ops playbooks, assets. | c8volt.info static hosting. | Docs build workflow, generated docs, Jekyll site, SFTP publication. | Docs can be rebuilt/published independently of user runtime. |
| CI build/test environment | Build, tests, coverage, docs validation. | GitHub Actions. | Go toolchain, Ruby docs toolchain, repository secrets where applicable. | Gate changes before release/docs publication. |
| User Camunda environment | Workflow runtime, API responses, process and incident state. | External system configured by base URLs and auth. | User-managed cluster and identity setup. | c8volt release changes must preserve configured-environment compatibility. |

## External System Collaboration

| External System | Purpose | Exchanged Content | Authoritative Fact | Failure Impact | Isolation / Substitute Boundary |
|-----------------|---------|-------------------|--------------------|----------------|---------------------------------|
| Camunda APIs | Read and mutate workflow resources and cluster facts. | Configured requests, selectors, process assets, variables, mutation commands, runtime observations. | Workflow runtime state and metadata. | Inspection/mutation commands fail or cannot verify outcome. | Effective config can point to local/dev/prod environments. |
| OAuth2 token endpoint | Provide bearer-style authentication where configured. | Client credentials/scopes and token responses. | Token issuance and auth policy. | Bootstrap/auth fails before remote command action. | Other auth modes are configured alternatives. |
| Cookie identity endpoint | Support session cookie-based auth where configured. | Session credentials and cookie/XSRF material. | Session authority. | Bootstrap/auth or request authorization fails. | Other auth modes are configured alternatives. |
| GitHub Actions | Build, test, docs, and release workflows. | Source checkout, test coverage, release tags, docs artifacts. | CI job status and release execution outcome. | Artifact or docs publication fails. | Local `make` targets provide similar developer validation but not publication. |
| GoReleaser/GitHub releases | Package and publish installable archives. | Binary builds, checksums, changelog, archive metadata. | Release artifact set. | Users cannot obtain expected archive from release workflow. | Local release target can build without publishing. |
| Static docs host via SFTP | Publish c8volt.info content. | Generated site files and hosting credentials. | Published website content. | Documentation site release fails. | CLI remains runnable without website availability. |

## Fact Sources and Observability

| Fact / Event | Authoritative Source | Observable Location | Consumers | Traceability Requirement |
|--------------|----------------------|---------------------|-----------|--------------------------|
| Effective command configuration | Local config/env/flags/profiles/defaults | CLI config output and bootstrap logs/errors | Operators, scripts, service adapters | Show enough context to avoid wrong-environment execution. |
| Workflow runtime state | Camunda external environment | CLI inspection output and mutation verification output | Operators, support engineers, pipelines, agents | Report the effective target and observed result. |
| Command capability contract | Executable command metadata | `capabilities` output and generated docs | Scripts, agents, docs users | Keep discovery, help, and docs aligned. |
| Ops playbook result | c8volt playbook execution against Camunda observations | Terminal output, JSON output, or report file | Operators, CI, audit consumers | Tie report to frozen targets and observed execution. |
| CI validation result | GitHub Actions | Workflow run status, coverage artifact, logs | Maintainers | Failing validation blocks confidence in release readiness. |
| Release artifact | GitHub release workflow and GoReleaser | Published archives and checksums | Users and pipelines | Include version/build metadata and starter artifacts. |
| Documentation publication | Docs workflow and static host | c8volt.info and workflow logs | Users and maintainers | Generated content should be traceable to command metadata and docs sources. |

## Operations and Release Boundaries

| Operational Concern | Responsible Boundary | Trigger | Affected Views | Architecture Consequence |
|---------------------|----------------------|---------|----------------|--------------------------|
| User environment setup | User plus configuration boundary | Config file, env, profile, or flags change. | Scenario, Process, Physical | c8volt can validate/use context but not manage external environments. |
| Command behavior validation | CI and local test boundary | Pull request, branch push, or local `make` target. | Development, Process | Behavior changes should be tested near command/service boundary. |
| Release packaging | Release boundary | Version tag. | Development, Physical | Binary distribution is the deployment unit. |
| Docs publication | Documentation site boundary | Main/docs workflow, release, or manual docs release. | Scenario, Development, Physical | Documentation follows generated command reference and playbook sources. |
| Camunda compatibility review | Service adapter and generated-client boundary | New supported version or upstream API change. | Logical, Process, Development, Physical | Version-specific behavior must stay below user contract where possible. |
| Ops workflow safety review | Operational playbook boundary | New or changed playbook. | Scenario, Logical, Process | Must preserve discover/freeze/plan/dry-run/confirm/verify/report semantics. |

## Physical View Gaps

| Gap | Affected Deployment / External Boundary | Why It Matters |
|-----|-----------------------------------------|----------------|
| No c8volt server deployment manifests are present. | CLI binary boundary | Architecture cannot claim any hosted runtime beyond local/CI binary execution. |
| External Camunda topology is not described. | Camunda external boundary | Cluster ownership, brokers, workers, and storage remain outside this repo. |
| Identity provider policies are not described. | External identity boundary | Auth modes are known, but rotation, scopes, sessions, and organizational policy are external. |
| Static hosting details after SFTP upload are not visible. | Documentation site boundary | Physical web hosting can only be described as generated static content published by workflow. |
| Ops report archival destination is not evidenced. | Fact sources and observability | Reports are command outputs/artifacts, not a proven audit storage system. |

## Prohibited Content

Do not write Kubernetes YAML, cloud resource manifests, machine sizes, service SKUs, deployment scripts, runbooks, or concrete infrastructure configuration here.
