<img src="./docs/logo/c8volt_orange_black_bkg_white_400x152.png" alt="c8volt logo" style="border-radius: 5px;" />

# c8volt - Yet Another Camunda 8 CLI Tool?

No, **c8volt** is different. Its design and development focus on operational effectiveness, ensuring that done is done.
There are plenty of operational tasks where you want to be sure that:

* A process instance was started – is it really active and running?
* A process instance was canceled – is it really in the cancelled state?
* A process instance tree was deleted – are all instances truly gone?
* A process variable was set – does it hold the correct value?

If some operation requires additional steps to reach the desired state, **c8volt** takes care of it for you by:
* Cancelling the root process instance when you want to cancel a child process instance.
* Deleting process instances by first cancelling them and then deleting them.
* Waiting until the process instance reaches the desired state (e.g., `CANCELED`)
* Traversing the process instance tree to perform operations like cancellation or deletion.

**c8volt** focuses on real operational use cases while still providing the familiar CLI functionality such as standard CRUD commands on various resources.
 
## Quick Start with c8volt

1. **Install Camunda 8.8 Run**  

Download [Camunda 8 Run](https://downloads.camunda.cloud/release/camunda/c8run), unpack and start it with `./start.sh`.
Use listed relevant URLs and default credentials to create the configuration for c8volt.
```bash
$ ./start.sh

System Version Information
--------------------------
Camunda Details:
  Version: 8.8.2
--------------------------
[...]
-------------------------------------------
Access each component at the following urls with these default credentials:
- username: demo
- password: demo

Operate:                    http://localhost:8080/operate
Tasklist:                   http://localhost:8080/tasklist
Identity:                   http://localhost:8080/identity

Orchestration Cluster API:  http://localhost:8080/v2/
[...]
```
2. **Install c8volt**

Download the latest release relevant to your OS from the [c8volt Releases](https://github.com/grafvonb/c8volt/releases) page and unpack it.
Here is an example for macOS ARM64:
```bash
$ wget -q --show-progress -c -O c8volt.tar.gz https://github.com/grafvonb/c8volt/releases/download/v0.1.61/c8volt_0.1.61_Darwin_arm64.tar.gz
$ tar -xvf c8volt.tar.gz
```
Check the version:
```bash
$ ./c8volt version
c8volt version 0.1.62, commit 5c38662a89a82ee82809752857a340129c20995e, built at 2025-11-01T16:46:41Z. Supported Camunda versions: 8.7, 8.8
```
3. **Configure c8volt**

Create a configuration file (YAML) in the folder where you unpacked c8volt with the name `config.yaml` or in `$HOME/.c8volt/config.yaml` with the minimal connection details.
c8run v8.8 uses no authentication by default for local development, so the minimal config looks like this:
```yaml
apis:
  version: "88"
  camunda_api:
    base_url: "http://localhost:8080/v2"
auth:
  mode: none
log:
  level: debug
```
4. **Run c8volt**

Test the connection and list cluster topology:
```bash
./c8volt get cluster-topology
```
or use explicit path to config file:
```bash
./c8volt get cluster-topology --config ./config-minimal.yaml
```
You should see output like this:
```bash
{
  "Brokers": [
    {
      "Host": "localhost",
      "NodeId": 0,
      "Partitions": [
        {
          "Health": "healthy",
          "PartitionId": 1,
          "Role": "leader"
        }
      ],
      "Port": 26501,
      "Version": "8.8.2"
    }
  ],
  "ClusterSize": 1,
  "GatewayVersion": "8.8.2",
  "PartitionsCount": 1,
  "ReplicationFactor": 1,
  "LastCompletedChangeId": ""
}
```
## Highlights

The comprehensive documentation is on its way, but here are some highlights of what **c8volt** can do for you.

### Embedded Deployment of BPMN Models

Quite often, we need to deploy some BPMN models to Camunda 8 for testing or operational tasks.
**c8volt** has embedded models in its binary that can be deployed on demand, without to use Camunda Modeler or other tools.

#### Embedded Deployment in Action

List embedded BPMN models:
```bash
$ ./c8volt embed list
processdefinitions/C87_DoubleUserTaskProcess.bpmn
processdefinitions/C87_MultipleSubProcessesParentProcess.bpmn
processdefinitions/C87_SimpleParentProcess.bpmn
processdefinitions/C87_SimpleUserTaskProcess.bpmn
processdefinitions/C87_SimpleUserTaskWithIncidentProcess.bpmn
processdefinitions/C88_DoubleUserTaskProcess.bpmn
processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn
processdefinitions/C88_SimpleParentProcess.bpmn
processdefinitions/C88_SimpleUserTaskProcess.bpmn
processdefinitions/C88_SimpleUserTaskWithIncidentProcess.bpmn
```
Deploy three embedded BPMN models to Camunda 8 that build a parent-child process instance tree:
```bash
$ ./c8volt embed deploy -f processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn,processdefinitions/C88_SimpleParentProcess.bpmn,processdefinitions/C88_SimpleUserTaskProcess.bpmn
found: 3
2251799813685772 <default> C88_MultipleSubProcessesParentProcess v1 vprocessdefinitions/C88_MultipleSubProcessesParentProcess.bpmn (2251799813685771)
2251799813685773 <default> C88_SimpleParentProcess v1 vprocessdefinitions/C88_SimpleParentProcess.bpmn (2251799813685771)
2251799813685774 <default> C88_SimpleUserTask_Process v1 vprocessdefinitions/C88_SimpleUserTaskProcess.bpmn (2251799813685771)
```
Check that the models are deployed:
```bash
$ ./c8volt get pd
found: 3
2251799813685772 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0
2251799813685773 <default> C88_SimpleParentProcess v1/v1.0.0
2251799813685774 <default> C88_SimpleUserTask_Process v1/v1.0.0
```
And run a process instance using standard run command:
```bash
INFO waiting for process instance with key 2251799813909860 to be started by workflow engine...
INFO process instance 2251799813909860 succesfully created (start registered at 2025-11-04T09:06:36.161Z and confirmed at 2025-11-04T09:06:39Z) using process definition id 2251799813685772, C88_MultipleSubProcessesParentProcess, v1, tenant: <default>
```
If you want to modify the embedded BPMN models, you can export them to a local folder with:
```bash
$ ./c8volt embed export -f processdefinitions/C88_SimpleParentProcess.bpmn -o ./exported_models
INFO exported 1 embedded resource(s) to "exported_models"
```
Modify it with Camunda Modeler or any text editor and deploy it back with "normal" deployment command:
```bash
$ ./c8volt deploy pd -f ./exported_models/processdefinitions/C88_SimpleParentProcess.bpmn
found: 1
2251799813906559 <default> C88_SimpleParentProcess v2 vC88_SimpleParentProcess.bpmn (2251799813906558)
```
And run a process instance of the modified model:
```bash
$ ./c8volt run pi -b C88_SimpleParentProcess
INFO waiting for process instance with key 2251799813909382 to be started by workflow engine...
INFO process instance 2251799813909382 succesfully created (start registered at 2025-11-04T09:03:27.05Z and confirmed at 2025-11-04T09:03:30Z) using process definition id 2251799813906559, C88_SimpleParentProcess, v2, tenant: <default>
```

### Running Process Instances

TBD

### Searching Process Instances

TBD

### Cancellation of Process Instances

Standard cancellation of process instances in Camunda 8 is asynchronous and does not guarantee that the instance is actually cancelled when the API call returns successfully.
**c8volt** provides a waiting mechanism that polls the process instance state until it reaches the desired `CANCELED` state or a timeout occurs.
It allows also forced cancellation a process instance even it is a child, by traversing the parent chain and cancelling its root ancestor. 

#### Cancellation in Action

Assuming you have following process instance tree (look at the sections above how to achieve that with c8volt's embedded deployment and run features):
```bash
$ ./c8volt get pi
found: 4
2251799813686374 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 ACTIVE s:2025-11-02T12:47:25.352+0000  p:<root> i:false
2251799813686383 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-02T12:47:25.352+0000  p:2251799813686374 i:false
2251799813686384 <default> C88_SimpleParentProcess v1/v1.0.0 ACTIVE s:2025-11-02T12:47:25.352+0000  p:2251799813686374 i:false
2251799813686392 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-02T12:47:25.352+0000  p:2251799813686384 i:false
```
Cancel the process instance `2251799813686384` (which is a child of `2251799813686374` and parent of `2251799813686392`):
```bash
$ ./c8volt cancel pi --key 2251799813686384
INFO cannot cancel, process instance with key 2251799813686384 is a child process of a root parent with key 2251799813686374
INFO you can use the --force flag to cancel the root process instance with key 2251799813686374 and all its child processes
```
You cannot cancel a child process instance directly. It is as standard behavior of Camunda 8.
**c8volt** has a flag `--force` to find the root ancestor process instance and cancel it instead, which also cancels all its child process instances:
```bash
$ ./c8volt cancel pi --key 2251799813686384 --force
INFO cannot cancel, process instance with key 2251799813686384 is a child process of a root parent with key 2251799813686374
INFO force flag is set, cancelling root process instance with key 2251799813686374 and all its child processes
INFO waiting for process instance with key 2251799813686374 to be cancelled by workflow engine...
INFO process instance 2251799813686374 currently in state ACTIVE; waiting...
INFO process instance 2251799813686374 currently in state ACTIVE; waiting...
INFO process instance 2251799813686374 currently in state ACTIVE; waiting...
INFO process instance with key 2251799813686374 was successfully cancelled
```
Standard behavior of **c8volt** is to wait until the process instance reaches the `CANCELED` state. 
You can check it with a special `walk` command that traverses the process instance tree. To get the whole tree use `--mode family`:
```bash
$ ./c8volt walk pi --key 2251799813686384 --mode family
2251799813686374 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 TERMINATED s:2025-11-02T12:47:25.352Z  e:2025-11-02T12:53:43.594Z p:<root> i:false ⇄ 
2251799813686383 <default> C88_SimpleUserTask_Process v1/v1.0.0 CANCELED s:2025-11-02T12:47:25.352+0000  e:2025-11-02T12:53:43.594+0000 p:2251799813686374 i:false ⇄ 
2251799813686384 <default> C88_SimpleParentProcess v1/v1.0.0 CANCELED s:2025-11-02T12:47:25.352+0000  e:2025-11-02T12:53:43.594+0000 p:2251799813686374 i:false ⇄ 
2251799813686392 <default> C88_SimpleUserTask_Process v1/v1.0.0 CANCELED s:2025-11-02T12:47:25.352+0000  e:2025-11-02T12:53:43.594+0000 p:2251799813686384 i:false
```

### Deletion

#### Deletion of Process Instances

TBD

#### Deletion of Process Definitions

TBD

### Walking Process Instance Trees

TBD

### Scripting Support

#### Error Codes

In case of errors, c8volt returns specific exit codes that can be used in scripts to handle different error scenarios.
If you do not want to deal with specific error codes, you can use the `--no-err-codes` flag to make c8volt always return exit code 0.

#### Command Pipelining

TBD

## Configuration

### Supported Camunda 8 Versions

**c8volt** supports the following Camunda 8 versions:

- 8.7.x
- 8.8.x

As the changes between Camunda 8.8+ and previous versions are significant, you need to specify the version you are using in the configuration file.
Here is an example configuration snippet for Camunda 8.8:
```yaml
app:
  camunda_version: "8.8"
```

### API Base URLs

**c8volt** needs to connect to Camunda 8 APIs. You need to provide the base URLs for the following APIs:
- Camunda 8 Orchestration Cluster API (formerly known as Zeebe API or Camunda API)
- Camunda 8 Operate API
- Camunda 8 Tasklist API

The base URL to Camunda 8 Orchestration Cluster API is usually in the form of `https://<host>:<port>/v2`, 
while the Operate and Tasklist APIs are usually in the form of `https://<host>:<port>`.
If you provide only the Camunda 8 Orchestration Cluster API base URL, **c8volt** will assume that Operate and Tasklist APIs 
are available under the same host and port without the `/v2` suffix.
Here is an example configuration snippet for API base URLs for Camunda 8 running locally:

```yaml
apis:
  camunda_api:
    base_url: "http://localhost:8080/v2"
  operate_api:
    base_url: "http://localhost:8080"
  tasklist_api:
    base_url: "http://localhost:8080"
```

### Choose authentication method

**c8volt** supports following authentication methods for connecting to Camunda 8 APIs:
- OAuth2 (OIDC)
- API Cookie (for local development with Camunda 8.7 Run only)
- No Authentication (for local development with Camunda 8.8+ Run only)

#### Authentication with OAuth2 (OIDC)

**c8volt** supports OAuth2 (OIDC) authentication with client credentials flow.
You need to provide the following settings:
* Token URL
* Client ID
* Client Secret
* Scopes (optional, depending on your identity provider and API setup)

Here is an example configuration snippet for OAuth2 authentication for Camunda 8 running locally with Keycloak:
```yaml
auth:
  mode: "oauth2"
  oauth2:
    token_url: "http://localhost:18080/auth/realms/camunda-platform/protocol/openid-connect"
    client_id: "c8volt"
    client_secret: "*******" # for local tests only, use environment variable C8VOLT_AUTH_OAUTH2_CLIENT_SECRET instead
    scopes:
      camunda_api: "profile"
      operate_api: "profile"
      tasklist_api: "profile"
```

#### Authentication with API Cookie (development with Camunda 8 Run only)

This method is only suitable for local development with Camunda 8 Run (up to version 8.7), 
as it uses the API cookie set by the web interface.

You need to provide the following settings:
* API base URL
* (optional) Username and Password (if not set, defaults to `demo`/`demo`)

Here is an example configuration snippet for API Cookie authentication for Camunda 8 Run running locally:
```yaml
app:
  camunda_version: "8.7"
apis:
  camunda_api:
    base_url: "http://localhost:8080/v2"
auth:
  mode: cookie
  cookie:
    base_url: "http://localhost:8080"
    username: "demo"
    password: "demo"
log:
  level: debug
```

#### No Authentication (development with Camunda 8.8+ Run only)

This method is only suitable for local development with Camunda 8.8+ Run, as it disables authentication entirely.
```yaml
app:
  camunda_version: "8.8"
apis:
  camunda_api:
    base_url: "http://localhost:8080/v2"
auth:
  mode: none
log:
  level: debug
```

### Ways to provide settings

c8volt uses [Viper](https://github.com/spf13/viper) under the hood. Configuration values can come from:

- Flags (`--auth-oauth2-client-id=...`)
- Environment variables (`C8VOLT_AUTH_OAUTH2_CLIENT_ID`)
- Config file (YAML)
- Defaults (hardcoded fallbacks)

### Precedence

When multiple sources define the same setting, the **highest-priority value wins**:

| Priority    | Source             | Example                               |
|-------------|--------------------|---------------------------------------|
| 1 (highest) | Command-line flags | `--auth-oauth2-client-id=c8volt`      |
| 2           | Environment vars   | `C8VOLT_AUTH_OAUTH2_CLIENT_ID=c8volt` |
| 3           | Config file (YAML) | `auth.oauth2.client_id: c8volt`       |
| 4 (lowest)  | Defaults           | `http.timeout: "30s"` (built-in)      |

### Default configuration file locations

When searching for a config file, c8volt checks these paths in order and uses the first one it finds:

| Priority | Location                                        | Notes                                                                                                                              |
|----------|-------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------|
| 1        | `./config.yaml`                                 | Current working directory                                                                                                          |
| 2        | `$XDG_CONFIG_HOME/c8volt/config.yaml`         | Skipped if `$XDG_CONFIG_HOME` is not set                                                                                           |
| 3        | `$HOME/.config/c8volt/config.yaml`            | XDG default on Linux/macOS                                                                                                         |
| 4        | `$HOME/.c8volt/config.yaml`                   | Legacy fallback                                                                                                                    |
| 5        | `%AppData%\c8volt\config.yaml` (Windows only) | `%AppData%` usually expands to `C:\Users\<User>\AppData\Roaming`<br>Example: `C:\Users\Alice\AppData\Roaming\c8volt\config.yaml` |

### File format

Config files must be **YAML**. You can inspect the effective configuration (after merging defaults, config file, env vars, and flags) with:
```bash
$ ./c8volt config show
```
You will see output like this:
``` yaml
app:
    tenant: ""
    camunda_version: "88"
apis:
    camunda_api:
        base_url: http://localhost:8080/v2
        key: camunda_api
        require_scope: true
    operate_api:
        base_url: http://localhost:8080
        key: operate_api
        require_scope: true
    tasklist_api:
        base_url: http://localhost:8080
        key: tasklist_api
        require_scope: true
auth:
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
    timeout: 23s
log:
    level: debug
```
You can output the config as empty (no values) template with:
```bash
$ ./c8volt config show --template
```
In case of any issues with loading or parsing the config file, use validation with:
```bash
$ ./c8volt config show --validate
```

### Environment variables

Each config key can also be set via environment variable.\
The prefix is `C8VOLT_`, and nested keys are joined with `_`. For
example:

-   `C8VOLT_AUTH_OAUTH2_CLIENT_ID`
-   `C8VOLT_AUTH_OAUTH2_CLIENT_SECRET`
-   `C8VOLT_HTTP_OAUTH2_TOKEN_URL`

### Security note

Sensitive fields such as `auth.oauth2.client_secret` are **always masked** when
the configuration is printed or logged. The raw values are still loaded and used internally, but they will never appear in output.

## Archive Section 
(for future documentation, might not work anymore in the current version, will be removed soon)

- **Delete active process instances by cancelling them first**
  ```bash
  ./c8volt delete pi --key <process-instance-key> --cancel
  ```

- **List process instances that are children (sub-processes) of other process instances**
  ```bash
  ./c8volt get pi --bpmn-process-id=<bpmn-process-id> --children-only
  ```

- **List process instances that are parents of other process instances**
  ```bash
  ./c8volt get pi --bpmn-process-id=<bpmn-process-id> --parents-only
  ```

- **List process instances that are children of orphan parent process instances**  
  (i.e., their parent process instance no longer exists)
  ```bash
  ./c8volt get pi --bpmn-process-id=<bpmn-process-id> --orphan-parents-only
  ```

- **List process instances for a specific process definition (model) and its first version**
  ```bash
  ./c8volt get pi --bpmn-process-id=<bpmn-process-id> --process-version=1
  ```

- **List process instances with incidents**
  ```bash
  ./c8volt get pi --incidents-only
  ```

- **List process instances without incidents**
  ```bash
  ./c8volt get pi --no-incidents-only
  ```

- **Recursive search (walk) process instances with parent–child relationships**
    - List all child process instances of a given process instance
      ```bash
      ./c8volt walk pi --mode children --start-key <process-instance-key>
      ```
    - List path from a given process instance to its root ancestor (top-level parent)
      ```bash
      ./c8volt walk pi --mode parent --start-key <process-instance-key>
      ```
    - List the entire family (parent, grandparent, …) of a given process instance (traverse up and down the tree)
      ```bash
      ./c8volt walk pi --mode family --start-key <process-instance-key>
      ```

- **List process instances in one line per instance (suitable for scripting)**  
  Works with all `get` commands.
  ```bash
  ./c8volt get pi --one-line
  ```

- **List process instances just by their keys (suitable for scripting)**  
  Works with all `get` commands.
  ```bash
  ./c8volt get pi --keys-only
  ```

- …and more to come:
- bulk operations (e.g., delete multiple process instances by filter)
- multiple Camunda 8 API versions support (currently 8.7, 8.8, 8.9 planned)
- or submit a proposal or contribute code on [GitHub](https://github.com/grafvonb/c8volt)

## Disclaimer

Use **c8volt** at your own risk. It can modify system state.

- Always create a verified backup before write operations, especially in production.
- Test commands in a non-production environment first.
- Batch operations and parallel runs can reduce system performance and availability.
- Review commands and flags before execution; dry-run where available.
- Ensure you have proper authorization. Changes may be irreversible.
- Network or API issues can leave partial state changes; validate results.
- Keep credentials secure and rotate them regularly.

**c8volt** is provided "AS IS", without warranties or conditions of any kind, as stated in the Apache License 2.0.

## Copyright

Copyright © 2025 [Adam Bogdan Boczek](https://boczek.info)

This project is licensed under the Apache License, Version 2.0.
See the LICENSE file in this repository for the full text.
