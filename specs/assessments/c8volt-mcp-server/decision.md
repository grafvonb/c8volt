# Decision: Focused Agent Access to Camunda Operations

- **Slug**: c8volt-mcp-server
- **Decided**: 2026-10-17
- **Verdict**: needs-clarification
- **Artifacts reviewed**: [intake.md](intake.md), [research.md](research.md), [problem.md](problem.md), [concept.md](concept.md)
- **Additional context reviewed**: [Project constitution](../../../.specify/memory/constitution.md), version 2.0.0.
- **Location**: `specs/assessments/c8volt-mcp-server/`, preserving the user's explicit location override.

## Scorecard

| Criterion | Rating | Justification |
| --- | --- | --- |
| Problem validity | weak | The issue states a plausible integration problem, but no concrete user workflow establishes material friction after accounting for current CLI discovery, structured results, and automation. [Research](research.md#users--demand), [problem](problem.md#problem-statement). |
| Evidence strength | weak | Technical reuse is well supported, but user demand, comparative benefit, and actual client barriers remain assumptions; these are the evidence needed to justify proceeding. [Research](research.md#evidence-quality--method). |
| Value vs. inaction | unknown | Existing alternatives remain available; lost time, blocked workflows, improvement thresholds, and ongoing support cost are unmeasured. [Problem](problem.md#cost-of-inaction). |
| Feasibility / appetite | unknown | Option B is bounded and has relevant prior art, but its weeks-sized appetite is explicitly provisional, with no agreed allowance, owner, or demonstrated fit. [Concept](concept.md#options). |
| Strategic fit | adequate | Improving agent use of existing operations fits the recorded project direction; preserving CLI behavior and operational truth aligns with constitution principles I, II, and V. Added complexity still needs justification against existing alternatives. [Research](research.md#prior-art), [constitution](../../../.specify/memory/constitution.md). |
| Risk posture | weak | Read-only scope limits mutation exposure, but client/environment boundaries, access expectations, result interpretation, ownership, and acceptance of scope narrowing remain unresolved. Identified risks are not yet demonstrated mitigations. [Concept](concept.md#assumptions-to-validate). |

## Verdict & Rationale

**Needs clarification.** The record supports a technically plausible direction, but does not yet justify specifying a new integration surface. Both problem validity and decision-relevant evidence are weak: the original request is real, while the remaining user pain and benefit over current automation have not been demonstrated. The unknown value and appetite scores prevent an honest claim that the recommended pilot is worth its cost or fits an accepted budget.

This does not contradict research's overall **medium** confidence. That confidence includes strong source evidence about existing capabilities and ecosystem precedents; the gate specifically needs evidence that a useful user problem remains and that the proposed scope addresses it. Technical availability cannot substitute for that evidence.

Option B remains a conditional candidate, not an accepted delivery scope. Its inspection-only experience cannot establish value for users whose essential need is to start or cancel processes, and it would not complete issue #80. The scope difference must be resolved explicitly rather than silently dropping the original state-changing workflows. [Concept recommendation](concept.md#recommendation).

A **kill** is not justified either: the record does not show that the need is absent, that an existing alternative fully satisfies it, or that costs necessarily exceed value. The missing evidence is specific enough to seek. No specification handoff is issued by this decision.

## Required Clarifications

The five items below consolidate the blocking questions. A maintainer's own concrete workflow can be valid user evidence; a broad market study is not required.

1. **Actual user and unmet workflow — research; problem Q1 and Q8.** [NEEDS CLARIFICATION: Who will use this, in which AI client, for what concrete Camunda workflow, and what remains difficult or impossible with current c8volt automation or an integration they already use?] Record a reproducible example, an observed attempt, or a specific first-hand account. Distinguish observed failure from preference for a different interface.
2. **Benefit and acceptance — research, then define; problem Q2 and Q7.** [NEEDS CLARIFICATION: What happens in that workflow today, and what observable improvement would make a pilot worthwhile?] Establish a baseline and an agreed measure such as setup effort, corrections, correct completion, or interpretation accuracy. A demonstrably blocked workflow is also meaningful evidence; no invented numerical target is required.
3. **Scope choice — shape; problem Q3.** [NEEDS CLARIFICATION: Is a read-only investigation useful enough to justify Option B, or are start/cancel and subsequent state verification essential to the first useful outcome?] Identify the indispensable workflow boundaries and explicitly accept or reject the pilot's narrowing relative to issue #80. Reconsider Option C if inspection cannot answer the user's need.
4. **Operating and access boundaries — define, then shape; problem Q4–Q7.** [NEEDS CLARIFICATION: Which client, environment, Camunda version, authorized data/tenant scope, and expected workload must the initial experience support, and which outcome distinctions matter to that workflow?] Define a bounded operating context; do not require every compatibility or protocol detail to be designed at this gate. If mutations are essential, their authorization and confirmation expectations must be included.
5. **Owner and appetite — shape; problem Q9.** [NEEDS CLARIFICATION: Who owns acceptance and ongoing support, what actual time or effort budget is available, and what scope will be excluded to stay within it?] The concept's provisional appetite labels are not commitments or estimates, and no staffing or funding requirement is inferred beyond identifying responsibility and an explicit limit.

## Next Stage & Reconsideration

- **Primary revisit stage**: research — `$speckit-assess-research slug=c8volt-mcp-server`, retaining `specs/assessments/` as the assessment root and bringing the concrete workflow evidence above.
- Reconcile that evidence into the problem definition and concept where it changes success measures, boundaries, or the recommended option. Existing artifact overwrite requirements still apply; this decision does not authorize replacing them.
- Re-run the decision gate after the blocking answers are recorded. A go requires at least adequate problem validity and evidence, plus a recommended concept whose useful scope, ownership, and appetite are defensible. Further specification details may remain open; the value proposition and first useful outcome may not.

The assessment remains open. This document neither closes GitHub issue #80 nor changes its requirements, and it does not start implementation.
