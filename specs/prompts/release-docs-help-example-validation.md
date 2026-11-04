# Release Docs And Help Example Validation Prompt

Use this prompt for a separately requested live release audit of README,
documentation, and command-help examples against a chosen Camunda 8.10
development cluster. The default is a findings report; documentation repairs are
a separate, explicitly requested step. VHS is entirely outside this workflow.
Editing or reviewing this template does not authorize its execution. Do not run
this live workflow as part of implementation validation.

Before using this prompt, follow `specs/prompts/AGENTS.md`. Do not treat prompt
templates as product behavior or release-change source material.

```text
Audit every c8volt command example in README.md, documentation, generated CLI
references, and command help against the intended Camunda 8.10 development
cluster. Also validate every nested ops command, including commands without a
public example. Do not scan, syntax-check, execute, or edit VHS tapes or their
supporting scripts. VHS maintenance is a separate workflow.

Run cluster operations only when the caller explicitly requests this live audit
for the specified target. Editing this template or implementing a feature does
not authorize a live run.

Inputs:
- Release under validation: <VERSION>
- Target Camunda release line: 8.10 only; record the actual patch/prerelease
- Absolute configuration path: <CONFIG_PATH>
- Configuration profile: <PROFILE>, or explicitly no profile
- Expected development cluster identity/base URL: <CLUSTER>
- Disposable fixture prefix: <FIXTURE_PREFIX>
- Authorized mutation boundary: <DISPOSABLE_SCOPE>
- Private report/evidence directory: <REPORT_DIR>
- Optional build output path: /tmp/c8volt-release-docs-help-verify

Default mode — report only:
Execute validation within the caller's authorized scope and report findings.
Do not edit README, documentation, command help, tests, or product code. Building
a temporary binary, capturing private evidence, and creating or cleaning up
explicitly authorized disposable fixtures are permitted validation activities.
Report-only describes repository edits; it does not imply cluster reads only.
This template does not itself grant permission to mutate a cluster. Reuse
explicit authorization already provided by the caller; resolve missing target
or mutation boundaries before dependent cluster operations. Continue independent
static inventory work while those inputs are missing.

Repair mode:
Only when the caller explicitly requests fixes, repair confirmed documentation
and help-example findings, regenerate affected docs, and rerun changed examples.
Keep runtime implementation changes out of scope; report product defects
separately. Do not commit, push, or publish unless asked.

README-only mode:
If explicitly requested, restrict the example inventory to README.md and consult
help only to check its examples for contradictions. The exhaustive command-help,
ops matrix, and feature-area coverage requirements below apply to the full audit,
not README-only mode. Report-only remains the default.

Scope and example rules:
1. Public executable examples target Camunda 8.10, including features available
   only on that release line. Flag ordinary setup examples still targeting 8.9
   or 8.8 for rewriting in repair mode. Preserve support documentation for 8.9
   and 8.8 and explicitly labeled compatibility examples; inventory those examples
   but exclude their older-target execution from this 8.10 audit. Check version
   hints such as "only c8.10+" or "new in c8.10" against the implementation.
2. Keep public examples generic, using placeholders for keys, tenant IDs, BPMN
   process IDs, and file paths. Substitute real values privately during testing.
   Record prerequisite data and setup steps. Missing fixtures are a validation
   blocker unless the documentation incorrectly describes those prerequisites;
   they are not automatically documentation defects.
3. Separate execution findings from editorial suggestions. Do not reduce working
   examples, add repetitive impact labels or warnings, or describe output layout
   merely to satisfy a style preference. Keep help focused on what commands do,
   selection, safety, compatibility, and completion semantics.
4. Prefer ordinary operator workflows. Flag exception-path options such as
   --no-wait in everyday examples for review; retain them where exceptional
   behavior is explicitly documented. Do not replace real mutation examples with
   dry-run-only examples.
5. Run the documented command shape with only necessary private substitutions
   and explicit test configuration. Preserve pipelines, flags, and interaction
   behavior. If extra selectors or limits are needed for safe execution, record
   that variant separately; do not count it as proof of the original example.
6. Mutations must affect only disposable resources within the authorized scope.
   Preview when supported, execute, then verify the post-condition. Creation and
   deployment are mutations too. Do not assume a tenant filter restricts explicit
   keys or every cleanup operation. If a broad command cannot be safely isolated,
   use an explicitly authorized disposable cluster or mark it blocked. Family
   traversal can include ancestors, siblings, and descendants outside the initial
   selection; --limit does not cap the expanded mutation scope. Check the entire
   affected family against the authorized boundary before mutation.
7. Bound runtime, output capture, and resource creation. Stop long-running or
   unbounded commands through the harness and record why they stopped. Watch and
   wait examples need a defined observation period and deliberate termination;
   do not leave background commands running.
8. Exercise documented interactive examples with terminal input and explicit
   confirmation responses. Do not silently add --auto-confirm or --automation
   to make them pass. Use those flags when they are part of the documented example.
9. Keep credentials and sensitive configuration out of reports and public docs.
   Retain only necessary redacted configuration evidence and fixture mappings.
10. Validate the current flag grammar against registrations and capabilities:
    bare properties select, --set-* supplies written values, and --with-* adds
    detail. Check --retries versus --set-retries, --timeout versus --set-timeout,
    --set-vars/--set-vars-file, --set-assignee, --set-bid, --with-stats, and the
    valueless --throw-bpmn-error action with --set-error-code. Renamed legacy
    spellings have no aliases. Report stale examples instead of silently fixing
    their syntax during execution; preserve literal data values and selectors.
11. Check each command's stated completion and output contract. Distinguish
    accepted requests from confirmed outcomes and process creation from process
    completion. Capture stdout and stderr separately: query and machine results
    use stdout; human mutation summaries and diagnostics use the configured
    logger; prompts use stderr. For supported JSON results, check the command's
    actual schema and require exactly one document through EOF. Do not assume
    every JSON-capable command uses the shared envelope.

Required workflow:
1. Inventory examples before executing them:
   - README.md and docs/**/*.md, including generated references
   - command Example metadata and help for every command, including hidden
     commands, nested commands, and documented aliases
   - setup snippets, pipelines, and local-only commands as well as cluster calls
   Exclude VHS assets and scripts entirely. Do not follow screencast links into
   VHS files. Deduplicate identical examples for execution while retaining every
   source location and any differences in surrounding prerequisites.
2. Build a temporary binary from the current checkout:
   GOCACHE=/tmp/c8volt-gocache go build -o /tmp/c8volt-release-docs-help-verify .
   Record the checkout commit and working-tree state. Use this binary as the
   equivalent of public c8volt examples.
3. Pin --config <CONFIG_PATH> and the selected --profile <PROFILE> on every
   cluster invocation; omit --profile only when no profile was explicitly
   selected. Check environment overrides that could change the effective target.
   Do not rely on config discovery beside a binary built under /tmp. Examples
   specifically testing configuration resolution need isolated local setup and
   an explicit check of the effective target before any cluster request.
   Inspect the effective settings privately with config show, which emits
   sanitized YAML, not JSON. Confirm the endpoint matches <CLUSTER> and
   app.camunda_version resolves to 8.10 before contacting it. Reuse the selected
   configuration's authentication mode: none, cookie, or OAuth2.
4. Run version and config validate, then config test-connection --json and get
   cluster version with the pinned configuration. Confirm the gateway and every
   reported broker use the 8.10 release line and reported partitions are healthy.
   Compare health values case-insensitively. A successful connection-test exit
   does not prove version compatibility: mismatches normally produce warnings,
   not failure exits. Treat mismatched or unknown versions, unhealthy topology,
   or failed connectivity as blockers and stop further cluster execution.
   Retain the static inventory and evidence of the blocker.
5. Capture capabilities --json and help to map the command surface and supported
   flags, output modes, and automation support. Compare source metadata,
   generated references, README, and live help for contradictions, obsolete
   flags, or stale examples. Keep normal error exit codes enabled in the audit
   harness; record any documented example that intentionally overrides
   them rather than treating exit zero alone as proof of success.
6. Give every example a stable ID and classify it as read-only lookup, search,
   preview, mutation, pipeline, local setup, or bounded watch/wait. Record its
   prerequisites, expected outcome, mutation scope, and verification method.
7. Create disposable fixtures within the authorized boundary as needed, using
   supported Camunda behavior. Track created definitions, instances, incidents,
   jobs, files, and other resources so their ownership and cleanup are clear.
   Do not use unrelated existing resources as mutation fixtures.
8. Execute every applicable example with real private substitutions. Capture
   exit status, separate stdout/stderr, duration, and evidence of the expected
   outcome. A successful exit alone is insufficient proof of a mutation. Run
   post-checks for cancellation, deletion, purge, repair, resolution, updates,
   deployment, and creation. Validate local-only snippets locally rather than
   requiring artificial cluster calls.
9. Build the complete ops matrix, including analyse, execute, purge, repair, and
   any additional groups present in the binary. Record every command and alias,
   8.10 support status, fixtures, impact, dry-run support, confirmation behavior,
   automation/JSON support, post-check, and cleanup requirements. For mutating
   commands, execute a preview when available and the real workflow on disposable
   data. For unsupported commands, verify the expected rejection before mutation
   where safe. Grouping commands need help/navigation validation, not mutation.
   Do not omit commands because they lack examples; report that gap separately.
10. Cover the following feature areas on owned fixtures, including commands
    without public examples. Record fixtures, scope, expected outcomes,
    verification, and any blocked cases:
    - Native user tasks: search, effective variables, assignment, unassignment,
      completion, and supported explicit-key collections.
    - Variable ownership: process-root, element-local, and task-owned updates;
      verify the intended scope and preservation of unrelated values.
    - Business IDs: creation, lookup/selection, post-creation assignment, and
      assignment with job completion, with the documented eligibility rules.
    - Suspension/resumption: inspect and expect suspended state, exact-key
      actions, and eligible descendants with --tree. Distinguish this scope
      from cancellation/deletion, which can expand to the whole family.
    - Definition lifecycle and cleanup: lifecycle state, --with-stats including
      separate suspended counts, and forced cleanup of unfinished instances.
      Definition disappearance alone does not prove deferred history deletion.
11. Clean up owned disposable resources within the authorized boundary and verify
    cleanup. Report residual resources, stopped processes, and cleanup failures.
    Never broaden cleanup to unrelated cluster data.
12. Produce the report before editorial repair. If fixes were explicitly
    requested, update source docs or command example metadata, not generated
    files directly; run make docs-content where applicable. Rerun each changed
    example, preserving before/after evidence. Run go test ./cmd -count=1 for
    command-help changes and go test ./docsgen -count=1 for generated-doc changes,
    plus git diff --check. Do not change runtime behavior to make an example pass.

Output expectations:
- Identify the version/commit, effective 8.10 configuration and observed cluster
  release versions, target identity, authorized scope, and mode.
- List all scanned README/docs/help surfaces, including hidden command coverage.
- Provide an example inventory with ID, all source locations, command shape,
  prerequisites, private substitution reference, status, exit result, expected
  outcome, post-check evidence, and proposed correction where applicable.
- Use passed, failed, blocked, or excluded status. Distinguish documented expected
  unsupported failures from unexpected failures. Mark older-version examples
  needing baseline updates separately from deliberate compatibility examples.
  Mark older-target execution explicitly as excluded, never passed. Do not defer
  applicable 8.10 examples merely because they use newer features. Explain all
  unexecuted cases.
- Provide the complete ops matrix, distinguishing help-only grouping checks,
  previews, real execution, expected unsupported rejection, and blocked cases.
- Include the feature-area coverage matrix with actual post-condition evidence
  and reasons for missing coverage; an example count alone is insufficient.
- Summarize total source occurrences, unique examples, executed examples,
  failures, blockers, exclusions, and coverage gaps. Do not claim complete runtime
  validation while applicable examples remain unexecuted.
- Separate confirmed documentation defects, product defects, missing environment
  prerequisites, and optional editorial suggestions.
- For critical mutations, report the exact scoped test, preview result,
  post-condition evidence, and cleanup outcome, including residual fixture data.
- Report actual checks run and any repairs explicitly authorized and completed.
- State that VHS was excluded entirely; do not produce a VHS review inventory.
```
