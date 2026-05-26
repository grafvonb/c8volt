# Logical View

**Input**: `.specify/memory/architecture-scenario-view.md`

**Purpose**: Derive capability boundaries, domain objects, states, relationships, and invariants from the scenario view.

## Architecture Intent

The logical architecture separates command intent, configuration authority, workflow-runtime facts, mutation planning, and operational reporting so c8volt can offer safe command outcomes without becoming the owner of Camunda runtime state.

## Core Tensions

| Tension | Current Tradeoff Direction | Logical Consequence |
|---------|----------------------------|---------------------|
| Command simplicity vs. runtime complexity | Commands expose user goals while logical capabilities absorb target discovery, enrichment, and version gating. | User-facing operations map to stable capabilities, not to upstream API shapes. |
| Local configuration authority vs. external runtime authority | Configuration determines how to talk to Camunda; Camunda determines workflow truth. | Config objects and runtime objects must remain logically distinct. |
| Preview semantics vs. execution semantics | Plans and observed outcomes are separate objects. | Dry-run output cannot be treated as a completed mutation. |
| Primitive commands vs. playbooks | Ops playbooks compose capabilities but do not own the underlying resource types. | Playbook state is an orchestration view over fixed targets and reports. |

## Stable Boundaries

| Boundary | Must Remain Stable Because | Explicitly Does Not Own |
|----------|----------------------------|-------------------------|
| Configuration boundary | Every remote operation depends on resolved version, auth, base URL, tenant, and output mode. | Camunda runtime facts or process lifecycle. |
| Workflow resource boundary | Process definitions, process instances, incidents, jobs, tenants, resources, and cluster facts are external observations. | Local persistence or independent workflow execution. |
| Mutation plan boundary | Risky operations require a clear separation between target selection, preview, confirmation, and submission. | Automatic approval or universal rollback. |
| Ops playbook boundary | High-level workflows must reuse lower-level resource capabilities. | Owning separate duplicate resource semantics. |
| Command contract boundary | Machines require stable contract metadata and output semantics. | Hidden command internals or upstream API documentation. |

## Change Axes

| Expected Change | Isolated By | Logical Impact |
|-----------------|-------------|----------------|
| Additional Camunda resource capabilities | Workflow resource boundary | Add capability semantics without collapsing into config or command rendering. |
| New mutation workflows | Mutation plan boundary | Preserve target selection, preview, confirmation, and observed outcome separation. |
| Expanded ops playbooks | Ops playbook boundary | Compose existing resource concepts and add reports without inventing duplicate domain objects. |
| Output and automation contract changes | Command contract boundary | Keep machine discovery aligned with actual command capabilities. |

## Invariants

| Invariant | Source Scenario / Object / State | Risk If Violated |
|-----------|----------------------------------|------------------|
| Runtime facts are authoritative only when observed from Camunda for the effective configuration context. | Scenario: Inspect workflow and cluster state; Object: Workflow resource | Local output could misrepresent current workflow state. |
| A mutation plan is not a mutation receipt. | Scenario: Destructive cleanup; Object: Mutation plan | Dry-run and completed execution would become indistinguishable. |
| Ops playbooks must freeze target meaning before execution. | Scenario: Ops playbook; Object: Operational playbook | Later discovery drift could change what the user approved. |
| Version support is part of operation validity. | Scenario: Mutate runtime state safely; Object: Version capability | Unsupported upstream actions could be attempted after user confirmation. |

## Non-goals / Anti-patterns

| Non-goal / Anti-pattern | Why It Is Out of Scope or Harmful |
|-------------------------|-----------------------------------|
| Treat local command output as a database | The repo shows no persistent store and Camunda is the fact source. |
| Model generated upstream schemas as the logical domain | Logical objects must be stable user/workflow concepts rather than version-specific wire shapes. |
| Give ops playbooks separate ownership of process, incident, or job facts | Playbooks coordinate resource capabilities and reports; they should not fork domain meaning. |
| Infer complete BPMN business semantics from fixture names | Embedded/sample definitions support smoke flows but do not prove production domain rules. |

## Capability Boundaries

| Capability / Boundary | Responsibility | Input | Output | Explicitly Does Not Own | Scenario Source |
|-----------------------|----------------|-------|--------|--------------------------|-----------------|
| Configuration and context | Resolve effective config, auth mode, tenant, version, timeout, logging, and output mode. | Flags, env, profile, config file, defaults. | Validated execution context. | Remote workflow state. | Configure and verify connectivity. |
| Command contract | Describe visible commands, flags, output modes, mutation behavior, and automation support. | Command tree and invocation mode. | Human or machine-readable capability document. | Upstream API capability enumeration. | Discover machine contract. |
| Workflow inspection | Read and enrich cluster, tenant, resource, process, incident, job, and variable facts. | Effective context and selectors. | Current observations in human or machine form. | Permanent local fact storage. | Inspect workflow and cluster state. |
| Workflow mutation | Plan, preview, submit, and verify state-changing operations. | Effective context, target selection, mutation input, confirmation mode. | Mutation plan, refusal, receipt, or observed result. | Business approval policies beyond command controls. | Mutate runtime state safely. |
| Process family traversal | Interpret parent/child process-instance scope for inspection and cleanup. | Process-instance selection. | Tree or flat relationship view. | BPMN execution semantics beyond observed relationships. | Diagnose process family scope. |
| Operational playbooks | Compose lower-level capabilities into repeatable discover-freeze-plan-execute-verify-report workflows. | Playbook-specific selection and controls. | Dry-run plan or execution report. | Separate resource authority. | Run high-level ops playbooks. |
| Embedded asset support | Provide built-in process definitions for fast-start and smoke workflows. | Embedded asset selection. | Deployable/exportable process asset. | Production process definition ownership. | Deploy and start workflow execution. |

