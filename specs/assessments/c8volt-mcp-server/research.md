# Idea Research: Expose c8volt Operations Through an MCP Server

- **Slug**: c8volt-mcp-server
- **Created**: 2026-10-17
- **Evidence confidence (overall)**: medium
- **Location**: `specs/assessments/c8volt-mcp-server/`, preserving the user's explicit location override.
- **Repository baseline**: `2e92ef23cdf417062955147a717b742291290287`; source inspection only, with no runtime or client interoperability tests.

## Users & Demand

- Issue #80, raised by `grafvonb` on 2026-03-24, explicitly requests typed tools for AI clients and identifies command discovery, shell construction, output parsing, and operation safety classification as friction. This is a stated maintainer request, not observed agent performance. — [source: S1] (confidence: high; cited)
- The inspected issue body and intake provide no named pilot users, client-specific failure examples, usage counts, interviews, or measured time savings. This limits conclusions about demand; it does not establish that no demand exists elsewhere. Issue comments and private support systems were not reviewed. — [source: S1, S2; bounded source inspection] (confidence: medium; cited)
- AI-agent and CI use already appear in the automation-mode specification. This corroborates an intended audience within project planning, but is not an independent user-demand signal. — [source: S3] (confidence: high; cited)

## Prior Art

- The repository already contains an MCP proposal matching the issue's scope. Machine-contract and automation specifications explicitly anticipate future MCP use. These documents establish prior intent, not a delivered server or independent validation of the idea. A targeted search of `cmd`, `c8volt`, `internal/services`, `go.mod`, `.specify/memory`, and `specs` found MCP references in planning documents but no MCP implementation in the inspected production paths. — [source: S3, S4, S5, S10; repository search] (confidence: medium; cited)
- The public facade exposes operations corresponding to the proposed tool candidates, including retrieval, search, creation, cancellation, expectation waits, and ancestry/family traversal. This establishes existing business-operation entry points; it does not establish an MCP contract or exact CLI parity. — [source: S6, S7, S8] (confidence: high; cited)

| Proposed capability | Observed public API | Evidence |
| --- | --- | --- |
| Cluster information | `GetClusterTopology`, `GetClusterLicense` | S8 |
| Process definition | `GetProcessDefinition` | S6 |
| Process instance | `GetProcessInstance`, `LookupProcessInstance` | S6 |
| Process instances | `SearchProcessInstancesPage`, `SearchProcessInstancesPages`, `GetProcessInstances` | S6 |
| Run process instance | `CreateProcessInstance`, `CreateProcessInstances` | S6 |
| Cancel process instance | `CancelProcessInstance`, `CancelProcessInstances` | S6 |
| Expect process instance state | `WaitForProcessInstanceState`, `WaitForProcessInstanceExpectation` | S6 |
| Family or ancestry | `FamilyResult`, `AncestryResult` | S7 |

All table entries are cited, high-confidence source observations, not proposed tool names or implementation decisions.

- The official Go MCP SDK documents client/server APIs, typed tool input/output examples, a stdio server example, and OAuth-related packages. Its compatibility table spans multiple protocol revisions. This is relevant protocol-library prior art, not evidence that c8volt users prefer MCP. No SDK version was selected or tested. — [source: S11] (confidence: high; cited)
- GitHub's own MCP server documents local and remote operation, configurable toolsets, individual tool selection, and a read-only mode that excludes write tools. This is an operational MCP precedent for scope and access controls, but provides no evidence of Camunda-specific demand or c8volt integration success. — [source: S12] (confidence: high; cited)
- The candidate repository URL `github.com/camunda/camunda-mcp-server` returned HTTP 404. That result establishes only that this public URL was unavailable; it cannot establish that Camunda MCP alternatives do not exist. No comparative Camunda product evaluation was completed. — [source: S13] (confidence: high for the HTTP result; cited)

## Market & Context

