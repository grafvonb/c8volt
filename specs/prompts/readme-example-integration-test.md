# README Example Integration Test Prompt

Use this prompt before every release after `README.md` has been updated. It
turns README command examples into a real integration test against the caller's
local/default Camunda development cluster.

Before using this prompt, follow `specs/prompts/AGENTS.md`. In particular, do
not treat files under `specs/prompts/` as release-change source material.

```text
Validate every executable example in README.md against the local/default c8volt
configuration and a real Camunda development cluster.

Inputs:
- Target document: README.md
- Default config: the repository's normal `config.yaml` resolution
- Cluster: caller's local/default Camunda development cluster
- Release version: <VERSION>
- Release date: <RELEASE_DATE>
- Expected Camunda minor version: <CAMUNDA_MINOR>, for example `8.9` or `8.10`
- Expected embedded fixture prefix: <FIXTURE_PREFIX>, for example `C89` for
  Camunda `8.9` or `C810` for Camunda `8.10` if that is the repository's
  fixture naming convention
- Optional build output path: /tmp/c8volt-readme-verify

Goal:
Prove that README examples are correct, copyable, bounded, and safe for a real
small development environment. If an example is wrong, stale, too broad,
unbounded, unsafe, or cannot be backed by real cluster state, report it in the
example/helper/documentation improvements report. Only update README, CLI help,
or helper-message documentation when the caller explicitly asks for those
documentation fixes.

Default mode:
This is a technical integration check, not a feature implementation task. Unless
the caller explicitly asks for documentation/helper edits, do not update files.
Never update product behavior or implementation code as part of this prompt run.
Prefer reporting findings. Documentation and helper-message changes are allowed
only when the caller asks this prompt runner to fix examples, help text, or docs
as part of the validation run.

Required safety rules:
1. Do not use `--no-wait` in README examples or everyday workflow snippets. It
   is an exception-path flag, not normal release documentation.
2. Do not run broad destructive examples against the cluster. Convert them to
   `--dry-run`, add a concrete fixture scope, or add a small `--limit`.
3. Bound list/search examples that may return many rows. Prefer `--limit 5` for
   display examples and `--limit 3` for piped mutation examples.
4. Avoid examples that mutate every active or completed instance in the cluster.
   Use a known fixture BPMN process ID, a real keyed target, `--dry-run`, or a
   small limit.
5. Do not delete process definitions or broad process-instance history unless
   the caller explicitly asks for destructive cleanup.
6. Do not leave long-running validation commands alive. If an example floods
   output or scans too much cluster state, stop it, fix the example, and rerun a
   bounded version.
7. Treat stale hardcoded runtime keys as documentation bugs. Replace them with
   placeholders such as `<process-instance-key>`, `<incident-key>`, or shell
   variables such as `$PROCESS_INSTANCE_KEY_A`, then validate the same syntax
   locally with real keys discovered during the run.
8. Treat stale fixture names as documentation bugs. Derive the expected fixture
   prefix from the default configured Camunda minor version and the repository's
   embedded fixture naming convention. For example, Camunda `8.9` currently uses
   `C89_*`; when the default moves to Camunda `8.10`, update examples to the new
   `8.10` fixture prefix instead of preserving `C89_*`.
9. Do not keep a README example that cannot be made discoverable and testable in
   the real local environment. Remove it or rewrite it to a supported workflow.

Required workflow:
1. Build a temporary validation binary from the current checkout:
   `GOCACHE=/tmp/c8volt-gocache go build -o /tmp/c8volt-readme-verify .`
   Use this binary as the runtime equivalent of README's `./c8volt` examples.
2. Confirm default configuration and connectivity before running any README
   example against the cluster:
   - `/tmp/c8volt-readme-verify version`
   - `/tmp/c8volt-readme-verify config validate`
   - `/tmp/c8volt-readme-verify config test-connection`
   Treat `config test-connection` as the first stop/go integration gate. It must
   load the intended config file and profile, reach the configured cluster, and
   report a healthy cluster version without warnings or mismatch diagnostics. If
   it reports any warning, connection problem, version mismatch, unexpected
   profile, or other configuration issue, stop and report the problem before
   validating README examples.
3. Capture machine-readable agent preflight data for validation logic before
   running README examples:
   - `/tmp/c8volt-readme-verify capabilities --json`
   - `/tmp/c8volt-readme-verify config show --json`
   - `/tmp/c8volt-readme-verify config test-connection --json`
   Use these JSON outputs as the agent's private source for command paths,
   visible flags, automation support, output modes, config resolution, profile
   selection, base URL, and cluster metadata. Do not rewrite README examples to
   JSON just because the agent used JSON internally; README examples should stay
   human-friendly unless a section is explicitly about automation or JSON output.
4. Treat the real cluster version from `config test-connection` as the first
   runtime truth before validating fixture examples:
   - Record the loaded config path, active profile, base URL, gateway version,
     broker version, broker count, partition count, and partition health from
     `config test-connection` output.
   - The gateway version, for example `8.9.0`, determines the real cluster minor
     being tested.
   - Use `/tmp/c8volt-readme-verify get cluster version` and
     `/tmp/c8volt-readme-verify get cluster topology` as follow-up checks when
     useful, but do not let them replace the initial `config test-connection`
     gate.
5. Derive the current default Camunda minor and fixture prefix before validating
   fixture examples:
   - Use the real cluster minor plus effective config to confirm the runtime
     target, for example `8.9` today or `8.10` after the next Camunda minor
     release.
   - Inspect embedded fixtures with `/tmp/c8volt-readme-verify embed list` and
     choose the fixture prefix that matches the default minor.
   - Do not assume `C89` is always current. It is only correct while the default
     runtime target is Camunda `8.9`.
6. Deploy small known fixtures before validating examples that require data:
   - `/tmp/c8volt-readme-verify embed deploy --all --run`
   - If a job example needs a real job key, deploy a tiny service-task BPMN
     fixture, start it, activate the job through the local API if needed, then
     validate `get job` and `update job` examples against that real key.
7. Extract README command blocks and classify each executable line:
   - direct read examples
   - mutation examples
   - dry-run examples
   - stdin/pipeline examples
   - JSON/automation examples
   - setup snippets that are intentionally illustrative, such as copying a
     config file or exporting OAuth environment variables
8. For placeholder examples, discover real values from the cluster and run the
   same command shape with substituted values:
   - process definition keys from `get pd --latest`
   - process instance keys from `run pi`, `get pi`, or fixture startup
   - user-task keys from user-task backed fixture instances
   - incident keys from `get incident --state active --limit 5` or
     `get pi --incidents-only --with-incidents --limit 5`
   - job keys from incident/job fixture output or job activation lookup
9. Validate every command shape in small batches. Keep a scratch log of:
   - README line or command family
   - substituted runtime values
   - pass/fail result
   - README/help/docs improvement to report or fix, if explicitly allowed
   - product/code/non-functional improvement to report separately, if any
10. When a README example fails, decide whether the problem is:
   - wrong syntax or wrong flag
   - wrong fixture/process ID for the default profile
   - broad/unbounded search
   - unsafe mutation
   - stale hardcoded runtime key
   - unavailable sample data
   - undocumented behavior drift
   Then report the issue. If the caller explicitly allowed documentation edits,
   fix the README/help/docs and rerun the corrected example.
11. If CLI help output contains examples that contradict the corrected README,
   report the mismatch. If the caller explicitly allowed documentation/helper
   edits, update the command help source and help-contract tests too. README and
   `--help` must not fight each other.
12. Run targeted automated validation after edits:
   - `go test ./cmd -count=1`
   - broader tests only when the edits touch behavior beyond docs/help strings
13. Audit the final documentation and help examples:
    - no README `--no-wait` examples
    - no stale fixture examples for the current default Camunda minor
    - no stale hardcoded runtime keys in README or non-test command help
    - no broad destructive pipelines
    - no unbounded examples that can flood a real dev cluster
    - no examples known to return empty input into a consuming pipeline

Suggested investigation commands:
- `awk '/^```bash/{in_block=1; next} /^```/{in_block=0} in_block && /^\.\/c8volt|^printf|^cp |^export /{print NR ":" $0}' README.md`
- `rg -n -- 'C88_|--no-wait|225179981|get resource --id|with-vars$|with-incidents$|--state active$|keys-only \|' README.md cmd -g '*.go'`
- `/tmp/c8volt-readme-verify capabilities --json`
- `/tmp/c8volt-readme-verify config show`
- `/tmp/c8volt-readme-verify config show --json`
- `/tmp/c8volt-readme-verify config test-connection`
- `/tmp/c8volt-readme-verify config test-connection --json`
- `/tmp/c8volt-readme-verify embed list`
- `/tmp/c8volt-readme-verify get cluster version`
- `/tmp/c8volt-readme-verify get cluster topology`
- `/tmp/c8volt-readme-verify get pd --latest`
- `/tmp/c8volt-readme-verify run pi -b <FIXTURE_PREFIX>_SimpleUserTask_Process`
- `/tmp/c8volt-readme-verify get pi --state active --limit 5`
- `/tmp/c8volt-readme-verify get incident --state active --limit 5`
- `/tmp/c8volt-readme-verify get pi --incidents-only --with-incidents --limit 5`
- `/tmp/c8volt-readme-verify capabilities --json`
- `go test ./cmd -count=1`

Known lessons from prior validation:
- Source checkouts may already contain a `./c8volt` directory, so use a
  temporary binary path while validating README command arguments.
- Default fixture examples must track the current default Camunda minor. At the
  time this prompt was written, Camunda `8.9` used `C89_*`; the next default
  minor after Camunda `8.9` is expected to be `8.10`, so examples should move to
  the repository's `8.10` fixture prefix when the product target moves.
- `get pi --with-vars`, `get pi --with-incidents`, and direct-incident filters
  can scan or render thousands of rows on a real cluster. Add a key, a fixture
  BPMN process ID, or a small `--limit`.
- Piping `get pi --state active --keys-only` into a mutation is too broad for a
  README. Add a fixture BPMN process ID and a small limit, or make the mutation
  a dry-run.
- A pipeline that produces no keys is not a valid example; pick a selector that
  exists in the default dev fixture set or rewrite the example.
- `get resource --id <resource-key>` should not appear in README unless the
  prompt runner can demonstrate how to discover a real resource key and fetch it
  successfully in the default environment.

Output expectations:
- If the caller did not explicitly authorize edits, do not update files. Produce
  reports only.
- If the caller explicitly authorized documentation/helper edits, update
  README.md when examples are wrong, unsafe, stale, unbounded, or not
  reproducible.
- If the caller explicitly authorized documentation/helper edits, update CLI
  help source and tests when user-facing help examples need the same correction
  as README.
- Report which command families were validated with real cluster calls.
- Report any setup-only snippets or external-profile examples that were not run
  and why.
- Report validation commands run, including `go test ./cmd -count=1`.
- Produce two distinct reports:
  1. Example, Helper, And Documentation Improvements:
     - README examples that should be added, removed, bounded, scoped, renamed,
       reordered, or clarified
     - CLI help examples or helper messages that should be brought into sync
       with the README
     - docs-only gaps found while validating real command usage
  2. Product, Code, And Non-Functional Improvements:
     - command behavior that is technically correct but painful in practice
     - examples or commands that take too long, page too much, or produce noisy
       output even when bounded
     - missing or incomplete log messages, unclear warnings, weak progress
       reporting, poor error messages, or output that is difficult for humans or
       agents to interpret
     - possible future code/test improvements, without implementing them unless
       the caller explicitly asks
- Do not commit unless asked.
```
