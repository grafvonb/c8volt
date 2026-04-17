---
title: "c8volt config show"
nav_exclude: true
---

[CLI Reference]({{ "/cli/" | relative_url }})
## c8volt config show

Show effective configuration

### Synopsis

Show the effective configuration with sensitive values sanitized.

Precedence follows one shared contract for all config-backed settings:
flag > env > profile > base config > default.

Use this command to inspect the values a command will actually use after
applying flags, environment variables, profile overlays, base config, and
defaults. Profile values overlay base config field by field and never override
an explicit flag or environment winner.

```
c8volt config show [flags]
```

### Examples

```
./c8volt config show --validate
active_profile: local
apis:
    camunda_api:
        base_url: http://localhost:8080/v2
        key: camunda_api
        require_scope: false
        version: v2
    operate_api:
        base_url: http://localhost:8080
        key: operate_api
        require_scope: false
        version: ""
    tasklist_api:
        base_url: http://localhost:8080
        key: tasklist_api
        require_scope: false
        version: ""
    versioning_disable: false
app:
    backoff:
        initial_delay: 1s
        max_delay: 0s
        max_retries: 0
        multiplier: 2
        strategy: exponential
        timeout: 30m0s
    camunda_version: "8.8"
    tenant: ""
auth:
    cookie:
        base_url: ""
        password: '*****'
        username: ""
    mode: oauth2
    oauth2:
        client_id: c8volt
        client_secret: '*****'
        scopes:
            camunda_api: profile
            operate_api: profile
            tasklist_api: profile
        token_url: http://localhost:18080/auth/realms/camunda-platform/protocol/openid-connect
http:
    timeout: 30s
log:
    format: plain
    level: info
    with_request_body: false
    with_source: false

INFO configuration is valid

# Inspect how flags override env/profile/config for the current command invocation
./c8volt --config ./config.yaml --profile prod --tenant ops-tenant config show

# Validate the effective config after env/profile/config resolution
C8VOLT_AUTH_MODE=oauth2 ./c8volt --config ./config.yaml config show --validate
```

### Options

```
  -h, --help       help for show
      --template   template configuration with values blanked out (copy-paste ready)
      --validate   validate the effective configuration and exit with an error code if invalid
```

### Options inherited from parent commands

```
  -y, --auto-confirm        auto-confirm prompts for non-interactive use
      --config string       path to config file
      --debug               enable debug logging, overwrites and is shorthand for --log-level=debug
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

* [c8volt config](c8volt_config)	 - Manage application configuration

