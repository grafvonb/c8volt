# Feature Specification: Advisory Camunda Version Check

**Feature Branch**: `222-camunda-version-check`

**Created**: 2026-10-30

**Status**: Draft

**Input**: [GitHub issue #222: add optional Camunda version check before API commands](https://github.com/grafvonb/c8volt/issues/222)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Notice a Version Mismatch Without Interrupting Work (Priority: P1)

As an operator, I want a warning when the connected gateway differs from my configured Camunda version so that I can correct my configuration while continuing my requested operation with the version I selected.

**Why this priority**: The warning makes a potentially confusing configuration mismatch visible without introducing a new compatibility gate or silently changing behavior.

**Independent Test**: Run a command against isolated gateway responses with matching and mismatched versions, and compare its operation, results, and exit status with the check disabled.

**Acceptance Scenarios**:

1. **Given** configured version 8.9 and gateway version 8.9.3, **When** an API-backed command reaches its first operational request, **Then** it checks the gateway once immediately beforehand and continues without a version warning.
2. **Given** configured version 8.9 and gateway version 8.10.3, **When** the check completes, **Then** it emits one WARN message, `version mismatch: configured 8.9, gateway 8.10.3; results may be incorrect; match config to gateway unless mismatch is intentional`, and continues using configured version 8.9.
3. **Given** a gateway whose major version differs from the configured major version, **When** the check completes, **Then** the same mismatch policy applies; the command does not switch versions or reject execution because of the check.
4. **Given** a command that paginates, processes multiple resources concurrently, or polls repeatedly, **When** it performs its operational requests, **Then** one version check precedes all of them and no page, resource, retry, or polling cycle triggers another check.
5. **Given** two separate command invocations using the same configuration, **When** each reaches an operational request, **Then** each performs its own check; the earlier result does not suppress the later check.
6. **Given** configured version 8.8, 8.9, or 8.10, **When** the matching, mismatch, and failed-check scenarios run, **Then** they follow the same advisory contract; existing invalid or unsupported configuration handling remains unchanged.

---

### User Story 2 - Continue When the Check Is Unavailable (Priority: P1)

As an operator, I want an unavailable or slow version check to produce a concise warning and let my operation proceed so that access restrictions or metadata failures do not become a new reason for command failure.

**Why this priority**: An advisory feature must remain bounded and must not prevent an otherwise usable operation.

**Independent Test**: Use isolated denied-access, stalled, malformed, and failed gateway responses, followed by successful and failed operational responses, and verify the warning, waiting bound, and original command outcome.

**Acceptance Scenarios**:

1. **Given** configured version 8.9 and denied access to version information, **When** the check fails, **Then** the command emits one WARN message, `version check: access denied; using configured 8.9`, and attempts its operation without retrying the check.
2. **Given** a version check that does not respond, **When** its short time limit expires, **Then** the command emits one WARN message, `version check: timeout; using configured 8.9`, and proceeds if the invocation remains active.
3. **Given** missing or unusable gateway version information, an unavailable version lookup, or another check failure, **When** the check ends, **Then** the command emits one compact warning with a bounded reason and continues with the configured version; it does not treat unknown information as a match.
4. **Given** a failed check followed by an operational failure or partial failure, **When** execution ends, **Then** the original operational failure class, partial results, and exit behavior remain intact; the advisory warning neither replaces nor duplicates the operational error.
5. **Given** cancellation or expiration of the overall invocation during the check, **When** execution stops, **Then** the command preserves its existing cancellation behavior and does not start new work after cancellation. Expiration of only the check's time limit does not cancel the operation.

---

### User Story 3 - Control the Check and Preserve Script Output (Priority: P2)

As an operator or automation author, I want to disable the check when needed and retain existing output contracts so that the advisory behavior does not interfere with local work or scripts.

**Why this priority**: Explicit control and clean result streams make default-on checking suitable for existing workflows.

**Independent Test**: Exercise enabled, disabled, local-only, locally rejected, and remote-discovery executions using isolated dependencies, while separately capturing results and configured diagnostic destinations.

**Acceptance Scenarios**:

1. **Given** an API-backed command with `--no-version-check`, **When** it executes, **Then** it makes no advisory version request and emits no version-check warning, while its normal operation remains unchanged. Explicit `--no-version-check=false` enables checking even if base/profile configuration or the environment disables it.
2. **Given** help, local application-version display, configuration inspection, or another local-only command, **When** it runs, **Then** it makes no advisory version request. A command rejected by local validation or ended before any operational request also makes none.
3. **Given** a dry-run or empty-scope workflow, **When** it needs remote discovery, **Then** it checks once before that discovery; if the workflow is entirely local, it does not check. The check adds no mutation or confirmation prompt.
4. **Given** a mismatch or failed check with human, JSON, or keys-only output, **When** the operation completes, **Then** results retain their established format and the warning uses the configured WARN logger, including inherited destination, formatting, and activity behavior. No warning contaminates result stdout.
5. **Given** quiet mode or a logger level that suppresses WARN, **When** a mismatch or failed check occurs, **Then** the warning is suppressed without disabling the check or changing explicitly requested machine output.
6. **Given** ordinary or verbose-only logging and a failed check containing raw transport details, **When** diagnostics are emitted, **Then** the warning contains no raw URL, method, body, credentials, resource details, or transport chain. Existing redacted DEBUG diagnostics remain available through either debug enablement path, subject to established precedence.
7. **Given** command help and user documentation, **When** an operator looks up version checking, **Then** they can find its default-on behavior, opt-out, warning-only policy, bounded waiting, and explicit statement that the configured version is never changed automatically.

### Edge Cases

- Patch-only changes are matches; numeric major/minor comparison distinguishes 8.9 from 8.10 without textual or decimal ordering mistakes. Broker versions do not determine the gateway comparison.
- Missing, blank, malformed, or excessively long observed version values produce a bounded failure reason when they cannot safely establish major/minor compatibility; they must not inject multiline diagnostics.
- A check attempt that fails still consumes the invocation's single check. Concurrent operational work must not race into multiple checks or start before the check finishes.
- A remotely discovered empty result still required an operational request and therefore a check; a wholly local empty preview did not. Rendering must not trigger another check.
- Commands explicitly requesting gateway version or topology remain API-backed commands. Their requested result remains independent of the advisory outcome, including when the advisory lookup fails. Disabling the advisory check does not suppress an explicitly requested gateway-version lookup.
- Existing invalid or unsupported configured-version rejection remains unchanged. The advisory check is not a way to bypass configuration validation.
- An operational failure after a mismatch remains an operational failure; a successful operation after a failed check remains successful. A warning never becomes an additional structured error envelope.
- Warning suppression does not imply check suppression. Unrelated existing diagnostics are outside the single version-warning count.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: API-backed CLI commands MUST enable the advisory check by default and attempt it once per invocation, immediately before the first operational request. All operational requests in that invocation MUST wait for that attempt to finish or reach its limit.
- **FR-002**: The comparison MUST use the effective configured Camunda major/minor version and the observed gateway major/minor version, ignoring patch differences and broker versions. A match MUST emit no version warning or normal/verbose success summary; DEBUG may report the outcome as specified in FR-016.
- **FR-003**: A mismatch MUST issue one WARN message identifying the configured and observed gateway versions, warning that results may be incorrect, and requesting correction unless the mismatch is intentional. Check failure MUST issue one WARN message with a concise bounded reason and the configured version being retained.
- **FR-004**: Warning message text MUST follow `version mismatch: configured <configured>, gateway <observed>; results may be incorrect; match config to gateway unless mismatch is intentional` or `version check: <reason>; using configured <configured>`. Denied access MUST use `access denied`; check timeout MUST use `timeout`. Logger-generated severity and timestamps MUST NOT be duplicated in message text.
- **FR-005**: The check MUST finish or time out within a two-second waiting budget, reduced by any shorter applicable request timeout or remaining invocation deadline. It MUST make no retries. Its own timeout MUST NOT cancel an otherwise active operational workflow.
- **FR-006**: A mismatch or check failure MUST NOT independently fail the command, change its exit code, alter requested operations, or select another Camunda adapter. Effective configuration MUST remain unchanged. Existing overall cancellation and deadline behavior MUST remain authoritative.
- **FR-007**: Pagination, concurrent batches, retries of operational requests, polling, and multiple operations within one invocation MUST NOT repeat the check. Each new invocation MUST have a fresh check decision, including after a previous check failed.
- **FR-008**: `--no-version-check` MUST disable only the advisory request and its warnings for that invocation. The flag MUST be available to all API-backed commands; explicit false MUST enable checking even when a lower-precedence source disables it. The equivalent `app.no_version_check` boolean MUST be supported in base YAML and selected profiles, with `C8VOLT_APP_NO_VERSION_CHECK` as its environment variable. Precedence MUST be explicit CLI flag > environment > selected profile > base config > default false. Explicit profile/environment false values MUST override lower-precedence true values; omitted profile values MUST inherit. Effective config and templates MUST expose the setting, and invalid boolean values MUST fail configuration resolution. It MUST NOT suppress an explicitly requested gateway-version result.
- **FR-009**: Local-only execution, local validation failures, and execution paths ending before any operational request MUST make zero advisory version requests. API-backed discovery, including discovery for previews or eventual empty results, MUST receive the normal single check.
- **FR-010**: Warnings MUST use the configured command-context WARN logger and honor its level, format, destination, inherited settings, activity routing, quiet suppression, and existing flag precedence. They MUST NOT use direct printing or a separate logger, and MUST NOT alter plain interactive prompts.
- **FR-011**: Human and machine result contracts MUST remain unchanged. JSON MUST retain exactly one existing command envelope where supported, including partial failures and empty results; keys-only MUST retain exactly its result keys or zero bytes. Version warnings MUST NOT become result payloads or trigger a second error renderer.
- **FR-012**: Ordinary and verbose-only diagnostics MUST exclude raw transport errors, URLs, HTTP methods, bodies, credentials, resource details, and joined error chains. Existing redacted DEBUG diagnostics MUST remain independently available through `--debug` and configured DEBUG level, without diagnostic records on result stdout.
- **FR-013**: Unavailable, missing, or unparseable gateway version information MUST be treated as check failure, not as a match or a reason for automatic version selection. Diagnostic values MUST be bounded and safe for a single-line message.
- **FR-014**: The same advisory contract MUST apply across supported Camunda versions 8.8, 8.9, and 8.10. Existing configuration validation and unsupported-version rejection MUST remain unchanged.
- **FR-015**: Command help, applicable command metadata, README guidance, and generated CLI documentation MUST describe the default, opt-out, comparison policy, bounded wait, warning-only failures, and continued use of configured versions.

- **FR-016**: Each attempted check MUST emit one summary through the configured DEBUG logger, independently of verbose and respecting quiet/flag precedence. It MUST include result, configured and gateway versions, retained adapter version, and major/minor comparison policy. Available topology facts MUST include sorted unique validated broker versions, number of reported brokers, cluster size, partition count, and replication factor. Failed checks MUST include the fixed failure category. Missing/unsafe versions MUST use `unknown`; unavailable or nonpositive counts MUST be omitted. Derive all facts from the existing lookup without additional requests or changes to WARN messages, comparison, command results, or exit behavior.

### Key Entities *(include if feature involves data)*

- **Configured version**: The effective Camunda major/minor selection already resolved for the invocation; remains authoritative for its operations.
- **Observed gateway version**: The gateway's reported version, including patch information when available; only its major/minor components determine a match.
- **Invocation check outcome**: Not needed, disabled, matched, mismatched, or failed, associated with one invocation and at most one advisory attempt.
- **Advisory warning**: A compact mismatch or failure explanation identifying the version retained, whose visibility follows the configured logging policy.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In every eligible invocation scenario, exactly one advisory attempt precedes the first operational request, even with multiple pages, concurrent resources, or polling; disabled and wholly local scenarios perform zero attempts.
- **SC-002**: Every matching major/minor scenario produces zero version warnings; every mismatch or failure scenario produces exactly one when WARN is enabled, and zero when suppressed. All three source-issue example messages match exactly before logger formatting.
- **SC-003**: An unresponsive version check releases an otherwise active invocation within the two-second budget, with no retry. Shorter applicable deadlines remain respected.
- **SC-004**: In every controlled comparison with checking enabled and disabled, the same operational responses produce the same selected-version behavior, command results, partial-result evidence, and exit status, provided the overall invocation remains active. No check outcome causes automatic version switching.
- **SC-005**: Every supported machine-output scenario remains directly consumable with zero advisory content in result stdout, exactly one structured result where applicable, and no extra keys or blank lines.
- **SC-006**: In all diagnostic acceptance scenarios, warnings follow configured visibility and destination, expose no raw transport details at ordinary or verbose-only levels, and remain distinguishable from independent operational failures.

## Assumptions

- The source is issue #222, retrieved on 2026-10-30, last updated at 13:41:35 UTC, with no comments. Its scope is CLI invocations; changing standalone library-consumer behavior is not required.
- Two seconds is the concrete interpretation of the issue's unspecified “short timeout.” It is a total waiting bound for the advisory attempt, not an extension of existing command deadlines, and introduces no new timeout-setting flag.
- The check uses the command's existing connection and authentication choices. Access to gateway version information is not assumed merely because an operational command is permitted. Selecting the version-information mechanism belongs in planning.
- Ordinary supported release-version representations are comparable by their numeric major/minor components. Unsupported or ambiguous representations must produce an advisory failure rather than guess compatibility; detailed parsing rules belong in planning.
- Existing quiet behavior suppresses WARN records; it does not turn checking off. The “warn once” requirement refers to the single version advisory, not unrelated bootstrap or operational warnings.
- Scope excludes automatic adapter selection, configuration repair, strict mismatch failure, repeated or background checking, cross-invocation caching, broker compatibility analysis, new authentication behavior, new diagnostic infrastructure, and changes to operational retry policy.
- Implementation validation uses only isolated fakes, mocks, test-owned HTTP responses, and isolated command/subprocess checks. It must exercise configured and inherited diagnostic destinations, active plain and structured formats, both debug entry points, quiet and machine-output combinations, request ordering/counts, timeout/cancellation, and original operational failure/partial-result behavior. Direct-print routing or raw-error leakage must fail those checks.
- Live Camunda, cluster, smoke, and environment-backed checks are not implementation completion gates. Any real-engine behavior not proven by isolated coverage must be reported accurately as unverified. Specification-only validation consists of document review, consistency, formatting, and local-link checks.
