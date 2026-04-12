---
title: "Camunda 8 CLI Use Cases"
description: "Use cases and FAQ for teams searching for a Camunda 8 CLI, BPMN deployment CLI, and process-instance operations tooling."
nav_order: 3
has_toc: true
---

# Camunda 8 CLI Use Cases

`c8volt` is a Camunda 8 CLI for teams that need to operate workflows from the terminal with more confidence than a bare API wrapper usually gives them.

This page is for common search intents such as:

- Camunda 8 CLI
- Camunda CLI for operations
- BPMN deployment CLI
- cancel process instance Camunda 8
- delete process instance Camunda 8
- inspect process tree Camunda 8
- wait for workflow state in CI

## Who c8volt is for

`c8volt` is useful for:

- platform and operations teams managing Camunda 8 environments
- developers deploying BPMN and testing workflow behavior locally
- support engineers inspecting live process-instance trees
- CI jobs that need a command-line workflow verification step
- tenant-aware environments that need explicit per-command or per-profile targeting

## Problems c8volt is built to solve

Many workflow teams can already send requests to Camunda 8 APIs. The harder part is knowing whether the requested action produced the outcome they actually need.

`c8volt` focuses on that gap:

- deploy BPMN and use it immediately
- start a process instance and confirm it becomes active
- walk ancestors, descendants, and full process families before acting
- cancel the correct root instance when direct child cancellation is not enough
- delete process-instance trees thoroughly
- wait for active, completed, canceled, terminated, or absent states in automation
- validate connection, auth, and profile settings before running destructive commands

## Typical workflow operations

### Deploy BPMN from the CLI

```bash
./c8volt deploy pd --file ./order-process.bpmn
./c8volt get pd --bpmn-process-id order-process --latest
```

This matches searches like "deploy BPMN Camunda 8 CLI", "Camunda CLI deploy BPMN", or "Camunda 8 deploy process definition command line".

### Run and verify a Camunda 8 process instance

```bash
./c8volt run pi -b order-process
./c8volt expect pi --key 2251799813685255 --state active
```

This is helpful when people search for "start process instance Camunda 8 CLI", "Camunda CLI run process instance", or "wait for process active Camunda".

### Cancel the right process instance

```bash
./c8volt cancel pi --key 2251799813711977 --force
./c8volt walk pi --key 2251799813711977 --family --tree
```

This supports searches around "cancel child process instance Camunda 8", "Camunda root process cancel", and "inspect process tree Camunda CLI".

### Delete process instances thoroughly

```bash
./c8volt delete pi --key 2251799813711967 --force
./c8volt delete pi --state completed --end-date-older-days 7 --auto-confirm
```

This targets operational cleanup flows behind searches such as "delete process instance Camunda 8 CLI" and "bulk cleanup completed workflows".

## FAQ

### Is c8volt a Camunda 8 CLI?

Yes. `c8volt` is a Camunda 8 CLI focused on BPMN deployment, process-instance control, process-tree inspection, and state verification.

### How is c8volt different from c8ctl?

`c8ctl` is Camunda's newer official Camunda 8 CLI, especially oriented around the broader Camunda lifecycle and newer platform capabilities. `c8volt` has a narrower and more operations-heavy focus: deploy, run, inspect trees, cancel safely, delete thoroughly, wait for state, and work well in scripts where outcome verification matters more than broad API coverage.

### What makes c8volt different from a basic API wrapper?

The main difference is outcome verification. `c8volt` is designed to wait, walk trees, escalate to the correct root instance when needed, and verify final state instead of stopping at "request accepted".

### Does c8volt work well in scripts and CI?

Yes. The CLI supports automation-friendly flags such as `--json`, `--keys-only`, `--auto-confirm`, `--workers`, `--quiet`, and `--no-wait` where appropriate.

## Where to go next

- [Project overview](./index.md)
- [CLI reference](./cli/index.md)
- [GitHub repository](https://github.com/grafvonb/c8volt)
- [Releases](https://github.com/grafvonb/c8volt/releases)
