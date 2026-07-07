# Development View

**Input**: repository facts; logical and process views only when they pass readiness validation.

**Purpose**: Architecture-level components, package boundary intent, contract/artifact semantics, dependency rules, and the required dependency matrix.

## Architecture Intent

The development architecture keeps command interaction, bootstrap context, operation facades, domain concepts, service adapters, generated clients, documentation generation, release support, and test/fixture support separated so c8volt can preserve stable command behavior while Camunda versions and operational workflows evolve.

## Core Tensions

| Tension | Chosen Architectural Direction | Development Consequence |
|---------|-------------------------------|-------------------------|
| Stable user contract vs. evolving upstream APIs | Isolate generated/versioned clients below service and facade boundaries. | Command behavior should not expose upstream wire-shape churn. |
| Interactive command breadth vs. consistency | Centralize command contract, rendering, config, confirmation, and automation patterns. | New command work should reuse the established command interaction model. |
| Root bootstrap needs vs. operation isolation | Allow command bootstrap to assemble config/auth/transport while operation handlers use facades. | Context setup remains explicit without letting command operations own remote collaboration. |
| High-level ops workflows vs. lower-level primitives | Treat playbooks as composition over existing resource capabilities. | Ops features should not fork process, incident, job, or resource semantics. |
| Documentation freshness vs. manual edits | Generate CLI reference from command metadata. | Behavior changes must drive documentation updates through the generation path. |

## Stable Boundaries

| Boundary | Must Remain Stable Because | Excluded Responsibility / Scope |
|----------|----------------------------|---------------------------------|
| Command interface layer | It owns command taxonomy, flags, help, prompting, rendering choices, output modes, and invocation lifecycle. | Generated API clients, remote resource persistence, or service adapter ownership. |
| Bootstrap context layer | Remote commands need effective configuration, auth, tenant/profile, timeout, and logging context before work starts. | Command-specific business behavior or workflow resource state. |
| Operation facade layer | Commands need grouped operation capabilities without coupling to internal implementation details. | CLI prompting/rendering details or generated-client details. |
| Domain concept layer | Services and facades need stable workflow and ops meanings that outlive upstream schema changes. | Concrete upstream wire schemas. |
| Service adapter layer | Remote collaboration, version selection, lookup, wait, workflow composition, and mutation mechanics belong below the facade. | User-facing command taxonomy or documentation publication. |
| Generated-client boundary | Upstream API version differences must remain isolated. | Public command/domain contract ownership. |
| Documentation and release support | Published docs and installable archives are derived from source behavior and release workflows. | Runtime command decisions or Camunda workflow state. |
| Test and fixture support | Verification and sample assets support confidence without becoming runtime authority. | Production process ownership or deployable runtime dependencies. |

## Change Axes

| Expected Change | Isolation Mechanism / Boundary Rule | Development Impact |
|-----------------|-------------------------------------|--------------------|
| New Camunda version support | Service adapter and generated-client boundaries | Add or adjust version adapters and compatibility checks without command-layer rewrites. |
| New command or flag | Command interface, command contract, and docs-generation boundaries | Update behavior tests and generated docs so help, capabilities, and published reference agree. |
| New ops playbook | Ops composition boundary | Compose existing resource capabilities and add reporting semantics without duplicating domain ownership. |
| New auth/config option | Bootstrap context and configuration boundaries | Extend context setup and validation while preserving command operation behavior. |
| Generated client refresh | Generated-client boundary | Validate service conversions and compatibility gates before exposing behavior changes. |
| Tooling dependency change | Development tooling and docs/release boundaries | Review whether the dependency affects core runtime, docs, or local automation before projecting architecture impact. |

## Invariants

| Invariant | Source Boundary / Contract / Dependency Rule | Risk If Violated |
|-----------|----------------------------------------------|------------------|
| Command operation handlers depend on operation facades rather than generated clients. | Repo Facts: First-Party Module Edges; Module Invocation Governance | Version-specific wire details would leak into user behavior. |
| Command bootstrap may depend on configuration, auth, and transport setup as an explicit context-building exception. | Repo Facts: First-Party Module Edges | Operation implementation and context setup would become indistinct. |
| Facades delegate to internal services and convert outcomes without owning CLI interaction. | Repo Facts: Development Structure Clues; First-Party Module Edges | Public operation APIs would become coupled to terminal UX. |
| Internal services may depend on generated clients; generated clients must not define command contracts. | Repo Facts: Module Invocation Governance | Upstream schema churn could break CLI contract stability. |
| Ops playbooks compose lower-level capabilities. | Repo Facts: Entry Points; Runtime and Process Clues; Development Structure Clues | Playbooks could create parallel resource semantics. |
| Generated CLI docs follow command metadata. | Repo Facts: Development Structure Clues; Dependency Governance Signals | Documentation could drift from executable behavior. |
| Test support remains outside runtime boundaries. | Repo Facts: Development Structure Clues | Runtime code could inherit test-only assumptions. |

