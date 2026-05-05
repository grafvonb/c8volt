---
title: "c8volt get process-instance"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt get process-instance

List or fetch process instances

### Synopsis

Get process instances by key or by search criteria.

Use direct lookup when you know a process-instance key, or combine search filters to inspect matching process instances by process definition, tenant, state, incidents, variables, jobs, user tasks, and time ranges.

Search results support interactive paging, scriptable JSON aggregation, and count-only workflows. Direct key lookup stays strict: missing keys return not-found.

Use --with-incidents with keyed or list/search output to include direct incident keys and messages under matching process-instance rows. Add --incident-message-limit <chars> to shorten human incident messages; JSON keeps full incident messages.

User-task based lookup resolves owning process instances through tenant-aware Camunda v2 user-task search first. On Camunda 8.8 and 8.9, not-found user-task results fall back to deprecated Tasklist V1 lookup for legacy user-task compatibility; Camunda 8.7 remains unsupported.

Run `c8volt get pi --help` for the complete flag reference.

```
c8volt get process-instance [flags]
```

### Examples

```
  ./c8volt get pi --bpmn-process-id <bpmn-process-id> --state active
  ./c8volt get pi --key <process-instance-key>
  ./c8volt get pi --state active
  ./c8volt get pi --state active --json
  ./c8volt get pi --state active --total
  ./c8volt get pi --has-user-tasks <user-task-key>
  ./c8volt get pi --state active --batch-size 250 --limit 25
  ./c8volt get pi --state active --limit 25 --auto-confirm
  ./c8volt get pi --incidents-only --with-incidents
  ./c8volt get pi --with-incidents --incident-message-limit 80
  ./c8volt get pi --key 2251799813711967 --with-incidents
  ./c8volt get pi --key 2251799813711967 --json
  ./c8volt get pi --key 2251799813711967 --with-incidents --json
  ./c8volt get pi --start-date-after 2026-01-01 --start-date-before 2026-01-31
  ./c8volt get pi --key 2251799813711967 --key 2251799813711977
```

### Options

```
  -n, --batch-size int32             number of process instances to fetch per page (max limit 1000 enforced by server) (default 1000)
  -b, --bpmn-process-id string       BPMN process ID to filter process instances
      --children-only                show only child process instances
      --end-date-after string        only include process instances with end date >= YYYY-MM-DD
      --end-date-before string       only include process instances with end date <= YYYY-MM-DD
      --end-date-newer-days int      only include process instances with end date N days old or newer (0 means today) (default -1)
      --end-date-older-days int      only include process instances with end date N days old or older (default -1)
      --fail-fast                    stop scheduling new instances after the first error
      --has-user-tasks strings       user task key(s) whose owning process instances should be fetched
  -h, --help                         help for process-instance
      --incident-message-limit int   maximum characters to show for human incident messages when --with-incidents is set; 0 disables truncation
      --incidents-only               show only process instances that have incidents
  -k, --key strings                  process instance key(s) to fetch
  -l, --limit int32                  maximum number of matching process instances to return or process across all pages
      --no-incidents-only            show only process instances that have no incidents
      --no-worker-limit              disable limiting the number of workers to GOMAXPROCS when --workers > 1
      --orphan-children-only         show only child instances with missing parents
      --parent-key string            parent process instance key to filter process instances
      --pd-key string                process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)
      --pd-version int32             process definition version
      --pd-version-tag string        process definition version tag
      --roots-only                   show only root process instances
      --start-date-after string      only include process instances with start date >= YYYY-MM-DD
      --start-date-before string     only include process instances with start date <= YYYY-MM-DD
      --start-date-newer-days int    only include process instances N days old or newer (0 means today) (default -1)
      --start-date-older-days int    only include process instances N days old or older (default -1)
  -s, --state string                 state to filter process instances: all, active, completed, canceled, terminated (default "all")
      --total                        return only the numeric total of matching process instances; capped backend totals are counted by paging
      --with-incidents               include direct incident keys and messages for keyed or list/search process-instance output
  -w, --workers int                  maximum concurrent workers when --batch-size > 1 (default: min(batch-size, GOMAXPROCS))
```

### Options inherited from parent commands

```
  -y, --auto-confirm       auto-confirm prompts for non-interactive use
      --automation         enable non-interactive mode for commands that explicitly support it
      --config string      path to config file
      --debug              enable debug logging
  -j, --json               output as JSON (where applicable)
      --keys-only          output keys only (where applicable)
      --log-level string   log level (debug, info, warn, error) (default "info")
      --no-indicator       disable transient terminal activity indicators
      --profile string     config active profile name to use (e.g. dev, prod)
  -q, --quiet              suppress output except errors
      --tenant string      tenant ID for tenant-aware command flows (overrides env, profile, and base config)
      --timeout duration   HTTP request timeout (default 30s)
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt get](c8volt_get)	 - Inspect cluster, process, tenant, and resource state

