<img src="./docs/logo/c8volt_orange_black_bkg_white_400x152.png" alt="c8volt logo" style="border-radius: 5px;" />

# c8volt - Yet Another Camunda 8 CLI Tool?

No, **c8volt** is different. Its design and development focus on operational effectiveness, ensuring that done is done.
There are plenty of operational tasks where you want to be sure that:

* A process was canceled – is it really in the cancelled state?
* A process tree was deleted – are all instances truly gone?
* A process instance was started – has it already reached the expected user task?
* A process variable was set – does it hold the correct value?

**c8volt** focuses on real operational use cases while still providing the familiar CLI functionality such as standard CRUD commands on various resources.
 
## Quick Start with c8volt

1. **Install Camunda 8.8 Run**  
   Download [Camunda 8 Run](https://downloads.camunda.cloud/release/camunda/c8run/8.8/), unpack and start it with `./start.sh`.
2. **Install c8volt**  
   Download the latest release from the [c8volt Releases](https://github.com/grafvonb/c8volt/releases) page and unpack it.
3. **Configure c8volt**  
   Create a configuration file (YAML) in the folder where you unpacked c8volt with the name `config.yaml` or in `$HOME/.c8volt/config.yaml` with the minimal connection and authentication details:
    ```yaml
    auth:
      mode: "cookie"
      cookie:
        base_url: "http://localhost:8080"
    
    apis:
      version: "8.8"
      camunda_api:
        base_url: "http://localhost:8080/v2"
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
    ```json
    {
      "Brokers": [
        {
          "Host": "192.168.178.88",
          "NodeId": 0,
          "Partitions": [
            {
              "Health": "healthy",
              "PartitionId": 1,
              "Role": "leader"
            }
          ],
          "Port": 26501,
          "Version": "8.8.0"
        }
      ],
      "ClusterSize": 1,
      "GatewayVersion": "8.8.0",
      "PartitionsCount": 1,
      "ReplicationFactor": 1,
      "LastCompletedChangeId": ""
    }  
    ```

## Highlights

c8volt simplifies various tasks related to Camunda 8, including these special use cases:

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

## Supported Camunda 8 APIs

- 8.7.x
- 8.8.x

## Configuration

### Choose authentication method

c8volt supports two authentication methods for connecting to Camunda 8 APIs:
- OAuth2 (OIDC)
- API Cookie (development with Camunda 8 Run only)

#### Authentication with OAuth2 (OIDC)

c8volt supports OAuth2 (OIDC) authentication with client credentials flow.
You need to provide the following settings:
* Token URL
* Client ID
* Client Secret
* Scopes (optional, depending on your identity provider and API setup)

Here is an example configuration snippet for OAuth2 authentication for Camunda 8 running locally with Keycloak:
```yaml
auth:
  mode: "oauth2" # options: "oauth2", "cookie"
  oauth2:
    token_url: "http://localhost:18080/auth/realms/camunda-platform/protocol/openid-connect"
    client_id: "c8volt"
    client_secret: "*******" # use environment variable c8volt_AUTH_CLIENT_SECRET if possible
    scopes:
      camunda_api: "profile"
      operate_api: "profile"
      tasklist_api: "profile"
```

#### Authentication with API Cookie (development with Camunda 8 Run only)

This method is only suitable for local development with Camunda 8 Run, as it uses the API cookie set by the web interface.
You need to provide the following settings:
* API base URL
* (optional) Username and Password (if not set, defaults to `demo`/`demo`)

Here is an example configuration snippet for API Cookie authentication for Camunda 8 Run running locally:
```yaml
auth:
  mode: "cookie" # options: "oauth2", "cookie"
  cookie:
    base_url: "http://localhost:8090"
    username: "demo"
    password: "demo"
```

### Connecting to Camunda's 8 APIs

To run c8volt, you need to configure the connection to your Camunda 8 APIs (Camunda, Operate, Tasklist) and authentication details.
With the introduction of Camunda v8.8 and unified APIs, some of these settings may become optional or redundant in the future.

c8volt expects the following API configurations:
* [Camunda 8 API](https://docs.camunda.io/docs/8.7/apis-tools/camunda-api-rest/camunda-api-rest-overview/) (required, formally known as Zeebe API), since v8.8 known as [Orchestration Cluster API](https://docs.camunda.io/docs/apis-tools/orchestration-cluster-api-rest/orchestration-cluster-api-rest-overview/)
* [Operate API](https://docs.camunda.io/docs/8.7/apis-tools/operate-api/overview/) (optional, required for some commands, if not set defaults to Camunda 8 API), since v8.8 deprecated in favor of unified Orchestration Cluster API
* [Tasklist API](https://docs.camunda.io/docs/8.7/apis-tools/tasklist-api-rest/tasklist-api-rest-overview/) (optional, required for some commands, if not set defaults to Camunda 8 API), since v8.8 deprecated in favor of unified Orchestration Cluster API

#### If you use Camunda 8 Run with API Cookie Authentication (Development only)

After starting Camunda 8 Run, you can find the API endpoint in the terminal output (default is `http://localhost:8080/v2`). 
(current, October 2025, gotcha: if you use `--port` flag, the terminal output still shows port 8080, but it actually runs on the port you specified).

Provide the following settings in the config file, environment variables, or flags:
```yaml
apis:
  version: "87"
  camunda_api:
    base_url: "http://localhost:8080/v2"
  operate_api:
    base_url: "http://localhost:8080"
  tasklist_api:
    base_url: "http://localhost:8080"
```

#### If you use Camunda 8 with OAuth2 (oidc) authentication

### Ways to provide settings

c8volt uses [Viper](https://github.com/spf13/viper) under the hood.
Configuration values can come from:

-   **Flags** (`--auth-client-id=...`)
-   **Environment variables** (`C8VOLT_AUTH_CLIENT_ID=...`)
-   **Config file** (YAML)
-   **Defaults** (hardcoded fallbacks)

### Precedence

When multiple sources define the same setting, the **highest-priority value wins**:

| Priority    | Source             | Example                          |
|-------------|--------------------|----------------------------------|
| 1 (highest) | Command-line flags | `--auth-client-id=cli-id`        |
| 2           | Environment vars   | `C8VOLT_AUTH_CLIENT_ID=env-id`   |
| 3           | Config file (YAML) | `auth.client_id: file-id`        |
| 4 (lowest)  | Defaults           | `http.timeout: "30s"` (built-in) |

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

Config files must be **YAML**. Example with core settings:

``` yaml
app:
  # Tenant identifier (optional, depending on your setup)
  tenant: ""

auth:
  # OAuth token endpoint
  token_url: "http://localhost:18080/auth/realms/camunda-platform/protocol/openid-connect"

  # Client credentials (use env vars if possible)
  client_id: "c8volt"
  client_secret: "" # do not store secrets in config files if possible, use environment variable C8VOLT_AUTH_CLIENT_SECRET instead

  # Scopes as key:value pairs (names -> scope strings)
  # Do not define if not in use or empty
  scopes:
    camunda_api: "profile"
    operate_api: "profile"
    tasklist_api: "profile"

http:
  # Go duration string (e.g., 10s, 1m, 2m30s)
  timeout: "30s"

apis:
  # Base URL for Camunda API (v8.8+: Orchestration Cluster API)
  camunda_api:
    base_url: "http://localhost:8080/v2"
  # Base URL for Operate API (leaves empty to default to camunda_api, deprecated in v8.8+)
  operate_api:
    base_url: "http://localhost:8081/v1"
  # Base URL for Tasklist API (leaves empty to default to camunda_api, deprecated in v8.8+)
  tasklist_api:
    base_url: "http://localhost:8082/v1"
```

### Environment variables

Each config key can also be set via environment variable.\
The prefix is `c8volt_`, and nested keys are joined with `_`. For
example:

-   `c8volt_AUTH_CLIENT_ID`
-   `c8volt_AUTH_CLIENT_SECRET`
-   `c8volt_HTTP_TIMEOUT`

### Security note

Sensitive fields such as `auth.client_secret` are **always masked** when
the configuration is printed (e.g. with `--show-config`) or logged.\
The raw values are still loaded and used internally, but they will never
appear in output.

### Example: Show effective configuration

You can inspect the effective configuration (after merging defaults,
config file, env vars, and flags) with:

```bash
$ ./c8volt --show-config
config loaded: /Users/adam.boczek/Development/Workspace/Boczek/Projects/c8volt/c8volt/config.yaml
{
  "Config": "",
  "App": {
    "Tenant": "",
    "Backoff": {
      "Strategy": "exponential",
      "InitialDelay": 500000000,
      "MaxDelay": 8000000000,
      "MaxRetries": 0,
      "Multiplier": 2,
      "Timeout": 30000000000
    }
  },
  "Auth": {
    "Mode": "",
    "OAuth2": {
      "TokenURL": "http://localhost:18080/auth/realms/camunda-platform/protocol/openid-connect",
      "ClientID": "******",
      "ClientSecret": "******",
      "Scopes": {
        "camunda_api": "profile",
        "operate_api": "profile",
        "tasklist_api": "profile"
      }
    },
    "Cookie": {
      "BaseURL": "http://localhost:8090",
      "Username": "******",
      "Password": "******"
    }
  },
  "APIs": {
    "Version": "87",
    "Camunda": {
      "Key": "camunda_api",
      "BaseURL": "http://localhost:8086/v2"
    },
    "Operate": {
      "Key": "operate_api",
      "BaseURL": "http://localhost:8081"
    },
    "Tasklist": {
      "Key": "tasklist_api",
      "BaseURL": "http://localhost:8082"
    }
  },
  "HTTP": {
    "Timeout": "23s"
  }
}
```


### c8volt in Action
Look here for practical examples of using c8volt for common tasks and special use cases.

#### Deleting an active process instance by cancelling it first, if the instance is active
List all process instances for a specific process definition:
```
$ c8volt get pi --bpmn-process-id=C87SimpleUserTask_Process --one-line
found: 12
2251799813685511 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:30.380+0000 i:false
2251799813685518 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:36.618+0000 i:false
2251799813685525 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:40.675+0000 i:false
2251799813685532 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:44.338+0000 i:false
2251799813685541 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:11:52.976+0000 i:false
2251799813685548 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:13:30.653+0000 i:false
2251799813685556 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:13:53.060+0000 i:false
2251799813685571 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:28.190+0000 e:2025-09-10T20:36:44.990+0000 p:2251799813685566 i:false
2251799813685582 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:33.364+0000 e:2025-09-09T20:27:55.530+0000 p:2251799813685577 i:false
2251799813685595 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:27.621+0000 p:2251799813685590 i:false
2251799813685606 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:33.533+0000 p:2251799813685601 i:false
2251799813685614 dev01 C87SimpleUserTask_Process v3 ACTIVE s:2025-09-10T10:03:12.700+0000 i:false
```
List only active process instances for the specific process definition in active state:
```
$ c8volt get pi --bpmn-process-id=C87SimpleUserTask_Process --one-line --state=active
filter: state=active
found: 10
2251799813685511 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:30.380+0000 i:false
2251799813685518 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:36.618+0000 i:false
2251799813685525 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:40.675+0000 i:false
2251799813685532 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:44.338+0000 i:false
2251799813685541 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:11:52.976+0000 i:false
2251799813685548 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:13:30.653+0000 i:false
2251799813685556 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:13:53.060+0000 i:false
2251799813685595 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:27.621+0000 p:2251799813685590 i:false
2251799813685606 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:33.533+0000 p:2251799813685601 i:false
2251799813685614 dev01 C87SimpleUserTask_Process v3 ACTIVE s:2025-09-10T10:03:12.700+0000 i:false
```
Try to delete an active process instance (will fail):
```
$ c8volt delete pi --key 2251799813685511
trying to delete process instance with key 2251799813685511...
Error deleting process instance with key 2251799813685511: unexpected status 400: {"status":400,"message":"Process instances needs to be in one of the states [COMPLETED, CANCELED]","instance":"dae2c2ce-58dd-4396-a948-4d57463168ed","type":"Invalid request"}
```
Try to delete an active process instance by forcing cancellation first (will succeed):
```
$ c8volt delete pi --key 2251799813685511 --cancel
trying to delete process instance with key 2251799813685511...
process instance with key 2251799813685511 not in state COMPLETED or CANCELED, cancelling it first...
trying to cancel process instance with key 2251799813685511...
process instance with key 2251799813685511 was successfully cancelled
waiting for process instance with key 2251799813685511 to be cancelled by workflow engine...
process instance "2251799813685511" currently in state "ACTIVE"; waiting...
process instance "2251799813685511" currently in state "ACTIVE"; waiting...
process instance "2251799813685511" currently in state "ACTIVE"; waiting...
process instance "2251799813685511" currently in state "ACTIVE"; waiting...
process instance "2251799813685511" currently in state "ACTIVE"; waiting...
process instance "2251799813685511" reached desired state "CANCELED"
process instance with key 2251799813685511 was successfully deleted
{
  "deleted": 1,
  "message": "Process instance and dependant data deleted for key '2251799813685511'"
}
```

#### Finding process instances with orphan (non-existing) parent process instances
List all process instances for a specific process definition:
```
$ c8volt get pi --bpmn-process-id=C87SimpleUserTask_Process --one-line
found: 12
2251799813685511 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:30.380+0000 i:false
2251799813685518 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:36.618+0000 i:false
2251799813685525 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:40.675+0000 i:false
2251799813685532 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:44.338+0000 i:false
2251799813685541 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:11:52.976+0000 i:false
2251799813685548 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:13:30.653+0000 i:false
2251799813685556 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:13:53.060+0000 i:false
2251799813685571 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:28.190+0000 e:2025-09-10T20:36:44.990+0000 p:2251799813685566 i:false
2251799813685582 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:33.364+0000 e:2025-09-09T20:27:55.530+0000 p:2251799813685577 i:false
2251799813685595 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:27.621+0000 p:2251799813685590 i:false
2251799813685606 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:33.533+0000 p:2251799813685601 i:false
2251799813685614 dev01 C87SimpleUserTask_Process v3 ACTIVE s:2025-09-10T10:03:12.700+0000 i:false
```
List all process instances for a specific process definition that are children of other process instances (look at p:...):
```
$ c8volt get pi --bpmn-process-id=C87SimpleUserTask_Process --one-line --children-only
filter: children-only=true
found: 4
2251799813685571 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:28.190+0000 e:2025-09-10T20:36:44.990+0000 p:2251799813685566 i:false
2251799813685582 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:33.364+0000 e:2025-09-09T20:27:55.530+0000 p:2251799813685577 i:false
2251799813685595 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:27.621+0000 p:2251799813685590 i:false
2251799813685606 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:33.533+0000 p:2251799813685601 i:false
```
List all process instances for a specific process definition that are children of orphan parent process instances (their parent process instance no longer exists):
```
$ c8volt get pi --bpmn-process-id=C87SimpleUserTask_Process --one-line --orphan-parents-only
filter: orphan-parents-only=true
found: 2
2251799813685571 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:28.190+0000 e:2025-09-10T20:36:44.990+0000 p:2251799813685566 i:false
2251799813685582 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:33.364+0000 e:2025-09-09T20:27:55.530+0000 p:2251799813685577 i:false
```

#### Listing process instances for a process definition in a specific version
List all process instances for a specific process definition:
```
$ c8volt get pi --bpmn-process-id=C87SimpleUserTask_Process --one-line
found: 11
2251799813685518 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:36.618+0000 i:false
2251799813685525 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:40.675+0000 i:false
2251799813685532 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:44.338+0000 i:false
2251799813685541 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:11:52.976+0000 i:false
2251799813685548 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:13:30.653+0000 i:false
2251799813685556 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T16:13:53.060+0000 i:false
2251799813685571 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:28.190+0000 e:2025-09-10T20:36:44.990+0000 p:2251799813685566 i:false
2251799813685582 dev01 C87SimpleUserTask_Process v2 CANCELED s:2025-09-09T19:10:33.364+0000 e:2025-09-09T20:27:55.530+0000 p:2251799813685577 i:false
2251799813685595 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:27.621+0000 p:2251799813685590 i:false
2251799813685606 dev01 C87SimpleUserTask_Process v2 ACTIVE s:2025-09-09T22:01:33.533+0000 p:2251799813685601 i:false
2251799813685614 dev01 C87SimpleUserTask_Process v3 ACTIVE s:2025-09-10T10:03:12.700+0000 i:false
```
List only process instances for version 1 of the process definition:
```
$ c8volt get pi --bpmn-process-id=C87SimpleUserTask_Process --one-line --process-version=1
found: 3
2251799813685518 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:36.618+0000 i:false
2251799813685525 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:40.675+0000 i:false
2251799813685532 dev01 C87SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-09-09T12:14:44.338+0000 i:false
```

Copyright © 2025 Adam Bogdan Boczek | [boczek.info](https://boczek.info)
