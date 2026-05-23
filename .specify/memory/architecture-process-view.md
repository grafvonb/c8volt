# Process View

**Input**: `.specify/memory/architecture-scenario-view.md`, `.specify/memory/architecture-logical-view.md`

**Purpose**: Derive runtime collaboration, handoffs, approvals, receipts, state advancement, and failure closure from scenario paths and logical boundaries.

## Architecture Intent

The runtime architecture is a short-lived CLI collaboration: resolve local context, authenticate outbound requests, perform scoped Camunda reads or mutations, render human or machine output, and close risky workflows through preview, confirmation, verification, and reporting rather than background state management.

## Core Tensions

| Tension | Current Tradeoff Direction | Process Consequence |
|---------|----------------------------|---------------------|
| Interactive use vs. unattended automation | Branch prompting and output behavior through explicit flags and automation support. | Runtime must reject unsupported automation instead of prompting unpredictably. |
| Remote eventual state vs. command completion | Wait/verify where documented, otherwise report submitted or observed outcome precisely. | Completion condition differs by operation and must be visible to users. |
| Multi-step playbooks vs. shifting remote data | Freeze target sets before execution. | Discovery and approval are tied to the same target meaning. |
| Local failures vs. remote failures | Separate bootstrap/config errors, local precondition errors, upstream errors, and observed closure failures. | Users can tell whether nothing was attempted, remote mutation failed, or verification did not close. |

## Stable Boundaries

| Boundary | Must Remain Stable Because | Explicitly Does Not Control |
|----------|----------------------------|-----------------------------|
| Bootstrap boundary | Commands need resolved config, logging, HTTP, and auth before remote behavior. | Upstream Camunda availability. |
| User approval boundary | Mutations and ops workflows rely on dry-run, confirmation, auto-confirm, or automation semantics. | External business approval systems. |
| Remote authority boundary | Camunda owns workflow state and decides request acceptance. | Internal Camunda execution, worker processing, or persistence. |
| Output boundary | Humans and machines consume different output modes. | Downstream script behavior after output is emitted. |

## Change Axes

| Expected Change | Isolated By | Process Impact |
|-----------------|-------------|----------------|
| New authentication modes or transport behavior | Bootstrap/auth boundary | Extend context setup without changing command scenario semantics. |
| New waits or verification steps | Mutation receipt boundary | Update completion semantics per operation. |
| New ops workflows | Playbook lifecycle boundary | Preserve discover-freeze-plan-validate-execute-verify-report shape. |
| More automation-enabled commands | Output and approval boundary | Expand explicit automation support metadata and reject unsupported paths. |

## Invariants

| Invariant | Source Scenario / Runtime Link | Risk If Violated |
|-----------|--------------------------------|------------------|
| Remote commands must not run before configuration and auth setup succeed. | Scenario: First connection; Runtime Link: Bootstrap to command execution | A command could target the wrong environment or fail after partial setup. |
| Dry-run exits before mutation submission. | Scenario: Destructive cleanup; Runtime Link: Approval to mutation | Preview would become unsafe. |
| Automation mode must be explicit and supported by the command. | Scenario: Automation discovery; Runtime Link: Command contract to execution | Non-interactive runs could hang or behave like interactive sessions. |
| Ops execution must operate on the approved frozen target meaning. | Scenario: Ops playbook; Runtime Link: Discovery to plan | Target drift could invalidate the user's decision. |

## Non-goals / Anti-patterns

| Non-goal / Anti-pattern | Why It Is Out of Scope or Harmful |
|-------------------------|-----------------------------------|
| Background daemon orchestration | Evidence shows per-invocation CLI processes, not resident workers. |
| Universal retry or rollback policy | The repo has operation-specific waits and failures; a single global compensation model is not evidenced. |
| Silent broad selection | Selection, paging, dry-run, keys-only, and confirmation controls are part of the safety model. |
| Rendering human output to feed machines | The machine path is explicit through JSON and keys-only modes. |

## Main Runtime Links

| Runtime Link | Trigger | Source | Target | Transferred Content / Fact | Completion Condition |
|--------------|---------|--------|--------|----------------------------|----------------------|
| CLI bootstrap | User or script invokes command. | Shell process | Configuration and transport context | Flags, env, profiles, config, logging/output flags, auth mode. | Context is valid and installed, or command fails before remote work. |
| Command dispatch | Bootstrap succeeds. | Command layer | Operation facade | User intent, selectors, payloads, output/automation controls. | Operation begins or local preconditions reject input. |
| Remote read | Inspection command or playbook discovery needs facts. | Operation/service boundary | Camunda external boundary | Selectors, tenant, version, auth, request context. | Current observation is rendered or passed to planning. |
| Mutation planning | Risky command or ops workflow has selected targets. | Workflow mutation boundary | User approval boundary | Proposed targets, safety checks, intended changes. | Plan is displayed, rejected, or moves to approval. |
| Mutation submission | User confirms, auto-confirm applies, or supported automation path authorizes. | Workflow mutation boundary | Camunda external boundary | Accepted plan and mutation request. | Remote response is received or upstream failure is reported. |
| Verification/wait | Operation promises confirmed outcome. | Workflow mutation boundary | Camunda external boundary | Target identifiers and expected state/fact condition. | Expected observation is reached or timeout/failure is reported. |
| Ops reporting | Playbook dry-run or execution completes. | Operational playbook boundary | User or automation output boundary | Frozen targets, plan, execution observations, report content. | Report/output is written or command reports failure. |
| Documentation/contract discovery | User or agent invokes capability/docs workflow. | Command contract boundary | User, script, or generated docs | Visible command taxonomy and contract metadata. | Machine-readable or generated documentation surface is produced. |

