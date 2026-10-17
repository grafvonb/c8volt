# Concept: Focused Agent Access to Camunda Operations

- **Slug**: c8volt-mcp-server
- **Created**: 2026-10-17
- **Inputs used**: [intake.md](intake.md), [research.md](research.md), [problem.md](problem.md)
- **Recommended option**: Option B — Bounded read-only MCP pilot, conditional on validating a concrete user workflow.
- **Location**: `specs/assessments/c8volt-mcp-server/`, as requested by the user.

## Options

The options below are shaping proposals, not approved scope. Appetite labels express provisional spending limits, not delivery estimates. The evidence contains no agreed budget or staffing commitment; each appetite requires confirmation by the decision owner. Existing operational entry points support considering a bounded experiment, but do not establish how long delivery will take. ([Research: Prior Art](research.md#prior-art); [problem: Q9](problem.md#open-questions))

### Option A — Use the existing automation experience

- **Sketch**: Users continue through the current CLI's machine-readable discovery, structured results, and non-interactive execution. A representative user workflow is tried with those existing capabilities to establish whether the reported friction remains and where assistance is needed. This is the smallest intervention and the no-new-product baseline; it may resolve a setup or usage gap without introducing another integration surface.
- **Appetite**: **Small (days), provisional** for a bounded workflow evaluation and guidance. No new server investment. [NEEDS CLARIFICATION: Confirm the allowed evaluation budget and user availability.]
- **Trade-offs**: Preserves current behavior and avoids additional compatibility support. It may adequately serve clients that can invoke the CLI, but it does not satisfy the issue's request for direct MCP access and may leave some clients unable to participate conveniently. Whether that limitation affects real users is unvalidated. ([Research: Market & Context](research.md#market--context))
- **Rabbit holes**: Turning targeted guidance into a general documentation rewrite; building custom shell integration for every client; repeatedly revisiting the same workflow without agreeing what constitutes sufficient evidence.

### Option B — Bounded read-only MCP pilot

- **Sketch**: A named pilot user accesses a small, agreed set of existing Camunda inspection workflows from a chosen MCP-capable client, without manually constructing shell commands. Candidate experiences are inspecting cluster information, finding process definitions or instances, and understanding process ancestry or family. The pilot centers on a complete useful investigation, not an arbitrary count of exposed operations, and compares the experience with the current CLI baseline. Changes to running processes remain outside this pilot.
- **Appetite**: **Medium (weeks), provisional ceiling** for a deliberately limited pilot. [NEEDS CLARIFICATION: Confirm a budget owner, actual time allowance, supported client/environment, and what can fit; do not treat this label as an estimate.]
- **Trade-offs**: Directly explores whether MCP improves discovery, input correctness, setup effort, and interpretation for an actual user. Limiting the pilot to inspection narrows the operational responsibilities under evaluation. It cannot validate mutation authorization or accepted-versus-completed mutation semantics, and it would not fulfill the whole original issue. Demand and compatibility remain assumptions. ([Intake: Idea](intake.md#idea-as-captured); [research: Evidence Against the Idea](research.md#evidence-against-the-idea))
- **Rabbit holes**: Expanding to every read operation; promising compatibility with all clients or environments; unbounded result enrichment and traversal; turning an individual pilot into shared hosted infrastructure; adding writes because the underlying operation already exists.

### Option C — Focused read-and-change MCP offering

- **Sketch**: Users carry out a focused operational loop from an MCP client: inspect existing processes, start or cancel an instance where authorized, and understand the resulting state. This is closest to the original issue's proposed experience. It retains a narrow selection of existing operations while requiring users to distinguish authorization, submission, completion, and failure throughout that loop.
- **Appetite**: **Medium (weeks), tentative budget category with substantial uncertainty**. It would compete for more of that budget than Option B; no evidence establishes that the full loop fits. [NEEDS CLARIFICATION: Confirm the actual allowance and essential workflows; if they do not fit, reconsider scope rather than assuming a months-long commitment.]
- **Trade-offs**: Evaluates end-to-end operational value and the issue's state-changing use cases. It also takes on unresolved access, confirmation, waiting, partial-outcome, and support expectations before comparative demand is established. Existing c8volt operations are relevant prior work, but their presence does not prove equivalent behavior in an AI-client experience. ([Research: Data & Constraints](research.md#data--constraints))
- **Rabbit holes**: Full CLI parity; bulk mutations and complex recovery; broad authentication or multi-user administration; long-running workflow management; treating every version/client combination as an initial support requirement.

No buy option is presented as a verified substitute: research did not establish a Camunda alternative that meets the intended users' needs. Existing integrations remain an unresolved comparison under problem Q8; one unavailable repository URL is not evidence that alternatives are absent. ([Research: Prior Art](research.md#prior-art))

## Recommendation

Recommend **Option B as the next concept to consider at the decision gate**, conditional on identifying a real inspection workflow that the current CLI experience does not serve adequately. It tests the issue's central claim about direct agent access while keeping scope bounded. The presence of existing c8volt operations and an MCP ecosystem supports investigating the option; neither establishes user demand or a delivery estimate. ([Research](research.md#prior-art))

Evaluate it against the problem's candidate measures: setup effort, correct workflow completion, manual corrections, and accurate interpretation of complete, partial, or failed inspection results. Compare equivalent tasks and environments with Option A. The pilot cannot establish correctness or benefit for state-changing operations, so those outcomes remain unevaluated rather than being credited to a successful read-only experience. ([Problem: Success Metrics](problem.md#success-metrics))

Before a build commitment, the decision owner needs a named user/client and workflow, an agreed comparison and success threshold, and an owner-approved appetite. These are unresolved problem Q1, Q2, and Q9, not new approval requirements for writing this assessment. If no material gap can be demonstrated or no owner accepts the budget, retain Option A. If the user's essential need is specifically a mutation workflow, reconsider Option C rather than running an inspection pilot that cannot answer the actual question.

This recommendation deliberately narrows the original issue for learning. It does **not** remove run, cancel, or expectation workflows from issue #80, claim that a pilot completes the issue, or authorize implementation. The subsequent decision stage must resolve whether to adopt the narrower scope or pursue the original operational loop.

## Out of Scope (for the recommended option)

- Replacing the CLI or changing established human and automation workflows.
- Full CLI parity or new Camunda business operations.
- Starting, canceling, deleting, or otherwise mutating process state through the pilot; validating mutation-specific confirmation and completion behavior is deferred with those workflows.
- General-purpose shared hosting, multi-user administration, or organization-wide access management.
- Supporting every MCP client, deployment environment, or Camunda version by default; the pilot's actual compatibility boundary remains to be agreed.
- Exhaustive enrichment, unbounded traversal, or broad bulk-operation coverage beyond the agreed inspection workflow.
- Treating a pilot as a production-wide release or as satisfaction of all issue #80 acceptance expectations.

The first two exclusions are inherited from the [problem non-goals](problem.md#non-goals). The remaining exclusions are proposed boundaries for Option B, not established user requirements. If they exclude the intended user's essential workflow, the recommendation must change.

## Assumptions to Validate

- **A1 — Useful inspection need**: A named user has a meaningful read-only workflow whose current integration friction persists despite existing discovery, structured output, and automation. Validate against problem Q1 and Q3; otherwise Option B lacks a useful subject.
- **A2 — Comparative benefit**: Direct MCP access can improve at least one agreed measure without degrading outcome interpretation or existing workflows. Baselines and improvement thresholds remain unknown under Q2; no benefit is claimed yet.
- **A3 — Narrow environment suffices**: An agreed client and operating context can represent the pilot's intended use without broad hosting or compatibility commitments. Resolve Q4 and Q6 before describing the pilot as representative.
- **A4 — Existing access is appropriate**: The user is authorized to inspect the relevant environments and tenant scopes, and the selected client is acceptable for the returned operational information. Read-only scope does not resolve Q5 automatically.
- **A5 — Results remain meaningful**: Users can interpret missing, partial, failed, and completed inspection results within the bounded workflow. Resolve relevant Q7 expectations; mutation outcomes remain deferred, not implicitly solved.
- **A6 — Ownership and budget exist**: Someone accepts the pilot's explicit spending limit, acceptance responsibility, and compatibility/support obligations. Resolve Q9; the appetite label is not an approved commitment.
- **A7 — The baseline is fair**: Comparison uses current c8volt capabilities and considers any existing integrations actually available to the user. Resolve Q8 rather than assuming MCP is uniquely necessary.
- **A8 — Scope narrowing is acceptable**: The decision owner accepts that a read-only pilot is a partial exploration of issue #80. If run/cancel workflows are the primary need, reconsider Option C and its additional unresolved responsibilities.

These assumptions carry forward all nine problem questions. None is asserted as validated, and no architecture, interface contract, or implementation task breakdown is selected here.
