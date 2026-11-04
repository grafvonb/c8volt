# Integration evidence (#338)

The separate aggregate data model was removed on 2026-10-15 at the operator's
request. No ValidationRequest, CoverageCatalog, SliceExecution, normalized
ScenarioEvidence or ReleaseSummary contract remains.

Use the existing integration/cli per-group records: profile readiness, command
stdout/stderr, fixture identities, family reports and summary.md. Batch and
draining reports retain observed states/counters; tenant reports retain declared
roles and observed isolation checks. Incident repair evidence records incident
keys together with their process-instance owners.

Small version/fixture helpers and credential-free tenant-role metadata remain
under integration/internal/validation. They do not orchestrate tests or reduce
coverage into a release verdict. Authentication stays in local profiles.
