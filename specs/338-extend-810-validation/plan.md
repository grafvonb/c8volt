# Implementation plan: direct Make execution for #338

Revised 2026-10-15. The operator rejected the added test-runner architecture.
This plan supersedes its earlier version.

1. Remove integration/runner and its dedicated policy/reduction types, catalogs,
   tests and normalized evidence registration/flushing hooks.
2. Keep the existing test targets. Have integration-test-all invoke them using
   recursive Make with -j1 after the startup summary and single confirmation.
   Preserve defaults and environment/command-line overrides.
3. Keep small existing version, fixture and tenant helpers. Enforce the requested
   minor in the existing checkProfileReadiness path alongside profile/server
   matching. Keep common nonmutating preflight ahead of mutation groups.
4. Retain all useful scenarios, fixture fixes, generation guards and C810 shell
   support. Preserve existing per-group summaries, logs and reports.
5. Update operator documentation and current feature artifacts. Preserve the
   historical validation log, including failures; do not reinterpret old results.

Validation for this change is source/diff inspection only, per explicit user
instruction. Do not run tests, the aggregate, or live commands. A future authorized
validation should check version refusal, serial group order, visible output,
error propagation and the retained scenarios.
