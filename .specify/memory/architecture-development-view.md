# Development View

**Input**: `.specify/memory/architecture-logical-view.md`, `.specify/memory/architecture-process-view.md`

**Purpose**: Derive architecture-level components, package boundary intent, contract/artifact semantics, and dependency rules from logical and process views.

## Architecture Intent

The development architecture keeps the CLI surface, operation facade, domain concepts, versioned remote services, configuration, generated clients, generated docs, and test support separated so command behavior can remain stable while Camunda versions and operational workflows evolve.

## Core Tensions

| Tension | Current Tradeoff Direction | Development Consequence |
|---------|----------------------------|-------------------------|
| Stable user contract vs. evolving upstream APIs | Isolate generated/versioned clients below service and facade boundaries. | Command behavior should not expose generated-client churn. |
| Feature breadth vs. command consistency | Centralize command contract, rendering, config, and confirmation patterns. | New commands should reuse established interaction semantics. |
| High-level ops workflows vs. lower-level primitives | Build playbooks as composition over existing capabilities. | Ops features should not fork process/incident/job ownership. |
| Documentation freshness vs. manual docs edits | Generate CLI reference from command metadata. | Behavior changes must drive documentation updates through the generation path. |

## Stable Boundaries

| Boundary | Must Remain Stable Because | Explicitly Must Not Own |
|----------|----------------------------|-------------------------|
| Command interface layer | It owns user interaction, flags, help, rendering choices, and command taxonomy. | Generated API clients or remote resource persistence. |
| Operation facade layer | It presents grouped operation capabilities to commands. | CLI prompting/rendering details or generated-client details. |
| Domain concept layer | It normalizes workflow and ops concepts for services and facade conversions. | Concrete upstream wire schemas. |
| Service adapter layer | It owns remote collaboration, version selection, lookups, waits, and workflow composition below the facade. | User-facing command taxonomy. |
| Configuration layer | It owns settings, precedence, validation, and examples. | Camunda runtime state. |
| Documentation generation layer | It owns generated command reference and static site inputs. | Source behavior definition. |

## Change Axes

| Expected Change | Isolated By | Development Impact |
|-----------------|-------------|--------------------|
| New Camunda version support | Service adapter and generated-client boundaries | Add or adjust version adapters without command-layer rewrites. |
| New command or flag | Command interface and command contract boundaries | Update tests and regenerate docs so help, capability discovery, and docs agree. |
| New ops playbook | Ops composition boundary | Compose existing resource capabilities and add report semantics without duplicating domain ownership. |
| New auth/config option | Configuration and bootstrap boundaries | Extend context setup and validation while preserving command behavior. |
| Generated client refresh | Generated-client boundary | Validate service conversions and compatibility gates. |

## Invariants

| Invariant | Source Boundary / Contract / Dependency Rule | Risk If Violated |
|-----------|----------------------------------------------|------------------|
| Commands depend on the operation facade rather than on generated clients. | Repo Facts: First-Party Module Edges | Version-specific wire details would leak into user behavior. |
| Internal services may depend on generated clients; generated clients must not define command contracts. | Repo Facts: Module Invocation Governance | Upstream schema churn could break CLI contract stability. |
| Ops playbooks compose lower-level capabilities. | Logical View: Operational playbook; Process View: Ops target freeze | Playbooks could create parallel process/incident semantics. |
| Generated CLI docs follow command metadata. | Repo Facts: Development Structure Clues; Dependency Governance Signals | Documentation could drift from executable behavior. |
| Test support remains outside runtime boundaries. | Repo Facts: Development Structure Clues | Runtime code could inherit test-only assumptions. |

## Non-goals / Anti-patterns

| Non-goal / Anti-pattern | Why It Is Out of Scope or Harmful |
|-------------------------|-----------------------------------|
| A second command framework for ops workflows | Ops should reuse the same CLI contract and rendering controls. |
| Direct command coupling to version-specific upstream clients | It would undermine version compatibility isolation. |
| Treat generated documentation as editable source of truth | Regeneration is the evidenced workflow for CLI reference. |
| Adding a local state store for workflow facts | The logical and process views keep Camunda authoritative. |

