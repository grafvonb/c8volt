# Research: Native Process Instance Variable Search

## Decision: Extend The Existing `get pi` Search Surface

**Rationale**: Issue #139 explicitly scopes the feature to `get process-instance` / `get pi`. The existing command already owns process-instance search flags, pagination, totals, enrichment, tenant behavior, command metadata, and docs generation. Extending that command keeps operators in the current workflow and avoids a parallel command family.

**Alternatives considered**: Add a new `get variable` or `search variable` command. Rejected because it would force users to compose a variable search with a second process-instance lookup and would duplicate command contract behavior already present in `get pi`.

## Decision: Parse Variable Filter Syntax In The Command Layer

**Rationale**: The variable-filter grammar is a CLI contract: comma splitting, quoting, shorthand forms, alias normalization, and local validation determine whether a command is accepted before remote search. That responsibility belongs near existing Cobra flag validation and search filter population.

**Alternatives considered**: Parse raw strings inside versioned services. Rejected because services should receive version-neutral domain intent, not user-facing flag syntax, and parser failures should be reported before remote clients are constructed where practical.

## Decision: Represent Variable Filters In The Version-Neutral Domain Filter

**Rationale**: `c8volt/process` should remain a thin facade, and internal process-instance services already receive `domain.ProcessInstanceFilter`. Adding a structured variable filter collection there keeps versioned request construction below the facade while allowing command tests and facade tests to assert stable intent.

**Alternatives considered**: Pass raw CLI strings through facade options. Rejected because that would leak command grammar into service adapters and make behavior harder to test independently.

## Decision: Build Native Requests In The v8.8 And v8.9 Process-Instance Adapters

**Rationale**: Supported Camunda versions can differ in generated request types and compatibility behavior. Native variable-search request construction belongs in the version package that owns those generated client shapes and existing process-instance search requests.

**Alternatives considered**: Create a shared request builder using generated types from one version. Rejected because generated type ownership must remain isolated and version-specific differences should stay explicit.

## Decision: Fail 8.7 Variable Search Explicitly

**Rationale**: The issue makes 8.7 out of scope for variable-search flags and forbids silent Operate fallback. The command or service path must detect variable filters and return a clear unsupported error before implying a valid empty result.

**Alternatives considered**: Filter locally by retrieving variables for each process instance. Rejected because it would not be native variable search, could be expensive, and would violate the no-Operate/no-fallback constraint.

## Decision: Preserve Native Serialized Value Semantics

**Rationale**: Users need access to native equality, existence, array, and pattern matching behavior. The CLI should validate expression shape and operator names, preserve arrays and quoted strings, and avoid rewriting values beyond documented shorthand and `$notin` normalization.

**Alternatives considered**: Parse all values into Go-native values and reserialize. Rejected for the first implementation because it risks changing user-intended serialized variable values and wildcard escaping.

## Decision: Treat Documentation As Part Of The Deliverable

**Rationale**: The new flags are compact but syntax-sensitive. Help text, README examples, generated CLI docs, and command metadata must agree so operators can use quoting, arrays, `$like`, `$notIn`, and `scopeKey` correctly.

**Alternatives considered**: Add tests only. Rejected because the constitution requires user-visible documentation changes whenever command behavior changes.
