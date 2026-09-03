---
title: "Analyse API Latency"
permalink: /ops/analyse-api-latency/
parent: "C8 Ops CLI"
nav_order: 1
has_toc: true
---

# c8volt ops analyse api-latency

## Purpose

API timeout reports often need a quick split between control-path reads, query reads, and direct keyed reads. `c8volt ops analyse api-latency` measures those paths in bounded stages without deploying, creating, cancelling, or deleting Camunda resources.

The result is diagnostic evidence, not a capacity benchmark. It compares measured read behavior, records safe findings, and states the limits of what a read-only sample can prove.

## Use When

- production safety matters more than write-path evidence
- topology, process-definition search, or process-instance search calls appear slow or unreliable
- direct keyed reads should be compared with search-derived reads without supplying keys by hand
- a compact Markdown or JSON report is needed for support or platform handoff

## Basic Usage

```bash
c8volt ops analyse api-latency
```

Generated reference: [ops analyse api-latency](/cli/c8volt_ops_analyse_api-latency).

## Best Variants

```bash
c8volt ops analyse api-latency --count 6 --workers 2
c8volt ops analyse api-latency --report-file api-latency.md
c8volt ops analyse api-latency --json --report-file api-latency.json
```

`--count` is the total primary sample-cycle budget across all stages. `--workers` is the maximum closed-loop worker count and the final stage width. The sample count must be large enough to exercise every displayed stage; for example, four workers require at least seven samples for stages 1, 2, and 4.

## What It Measures

Each primary read-only cycle measures cluster topology, process-definition search, and process-instance search. When a measured search returns a reusable key, the command also measures one direct keyed read for that resource type.

Missing keys, disappearing resources, and unsupported keyed reads are reported as unavailable evidence instead of successful measurements. Camunda 8.7 does not support the process-instance keyed-read evidence used by this diagnostic; the rest of the read-only analysis still runs.

## Output And Reports

Default output shows the read-only scope, request failures, slowest path, load effect, findings, next investigation, and outcome. Latency is written in plain language, such as "95% completed within".

Use `--verbose` for the stage plan and detailed measurements.

Use `--json` for machine-readable output. Use `--report-file` to save a Markdown or JSON report; the filename extension selects the format unless `--report-format` is set.

Reports and standard output include safe profile, tenant, Camunda version, stage, finding, and limitation evidence. They exclude endpoints, access tokens, authorization headers, client secrets, variables, payloads, raw response bodies, and unbounded upstream error strings.

## Read-Only Limits

This command performs no mutation, including on failure, timeout, and cancellation paths. It cannot prove write-path health, exporter health, end-to-end process execution health, capacity, or overall cluster health.

Completed diagnostics exit successfully when all planned stages finish with usable evidence, even if the evidence includes abnormal latency, timeouts, backpressure, or other classified request errors. Invalid flags, incomplete execution, and report write failures return the established command error output.