## Architecture-Level Components

| Component / Capability Package | Responsibility | Input / Output Boundary | Collaborators | Explicitly Must Not Own | Source View Evidence |
|--------------------------------|----------------|-------------------------|---------------|--------------------------|----------------------|
| Command interface | Command taxonomy, flags, help, confirmation, rendering, and invocation lifecycle. | Shell/stdin/config flags to operation calls and stdout/stderr/exit status. | Configuration, operation facade, command contract, docs generation. | Remote API schemas or workflow state storage. | Scenario: CLI contract boundary; Process: Bootstrap and approval boundaries. |
| Operation facade | Stable grouped operations for workflow resources and ops workflows. | Command intent to capability calls and domain outcomes. | Command interface, domain concepts, service adapters. | User prompting or generated-client selection details. | Logical: Capability boundaries; Repo Facts: Public facade boundary. |
| Domain concepts | Shared workflow, process-family, incident, job, tenant, resource, and report meanings. | Service observations to stable operation results. | Service adapters and facade conversions. | Upstream wire schemas or command help. | Logical: Domain Objects and Relationships. |
| Service adapters | Remote reads/mutations, version-aware behavior, lookup/wait flows, playbook composition. | Effective context and operation inputs to Camunda collaboration and results. | Generated clients, config, HTTP/auth, domain concepts. | CLI presentation or docs publication. | Process: Main Runtime Links; Repo Facts: Internal services. |
| Configuration and transport | Config precedence, validation, auth setup, HTTP timeout/logging, tenant/profile context. | Flags/env/files/profiles/defaults to validated execution context. | Command bootstrap, service adapters, auth providers. | Workflow resource ownership. | Logical: Effective configuration; Process: Bootstrap boundary. |
| Generated-client boundary | Version-specific upstream request/response access. | Service adapter calls to Camunda/OAuth HTTP interactions. | Service adapters and code generation workflow. | User-visible command contract. | Repo Facts: Generated API-client boundary. |
| Documentation and release support | Generated CLI reference, static docs site, release artifact metadata. | Command metadata and README/docs inputs to site and release outputs. | Command interface, CI/release workflows. | Runtime command behavior. | Repo Facts: Documentation and release boundary. |
| Test and fixture support | Fake servers, subprocess execution, embedded/sample process assets, and behavior guards. | Test scenarios and sample assets to verification evidence. | Command, facade, service, and docs-generation tests. | Production process ownership. | Repo Facts: Test support and embedded asset clues. |

## Package Boundary Intent

| Package / Boundary | Abstraction Level | Owned Concepts | May Depend On | Must Not Depend On | Evolution Rule |
|--------------------|-------------------|----------------|---------------|--------------------|----------------|
| Command interface | User interaction | Command paths, flags, rendering, confirmation, machine discovery. | Operation facade, configuration, shared rendering/error helpers. | Version-specific generated clients. | Add commands through existing command contract and docs-generation patterns. |
| Operation facade | Application capability | Grouped workflow and ops operations. | Domain concepts and service adapters. | CLI prompt/render internals. | Keep command-visible operations stable across service implementation changes. |
| Domain concepts | Stable semantic model | Workflow resource meanings, process-family concepts, operation reports. | Standard libraries and shared type helpers. | Upstream generated DTOs as public contract. | Add concepts only when they represent user/workflow semantics. |
| Service adapters | Remote collaboration | Version-aware clients, searches, lookups, waits, and playbook workflows. | Config, HTTP/auth, generated clients, domain concepts. | Command help and docs generation. | Keep upstream version differences contained here. |
| Generated clients | Wire compatibility | Camunda/OAuth HTTP client shapes. | Code generation inputs/toolchain. | Command and domain ownership. | Refresh through generation workflow and validate conversions. |
| Configuration | Execution context | Config schema, precedence, validation, examples. | CLI flags/env and transport/auth setup. | Remote workflow facts. | Preserve precedence and explicit validation behavior. |
| Documentation generation | Published reference | Command docs, static site build inputs, release docs. | Command metadata and docs tooling. | Runtime decision logic. | Regenerate rather than hand-edit derived CLI reference. |