## Non-goals / Anti-patterns

| Non-goal / Anti-pattern | Why It Is Out of Scope or Harmful |
|-------------------------|-----------------------------------|
| A second command interaction model for ops workflows | Ops should reuse the same command contract, output, confirmation, and rendering controls. |
| Direct command coupling to version-specific upstream clients | It would undermine version compatibility isolation. |
| Treating generated documentation as editable source of truth | Regeneration is the evidenced workflow for CLI reference. |
| Treating generated-client presence as supported user behavior | Supported behavior requires command/config/docs evidence, not only generated code. |
| Adding a local state store for workflow facts | Repository evidence keeps Camunda authoritative for workflow runtime state. |
| Treating auxiliary tooling dependencies as core runtime dependencies without source evidence | Tooling manifests can support local automation without changing CLI architecture. |

## Architecture-Level Components

| Component / Capability Package | Responsibility | Input / Output Boundary | Collaborators | Excluded Responsibility / Scope | Source Reference |
|--------------------------------|----------------|-------------------------|---------------|---------------------------------|------------------|
| Command interface | Command taxonomy, flags, help, prompting, rendering, output modes, and invocation lifecycle. | Shell/stdin/config flags to operation calls and stdout/stderr/exit status. | Bootstrap context, operation facade, command contract, docs generation. | Remote API schemas or workflow state storage. | Repo Facts: Entry Points; User-Visible Behaviors; Development Structure Clues |
| Bootstrap context | Effective configuration, auth setup, HTTP transport, timeout/logging, tenant/profile, and pre-command validation. | Flags/env/files/profiles/defaults to validated execution context. | Command interface, configuration, auth providers, transport services. | Workflow resource ownership or command-specific remote behavior. | Repo Facts: Runtime and Process Clues; First-Party Module Edges |
| Operation facade | Stable grouped operations for workflow resources and ops workflows. | Command intent to capability calls and domain outcomes. | Command interface, domain concepts, service adapters. | User prompting or generated-client selection details. | Repo Facts: Public facade boundary; First-Party Module Edges |
| Domain concepts | Shared workflow, process-family, incident, job, tenant, resource, and report meanings. | Service observations to stable operation results. | Service adapters and facade conversions. | Upstream wire schemas or command help. | Repo Facts: Data and State Clues; Development Structure Clues |
| Service adapters | Remote reads/mutations, version-aware behavior, lookup/wait flows, and playbook composition. | Effective context and operation inputs to Camunda collaboration and results. | Generated clients, config/auth/transport, domain concepts. | CLI presentation or docs publication. | Repo Facts: Internal services; Runtime and Process Clues |
| Generated-client boundary | Version-specific upstream request/response access. | Service adapter calls to external HTTP interactions. | Service adapters and code generation workflow. | User-visible command or domain contract. | Repo Facts: Generated API-client boundary; Generated client adapters |
| Documentation and release support | Generated CLI reference, static docs site, release artifact metadata, and docs publication inputs. | Command metadata and docs/release inputs to published references and archives. | Command interface, CI/release workflows. | Runtime command behavior or workflow state. | Repo Facts: Documentation and release boundary; Physical / Deployment Clues |
| Test and fixture support | Fake servers, subprocess execution, embedded/sample process assets, and behavior guards. | Test scenarios and sample assets to verification evidence. | Command, facade, service, and docs-generation tests. | Production process ownership. | Repo Facts: Test support and embedded asset clues |

## Package Boundary Intent

