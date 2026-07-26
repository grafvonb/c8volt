# Contract: Command Coverage

## Inventory Source

The suite must flatten the command tree from:

```sh
c8volt capabilities --json
```

The current inventory contains 55 command nodes:

```text
cancel
cancel process-instance
capabilities
config
config show
config template
config test-connection
config validate
delete
delete process-definition
delete process-instance
deploy
deploy process-definition
embed
embed deploy
embed export
embed list
expect
expect process-instance
get
get cluster
get cluster license
get cluster topology
get cluster version
get element
get incident
get job
get process-definition
get process-instance
get resource
get tenant
ops
ops analyse
ops analyse slow-process-instances
ops execute
ops execute retention-policy
ops execute smoke-test
ops purge
ops purge all-process-definitions
ops purge orphan-process-instances
ops purge process-instances-with-incidents
ops repair
ops repair incident
ops repair process-instance
resolve
resolve incident
resolve process-instance
run
run process-instance
update
update job
update process-instance
version
walk
walk process-instance
```

## Required Coverage Groups

| Group | Command paths |
| --- | --- |
| `cancel` | `cancel`, `cancel process-instance` |
| `capabilities` | `capabilities` |
| `config` | `config`, `config show`, `config template`, `config test-connection`, `config validate` |
| `delete` | `delete`, `delete process-definition`, `delete process-instance` |
| `deploy` | `deploy`, `deploy process-definition` |
| `embed` | `embed`, `embed deploy`, `embed export`, `embed list` |
| `expect` | `expect`, `expect process-instance` |
| `get` | `get`, `get cluster`, `get cluster license`, `get cluster topology`, `get cluster version`, `get element`, `get incident`, `get job`, `get process-definition`, `get process-instance`, `get resource`, `get tenant` |
| `ops analyse` | `ops analyse`, `ops analyse slow-process-instances` |
| `ops execute` | `ops execute`, `ops execute retention-policy`, `ops execute smoke-test` |
| `ops purge` | `ops purge`, `ops purge all-process-definitions`, `ops purge orphan-process-instances`, `ops purge process-instances-with-incidents` |
| `ops repair` | `ops repair`, `ops repair incident`, `ops repair process-instance` |
| `resolve` | `resolve`, `resolve incident`, `resolve process-instance` |
| `run` | `run`, `run process-instance` |
| `update` | `update`, `update job`, `update process-instance` |
| `version` | `version` |
| `walk` | `walk`, `walk process-instance` |

## Coverage Entry Requirements

Each command path must map to one coverage entry with:

- command path
- group
- scenario owner file
- aliases to exercise
- flags to exercise
- output modes to exercise
- version expectations
- destructive classification
- setup data requirements
- cleanup/reporting expectation

Leaf command entries must cover every command-local flag returned by capabilities. Grouping command entries must cover help/discovery and no-argument behavior.

## Inventory Validation

The suite must fail when:

- live inventory count differs from expected count and coverage entries have not been updated
- a live command path has no coverage entry
- a coverage entry references a command path absent from the live inventory
- a leaf command exposes a flag that is not included in the coverage entry
- a coverage entry claims an output mode that the command no longer supports

The failure output must list missing paths, stale paths, and missing flags by command path.
