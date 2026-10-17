# Tasks: Camunda 8.10 integration support (#338)

Revised 2026-10-15 to follow the requested direct Make approach. The original
T001–T003 runner/policy work is removed; historical execution remains in
progress.md and validation.md.

- [x] Keep version-matched 8.10 fixtures and all existing baseline, volume and
  real-state scenarios, including user-task and batch/tenant additions.
- [x] Keep fixture isolation, exact eventual batch accounting, incident-owner
  repair selection, and the 8.10 nullable deletion-batch adapter fix.
- [x] Keep API generation guards and C810 shell-suite support.
- [x] Remove the Go runner, normalized evidence hooks and aggregate policy.
- [x] Restore direct serial Make group execution with visible output, defaults,
  startup summary, confirmation and observed/requested version gates.
- [x] Align current operator and feature documentation; preserve historical logs.
- [ ] T010 Revalidate the direct Make workflow when explicitly authorized.
  Verify group output/order, version mismatch refusal, failure propagation and
  the retained live scenarios. Record actual skips and unavailable tenant or
  older-version coverage without claiming complete release certification.

No tests were run for the runner removal, as explicitly requested.
