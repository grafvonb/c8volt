---
title: "c8volt update job"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt update job

Update a job by key

### Synopsis

Update a Camunda job by key.

The command supports retries and timeout updates for Camunda 8.8 and 8.9. It builds a pre-mutation plan, supports --dry-run previews, asks for confirmation before material interactive mutations, and can return after acceptance with --no-wait. Retry updates are confirmed by reading the job by key by default; timeout updates report submitted milliseconds without deadline confirmation. JSON mutations require --dry-run, --auto-confirm, or --automation, and --json cannot be combined with --verbose. Camunda 8.7 returns an unsupported-version error before mutation.

```
c8volt update job [flags]
```

### Examples

```
  ./c8volt update job --key 2251799813711967 --retries 3
  ./c8volt update job --key 2251799813711967 --timeout 5m
  ./c8volt update job --key 2251799813711967 --retries 3 --timeout 5m
  ./c8volt update job --key 2251799813711967 --retries 3 --dry-run
  ./c8volt update job --key 2251799813711967 --retries 3 --auto-confirm
  ./c8volt update job --key 2251799813711967 --retries 3 --no-wait
  ./c8volt --json update job --key 2251799813711967 --retries 3 --dry-run
  ./c8volt --json update job --key 2251799813711967 --retries 3 --auto-confirm
```

### Options

```
      --dry-run          preview job updates without submitting mutation
  -h, --help             help for job
      --key string       job key to update
      --no-wait          return after the update request is accepted without retry confirmation
      --retries int32    retry count to set on the job
      --timeout string   timeout duration to submit for the job, for example 60s, 5m, or 1h
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
  -v, --verbose            show additional output
```

### SEE ALSO

* [c8volt update](c8volt_update)	 - Update existing resources

