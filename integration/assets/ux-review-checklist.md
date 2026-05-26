# UX Review Checklist

Use this checklist while reviewing `report.md` and command evidence from a
release integration suite. Capture concrete command examples and log file paths
for every finding.

## Grammar And Rhythm

- Command families use consistent nouns: process-instance, process-definition,
  incident, job, variable, tenant, cluster.
- Aliases feel predictable and do not conflict with common operator intent.
- Dry-run output, mutation output, and confirmation text use the same vocabulary.
- Error messages name the command, selected scope, and blocked prerequisite.

## Automation

- `--json`, `--automation`, `--keys-only`, and `--auto-confirm` behave
  consistently on comparable commands.
- Non-interactive runs do not prompt for human recovery choices.
- Stdin key workflows accept only clean key streams and reject mixed output.
- Report file paths and command-generated audit reports are announced clearly.

## Safety

- Destructive commands show selected scope before mutation.
- Dry-run output proves no mutation was submitted.
- Real mutation output includes enough evidence to identify affected resources.
- Unsupported-version paths fail before mutation with a clear diagnostic.

## Bounded Output

- List/search commands use limits or page controls in normal operator paths.
- Verbose details stay behind `--verbose`, reports, or JSON where appropriate.
- Large batches summarize totals while preserving traceable evidence.

## Follow-Up Candidates

- Missing suite data generator for an important workflow.
- Missing command report format or weak report evidence.
- Inconsistent flag name, alias, wording, or confirmation behavior.
- Unclear remediation text for Camunda runtime failures.
- Useful operator workflow not represented by docs/help examples.