## Contracts and Artifacts

| Contract / Artifact | Semantics | Producer | Consumer | Lifecycle | Architecture Consequence |
|---------------------|-----------|----------|----------|-----------|--------------------------|
| CLI command contract | Visible commands, flags, output modes, mutation and automation metadata. | Command interface | Humans, scripts, agents, docs generation | Changes with command behavior. | Must be tested and regenerated into docs when user-facing surface changes. |
| Effective configuration | Resolved environment and execution context. | Configuration layer | Bootstrap, service adapters, command rendering | Built per invocation. | Remote behavior depends on validated context. |
| Operation result | Stable outcome of inspection, mutation, or playbook. | Facade/service adapters | Command rendering and automation output | Produced per command. | Keeps generated-client details below architecture boundary. |
| Generated client set | Version-specific upstream access layer. | Code generation workflow | Service adapters | Refreshed when upstream specs/support change. | Requires compatibility review for supported Camunda versions. |
| Generated CLI reference | Published command documentation. | Documentation generation | Users and website | Regenerated from command metadata. | Docs must follow executable command behavior. |
| Release archive | Installable CLI distribution. | Release workflow | End users and pipelines | Built from tagged releases. | Runtime deployment is a single binary plus documentation/example config. |
| Ops report | Audit-oriented playbook receipt. | Operational playbooks | Operators, CI, agents | Produced by dry-run or execution flows. | Supports review but not evidenced as long-term storage. |

## Dependency Rules

| Rule | Allowed Direction | Forbidden Direction | Reason | Risk If Violated |
|------|-------------------|---------------------|--------|------------------|
| Command to facade | Command interface to operation facade | Command interface to generated clients | Keeps CLI contract stable across upstream versions. | Commands become version-fragile. |
| Facade to services | Operation facade to service adapters and domain concepts | Service adapters to command rendering | Keeps user interaction outside remote implementation. | Services would start controlling UX inconsistently. |
| Services to generated clients | Service adapters to generated-client boundary | Generated clients to facade or command contract | Keeps generated API churn isolated. | Upstream schema changes leak into public behavior. |
| Ops as composition | Operational playbooks to workflow inspection/mutation capabilities | Ops playbooks owning duplicate resource semantics | Keeps high-level workflows consistent with low-level commands. | Divergent behavior between primitive commands and playbooks. |
| Docs from commands | Documentation generation reads command metadata | Generated docs defining runtime behavior | Keeps docs and help aligned. | Stale or contradictory user documentation. |
| Tests outside runtime | Runtime code may use shared production helpers; tests may use test support | Production code depending on test fixtures | Preserves deployable binary independence. | Test-only assumptions enter production. |

## Development View Gaps

| Gap | Affected Component / Boundary | Why It Matters |
|-----|-------------------------------|----------------|
| No repository-first module invocation spec exists. | Dependency Rules | These rules are inferred from observable structure rather than a generated dependency-governance artifact. |
| Complete import graph was not formalized in this pass. | Package Boundary Intent | A future repository-first matrix should validate or refine allowed directions. |
| Node auxiliary artifact lacks manifest evidence. | Documentation/release support | It should not drive an architecture conclusion without a clear toolchain source. |
| Generated-client provenance is partly script-based. | Generated-client boundary | Upstream spec refresh policy may need stricter documentation if client generation changes. |

## Prohibited Content

Do not write source file paths, concrete package trees, classes, functions, implementation tasks, framework-specific wiring, or code generation notes here.