## Handoffs and Approvals

| Handoff / Approval | From | To | Meaning | Accepted Path | Rejected / Returned Path |
|--------------------|------|----|---------|---------------|--------------------------|
| Config to command | Bootstrap | Command execution | Effective context is ready. | Continue with remote or local command. | Fail with validation/bootstrap error. |
| Dry-run plan to user | Mutation planning | User or automation caller | Planned effect is observable without mutation. | Exit successfully for dry-run, or rerun/continue with confirmation controls. | User revises selection or abandons command. |
| Interactive confirmation | Mutation planning | Human user | User accepts planned mutation. | Submit mutation. | Abort locally before remote side effect. |
| Automation authorization | Command contract | Non-interactive command run | Command is known to support unattended behavior. | Run without prompts and render deterministic output. | Reject unsupported automation path. |
| Ops target freeze | Discovery | Playbook plan | Candidate set becomes the approved scope. | Plan, dry-run, or execute against frozen set. | Unsafe/invalid discovery blocks execution. |

## Receipts and User Participation

| Receipt / Participation Point | Sender | Receiver | Content | User Action | Architecture Consequence |
|-------------------------------|--------|----------|---------|-------------|--------------------------|
| Config validation result | Configuration boundary | User/script | Effective or invalid config context. | Fix config or proceed. | Prevents remote work under ambiguous context. |
| Inspection output | Workflow inspection | User/script | Current runtime facts in selected rendering mode. | Decide next command or consume data. | Keeps c8volt read-only until mutation is chosen. |
| Dry-run output | Mutation planning | User/script | Target scope and proposed effects. | Approve by rerun/confirmation or stop. | Separates plan from receipt. |
| Mutation result | Workflow mutation | User/script | Submitted, confirmed, refused, or failed outcome. | Continue diagnosis or close workflow. | Provides failure closure at command boundary. |
| Ops report | Operational playbook | User/script/audit consumer | Plan and observed workflow outcome. | Store, review, or feed pipeline. | Makes multi-step workflows reviewable after execution. |
| Capability document | Command contract | Agent/script | Supported command contract and automation metadata. | Choose supported command path. | Avoids help-text scraping and unsupported automation. |

## Failure, Degradation, and Closure

| Failure / Branch | Detection Boundary | Responsible Boundary | Degradation or Compensation | User-Visible Result | Closure Condition |
|------------------|--------------------|----------------------|-----------------------------|---------------------|-------------------|
| Missing or invalid config | Bootstrap | Configuration and context | Stop before remote work. | Validation/bootstrap error. | User fixes config or chooses local-only command. |
| Unsupported Camunda version for mutation | Operation validation | Workflow mutation | Stop before mutation. | Unsupported-version error. | User changes configured version/environment or operation. |
| Unsafe destructive target scope | Mutation planning | Workflow mutation | Refuse or require force/confirmation path where supported. | Dry-run/refusal output showing scope issue. | User narrows or explicitly approves supported path. |
| Remote read or mutation failure | Remote authority boundary | Workflow inspection/mutation | Report upstream failure; do not invent state. | Error or partial observed result depending on operation. | User retries, inspects, or changes environment. |
| Verification does not observe expected closure | Verification/wait boundary | Workflow mutation | Report unclosed condition. | Timeout/failure instead of false success. | User inspects current state. |
| Unsupported automation command | Command contract boundary | Command contract | Reject before interactive prompt. | Error directing user to capabilities. | User chooses supported command or removes automation. |
| Docs/release publication failure | CI/release boundary | Documentation/release workflow | CI job fails; artifacts not published. | GitHub Actions failure. | Maintainer reruns after correcting inputs/secrets/tooling. |

## Process Gaps

| Gap | Affected Runtime Link / Scenario | Why It Matters |
|-----|----------------------------------|----------------|
| Universal remote retry semantics are not proven. | Remote read and mutation | Architecture must avoid claiming all operations retry or compensate equally. |
| External Camunda and identity uptime/ownership are not represented. | Bootstrap, remote read, remote mutation | Process view cannot assign operational responsibility outside the CLI boundary. |
| Report retention after writing is not shown. | Ops reporting | Reports are receipts, not an evidenced audit repository. |
| Exact partial-failure semantics vary by playbook/command. | Mutation and ops workflows | Architecture should require explicit reporting rather than a single rollback statement. |

## Prohibited Content

Do not write call stacks, queue names, retry counts, thread/process details, endpoint sequences, workflow engine configuration, or orchestration code here.