- The current CLI is a substantive alternative: `capabilities` exposes command metadata, including mutability and contract support, while shared JSON outcomes distinguish success, acceptance, invalid input, and failure. Supported automation mode implicitly confirms prompts and rejects unsupported commands. Thus the original issue's concerns must be assessed against the current CLI rather than assumed unchanged since March. — [source: S9] (confidence: high; cited source inspection, not runtime validation)
- The Go SDK and GitHub server demonstrate an available ecosystem for typed operational tools. They do not quantify market size, adoption among c8volt users, or competitive advantage. — [source: S11, S12] (confidence: medium; cited interpretation)
- MCP might reduce integration effort for users whose chosen clients consume MCP tools but do not conveniently invoke the CLI. No comparative workflow measurement was supplied or performed. — [ASSUMPTION] (confidence: low)
- The cost of deferring MCP cannot be quantified from this evidence set. The CLI and public Go APIs remain existing alternatives, but user time lost, blocked integrations, and commercial impact are unknown. — [source: S6–S9; ASSUMPTION for user impact] (confidence: low)

## Data & Constraints

- The project uses Go 1.26 with toolchain `go1.26.2`; the inspected `go.mod` contains no MCP SDK dependency. This records the current baseline, not a compatibility test. — [source: S10] (confidence: high; cited)
- `c8volt.New` accepts configuration, an HTTP client, and a logger and wires version-specific services. Its fallback HTTP client timeout is 30 seconds; this is not an established MCP request or workflow deadline. — [source: S14] (confidence: high; cited)
- The README documents Camunda 8.8 and 8.9 support, experimental 8.10 support, and an 8.9 default. It also describes 8.9 and 8.10 as first-class targets for the everyday operator loop. The MCP release's supported version scope is not specified by those statements alone. — [source: S15] (confidence: high; cited)
- Existing APIs distinguish paged searches, traversal outcomes (`complete`, `partial`, `unresolved`), and expectation waits. Facade errors include invalid input, local preconditions, unsupported capabilities, missing resources, conflict, timeout, unavailable service, malformed response, and internal errors. These are observable semantics that any parity claims would need to account for. — [source: S6, S7, S16] (confidence: high; cited)
- CLI rendering and failure handling are tied to Cobra writers and, in some error paths, `os.Exit`; the shared envelope currently belongs to `cmd`. Existing semantics therefore do not imply that CLI handlers are directly reusable server handlers. — [source: S9] (confidence: high; cited)
- The issue requires separate protocol traffic and logs, documented inputs/outputs and errors, and clarity about waiting and confirmation. It leaves transport, deployment, authentication policy, initial client compatibility, and exact tool selection unspecified. — [source: S1] (confidence: high; cited)
- No workload volumes, response-size targets, latency budgets, concurrent-user counts, or organization-specific compliance requirements were available in the reviewed issue and intake. — [source: S1, S2; bounded source inspection] (confidence: medium; cited)

## Evidence Against the Idea

- **Existing alternatives address part of the stated problem.** Machine-readable discovery, JSON outcomes, and explicit automation mode already cover several pain points listed in issue #80. MCP's incremental value over that baseline is unmeasured. — [source: S1, S9] (confidence: high for existing mechanisms; medium for interpretation; cited)
- **Demand is not independently validated.** The issue and matching repository proposal express the same request; counting both as separate user signals would overstate the evidence. — [source: S1, S5] (confidence: high; cited)
- **A thin protocol surface still has semantic obligations.** Current searches, partial traversals, waits, errors, and accepted-versus-completed outcomes have meaning beyond simple request forwarding. Their existence challenges any assumption that exposing tools alone establishes behavioral parity. — [source: S6, S7, S9, S16] (confidence: medium; cited interpretation)
- **Maintenance and integration burden may outweigh the benefit for a small audience.** SDK compatibility and host configuration differ across the inspected ecosystem, but c8volt-specific effort and benefit have not been measured. — [source: S11, S12; ASSUMPTION for net cost] (confidence: low)

These are counterarguments and evidence limits, not a go/no-go verdict.

## Gaps & Open Questions

