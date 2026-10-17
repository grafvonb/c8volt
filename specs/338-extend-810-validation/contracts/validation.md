# Existing Make integration interface

Updated 2026-10-15; supersedes the removed runner contract.

- `make integration-test-all` confirms once, then directly calls existing common,
  batch/tenant, baseline, volume and real-state targets through serial Make.
- Default profiles: c810local. Default requested minor: 8.10. Explicit environment
  or Make command-line settings override defaults. Direct group targets retain
  their existing profile selection behavior.
- Startup output lists profile, requested version, work directory selection,
  tenant requirements, build mode and timeouts. Test output remains visible.
- Common preflight runs before mutation groups. Existing readiness checks compare
  configured and requested minors with the observed gateway version.
- Normal Make/Go exit status controls success/failure; stop at the first failing
  group. No aggregate manifest, release.json, reducer or certification verdict.
- Existing per-group reports remain. Default work directories are separate
  retained temporary directories. An explicit C8VOLT_IT_WORKDIR is an exact group
  directory and can overwrite reports if reused across groups.
- Optional tenant scenarios skip without prerequisites. Explicitly required
  logical/physical coverage fails when its role metadata is missing/invalid.
- Keep all existing timeout, binary, verbosity and automation controls.
