---
title: "Runtime Elements"
permalink: /cli/runtime-elements/
parent: "CLI Reference"
nav_order: 2
nav_exclude: false
has_toc: true
---

# Runtime elements and listeners

Inspect runtime elements when a process is slow or blocked and its process-instance row does not explain why. Examples target Camunda 8.10.

## Inspect an instance

```bash
c8volt get process-instance --key <process-instance-key> --with-elements --with-listeners
c8volt get element --pi-key <process-instance-key> --state active --limit 10
c8volt walk process-instance --key <process-instance-key> --with-elements
```

`--with-listeners` requires `--with-elements` on process-instance and walk commands. `get element` can include listeners directly. Search predicates combine with AND.

## Read variables

```bash
c8volt get element --key <element-instance-key> --with-vars
c8volt get user-task --key <user-task-key> --with-vars
```

Element reads return variables owned directly by that element; parent scopes are excluded. User-task reads return effective variables, including inherited values.

## Update variables

```bash
c8volt update element --key <element-instance-key> --set-vars-file vars.json --dry-run
c8volt update user-task --key <user-task-key> --set-vars-file vars.json --dry-run
```

Element updates write locally. User-task updates preserve existing owning scopes and create new names at the task's element scope. Process-instance updates write to each selected instance's root scope. Previews show the destinations; normal execution confirms requested values there. Use `--no-wait` to skip read-back where supported.

## Read listener timing

| Label | Meaning |
| --- | --- |
| `s:` | Job creation time, including waiting before worker execution |
| `e:` | Recorded job end time |
| `d:` | Deadline, shown only for an activated job |
| `dur:` | Time from creation to end, or elapsed time for a pending job |

Unavailable timestamps are omitted. Listener duration includes worker waiting time. Each job or task retains its own observed business ID (c8.10+); a parent ID is not substituted.

For duration-based diagnosis, use the [slow-instance playbook](/ops/analyse-slow-process-instances/).

[get element](/cli/c8volt_get_element) · [update element](/cli/c8volt_update_element) · [update user-task](/cli/c8volt_update_user-task) · [walk process-instance](/cli/c8volt_walk_process-instance)
