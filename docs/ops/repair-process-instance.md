---
title: "Repair Process Instance"
permalink: /ops/repair-process-instance/
parent: "C8 Ops CLI"
nav_order: 7
has_toc: true
---

# c8volt ops repair process-instance

## The Problem

Support work often starts from process-instance keys or process-instance filters, not incident keys. The operator needs c8volt to discover the active incidents for those instances, skip instances that have nothing repairable, dedupe shared scopes, and then run the same careful variable, job, incident-resolution, and confirmation steps.

## The Promise

`c8volt ops repair process-instance` selects process instances by key, stdin, or process-instance search filters, freezes the repairable process-instance and incident set, then reuses the incident repair workflow for variables, related jobs, incident resolution, confirmation, and audit reporting.

Aliases: `pi`, `pis`, `process-instances`.

## Use When

- repairing all active incidents associated with one known process instance
- previewing a bounded search of incident-bearing process instances
- narrowing repair to direct active incidents with `--direct-incidents-only`
- applying the same variable payload to every matched repair scope
- producing an audit trail that distinguishes repaired, skipped, and duplicate process-instance targets

## Command At A Glance

```bash
c8volt ops repair process-instance --key <process-instance-key> --dry-run
c8volt ops repair process-instance --key <process-instance-key> --retries 3 --job-timeout 5m --dry-run
c8volt ops repair process-instance --state active --limit 5 --dry-run
c8volt ops repair process-instance --direct-incidents-only --state active --limit 5 --dry-run
printf '%s\n' "$PI_KEY_A" "$PI_KEY_B" | c8volt ops repair process-instance - --dry-run
c8volt ops repair process-instance --key <process-instance-key> --auto-confirm --report-file repair-process-instance.md
c8volt --automation --json ops repair process-instance --key <process-instance-key> --dry-run
```

## Built From Lower-Level Commands

This is the conceptual flow. The implemented command calls c8volt services directly and freezes the target set before mutation.

```bash
c8volt get pi --key <process-instance-key> --with-incidents
c8volt get pi <process-instance-filters> --incidents-only --keys-only
c8volt update pi --key <process-instance-key> --vars <json>
c8volt update job --key <job-key> --retries <count>
c8volt update job --key <job-key> --timeout <duration>
c8volt resolve incident --key <incident-key>
```

Keyed mode and search mode are mutually exclusive. With no keys and no stdin, the command uses process-instance search mode and automatically limits discovery to incident-bearing process instances. `--direct-incidents-only` adds a stricter direct active incident match and can be combined with incident-state, incident-error-type, and incident-error-message filters.

## Workflow

```text
read process-instance keys or search filters
        |
        v
discover incident-bearing process instances
        |
        v
discover active incidents for the frozen set
        |
        v
skip process instances without active repair targets
        |
        v
dedupe variable scopes and related jobs
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
run the shared incident repair steps
        |
        v
write optional audit report
```

## Dry Run

`--dry-run` searches or reads process-instance targets, discovers their active incidents, and builds the repair plan without updating variables, changing jobs, or resolving incidents. Human output shows process-instance count, skipped process-instance count, incident count, related job count, variable scope count, and the planned final outcome.

Verbose output can list frozen process-instance keys, skipped keys, incident keys, job keys, and planned variable scopes.

## Real Execution

Without `--dry-run`, interactive runs first execute the same plan as a preflight. If no active repair targets are found, the command reports the plan and submits no mutation. Otherwise it asks for confirmation, freezes the planned process-instance set, and runs repair against the discovered incident keys.

Variable updates are applied once per unique process-instance scope before dependent incident resolution. Related job retry and timeout updates run only where the incident has a related job. `--no-wait` returns after repair mutations are accepted without waiting for incident or retry confirmation.

## Reports

Reports use schema version `ops.repair.v1`. They include command metadata, discovery mode, process-instance filters, direct-incident filter settings, frozen process-instance and incident keys, skipped process-instance keys, variable scopes, job applicability, per-incident plan statuses, remaining active incidents when checked, notices, errors, automation flags, timestamps, duration, and final outcome.

Report format is inferred from `--report-file` unless `--report-format markdown|json` is supplied.

## Demo

The VHS source is `demos/vhs/ops-repair-process-instance.tape`.

```bash
c8volt ops repair process-instance --direct-incidents-only --state active --limit 1 --dry-run
c8volt ops repair process-instance --direct-incidents-only --state active --limit 1 --auto-confirm --report-file /tmp/c8volt-vhs/reports/repair-process-instance.md
```

## Failure And Safety Notes

- `--key` means process-instance key; it cannot be combined with process-instance search filters.
- Stdin `-` can be combined with repeated `--key`, but not with search filters.
- Search mode automatically selects incident-bearing process instances.
- `--direct-incidents-only` filters process instances by direct incident fields before repair.
- Process instances with no active incidents are skipped and reported.
- `--vars` and `--vars-file` are mutually exclusive and must contain a JSON object.
- `--json` cannot be combined with `--verbose` for this command.

## Related Commands

- [ops repair process-instance](/cli/c8volt_ops_repair_process-instance/)
- [get process-instance](/cli/c8volt_get_process-instance/)
- [get incident](/cli/c8volt_get_incident/)
- [update process-instance](/cli/c8volt_update_process-instance/)
- [update job](/cli/c8volt_update_job/)
- [resolve incident](/cli/c8volt_resolve_incident/)