| Package / Boundary | Abstraction Level | Owned Concepts | May Depend On | Must Not Depend On | Evolution Rule |
|--------------------|-------------------|----------------|---------------|--------------------|----------------|
| Command interface | User interaction | Command paths, flags, rendering, confirmation, machine discovery. | Operation facade, bootstrap context, shared rendering/error helpers. | Version-specific generated clients. | Add commands through existing command contract and docs-generation patterns. |
| Bootstrap context | Invocation setup | Effective config, auth, HTTP transport, profile/tenant, logging, validation. | Configuration, auth/transport services, command context helpers. | Workflow resource semantics or generated-client operation details. | Extend setup without moving remote operation logic into commands. |
| Operation facade | Application capability | Grouped workflow and ops operations. | Domain concepts and service adapters. | CLI prompt/render internals. | Keep command-visible operations stable across service implementation changes. |
| Domain concepts | Stable semantic model | Workflow resource meanings, process-family concepts, operation reports. | Standard libraries and shared production helpers. | Upstream generated DTOs as public contract. | Add concepts only when they represent user/workflow semantics. |
| Service adapters | Remote collaboration | Version-aware clients, searches, lookups, waits, and playbook workflows. | Config/auth/transport, generated clients, domain concepts, shared production helpers. | Command help and docs generation. | Keep upstream version differences contained here. |
| Generated clients | Wire compatibility | External API client shapes. | Code generation inputs/toolchain. | Command and domain ownership. | Refresh through generation workflow and validate conversions. |
| Configuration | Execution context | Config schema, precedence, validation, and examples. | CLI flags/env and transport/auth setup. | Remote workflow facts. | Preserve precedence and explicit validation behavior. |
| Documentation generation | Published reference | Command docs, static site build inputs, release docs. | Command metadata and docs tooling. | Runtime decision logic. | Regenerate rather than hand-edit derived CLI reference. |
| Test and fixture support | Verification support | Fake servers, integration guards, subprocess helpers, embedded/sample assets. | Runtime packages under test and test-only helpers. | Production runtime dependencies. | Keep test-only helpers outside deployable runtime boundaries. |

## Contracts And Artifacts

| Contract / Artifact | Semantics | Producer | Consumer | Lifecycle | Architecture Consequence |
|---------------------|-----------|----------|----------|-----------|--------------------------|
| CLI command contract | Visible commands, flags, output modes, mutation and automation metadata. | Command interface | Humans, scripts, agents, docs generation | Changes with command behavior. | Must be tested and regenerated into docs when user-facing surface changes. |
| Effective configuration | Resolved environment and execution context. | Bootstrap context and configuration layer | Command execution and service adapters | Built per invocation. | Remote behavior depends on validated context. |
| Operation result | Stable outcome of inspection, mutation, or playbook. | Facade/service adapters | Command rendering and automation output | Produced per command. | Keeps generated-client details below architecture boundary. |
| Generated client set | Version-specific upstream access layer. | Code generation workflow | Service adapters | Refreshed when upstream specs/support change. | Requires compatibility review for supported Camunda versions. |
| Generated CLI reference | Published command documentation. | Documentation generation | Users and website | Regenerated from command metadata. | Docs must follow executable command behavior. |
| Release archive | Installable CLI distribution. | Release workflow | End users and pipelines | Built from tagged releases. | Runtime deployment is a single binary plus documentation/example config. |
| Ops report | Audit-oriented playbook receipt. | Operational playbooks | Operators, CI, agents | Produced by dry-run or execution flows. | Supports review but is not evidenced as long-term storage. |
| Auxiliary tooling manifest | Local tooling dependency declaration. | Repository tooling | Developers or maintenance automation | Changes with tooling needs. | Does not become core runtime architecture without source evidence. |

## Dependency Rules

| Rule | Allowed Direction | Forbidden Direction | Reason | Risk If Violated |
|------|-------------------|---------------------|--------|------------------|
| Command operation to facade | Command interface to operation facade | Command operation handlers to generated clients | Keeps CLI contract stable across upstream versions. | Commands become version-fragile. |
| Command bootstrap to context setup | Command bootstrap to configuration/auth/transport setup | Bootstrap owning workflow resource semantics | Keeps execution context assembly explicit and bounded. | Context setup starts duplicating operation behavior. |
| Facade to services | Operation facade to service adapters and domain concepts | Service adapters to command rendering | Keeps user interaction outside remote implementation. | Services would start controlling UX inconsistently. |
| Services to generated clients | Service adapters to generated-client boundary | Generated clients to facade or command contract | Keeps generated API churn isolated. | Upstream schema changes leak into public behavior. |
| Ops as composition | Operational playbooks to workflow inspection/mutation capabilities | Ops playbooks owning duplicate resource semantics | Keeps high-level workflows consistent with low-level commands. | Divergent behavior between primitive commands and playbooks. |
| Docs from commands | Documentation generation reads command metadata | Generated docs defining runtime behavior | Keeps docs and help aligned. | Stale or contradictory user documentation. |
| Tests outside runtime | Runtime code may use shared production helpers; tests may use test support | Production code depending on test fixtures | Preserves deployable binary independence. | Test-only assumptions enter production. |
| Tooling dependencies stay scoped | Tooling manifests to local automation or docs/release support when evidenced | Auxiliary tooling dependencies defining CLI runtime behavior without source evidence | Prevents build/tooling artifacts from becoming architecture claims. | Runtime requirements become overstated or brittle. |

## Dependency Matrix

