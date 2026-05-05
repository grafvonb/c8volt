# Research: Process Instance Incident Expectation

## Decision: Represent incident matching as an expectation alongside state matching

**Rationale**: The CLI must allow `--incident` alone and together with `--state`. Modeling both as requested expectations lets command validation require at least one expectation while the wait loop checks all requested conditions for each selected process instance.

**Alternatives considered**:
- Add a separate `expect incident` command. Rejected because the issue explicitly asks to extend `expect process-instance` / `expect pi` and avoid a parallel command flow.
- Treat incident as a process-instance search filter only. Rejected because expectations must wait until selected process instances reach the requested marker, not only filter an initial search result.

## Decision: Use exact string validation for `--incident true|false`

**Rationale**: The issue requires accepted values exactly `true` and `false`. Parsing through a strict helper keeps invalid values like `maybe`, `1`, or `TRUE` on the clear invalid-input path and avoids hidden aliases.

**Alternatives considered**:
- Use permissive boolean parsing. Rejected because it would accept values outside the contract.
- Make `--incident` a boolean flag. Rejected because the command needs both true and false expectations.

## Decision: Extend the existing service waiter path

**Rationale**: The current `--state` behavior, absent semantics, canceled/terminated compatibility, worker controls, fail-fast, logging, and timeout behavior already live in the process facade and service waiter path. Extending that path keeps behavior consistent and minimizes new code.

**Alternatives considered**:
- Poll through command-layer `LookupProcessInstance` calls. Rejected because it would duplicate wait behavior and risk drifting from existing state semantics.
- Add version-specific wait loops. Rejected because v87/v88/v89 already delegate state waiting to the shared waiter.

## Decision: Missing instances only satisfy state absence, never incident false

**Rationale**: `--state absent` has established special semantics, but the issue explicitly says a missing process instance must not be treated as `--incident false`. The wait matcher must therefore distinguish state absence from incident absence.

**Alternatives considered**:
- Treat missing as incident false because no incident marker is visible. Rejected because it violates the issue and would let scripts falsely succeed.

## Decision: Update docs from command metadata and docs generation

**Rationale**: Help text and generated CLI docs are user-facing behavior in this repository. Updating command metadata and running the docs generation path keeps README, docs index, and generated CLI pages aligned.

**Alternatives considered**:
- Hand-edit only README examples. Rejected because generated CLI docs would drift from command help.
