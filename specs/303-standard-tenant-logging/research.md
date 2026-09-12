# Research: Standard Tenant Logging

## 1. Emission mechanism

**Decision**: Call `printOpsDurableLine(cmd, line.Text, line.Warn)` from both existing loops in `cmd/processinstance_mutation_progress.go`.

**Rationale**: `cmd/ops_progress_render.go` already maps the warning boolean to logger WARN and otherwise INFO. It returns after invoking the attached logger, including when filtering suppresses the record. Without a logger it writes the raw line to `cmd.ErrOrStderr()`. This exactly supplies the issue's requested formatting, filtering, severity, and test fallback behavior.

**Alternatives considered**: Direct logger calls duplicate existing fallback handling. A new helper or changes to shared logging broaden scope. Retaining raw writes continues to discard severity and bypass configuration.

## 2. Eligibility and lifecycle

**Decision**: Preserve existing guards and mark-before-emission ordering.

**Rationale**: The progress emitter additionally requires a durable stderr channel in human, verbose, or debug mode. Both paths use `shouldRenderTenantContextHuman` and the existing rendered marker; the confirmation emitter also checks `flagCmdAutomation`. Shared eligibility excludes zero context, quiet, automation, and non-one-line rendering. Logging thresholds are downstream of these guards. Marking occurs before logging, so suppression must not cause replay.

**Alternatives considered**: Marking only after a visible record changes deduplication. Consolidating the two guard sets risks changing caller-specific behavior.

## 3. Message policy and dry-run behavior

**Decision**: Reuse `tenantContextHumanLines` unchanged and leave selector dry-run rendering unchanged.

**Rationale**: `cmd/cmd_tenant_context.go` already owns text, override provenance, line order, and `Warn`. Multiple affected tenants produce a warning-marked summary, and other tenant warnings retain their existing classification. `renderCancelSearchTenantContext` and `renderDeleteSearchTenantContext` dispatch dry-run previews to `renderTenantContext`; only ordinary confirmation context reaches the targeted stderr renderer.

**Alternatives considered**: Inferring severity from message text changes policy. Logging every tenant renderer would change dry-run and unrelated command contracts.

## 4. Regression evidence

**Decision**: Use attached-logger tests for both emitters plus existing delete/cancel command and terminal fixtures.

**Rationale**: Existing progress and confirmation tests commonly exercise the no-logger fallback and therefore cannot prove logger severity or formatting. Plain/JSON format and INFO/WARN/ERROR thresholds expose the defect without live backend access. Existing selector tests cover empty/sparse pages and command modes; terminal fixtures prove prompt routing using terminal stdin.

**Alternatives considered**: Only checking substrings in a combined output buffer misses stream contamination and severity. Only helper tests miss guard/wiring regressions. Live destructive validation is unnecessary given existing service stubs and subprocess fixtures.

The existing fixture can attach `logging.New(logging.LoggerConfig{Level: level, Format: format, Writer: stderr})` using `logging.ToContext`. Standard JSON records contain `time`, `level`, and `msg`, with optional `source`; decode them through EOF. Plain format includes a timestamp, so validate its standard prefix separately from exact severity/message content. Root setup in `cmd/root.go` supplies the activity-wrapped stderr writer to the logger. Reuse `testx.NewCmdTerminalRunner()` for real terminal input and independent output streams.

## 5. Documentation and dependencies

**Decision**: Update README and delete/cancel help metadata, regenerate docs, and reuse all existing dependencies.

**Rationale**: The observable diagnostic format changes, so constitution documentation requirements apply. `go.mod` declares Go 1.26 with go1.26.2; Cobra and `toolx/logging` already provide the needed behavior. No technology selection or external integration is introduced.

**Alternatives considered**: Calling this internal-only would omit a user-visible change. Hand editing generated CLI documentation would violate repository rules.

All research questions are resolved through repository evidence; no external library recommendations or version changes are required.