## Domain Objects and Relationships

| Object | Meaning | Owning Capability | Key Relationships | Fact Source | Invariants |
|--------|---------|-------------------|-------------------|-------------|------------|
| Effective configuration | The resolved local context for a command invocation. | Configuration and context | Selects Camunda environment, auth mode, tenant, version, timeout, logging, and output shape. | Repo Facts: Data and State Clues | Must be validated before remote behavior is trusted. |
| Command contract | The visible machine and human surface of the CLI. | Command contract | Describes commands that invoke workflow inspection, mutation, and playbooks. | Repo Facts: Entry Points; User-Visible Behaviors | Must align with actual command behavior. |
| Workflow resource | A Camunda-owned runtime or metadata concept observed through c8volt. | Workflow inspection and mutation | Includes definitions, instances, incidents, jobs, tenants, resources, and cluster facts. | Repo Facts: Data and State Clues | Camunda remains authoritative. |
| Process family | Related process-instance scope used for traversal, cancellation, and deletion decisions. | Process family traversal | Expands from selected instances into ancestor/descendant context. | Repo Facts: User-Visible Behaviors | Scope must be visible before risky family actions. |
| Mutation plan | A proposed state change against a selected scope. | Workflow mutation | Precedes confirmation, submission, and receipt. | Repo Facts: Runtime and Process Clues | Dry-run plan must not imply mutation occurred. |
| Mutation receipt | The command-visible outcome after a remote change or refusal. | Workflow mutation | Follows execution or safety rejection. | Repo Facts: Runtime and Process Clues | Must report observed result or reason for non-execution. |
| Operational playbook | A named higher-level workflow over lower-level capabilities. | Operational playbooks | Freezes targets, builds plans, executes, verifies, and reports. | Repo Facts: Entry Points; Runtime and Process Clues | Must not duplicate lower-level domain ownership. |
| Audit-oriented report | A structured or rendered account of ops workflow plan/execution. | Operational playbooks | Produced from frozen targets and observed results. | Repo Facts: Data and State Clues | Does not imply external archival policy. |

## State and Lifecycle

| Object / Flow | State | Entered When | Exited When | Forbidden Transition | Responsible Boundary |
|---------------|-------|--------------|-------------|----------------------|----------------------|
| Effective configuration | Unresolved | CLI process starts. | Flags/env/profile/config/defaults are normalized and validated. | Remote command execution before validation. | Configuration and context |
| Effective configuration | Ready | Validation and remote service setup succeed. | Command completes or fails. | Claiming remote state without context. | Configuration and context |
| Mutation flow | Selected | Targets are identified by key, stdin, filters, or playbook discovery. | Scope is planned or rejected. | Submitting mutation before target meaning is known. | Workflow mutation |
| Mutation flow | Planned | Previewable effect is available. | User confirms, automation authorizes, or dry-run exits. | Treating plan as receipt. | Workflow mutation |
| Mutation flow | Submitted | Confirmation/automation path sends remote mutation. | Receipt or observed failure is reported. | Silent partial success. | Workflow mutation |
| Ops playbook | Frozen | Candidate targets are discovered and fixed for the run. | Plan is executed, dry-run exits, or validation rejects. | Re-discovering a broader target set after approval. | Operational playbooks |
| Workflow resource observation | Current observation | c8volt reads from Camunda under effective context. | Output is rendered or used for next operation. | Reusing observation as durable truth without re-check. | Workflow inspection |

## Logical Decisions

| Decision | Scope | Owner / Boundary | Affected Objects or Flows | Consequence |
|----------|-------|------------------|---------------------------|-------------|
| Use Camunda as workflow fact authority. | Runtime and metadata facts | Workflow resource boundary | Workflow resource, process family, mutation receipt | c8volt observes and mutates but does not persist workflow truth. |
| Keep config and version support as preconditions. | All remote commands | Configuration and context | Effective configuration, version capability | Invalid or unsupported operations fail before mutation. |
| Expose both primitive commands and composed playbooks. | User workflows | Operational playbooks | Operational playbook, mutation plan, report | Users can choose manual control or higher-level repeatability. |
| Preserve machine contract discovery. | Automation | Command contract | Command contract, CLI scenarios | Scripts and agents use metadata instead of scraping help text. |

## Logical Gaps

| Gap | Affected Capability / Object | Why It Matters |
|-----|------------------------------|----------------|
| Complete upstream resource schema semantics are not an architecture source here. | Workflow resource | Logical model must avoid generated DTO or field-level claims. |
| External identity policy is not evidenced. | Effective configuration | Auth modes are known, but token/session governance remains external. |
| Universal compensation semantics are not evidenced. | Mutation receipt | Architecture cannot claim rollback; it can only require explicit reporting and closure. |
| Report retention and audit custody are not evidenced. | Audit-oriented report | Reports exist, but ownership after writing is outside repo evidence. |

## Prohibited Content

Do not write classes, DTOs, database tables, fields, method names, endpoints, schemas, or implementation data structures here.
