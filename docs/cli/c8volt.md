---
title: "c8volt"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt

c8volt: Camunda 8 Operations CLI

### Synopsis

c8volt: Camunda 8 Operations CLI.

Built for Camunda 8 operators and developers who need confirmation, not guesses.
c8volt focuses on operational workflows such as deploying BPMN models, starting process instances,
waiting for state transitions, walking process trees, cancelling safely, and deleting thoroughly.

Tenant-aware process-instance flows use one effective tenant context per command execution.
Supported wrong-tenant lookups resolve as not found. Current process-instance runtime support
is implemented for Camunda 8.7, 8.8, and 8.9 through the repository's versioned service
factories and facades, with the same repository command-family coverage on 8.9 that already
exists on 8.8.

Refer to the documentation at https://c8volt.info for more information.

```
c8volt [flags]
```

### Options

```
  -y, --auto-confirm        auto-confirm prompts for non-interactive use
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
  -h, --help                help for c8volt
  -j, --json                output as JSON (where applicable)
      --keys-only           output as keys only (where applicable), can be used for piping to other commands
      --log-format string   log format (json, plain, text) (default "plain")
      --log-level string    log level (debug, info, warn, error) (default "info")
      --log-with-source     include source file and line number in logs
      --no-err-codes        suppress error codes in error outputs
      --profile string      config active profile name to use (e.g. dev, prod)
  -q, --quiet               suppress all output, except errors, overrides --log-level
      --tenant string       tenant ID for tenant-aware command flows (overrides env, profile, and base config)
  -v, --verbose             adds additional verbosity to the output, e.g. for progress indication
```

### SEE ALSO

* [c8volt cancel](c8volt_cancel)	 - Cancel resources
* [c8volt config](c8volt_config)	 - Manage application configuration
* [c8volt delete](c8volt_delete)	 - Delete resources
* [c8volt deploy](c8volt_deploy)	 - Deploy resources
* [c8volt embed](c8volt_embed)	 - Manage embedded resources
* [c8volt expect](c8volt_expect)	 - Expect resources to be in a certain state
* [c8volt get](c8volt_get)	 - Get resources
* [c8volt run](c8volt_run)	 - Run resources
* [c8volt version](c8volt_version)	 - Print version information
* [c8volt walk](c8volt_walk)	 - Traverse (walk) the parent/child graph of resource type