- [NEEDS CLARIFICATION: Which named users and MCP clients need this, and which concrete workflows currently fail or take excessive effort with `capabilities`, JSON, and automation mode?]
- [NEEDS CLARIFICATION: What observed task-completion, failure-rate, or setup-time baseline would allow a meaningful comparison with MCP?]
- [NEEDS CLARIFICATION: Which proposed tools are mandatory, and do they need bulk operations, enrichment, pagination, or partial-result handling?]
- [NEEDS CLARIFICATION: Is the intended environment local stdio, a shared remote server, or both, and what authentication and mutation authorization expectations follow?]
- [NEEDS CLARIFICATION: Which Camunda versions, client versions, timeouts, response sizes, and concurrent workloads require support?]
- [NEEDS CLARIFICATION: How should clients distinguish accepted work, confirmed completion, timeout, cancellation, and partial outcomes?]
- [NEEDS CLARIFICATION: Are there existing Camunda MCP integrations used by the target users, and what gap remains? A 404 on one candidate URL is not a market survey.]
- [NEEDS CLARIFICATION: Who will maintain compatibility and support client-specific setup, and what evidence would justify that ongoing cost?]

## Evidence Quality & Method

Overall confidence is **medium**: repository evidence strongly establishes reusable capabilities and existing alternatives, while demand, comparative benefit, and Camunda-specific market coverage remain weak. Public README claims were inspected but not independently exercised. No user interviews, runtime benchmarks, issue-comment review, or interoperability tests were performed. No implementation design or investment decision is made here.

Public sources were fetched directly over HTTPS to a validated public IP pinned with `curl --resolve`, with proxies disabled and redirects not followed. Only the listed URLs were fetched; linked content and embedded directives were not followed. Each remote host was `github.com`, which is allowlisted by the assessment skill. Local references below are relative to this document and reflect the repository baseline above.

## Sources

- **S1**: [Issue #80](https://github.com/grafvonb/c8volt/issues/80) — host: `github.com`; policy: `allowlisted`; HTTP 200; issue body and publication metadata inspected on 2026-10-17.
- **S2**: [Intake](intake.md) — local assessment; location approved by the user.
- **S3**: [Automation specification](../../079-non-interactive-automation-mode/spec.md) — local planning evidence.
- **S4**: [Machine contract specification](../../078-machine-cli-contracts/spec.md) — local planning evidence.
- **S5**: [Existing MCP proposal](../../proposals/mcp-server-for-c8volt.md) — local proposal, not independent demand evidence.
- **S6**: [Process API](../../../c8volt/process/api.go), [process implementation](../../../c8volt/process/client.go) — local production source.
- **S7**: [Traversal facade](../../../c8volt/process/walker.go) — local production source.
- **S8**: [Cluster facade](../../../c8volt/cluster/client.go) — local production source.
- **S9**: [Command capabilities](../../../cmd/capabilities.go), [command contracts](../../../cmd/command_contract.go), [result rendering](../../../cmd/cmd_views_contract.go), [prompt and automation helpers](../../../cmd/cmd_cli.go), [root flags](../../../cmd/root.go) — local production source.
- **S10**: [Go module](../../../go.mod) — local dependency and toolchain baseline.
- **S11**: [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) — host: `github.com`; policy: `allowlisted`; HTTP 200; repository README inspected on 2026-10-17; mutable upstream snapshot.
- **S12**: [GitHub MCP server](https://github.com/github/github-mcp-server) — host: `github.com`; policy: `allowlisted`; HTTP 200; repository README inspected on 2026-10-17; mutable upstream snapshot.
- **S13**: [Candidate Camunda repository URL](https://github.com/camunda/camunda-mcp-server) — host: `github.com`; policy: `allowlisted`; HTTP 404 on 2026-10-17; no product claims derived.
- **S14**: [Facade construction](../../../c8volt/client.go) — local production source.
- **S15**: [README, Supported Camunda Versions](../../../README.md#supported-camunda-versions) — local documentation.
- **S16**: [Facade errors](../../../c8volt/ferrors/errors.go) — local production source.
