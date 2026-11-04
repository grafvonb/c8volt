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
