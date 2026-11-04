# Internal models and invariants

No new persistent schema, CLI payload or required public interface method is planned.
Existing public model fields/tags and error classes remain authoritative.

## Mechanical JSON values (T001)

Two separate operations: normalization for exact existing JSON comparisons, and validated
snapshotting for ownership. Preserve `json.Number`, large integers, scalar spelling where
current comparisons distinguish it, typed maps/slices/arrays, raw JSON, custom marshalers,
nil and empty containers. Invalid/cyclic/non-JSON input must retain the owning operation's
existing rejection class/text. Callers do not gain a new acceptance/rejection rule because
helper code moved. Copying accepted mutable values cannot preserve caller aliases. Keep
concrete representation when publicly established; a normalization helper is not that copier.

## Worker result slots (T002)

Existing pool input index -> output slot and aggregate errors remain unchanged. Worker
normalization/staggering/fail-fast/context rules and DEBUG records remain the same. Legacy
public Reports/Reporter and facade options keep their exact identity; internal services use
a neutral execution package. No public options type is relocated or redefined.

## Bounded failure facts (T003–T004)

Each formatter supplies an allowlisted reason for each qualifying outcome. Accumulate first-
seen reason order and counts; display the existing first three reasons and remaining-count
bucket. The original service error remains separate for machine evidence and exit classes.
A result-emission fact distinguishes not attempted from already attempted/completed so failure
handling never emits a second envelope. Bootstrap configuration failures retain their existing
fallback owner; configured logger routing never falls back to a new/global logger.

## Task/scope index (T005)

- Scope index: unique scope key -> validated target.
- Task index: task key -> established ordered scope sequence.
- Association index: (scope key, task key) -> independent ordered variable-name slice.

Build once from the validated frozen plan; reject the same duplicates/conflicts/missing edges
as before. Indexes are local, immutable during worker execution and not serialized. Ordered
source slices remain the source of result/verification ordering; no backend cache is added.

## Lifecycle evidence (T006–T009)

Keep separate public suspension/resumption plan and result types. Internal shared mechanics
may project common key, submission, status, last-state, suspended-date and has-observation facts.
Read ports accept context and existing CallOptions; mutation ports remain endpoint-specific.
Acceptance, rejection, unknown submission, no submission and observed state are independent axes.
An action policy supplies explicit eligibility and success/conflict predicates without deriving
submission from a read. Suspension success is SUSPENDED; resume success is ACTIVE or COMPLETED.
An empty collection or dry-run cannot acquire accepted submission; unstarted slots remain so.

## Variable-page evidence (T010–T011)

Keep effective-variable required totals distinct from root/local optional totals and strict
scope metadata. Each raw decoder preserves presence, raw count before filtering, request
position, end cursor, reported total kind, tenant/owner identity and truncation. Do not synthesize
zero totals for absent metadata, stop on sparse selected pages, or normalize endpoint schema
mismatches into success. Existing traversal computes continuation/completeness, not adapters.

## Selector and deployment preparation (T012–T013)

Selector request/result carries existing ordered normalized BPMN IDs, version/tag/latest mode,
exact matches, missing IDs and near matches. Internal strategy owns backend calls; facade
adapters only translate current public reader inputs/options/results. Command prompt/render
policy consumes the facts unchanged.

Deployment run preparation uses existing deployment metadata and original resource bytes.
Preserve key-before-definition-ID-before-BPMN-fallback precedence, unit order, duplicate handling,
tenant target and parse/no-identifier errors. Return current public creation-input values through
an additive function. Deployment results are emitted at their existing stage before starts.

## Test contracts, organization and tooling (T014–T018)

Shared test cases carry constructor, expected capability and wire-contract policy, never real
credentials or live destinations. Version-specific assertions remain local. File movement
changes no type/method/tag identity. An optional docs release-version input preserves the
manual/automatic build distinction; a shared action receives existing SFTP environment only
at execution, while tests use dummy values and fake executables. Public scaffolding remains
compatible; only private/internal unused symbols are eligible for deletion after reference audit.