| From Boundary / Component | To Boundary / Component | Allowed? | Constraint / Rule Source | Architecture Consequence |
|---------------------------|-------------------------|----------|--------------------------|--------------------------|
| Command interface | Operation facade | Yes | Repo Facts: First-Party Module Edges; Module Invocation Governance | Command operations stay separated from service implementation details. |
| Command interface | Bootstrap context | Yes, for invocation setup | Repo Facts: First-Party Module Edges; Runtime and Process Clues | Config/auth/transport setup remains a bounded command-layer responsibility. |
| Command interface | Generated-client boundary | No | Repo Facts: Module Invocation Governance | Upstream wire shapes must not define command behavior. |
| Bootstrap context | Configuration and transport/auth services | Yes | Repo Facts: First-Party Module Edges | Remote work starts from a validated execution context. |
| Operation facade | Service adapters | Yes | Repo Facts: Public facade boundary; First-Party Module Edges | Public operations can delegate without exposing internals to commands. |
| Operation facade | Command rendering/prompting | No | Repo Facts: Development Structure Clues; Module Invocation Governance | Facades remain application capability surfaces, not terminal UX owners. |
| Service adapters | Domain concepts | Yes | Repo Facts: First-Party Module Edges; Development Structure Clues | Remote observations are normalized before command-visible output. |
| Service adapters | Generated-client boundary | Yes | Repo Facts: Generated API-client boundary; First-Party Module Edges | Version-specific upstream interaction remains below services. |
| Generated-client boundary | Command interface or operation facade | No | Repo Facts: Module Invocation Governance | Generated code cannot become public command contract. |
| Documentation generation | Command interface | Yes | Repo Facts: Documentation and release boundary; Module Invocation Governance | Generated docs follow executable command metadata. |
| Generated CLI reference | Runtime command behavior | No | Repo Facts: Module Invocation Governance | Published docs remain derived artifacts, not behavior authority. |
| Runtime code | Test and fixture support | No | Repo Facts: Development Structure Clues | Deployable runtime avoids test-only assumptions. |
| Auxiliary tooling manifest | Core CLI runtime | Not proven | Repo Facts: Build Manifest Detection; Development-Owned Dependency Governance Signals | Tooling dependencies require additional source evidence before becoming runtime constraints. |

## Source Traceability

| Architecture Conclusion | Source Type | Source Reference | Confidence |
|-------------------------|-------------|------------------|------------|
| Command behavior is owned by the command interface and operation facades rather than generated clients. | Repo facts | Entry Points; User-Visible Behaviors; First-Party Module Edges; Module Invocation Governance | High |
| Bootstrap context is an allowed command-layer dependency separate from operation implementation. | Repo facts | Runtime and Process Clues; First-Party Module Edges | High |
| Internal service adapters own remote collaboration and generated-client use. | Repo facts | System Boundaries; Development Structure Clues; First-Party Module Edges | High |
| Generated clients must stay below service boundaries and not define public command/domain contracts. | Repo facts | Generated API-client boundary; Module Invocation Governance | High |
| Ops playbooks are composition over lower-level capabilities. | Repo facts | Entry Points; User-Visible Behaviors; Runtime and Process Clues | High |
| Generated CLI documentation follows command metadata. | Repo facts | Documentation and release boundary; Development Structure Clues; Dependency Governance Signals | High |
| Test and fixture support is verification support, not production resource ownership. | Repo facts | Data and State Clues; Development Structure Clues | Medium |
| Auxiliary Node tooling is detected but not proven as core runtime architecture. | Repo facts | Build Manifest Detection; Development-Owned Dependency Governance Signals | Medium |
| Logical and process views were not used as supporting evidence for new conclusions in this refresh. | Validator result and repo-facts gap | Evidence Gaps | High |

## Development View Gaps

| Gap | Affected Component / Boundary | Why It Matters |
|-----|-------------------------------|----------------|
| No repository-first module invocation spec exists. | Dependency Rules and Dependency Matrix | These rules are inferred from observable repository structure rather than a generated dependency-governance artifact. |
| Complete import graph was not formalized in this pass. | Package Boundary Intent | A future repository-first matrix should validate or refine allowed directions. |
| Logical and process views are present but not validator-ready. | Source Traceability | This refresh treats them as context only and relies on repo facts for new development conclusions. |
| Auxiliary Node tooling has limited source evidence. | Documentation/release or local tooling support | It should not drive a core runtime architecture conclusion without clearer use in build or source workflows. |
| Generated-client provenance is partly script-based. | Generated-client boundary | Upstream spec refresh policy may need stricter documentation if client generation changes. |

## Prohibited Content

Do not write source file paths, concrete package trees, classes, functions, implementation tasks, framework-specific wiring, or code generation notes here.
