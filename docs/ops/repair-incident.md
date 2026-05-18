---
title: "Repair Incident"
permalink: /ops/repair-incident/
parent: "C8 Ops CLI"
nav_order: 6
has_toc: true
---

# c8volt ops repair incident

## The Problem

Incident repair is rarely one API call. Operators often need to inspect the incident, update variables on the affected process-instance scope, restore retries or timeout on the related job when one exists, resolve the incident, confirm it cleared, and keep an audit trail of every decision.

## The Promise

`c8volt ops repair incident` turns that remediation chain into a fixed-target workflow. It accepts explicit incident keys, stdin keys, or incident filters, freezes the incident set before mutation, plans variable, job, and resolution steps, then reports what was planned, skipped, submitted, confirmed, or failed.

Aliases: `inc`.

## Use When

- repairing one known incident key from incident-aware process-instance output
- repairing a small filtered set of active incidents after previewing the match
- restoring job retries or timeout before resolving an incident
- setting process-instance-scope variables once per unique scope before dependent incident resolution
- producing a Markdown or JSON audit report for operator handoff

## Command At A Glance

```bash
c8volt ops repair incident --key <incident-key> --dry-run
c8volt ops repair incident --key <incident-key> --retries 3 --job-timeout 5m --dry-run
c8volt ops repair incident --state active --error-type io_mapping_error --limit 5 --dry-run
printf '%s\n' "$INCIDENT_KEY_A" "$INCIDENT_KEY_B" | c8volt ops repair incident - --dry-run
c8volt ops repair incident --key <incident-key> --vars '{"approved":true}' --dry-run
c8volt ops repair incident --key <incident-key> --auto-confirm --report-file repair-incident.md
c8volt --automation --json ops repair incident --key <incident-key> --dry-run
```

## Built From Lower-Level Commands

This is the conceptual flow. The implemented command calls c8volt services directly and freezes the target set before mutation.

```bash
c8volt get incident --key <incident-key>
c8volt get incident <incident-filters>
c8volt update pi --key <process-instance-key> --vars <json>
c8volt update job --key <job-key> --retries <count>
c8volt update job --key <job-key> --timeout <duration>
c8volt resolve incident --key <incident-key>
```

Keyed mode and search mode are mutually exclusive. `--key` and stdin `-` select incident keys. Search mode uses incident filters such as `--state`, `--error-type`, `--error-message`, `--bpmn-process-id`, `--pi-key`, `--root-key`, `--flow-node-id`, creation-time bounds, `--batch-size`, and `--limit`.

## Workflow

```text
read incident keys or search filters
        |
        v
discover and freeze incident targets
        |
        v
dedupe process-instance variable scopes
        |
        v
plan variable, job, resolution, and confirmation steps
        |
        +--> --dry-run: report plan, mutate nothing
        |
        v
confirm, auto-confirm, or automation-confirm
        |
        v
update requested variables once per scope
        |
        v
update related job retries and timeout where applicable
        |
        v
resolve each incident
        |
        v
confirm clearance unless --no-wait is set
        |
        v
write optional audit report
```

## Dry Run

`--dry-run` resolves the target set and builds the full repair plan without updating variables, changing jobs, or resolving incidents. Human output emphasizes incident count, process-instance count, related job count, variable scope count, and whether any incidents have no related job.

Verbose output can list frozen incident keys, process-instance keys, job keys, and planned variable scopes.

## Real Execution

Without `--dry-run`, interactive runs first execute the same plan as a preflight and ask for confirmation. `--auto-confirm` and `--automation` allow supported unattended repair.

When `--vars` or `--vars-file` is supplied, the variables are applied once per unique process-instance scope before incident resolution. If a variable update fails for a scope, dependent incident resolution is blocked for that scope. Job retry and timeout steps run only when the incident has a related job; incidents without related jobs are reported and still proceed to incident resolution.

## Reports

Reports use schema version `ops.repair.v1`. They include command metadata, discovery mode, incident filters, frozen incident and process-instance keys, variable scopes, job applicability, per-incident plan statuses, remaining active incidents when checked, notices, errors, automation flags, timestamps, duration, and final outcome.

Report format is inferred from `--report-file` unless `--report-format markdown|json` is supplied.

## Demo

The VHS source is `demos/vhs/ops-repair-incident.tape`.

```bash
c8volt ops repair incident --state active --limit 1 --dry-run
c8volt ops repair incident --state active --limit 1 --auto-confirm --report-file /tmp/c8volt-vhs/reports/repair-incident.md
```

## Failure And Safety Notes

- `--key` means incident key; it cannot be combined with search filters.
- Stdin `-` can be combined with repeated `--key`, but not with search filters.
- `--vars` and `--vars-file` are mutually exclusive and must contain a JSON object.
- `--retries 0` skips retry restoration; it does not set retries to zero.
- Existing report files are preserved for dry-run and preflight-style report planning.
- `--json` cannot be combined with `--verbose` for this command.

## Related Commands

- [ops repair incident](/cli/c8volt_ops_repair_incident/)
- [get incident](/cli/c8volt_get_incident/)
- [update process-instance](/cli/c8volt_update_process-instance/)
- [update job](/cli/c8volt_update_job/)
- [resolve incident](/cli/c8volt_resolve_incident/)
