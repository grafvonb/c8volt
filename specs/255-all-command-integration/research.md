# Research: All-Command Integration Suite

## Decision: Use `capabilities --json` As Command Inventory Oracle

**Rationale**: The command contract is already generated from the live Cobra tree and includes paths, aliases, flags, output modes, mutation classification, contract support, and automation support. Using it prevents a stale hand-maintained command list from becoming the suite's source of truth.

**Alternatives considered**: Parsing command source files was rejected because it would duplicate Cobra behavior and miss runtime annotations. Hard-coding the 55-command list alone was rejected because future command changes would not be detected reliably.

## Decision: Run The Built CLI As A Subprocess

**Rationale**: The integration suite must validate the same path operators use: root config loading, persistent flags, profile selection, auth, stdout, stderr, prompts, exit codes, and render modes. Subprocess execution catches wiring bugs that direct facade calls would miss.

**Alternatives considered**: Calling command handlers directly was rejected because global flag state and config bootstrap behavior are core to the suite's purpose. Calling facades directly was rejected because facade tests already cover that level and would not prove CLI behavior.

## Decision: Use Default Local Configuration Only

**Rationale**: The suite is meant to prove real local operator configuration. Generated configs and auth overrides would hide configuration problems and violate the user's requirement.

**Alternatives considered**: Reusing existing env-driven integration helpers was rejected for this suite because they can override auth mode and config source. Passing `--config` to a generated fixture config was rejected because it would test a separate setup.

## Decision: Treat Selected Clusters As Disposable And Dirty-Tolerant

**Rationale**: Release validators may run against a clean cluster or an already dirty local cluster. The suite should not require exclusive ownership and is explicitly allowed to mutate pre-existing data in the chosen disposable target.

**Alternatives considered**: Cleaning the cluster before the suite was rejected because it would both hide dirty-cluster behavior and itself be a broad destructive precondition. Requiring empty global counts was rejected because unrelated cluster data is valid.

## Decision: Prefer c8volt Commands And Embedded BPMN For Setup

**Rationale**: The suite should prove c8volt can create and manipulate the data it later inspects. Embedded BPMN keeps fixtures versioned and repository-owned.

**Alternatives considered**: Creating all setup through Camunda APIs was rejected because it would bypass c8volt workflows. Editing existing embedded BPMN in place was rejected because fixtures are shared assets and changes could destabilize existing tests.

## Decision: Record Setup Gaps As Proposal Reports

**Rationale**: Some states may require APIs or fixtures that c8volt cannot currently produce. Recording those gaps creates actionable product or fixture proposals without blocking the integration suite from covering existing commands.

**Current status**: This was the original 255 design. It is now deprecated by `specs/integration-test-responsibility.md`; future integration work should keep runtime evidence in tests and maintain missing setup or fixture needs in spec-owned gap artifacts.

**Alternatives considered**: Failing immediately on every unavailable setup path was rejected because it would prevent broad coverage. Silently using direct APIs was rejected because it would hide missing command capabilities.

## Decision: Keep Artifacts Under `integration/`

**Rationale**: The suite is harness material, not public documentation or normal feature implementation context. Keeping rules, code, and evidence under `integration/` aligns with existing guardrails and avoids generated-doc confusion.

**Alternatives considered**: Placing the rules under `docs/` was rejected because `docs/` contains generated and public-facing assets. Placing the rules under `specs/` only was rejected because the context must be reusable for reruns beyond this feature.
