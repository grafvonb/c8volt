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

Element variables belong directly to that element; parent scopes are excluded. User-task variables can include inherited values. Variable updates preserve existing owning scopes; element updates always write locally.

## Read listener timing

| Label | Meaning |
| --- | --- |
| `s:` | Job creation time, including waiting before worker execution |
| `e:` | Recorded job end time |
| `d:` | Deadline, shown only for an activated job |
| `dur:` | Time from creation to end, or elapsed time for a pending job |

Unavailable timestamps are omitted. Listener duration includes worker waiting time. Each job or task retains its own observed business ID (**only c8.10+**); a parent ID is not substituted.

For duration-based diagnosis, use the [slow-instance playbook](/ops/analyse-slow-process-instances/). Exact flags are in [get element](/cli/c8volt_get_element) and [walk process-instance](/cli/c8volt_walk_process-instance).
