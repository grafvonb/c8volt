# Release Docs And Help Example Validation Prompt

Use this prompt before publishing a c8volt release when all documentation and
CLI help examples need a real integration pass against the local/default Camunda
development cluster. It is the canonical prompt for release example validation;
use README-only mode when only `README.md` examples need to be checked.

Before using this prompt, follow `specs/prompts/AGENTS.md`. In particular, do
not treat files under `specs/prompts/` as release-change source material.

```text
Perform a deep integration audit of every c8volt command example shown in
README.md, docs, generated documentation, and CLI help output. Syntax-review
VHS demo tapes separately without changing their recorded flow. Validate
examples against the real local Camunda development cluster configured by the
repository's normal `config.yaml` resolution.

Inputs:
- Target release train: v4
- Target Camunda version: v89 only
- Default config: the repository's normal `config.yaml` resolution
- Cluster: caller's local/default Camunda development cluster
- Optional build output path: /tmp/c8volt-release-docs-help-verify

Goal:
Make all examples copyable, generic, bounded, operator-friendly, and true
against the real v89 cluster. Examples in user-facing docs and help must be
agnostic placeholders for real values, not embedded fixture names, stale runtime
keys, or one developer's local IDs. Use concrete real values only in the private
validation run. VHS tapes are the exception: they are real recorded examples
and may keep concrete IDs, keys, fixture names, and timing-specific flow.
Ops examples must represent the real operational workflow, including destructive
execution examples where deletion, purge, cancellation, or mutation is the core
use case. Do not reduce ops documentation to dry-run-only examples.

Default mode:
This is a deep integration validation and repair task. Fix findings in README,
docs, and CLI help examples. Update the relevant source docs, generated-doc
source, CLI help source, and tests as needed, then rerun the corrected examples.
Do not refactor, genericize, reorder, or otherwise update VHS tapes as part of
this prompt run. For VHS tapes, review only whether the c8volt command syntax is
valid; warn at the end which tape files need human review.

README-only mode:
If the caller explicitly asks for README-only validation, restrict the executable
example scan to `README.md` and compare CLI help only for contradictions. In
README-only mode, report findings by default and edit README/help/docs only when
the caller explicitly asks for fixes. Do not update product behavior or
implementation code as part of README-only validation. Use caller-provided
values for `<VERSION>`, `<RELEASE_DATE>`, `<CAMUNDA_MINOR>`, and
`<FIXTURE_PREFIX>` when they are relevant to the README examples being checked.

Required safety and example rules:
1. Validate every example against the real cluster before keeping it.
2. Use v89 only. Remove or rewrite v87/v88 examples because they are not
   testable against the target environment for this release.
3. Keep docs and help examples generic. Use placeholders such as
   `<process-instance-key>`, `<process-definition-key>`, `<incident-key>`,
   `<job-key>`, `<resource-key>`, `<tenant-id>`, and
   `<bpmn-process-id>`.
4. Do not publish examples that mention embedded fixture IDs such as
   `C89_SimpleUserTask`. The validation run may use fixtures privately
   to discover real keys and prove syntax, but public docs/help must stay
   environment-agnostic.
5. Keep only a small number of examples per command. Prefer the standard
   operator path over many flag variants.
6. Do not show unusual or exception-path flags in normal examples, including
   `--no-wait`. Keep them out of README, docs, and everyday help examples
   unless a section explicitly documents exceptional behavior.
7. Bound examples that can scan or print too much cluster state. Prefer small
   limits such as `--limit 5`, a concrete key placeholder, or a narrow selector.
8. Include destructive examples for ops commands where destruction or mutation
   is the core use case. A dry-run-only example is not representative for purge,
   cancel, delete, resolve, repair, or update workflows.
9. Prefer preview-then-execute examples for risky operations: show the
   `--dry-run` or preview command first, then the real destructive command.
10. Clearly mark every docs/help example by operator impact when the surrounding
    format allows it:
    - read-only: observes cluster state only
    - harmless: creates disposable demo/test data or writes only local reports
    - destructive: deletes, purges, cancels, resolves, repairs, updates, or
      otherwise mutates cluster state
11. Destructive examples must be explicitly labeled destructive in docs/help and
    must include a short warning or note explaining the affected scope.
12. Critical mutation commands must be tested for real against the dev cluster,
    including destructive operations. The caller allows the validation run to
    delete process instances, process definitions, and other test data on this
    dev cluster.
13. The real cluster validation should focus especially on destructive examples:
    prove that the preview command works, the destructive command works on
    disposable scoped data, and the post-check confirms the expected impact.
14. Do not leave long-running commands alive. If an example scans too much data,
    floods output, or takes too long for normal operator use, stop it, rewrite
    the example to be bounded or remove it.
15. When validating interactive destructive commands, answer confirmation
    prompts explicitly in the test harness. Do not add `--auto-confirm` to
    public examples unless the section is explicitly about automation.
16. Treat examples that require unavailable sample data as documentation bugs.
    Create temporary fixture data privately for validation, then publish a
    generic placeholder command shape.
17. Exclude VHS tapes from generic-placeholder cleanup. VHS tapes may use
    concrete process IDs, keys, fixture names, shell timing, and typed
    confirmations because they are real recorded demos.
18. Do not change VHS flow. Do not replace concrete VHS values with
    placeholders, do not add or remove VHS flags, do not reorder VHS steps, and
    do not rewrite VHS output rendering as part of this prompt run.
19. For VHS tapes, check only c8volt command syntax: command path, subcommand
    names, flags, required arguments, and obvious shell quoting needed for the
    recorded command to be accepted. If a VHS command looks syntactically wrong,
    list the tape at the end for manual review instead of editing it.

Required workflow:
1. Build a temporary validation binary from the current checkout:
   `GOCACHE=/tmp/c8volt-gocache go build -o /tmp/c8volt-release-docs-help-verify .`
   Use this binary as the runtime equivalent of public `c8volt` examples.
2. Confirm default configuration and connectivity before validating examples:
   - `/tmp/c8volt-release-docs-help-verify version`
   - `/tmp/c8volt-release-docs-help-verify config validate`
   - `/tmp/c8volt-release-docs-help-verify config test-connection`
   Stop if the config is not the intended local dev cluster or if the cluster is
   not a healthy v89 environment.
3. Capture private machine-readable preflight data:
   - `/tmp/c8volt-release-docs-help-verify capabilities --json`
   - `/tmp/c8volt-release-docs-help-verify config show --json`
   - `/tmp/c8volt-release-docs-help-verify config test-connection --json`
   Use these outputs to understand command paths, flags, config resolution,
   active profile, base URL, and cluster metadata. Do not convert public
   examples to JSON unless the surrounding section is explicitly about
   automation or JSON output.
4. Extract all executable examples from:
   - `README.md`
   - `docs/**/*.md`
   - generated docs source or generated docs output used by the site
   - CLI help output from every command family, including nested ops commands
   In README-only mode, extract executable examples from `README.md` only, then
   check help output just far enough to detect contradictions with README.
5. Extract c8volt commands from `demos/vhs/**/*.tape` into a separate VHS syntax
   review list. Do not include VHS tapes in the big docs/help genericization or
   example-reduction pass.
6. Include command help in the audit:
   - run `c8volt --help`
   - run help for every top-level command
   - run help for every subcommand and ops command family
   - enumerate every nested `ops` command and alias from help output, including
     every `ops execute ...`, `ops purge ...`, `ops repair ...`, and future ops
     subcommand that appears in the binary
   - compare help examples with README and docs examples so they do not
     contradict each other
7. Classify every docs/help example:
   - read-only lookup
   - bounded search/list
   - dry-run preview
   - normal mutation
   - critical mutation
   - stdin or pipeline
   - setup-only snippet
8. Classify VHS tape commands separately as syntax-only review items. Preserve
   their concrete values and recorded flow.
9. Create or discover temporary data as needed:
   - deploy embedded fixtures privately when useful
   - start process instances to obtain real keys
   - create incidents or jobs creatively through supported cluster behavior
   - discover process definition, process instance, incident, job, resource, and
     tenant values from the cluster
   - clean up temporary data after validation when practical
10. For placeholder examples, run the same command shape with real values
   substituted privately. Public examples must remain placeholder-based.
11. Test critical mutation behavior against the real dev cluster, with special
    focus on destructive examples that remain in docs/help:
   - purge/delete/cancel/resolve commands
   - broad ops cleanup commands
   - commands that remove process definitions or process instances
   - commands that change jobs, variables, incidents, or task state
   Prefer isolated fixture data, preview first, real mutation second, and a
   post-check proving the expected cluster impact.
12. Build and execute a complete ops validation matrix:
    - include every nested `ops` command discovered from help, not just commands
      that currently have docs/help examples
    - record command path, aliases, support status for the target v89 cluster,
      impact class, required fixture/setup data, dry-run support, automation or
      JSON support, confirmation behavior, and cleanup expectations
    - for read-only or dry-run-only ops commands, run the command and verify the
      observed output is bounded and actionable
    - for supported mutating ops commands, run a preview/dry-run when available,
      then run the real mutation on disposable scoped data and verify the
      expected post-condition
    - for unsupported or intentionally blocked ops commands, verify failure
      happens before mutation with a clear diagnostic
    - if an ops command has no public example but is part of the binary, still
      validate it and decide whether it needs a docs/help example, an explicit
      omission, or a follow-up issue
13. Decide for each docs/help example whether to keep, reduce, rewrite, move behind a
    warning, or remove:
    - keep only examples that represent normal operator workflows
    - include destructive execution examples for ops commands where the real
      workflow is destructive
    - reduce repeated flag variants
    - remove untestable v87/v88 examples
    - remove nonstandard `--no-wait` style examples from normal docs/help
    - replace hardcoded fixture IDs and runtime keys with placeholders
    - label examples as read-only, harmless, or destructive where the format
      allows it
    - add explicit destructive warnings for destructive examples that remain
14. For each VHS tape command, verify only that the c8volt command path, flags,
    and required arguments are syntactically valid for the current binary. Do
    not run a VHS rewrite. Do not edit tape files. If a VHS command is stale,
    ambiguous, v87/v88-only, or syntactically suspicious, record the tape path
    and reason for the final VHS review warning section.
15. If edits are needed, update source files rather than generated output
    when a generation path exists, then regenerate affected docs.
16. After edits, rerun every changed docs/help example against the real cluster.
17. Run targeted automated validation:
    - `go test ./cmd -count=1`
    - `go test ./docsgen -count=1` when generated documentation or docs links
      changed
    - broader tests only when product behavior changed

Suggested investigation commands:
- `rg -n -- 'C87_|C88_|C89_|--no-wait|225179981|<process|<incident|<job|<resource|ops ' README.md docs cmd demos/vhs -g '*.md' -g '*.go' -g '*.tape'`
- `rg -n -- 'Example|Examples|Use:|Aliases:|ops execute|ops purge|ops repair' cmd docs README.md demos/vhs`
- `/tmp/c8volt-release-docs-help-verify capabilities --json`
- `/tmp/c8volt-release-docs-help-verify config show`
- `/tmp/c8volt-release-docs-help-verify config test-connection`
- `/tmp/c8volt-release-docs-help-verify embed list`
- `/tmp/c8volt-release-docs-help-verify embed deploy --all --run`
- `/tmp/c8volt-release-docs-help-verify get cluster version`
- `/tmp/c8volt-release-docs-help-verify get cluster topology`
- `/tmp/c8volt-release-docs-help-verify get pd --latest --limit 5`
- `/tmp/c8volt-release-docs-help-verify get pi --state active --limit 5`
- `/tmp/c8volt-release-docs-help-verify get incident --state active --limit 5`
- `/tmp/c8volt-release-docs-help-verify ops execute smoke-test --dry-run`
- `/tmp/c8volt-release-docs-help-verify ops execute retention-policy --retention-days 90 --dry-run`
- `/tmp/c8volt-release-docs-help-verify ops purge orphan-process-instances --dry-run`
- `/tmp/c8volt-release-docs-help-verify ops purge process-instances-with-incidents --dry-run`
- `/tmp/c8volt-release-docs-help-verify ops purge all-process-definitions --dry-run`
- `/tmp/c8volt-release-docs-help-verify ops execute retention-policy --retention-days 90`
- `/tmp/c8volt-release-docs-help-verify ops purge orphan-process-instances`
- `/tmp/c8volt-release-docs-help-verify ops purge process-instances-with-incidents`
- `/tmp/c8volt-release-docs-help-verify ops purge all-process-definitions`
- enumerate all ops commands from help before finalizing:
  `/tmp/c8volt-release-docs-help-verify ops --help`,
  `/tmp/c8volt-release-docs-help-verify ops execute --help`,
  `/tmp/c8volt-release-docs-help-verify ops purge --help`,
  `/tmp/c8volt-release-docs-help-verify ops repair --help`, then run help for
  every nested command listed by those outputs
- `go test ./cmd -count=1`
- `go test ./docsgen -count=1`

Output expectations:
- Report the docs/help/VHS surfaces scanned.
- Report every command family validated with real cluster calls.
- Report every nested `ops` command discovered, how it was classified, and
  whether it was validated by dry-run, real execution, expected unsupported
  failure, or explicit documented omission.
- Report examples removed or rewritten because they were v87/v88-only,
  fixture-specific, stale, unbounded, unsafe, too slow, or too chatty.
- Report examples kept as placeholders and the private real values used to
  validate them, without turning those real values into public docs.
- Report destructive examples included in public docs/help and confirm their
  destructive labels or warning text.
- Report destructive examples intentionally excluded only when they are too
  broad, too slow, unbounded, or not representative of normal operator use.
- Report validation commands run, including cluster preflight and Go tests.
- Report VHS tapes checked for c8volt command syntax.
- Report VHS files that need human review, and do not edit those tapes.
- End with this section:

  Critical Mutation Cases Tested:
  - Command:
  - Real test performed:
  - Result:
  - Public docs/help decision: included with destructive warning, or excluded
  - Reason:

  VHS Files Needing Manual Review:
  - Tape:
  - Command:
  - Reason:
```
