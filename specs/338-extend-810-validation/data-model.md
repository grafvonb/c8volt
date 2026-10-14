# Data Model: Release Validation Evidence

These are integration-tooling records, not production domain types. All new machine evidence has `schemaVersion: 1`. Existing family reports remain available.

## ValidationRequest

Fields: `runId`, `mode` (exploratory/release), `requestedMinor`, selected profile names, required tenant coverage (none/logical/physical), catalog revision, start time, and absolute evidence root.

Rules: release mode requires a supported canonical minor; profiles refer only to default local configuration. No credentials, tokens, or full config dumps are serialized. Required coverage is frozen before execution. One requested minor per strict invocation; mixed-profile exploratory runs retain per-version results. Foreign-minor profiles cannot satisfy a release request.

## ValidationTarget

Fields: profile name, expected minor, observed full server version, readiness outcome/reason, declared cluster group, logical tenant ID, physical tenant ID, and role assignments. Store sanitized route identity rather than credential-bearing URLs.

Rules: version is parsed and verified before mutation; unknown/mismatched versions cannot be ready. Logical pairs must be distinct tenants within a declared physical endpoint. Physical pairs must identify distinct physical tenants within a declared cluster. URLs alone do not establish cluster identity. Positively observed fixtures substantiate each role before negative assertions.

## CoverageRequirement

Fields: stable scenario ID, slice/family, applicable minors, required flag, minimum evidence level, expected target roles, and optional accepted gap ID.

Rules: scenario identities are unique within the catalog; every applicable family has explicit minimum scenarios. No runtime wildcard waivers. Unsupported 8.8 real-state scenarios are not required. Controlled proof never satisfies a live post-state requirement.

## PrerequisiteGap

Fields: stable ID, affected scenario IDs/minors, missing prerequisite, blocked proof, rationale, owner/follow-up reference when available, and review status.

Rules: a gap may describe nonrequired extended proof but cannot remove an entire required family or convert product failure into success. Unrecognized gaps are not accepted automatically. Historical C89 gaps are reassessed for each minor rather than copied as validation evidence.

## SliceExecution

Fields: run ID, slice ID, target/profile group, start/end time, child evidence directory, process exit code, termination reason, and selected test names.

States: `planned -> running -> finished | failed | interrupted`. Planned or running slices at evaluation time are incomplete. Separate child directories prevent overwrites; duplicate completion records are invalid. An exit code of zero does not establish scenario success without matching evidence.

## ScenarioEvidence

Fields: run ID, slice ID, stable scenario ID, profile/role group, observed version, execution outcome, evidence level, before/after observations or paths, resource IDs, optional gap ID, failure class, and cleanup outcome.

Outcomes: `passed`, `skipped`, `blocked`, `failed`, `unsupported`.
Evidence levels: `live-post-state`, `submission-only`, `dry-run`, `controlled`, `read-only`.
Failure classes: `product`, `harness`, `fixture`, `environment`, `authorization`, `version`, `incomplete-evidence`.

Rules: passed evidence must meet the catalog minimum. Submission success must not imply completion. Read-only positive/negative tenant assertions include confirmed resource ownership; absence alone is insufficient. Cleanup is separate and cannot erase a scenario failure. Paths remain beneath the invocation root; secrets are redacted.

## ReleaseSummary

Fields: run ID, mode, requested minor, actual versions, per-version/family counters, missing requirements, gap list, failure list, verdict, and `coverageComplete`.

Verdicts: `running`, `passed`, `passed_with_gaps`, `failed`, `incomplete`.

Transitions: initial `running`; any unmet required check or failed child produces `failed`; interruption or unfinished mandatory child produces `incomplete`; all requirements met with nonrequired accepted gaps produces `passed_with_gaps`; all applicable proof complete produces `passed`.

`coverageComplete` is true only for `passed`. `passed_with_gaps` means declared required checks passed, not certification of the blocked behaviors. Counters and lists derive from expected requirements joined to current-run evidence, never only from files that happen to exist. Stale or foreign-run evidence is rejected.
