---
title: "c8volt update"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update

Update existing resources

### Synopsis

Change variables, user-task assignment or completion, and job retries or outcomes.

Use --set-* for values to write. Preview with --dry-run where supported, then
confirm the prepared plan. Choose a resource command for its supported actions.

```
c8volt update [flags]
```

### Examples

```
  c8volt update process-instance --key <process-instance-key> --set-vars-file vars.json --dry-run
  c8volt update job --key <job-key> --set-retries 3 --dry-run
  c8volt update user-task --key <user-task-key> --set-assignee alice --dry-run
```

### Options

```
  -h, --help   help for update
```

### Options inherited from parent commands

```
      --all-tenants          search all accessible tenants; cannot combine with --tenant
  -y, --auto-confirm         auto-confirm prompts for non-interactive use
      --automation           enable non-interactive mode for commands that explicitly support it
      --config string        path to config file
      --debug                enable debug logging
  -j, --json                 output as JSON (where applicable)
      --keys-only            output keys only (where applicable)
      --log-level string     log level (debug, info, warn, error) (default "info")
      --no-indicator         disable transient terminal activity indicators
      --no-version-check     skip the automatic advisory gateway version check
      --profile string       config active profile name to use (e.g. dev, prod)
  -q, --quiet                suppress output except errors
      --request-rate float   maximum shared request starts per second (fractions allowed; 0 disables pacing)
      --tenant string        tenant for discovery and creation; explicit keys use backend authorization
      --timeout duration     HTTP request timeout (default 30s)
  -v, --verbose              show additional output
```

### SEE ALSO

* [c8volt]({{ "/cli/c8volt/" | relative_url }})	 - Operate Camunda 8 workflows from the command line
* [c8volt update element]({{ "/cli/c8volt_update_element" | relative_url }})	 - Update variables at explicit element scopes
* [c8volt update job]({{ "/cli/c8volt_update_job" | relative_url }})	 - Update a job or complete selected jobs
* [c8volt update process-instance]({{ "/cli/c8volt_update_process-instance" | relative_url }})	 - Update process-instance variables or assign a business ID
* [c8volt update user-task]({{ "/cli/c8volt_update_user-task" | relative_url }})	 - Update variables, complete, assign or unassign user tasks by key

