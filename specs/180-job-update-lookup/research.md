# Research: Job Lookup And Updates

## Decision: Add `get job` Under The Existing Get Root

Create `cmd/get_job.go` and register `c8volt get job --key <job-key>` below the existing `get` command family.

**Rationale**: Operators already use `get pi --with-incidents` to discover `jobKey`; the natural follow-up is a read-only `get job` command that returns the job detail for that key. Keeping this under `get` preserves command semantics and discoverability.

**Alternatives considered**:

- Add job fields only to `get pi --with-incidents`: rejected because the issue requires direct job inspection by key.
- Add a facade-only helper without a CLI command: rejected because the feature is user-facing CLI behavior.

## Decision: Add `update job` Under The Existing Update Root

Create `cmd/update_job.go` and register `c8volt update job --key <job-key>` below the existing `update` command family.

**Rationale**: The repository already has `update pi --vars`; job updates are also state-changing and should use the same command family and metadata conventions.

**Alternatives considered**:

- Add mutation flags to `get job`: rejected because `get` should remain read-only.
- Add a new root command such as `job update`: rejected because it would diverge from the existing `update` command family.

## Decision: Create A Dedicated Job Service Package

Add `internal/services/job` with shared `api.go`, `factory.go`, versioned `v87`, `v88`, and `v89` services, and compile-time API conformance checks.

**Rationale**: The issue explicitly requires job lookup, job update, and confirmation to stay inside a job service boundary. Existing services follow this package/factory/version layout, and a dedicated package prevents job behavior from leaking into process-instance or incident services.

**Alternatives considered**:

- Add job lookup to `internal/services/processinstance`: rejected because the issue forbids mixing job functionality into process-instance services.
- Add job lookup to `internal/services/incident`: rejected because incident output only supplies `jobKey`; job inspection and mutation are separate behavior.
- Implement generated-client calls in `cmd`: rejected because it bypasses service versioning, error mapping, and test patterns.

## Decision: Use Generated Job Search For Lookup

Implement lookup by key through generated `SearchJobs` behavior in v8.8 and v8.9.

**Rationale**: The issue identifies `SearchJobs` as the lookup mechanism. Search results can return the fields needed for diagnosis and can be reused by retries confirmation.

**Alternatives considered**:

- Add a direct job-get endpoint: rejected because the issue specifies generated search and generated clients expose search behavior.
- Infer job status from incident search only: rejected because the command must inspect jobs directly by key.

## Decision: Use Generated Job Update For Supported Mutations

Implement v8.8/v8.9 updates with generated `PATCH /jobs/{jobKey}` behavior and a changeset containing only retries and timeout.

**Rationale**: Generated job update requests support the required update fields. Limiting the command to retries and timeout keeps scope aligned with the issue and avoids exposing fail-job concepts such as `retryBackOff`.

**Alternatives considered**:

- Expose `retryBackOff`: rejected because it belongs to fail-job behavior and is out of scope.
- Add job variables, fail, complete, or BPMN error commands now: rejected because each is explicitly out of scope.

## Decision: Confirm Retries Only

When retries are supplied and `--no-wait` is not supplied, poll job lookup until the requested retry count is observed or waiter exhaustion occurs. Timeout-only updates return submitted/accepted output after mutation acceptance and do not compare deadline timestamps.

**Rationale**: Retry count has a stable read-model predicate. Timeout confirmation would require comparing an observed deadline timestamp against client-side timing assumptions, which is vulnerable to time drift and read-model timing differences. The command should not claim confirmation where the predicate is not reliable.

**Alternatives considered**:

- Confirm timeout using a deadline tolerance: rejected because tolerance-based confirmation can produce brittle tests and false confidence under clock drift.
- Skip all confirmation for job updates: rejected because retries can be confirmed reliably and the constitution favors operational proof where available.

## Decision: Plan Job Updates Before Mutation And Support `--dry-run`

Before submitting `update job`, load the current job through the same lookup path as `get job --key`, build a compact update plan, and use that plan for dry-run output, no-op detection, and interactive confirmation.

**Rationale**: Job updates are state-changing and should follow the newer mutation UX established for process-instance variable updates. A plan lets operators see retries before/after, timeout submission intent, and no-op retry-only requests before mutation. It also keeps shell and JSON automation behavior predictable.

**Alternatives considered**:

- Submit immediately without a plan: rejected because it does not match the current mutation safety pattern for update commands.
- Compare timeout requests to observed deadlines for no-op detection: rejected because timeout equality is timing-sensitive and conflicts with the retries-only confirmation decision.
- Make `--dry-run` a command-only mock without loading current state: rejected because retry no-op detection and useful plan output require the visible job state.

## Decision: Keep JSON Output Stable For Job Updates

Reject `--json --verbose` for `update job`, including dry-run mode, and require `--auto-confirm` or `--automation` for non-dry-run JSON updates that would mutate state.

**Rationale**: JSON output should be one stable machine-readable view. Prompt text or verbose log lines around JSON make automation brittle, and the process-instance variable update behavior established the same contract for state-changing updates.

**Alternatives considered**:

- Let `--verbose` add detail to JSON output: rejected because JSON should already return the full payload.
- Prompt during non-dry-run JSON updates: rejected because prompts intermixed with JSON are poor automation UX.
- Allow `--json --verbose` only for dry-run: rejected because it creates inconsistent rules for the same command.

## Decision: Keep `--no-wait` As Submitted/Accepted Output

When `--no-wait` is supplied, return after the mutation request is accepted and skip retries confirmation.

**Rationale**: This mirrors existing submitted/accepted output behavior and gives automation a lower-latency path without claiming read-model confirmation.

**Alternatives considered**:

- Ignore `--no-wait` for retries updates: rejected because the issue explicitly requires the option.
- Poll once even with `--no-wait`: rejected because it makes the option ambiguous.

## Decision: Update Docs Through Existing Generation Flow

Update Cobra help/examples and README, then regenerate generated CLI documentation with the repository's docs tooling.

**Rationale**: The commands, flags, version support, no-wait behavior, and timeout submitted-only caveat are user-visible. Generated docs should be derived from command metadata and examples.

**Alternatives considered**:

- Hand-edit generated docs only: rejected because it risks drift from source command metadata.
