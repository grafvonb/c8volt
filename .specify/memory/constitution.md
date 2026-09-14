# c8volt Constitution

## Core Principles

### I. Operational Proof Over Intent
Every user-facing command MUST verify the operational outcome it claims unless an
explicit opt-out flag already exists for that workflow. Features MUST preserve the
project's "done is done" standard: report success only after the observable target
state is reached or clearly state that confirmation was skipped. This keeps c8volt
trustworthy for administrators and automation.

### II. CLI-First, Script-Safe Interfaces
Behavior MUST be exposed through stable CLI commands, flags, exit codes, and text or
structured output that work in interactive and scripted environments. New commands
MUST follow existing Cobra layouts, naming, and flag propagation rules already used in
the repository. Changes that break documented command behavior require an explicit
compatibility note in the plan and accompanying documentation updates.

### III. Validation Proportional to the Change
Validation MUST address the behavior and risk of the actual diff. A commit, merge,
Spec Kit command, or documentation update alone MUST NOT trigger runtime tests.

- Documentation-only changes (including specifications, plans, tasks, governance,
  prose, and comments with no executable effect) MUST use relevant lightweight
  checks, such as diff review, formatting, and changed-link validation. Do not run
  `go test` or `make test` solely for these changes. If documentation generation or
  examples are affected, run the relevant generator or example check only when it
  provides useful validation of the change.
- For executable code, configuration, dependency, build, or generated-code changes,
  contributors MUST select the smallest checks that meaningfully cover the affected
  behavior. Start with targeted tests in the changed package or execution path.
  Reuse existing coverage; add or update tests when behavior changes or a regression
  lacks coverage. Do not add tests merely to mirror implementation or satisfy a
  per-file quota.
- Run the full race-enabled suite (`make test`) when a change affects shared runtime
  behavior across packages, concurrency, dependencies or generated clients with
  broad impact, or when targeted failures leave broader regressions unresolved.
  A focused change adequately covered by targeted checks does not require the full
  suite. State the concrete reason when selecting broader validation.
- After relevant checks pass, do not repeat or broaden them merely to commit, merge,
  or mark a task complete. Repeat checks only when subsequent relevant changes,
  failures, or new evidence invalidate the earlier result. Documentation edits after
  a successful test run do not invalidate that result.
- Report the checks actually performed and any material validation gaps. Do not
  present skipped, unavailable, or failing checks as passing. When automated coverage
  is impractical, explain the gap and use a focused manual check where useful.

These rules replace older blanket test-before-commit wording in feature artifacts
or workflow guidance. A specific acceptance test remains required for the behavior
it covers; it does not make a planning-only or documentation-only commit require
runtime tests. This keeps validation useful without spending time on unrelated work.

### IV. Documentation Matches User Behavior
User-visible command changes MUST update `README.md` and any relevant generated CLI
documentation in the same unit of work. Examples, flags, defaults, and caveats MUST
match shipped behavior so operators do not need to infer hidden rules from code. If a
change is internal-only, documentation may stay unchanged, but that decision MUST be
explicit in the plan or task list.

### V. Small, Compatible, Repository-Native Changes
Work MUST reuse existing repository patterns, dependencies, and command structures
before introducing new abstractions. Feature slices, PRDs, and tasks MUST stay small,
dependency-ordered, and independently verifiable so work can be implemented in short
iterations. New complexity requires a written justification in the implementation plan,
including why simpler repository-native alternatives were rejected.

## Project Constraints

- The canonical implementation stack is Go with Cobra-based CLI commands and the
  existing internal service layout.
- Command trees and subcommands MUST mirror established repository structure rather
  than inventing parallel hierarchies.
- Configuration-sensitive commands MUST honor root-level flag resolution and explicit
  kubeconfig or config-file persistence rules documented in repository guidance.
- Changes that affect operational state transitions, process polling, or bulk actions
  MUST preserve deterministic exit behavior and user-facing status reporting.

## Delivery Workflow

- Specifications MUST describe independently testable user stories ordered by
  delivery priority.
- Implementation plans MUST include a Constitution Check that confirms operational
  verification, CLI compatibility, proportionate validation, documentation impact, and any
  justified complexity.
- Task lists MUST include concrete validation tasks and documentation tasks whenever a
  story changes user-visible behavior.
- Reviews MUST block merges when constitution requirements are unmet, even if code is
  otherwise functional.

## Governance

This constitution supersedes informal local practice for planning, implementation,
review, and documentation in this repository. Amendments require: (1) updating this
file, (2) recording sync impact for review, and (3) identifying conflicting dependent
guidance. Dependent workflows MUST read the current constitution; copied older rules
do not override it. Remove the temporary sync report before committing the amendment. Versioning follows semantic rules for
governance: MAJOR for incompatible principle changes or removals, MINOR for new
principles or materially stronger obligations, PATCH for clarifications that do not
change expected behavior. Compliance review is required in every feature plan and code
review, with unresolved exceptions documented under the plan's complexity or risk
tracking section.

**Version**: 2.0.0 | **Ratified**: 2026-03-15 | **Last Amended**: 2026-09-13
