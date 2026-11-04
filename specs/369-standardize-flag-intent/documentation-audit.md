# v5 documentation and help audit

Date: 2026-11-04. Scope: authored repository documentation, command help and examples, configuration samples, demo sources, and maintained invocation fixtures.

## Findings and changes

- README reduced from 960 to 134 lines. It now opens with v5 and Camunda 8.10, followed by setup, ops playbooks, common resource operations, flag intent, and scripting rules.
- Help prose reduced from approximately 9,600 to 2,500 words. Common examples replace long lists of flag combinations and feature-specific implementation detail. Essential scope, confirmation, assignment, and completion distinctions remain.
- Each ops playbook uses the same short structure: purpose, examples, execution rules, and reference links. Shared controls are explained once in the ops index.
- Configuration examples and recording sources now target 8.10. The built-in adapter default remains 8.9 and is documented explicitly; runtime defaults were not changed.
- Version-specific help and product guidance use “only c8.10+”. Compatibility release prompts and historical feature evidence retain their actual version context.
- Both authored guides inside `docs/cli/` were reviewed separately from generated command pages. Generated references and the homepage were refreshed through `make docs-content`.
- Current write, enrichment, and action examples follow the implemented v5 grammar. Retry selection is distinct from retry mutation; HTTP timeout is distinct from job timeout; BPMN errors use a bare action and an explicit code flag. Repair zero retries still means skip restoration.
- Recording config, environment names, process IDs, and fixture paths agree on 8.10. Earlier GIFs are not presented as v5 examples; regenerating recordings requires a separate live workflow.
- The offline example guard recognizes the sample `order` process and `order.bpmn` file as requiring setup, rather than assuming they exist in a release target. Variable-file examples use the existing `vars.json` fixture convention.

## Scan coverage

The static scan read 2,678 authored text files, including 1,297 documentation files. It excluded generated references, generated clients, upstream API snapshots, and binary media from editorial rewriting. Historical specs/tasks/memory and negative migration tests were inspected but preserved.

No obsolete accepted flag spelling was found in maintained documentation, demo tapes, or scripts. Remaining legacy spellings belong to historical evidence or intentional migration/rejection coverage. Local file links and published site routes resolve against repository content.

The built CLI rendered 69 help pages: the root, all 62 public command nodes, and six framework help/completion pages. Help produced no diagnostic output. `capabilities --json` supplied the command inventory locally; no Camunda operation was executed.

## Validation

Passed:

- Targeted command help, metadata, canonical-flag, and documentation-generation checks.
- `go test ./cmd -count=1` — full isolated command suite, including existing runtime, logger, prompt, and migration coverage.
- `go test ./cmd ./config ./docsgen -count=1` — complete command, configuration, and generator package checks after final review.
- `go test ./integration/cli -run '^TestExampleLiteralResourceKeyGuard$' -count=1` — untagged, text-only helper; no integration build tag or live suite.
- `make docs-content` and a local CLI build.
- `bash -n` on the recording helpers and updated observable-run smoke script.
- Static flag-spelling, local-link, fixture-path, and command-help inventory checks.
- Normalized Go source comparison across 65 changed command files: execution logic, accepted flag names, types, and defaults remain unchanged; differences are help, examples, descriptions, and completion wording.
- `gofmt` and `git diff --check`.

Tests cleared inherited integration/configuration controls and build tags and used a temporary Go build cache. No live cluster, integration suite, smoke workflow, or recording was run. This audit establishes documentation consistency and isolated CLI behavior, not new real-engine compatibility evidence.

## Final diff review

The follow-up review corrected limit and worker descriptions that incorrectly advertised explicit zero as unlimited or automatic. Commands that distinguish omission from zero now say to omit the flag. Existing zero-value rejection tests remain intact, with focused help checks added for the wording.

The local example checker now also validates pipeline command segments, rejects unexpected positional arguments, and accepts canonical attached error-code values. Dedicated negative cases verify that malformed or legacy BPMN-error examples cannot pass the check. No command execution logic was changed.

## Subpage implementation review — follow-up

The earlier counts and editorial choices record the initial audit. Subsequent README revisions restore the v5 feature scope, “done is done”, standard commands before family cleanup, and benefit-led c8ctl positioning. Version notes are now plain parenthetical hints.

Reviewed all 15 authored subpages plus README against command registration, validation, configuration resolution, renderers, and service workflows. The 63 generated command references were regenerated from source; all 69 public/framework help pages rendered locally without diagnostics. Generated clients, upstream API snapshots, vendored site dependencies, and historical feature evidence were not rewritten.

Corrections and clarifications:

- Config lookup starts in the working directory, not the executable directory. README and the configuration guide now match `cmd/root_config.go`.
- Runtime-variable guidance distinguishes process-instance root scope, exact element-local scope, and task effective/owning scopes, including new task-local names and read-back confirmation.
- Repair guides state the default retry count of 1, zero-as-skip behavior, active-incident selection, and variable failures blocking dependent resolution.
- Cleanup guides explain what each limit counts: incidents, ended retention seeds, inspected orphan candidates, or definition versions. Retention uses end dates and may expand to newer relatives.
- Smoke-test dry-run checks connectivity without mutations. No-wait affects cleanup confirmation; deployment, creation visibility, and traversal proof remain.
- Full process-definition/history purge requires c8.9+; suspended-instance handling requires c8.10+. Deferred history deletion remains distinct from definition disappearance.
- Shared report guidance distinguishes mutation audit files from analysis results and recommends a fresh report path for previews.
- Every ops playbook retains the same purpose → examples → execution → shared controls → references structure. Execution explains selection before mutation and confirmation. All eight remain under 200 words, including front matter and references.

Validation passed:

- Targeted command example/help, configuration validation/redaction, and config-precedence tests.
- Targeted isolated ops-service tests for repair, smoke-test proof/cleanup, orphan and definition purge, retention, and business-ID cleanup.
- Retention-discovery service tests for end-date selection and eligible-seed limits.
- `go test ./docsgen ./config -count=1` and `make docs-content`.
- Local validation of the README config and the guide's local, OAuth, and profile configurations, using only the read-only `config validate` command and a dummy OAuth secret.
- All 69 help pages and the capability inventory from a fresh local binary.
- Static checks of 51 command-example lines, 93 local links/routes, Markdown fences, canonical flags, version hints, README section order, and generated homepage synchronization.
- Final diff and whitespace review. No production Go code or command behavior changed.

All service validation used isolated test-owned dependencies. No live Camunda, integration, smoke-test workflow, or recording was executed; real-engine behavior was not revalidated.
