<img src="./logo/c8volt_orange_black_bkg_white_400x152.png" alt="c8volt logo" style="border-radius: 5px;" />

# c8volt: Camunda 8 Operations CLI
The tool for Camunda 8 admins and developers to verify outcomes.

[**c8volt**](https://github.com/grafvonb/c8volt)'s design and development focus on operational effectiveness, ensuring that **done is done**.
There are plenty of operational tasks where you want to be sure that:

- A process instance was started – is it really active and running?
- A process instance was canceled – is it really in the cancelled state?
- A process instance tree was deleted – are all instances truly gone?

If some operation requires additional steps to reach the desired state, **c8volt** takes care of it for you by:
- Running multiple process instances concurrently, with configurable number of workers.
- Cancelling the root process instance when you want to cancel a child process instance.
- Deleting process instances by first cancelling them and then deleting them.
- Waiting until the process instance reaches the desired state.
- Traversing the process instance tree to perform operations like cancellation or deletion.
- and much more...

Bulk operations like running multiple process instances, cancelling or deleting multiple process instances are supported out-of-the-box.

Proper exit codes, various output formats and piping capabilities make **c8volt** suitable for scripting and automation.

**c8volt** focuses on real operational use cases while still providing the familiar CLI functionality such as standard CRUD commands on various resources.

## Not Already Convinced?

Here two short examples of what **c8volt** can do for you.

### Running a Process Instance with Activation Confirmation 

You want to start a process instance and be sure that it is really active and running. Standard use case, isn’t it?

Let's list latest deployed process definitions first:
```bash
$ ./c8volt get pd --latest
2251799813686015 <default> C88_DoubleUserTask_Process v1/v1.0.0
2251799813743379 <default> C88_MultipleSubProcessesParentProcess v5/v1.0.4
2251799813687133 <default> C88_SimpleParentProcess v2/v1.0.1
2251799813686018 <default> C88_SimpleUserTaskWithIncident_Process v1/v1.0.0
2251799813686017 <default> C88_SimpleUserTask_Process v1/v1.0.0
found: 5
```
Now start a process instance of `C88_SimpleUserTask_Process` (**c8volt** automatically picks the latest version) and wait until it is active:
```bash
$ ./c8volt run pi -b C88_SimpleUserTask_Process
INFO waiting for process instance of 2251799813686017 with key 2251799814015823 to be started by workflow engine...
INFO process instance 2251799814015823 succesfully created (start registered at 2025-11-13T05:07:46.105Z and confirmed at 2025-11-13T05:07:49Z) using process definition id 2251799813686017, C88_SimpleUserTask_Process, v1, tenant: <default>
```
Yes, it's really started and active! **c8volt** assured that by polling the process instance state until it reached `ACTIVE`. 

Awaiting the confirmation is default behavior of **c8volt** when starting process instances and other operations.

**DONE IS DONE!**

However, if you don't want that, you can use the `--no-wait` flag, to switch off the waiting mechanism:
```bash
$ ./c8volt run pi -b C88_SimpleUserTask_Process --no-wait
INFO process instance creation with the key 2251799814016994 requested at 2025-11-13T05:15:46Z (run not confirmed, as no-wait is set) using process definition id 2251799813686017, C88_SimpleUserTask_Process, v1, tenant: <default>
```

### Cancelling a Process Instance Tree
Here something more complex: You have a process instance tree with parent-child relationships and you want to cancel a mid-child process instance.

Let's see the current process instance family tree first:
```bash
$ ./c8volt walk pi --key 2251799813711967 --family
2251799813711967 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 ACTIVE s:2025-11-08T22:21:09.617Z p:<root> i:false ⇄ 
2251799813711976 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T22:21:09.617+0000 p:2251799813711967 i:false ⇄ 
2251799813711977 <default> C88_SimpleParentProcess v2/v1.0.1 ACTIVE s:2025-11-08T22:21:09.617+0000 p:2251799813711967 i:false ⇄ 
2251799813711985 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T22:21:09.617+0000 p:2251799813711977 i:false
```
The relationship between the process instances can be visualized like this:
```
2251799813711967  C88_MultipleSubProcessesParentProcess  v1/v1.0.0  (root)
├─ 2251799813711976  C88_SimpleUserTask_Process          v1/v1.0.0
└─ 2251799813711977  C88_SimpleParentProcess             v2/v1.0.1
   └─ 2251799813711985  C88_SimpleUserTask_Process       v1/v1.0.0
```
Try to cancel the mid-child process instance `2251799813711977`:
```bash
$ ./c8volt cancel pi --key=2251799813711977
You are about to cancel 1 process instance(s)? [y/N]: y
INFO cancelling process instances requested for 1 unique key(s) using 1 worker(s)
INFO cannot cancel, process instance with key 2251799813711977 is a child process of a root parent with key 2251799813711967
INFO use --force flag to cancel the root process instance with key 2251799813711967 and all its child processes
INFO cancelling 1 process instance(s) completed: 0 succeeded or already cancelled/teminated, 1 failed
```
You cannot cancel a child process instance directly. It is as standard behavior of Camunda 8. You need to cancel the root process instance to cancel all its children.

**c8volt** provides a flag `--force` to automatically find and cancel the root process instance, thus all its child processes:
```bash
$ ./c8volt cancel pi --key=2251799813711977 --force
You are about to cancel 1 process instance(s)? [y/N]: y
INFO cancelling process instances requested for 1 unique key(s) using 1 worker(s)
INFO cannot cancel, process instance with key 2251799813711977 is a child process of a root parent with key 2251799813711967
INFO force flag is set, cancelling root process instance with key 2251799813711967 and all its child processes
INFO waiting for process instance with key 2251799813711967 to be cancelled by workflow engine...
INFO process instance 2251799813711967 currently in state ACTIVE; waiting...
INFO process instance 2251799813711967 currently in state ACTIVE; waiting...
INFO process instance 2251799813711967 currently in state ACTIVE; waiting...
INFO process instance with key 2251799813711967 was successfully (confirmed) cancelled
INFO cancelling the family of 1 process instance(s) completed: 1 succeeded or already cancelled/teminated, 0 failed
```
What has happened?
1. **c8volt** detected that the specified process instance 2251799813711977 is a child and found its root ancestor 2251799813711967.
2. It cancelled the root process instance 2251799813711967, which in turn cancelled all its child process instances.
3. It waited until the root process instance reached the `CANCELED` (in C8.8 `TERMINATED`) state.

Let's check the family tree again:
```bash
$ ./c8volt walk pi --key 2251799813711967 --family
2251799813711967 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 TERMINATED s:2025-11-08T22:21:09.617Z e:2025-11-09T08:14:00.681Z p:<root> i:false ⇄ 
2251799813711976 <default> C88_SimpleUserTask_Process v1/v1.0.0 CANCELED s:2025-11-08T22:21:09.617+0000 e:2025-11-09T08:14:00.681+0000 p:2251799813711967 i:false ⇄ 
2251799813711977 <default> C88_SimpleParentProcess v2/v1.0.1 CANCELED s:2025-11-08T22:21:09.617+0000 e:2025-11-09T08:14:00.681+0000 p:2251799813711967 i:false ⇄ 
2251799813711985 <default> C88_SimpleUserTask_Process v1/v1.0.0 CANCELED s:2025-11-08T22:21:09.617+0000 e:2025-11-09T08:14:00.681+0000 p:2251799813711977 i:false
```
The whole process instance tree is now cancelled!

**DONE IS DONE!**

## CLI Reference

The full command/flag reference generated from the code is available in the [c8volt CLI Reference](./cli/index.md).

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

Download the latest release relevant to your OS from the [c8volt Releases](https://github.com/grafvonb/c8volt/releases) page, unpack it and check the version:
```bash
$ ./c8volt version
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
  level: info
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

The comprehensive documentation is on its way, but here are some highlights of what **c8volt** can do for you, with concrete usage examples.

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
Deploy all embedded BPMN models to Camunda 8 at once:
```bash
$ ./c8volt embed deploy --all
2251799813686014 <default> C88_MultipleSubProcessesParentProcess v1 vprocessdefinitions/C88_MultipleSubProcessesParentProcess.bpmn (2251799813686013)
2251799813686015 <default> C88_DoubleUserTask_Process v1 vprocessdefinitions/C88_DoubleUserTaskProcess.bpmn (2251799813686013)
2251799813686016 <default> C88_SimpleParentProcess v1 vprocessdefinitions/C88_SimpleParentProcess.bpmn (2251799813686013)
2251799813686017 <default> C88_SimpleUserTask_Process v1 vprocessdefinitions/C88_SimpleUserTaskProcess.bpmn (2251799813686013)
2251799813686018 <default> C88_SimpleUserTaskWithIncident_Process v1 vprocessdefinitions/C88_SimpleUserTaskWithIncidentProcess.bpmn (2251799813686013)
found: 5
```
Check that the models are deployed:
```bash
$ ./c8volt get pd
2251799813686015 <default> C88_DoubleUserTask_Process v1/v1.0.0
2251799813686014 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0
2251799813686016 <default> C88_SimpleParentProcess v1/v1.0.0
2251799813686018 <default> C88_SimpleUserTaskWithIncident_Process v1/v1.0.0
2251799813686017 <default> C88_SimpleUserTask_Process v1/v1.0.0
found: 5
```
And run a process instance using standard run command:
```bash
$ ./c8volt run pi -b C88_MultipleSubProcessesParentProcess
INFO waiting for process instance of 2251799813686014 with key 2251799813686587 to be started by workflow engine...
INFO process instance 2251799813686587 succesfully created (start registered at 2025-11-08T19:28:52.116Z and confirmed at 2025-11-08T19:28:59Z) using process definition id 2251799813686014, C88_MultipleSubProcessesParentProcess, v1, tenant: <default>
```
If you want to modify the embedded BPMN models, you can export them to a local folder with:
```bash
$ ./c8volt embed export -f processdefinitions/C88_SimpleParentProcess.bpmn -o ./exported_models
INFO exported 1 embedded resource(s) to "exported_models"
```
Modify it with Camunda Modeler or any text editor and deploy it back with "normal" deployment command:
```bash
$ ./c8volt deploy pd -f ./exported_models/processdefinitions/C88_SimpleParentProcess.bpmn
2251799813687133 <default> C88_SimpleParentProcess v2 vC88_SimpleParentProcess.bpmn (2251799813687132)
found: 1
```
And run a process instance of the modified model:
```bash
$ ./c8volt run pi -b C88_SimpleParentProcess
INFO waiting for process instance of 2251799813687133 with key 2251799813687256 to be started by workflow engine...
INFO process instance 2251799813687256 succesfully created (start registered at 2025-11-08T19:33:12.933Z and confirmed at 2025-11-08T19:33:16Z) using process definition id 2251799813687133, C88_SimpleParentProcess, v2, tenant: <default>
```

### Running Process Instances

A process instance can be started for a deployed BPMN model by its process definition key or BPMN process ID.
In case of the latter you can also specify a version of the process definition to use. If no version is specified, the latest version is used by default.

Running a process instance is asynchronous in Camunda 8, meaning that when the API call returns successfully, the process instance might not be actually started yet.
**c8volt** provides a waiting mechanism that polls the process instance state until it reaches the desired `ACTIVE` state or a timeout occurs.
You can switch off the waiting mechanism using the `--no-wait` flag.

**c8volt** also supports starting multiple process instances concurrently by specifying the `--count` (`-n`) flag and the number of workers with `--workers` (`-w`) flag.

#### Running Process Instances in Action

First get the deployed process definitions:
```bash
$ ./c8volt get get pd
2251799813686015 <default> C88_DoubleUserTask_Process v1/v1.0.0
2251799813686014 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0
2251799813687133 <default> C88_SimpleParentProcess v2/v1.0.1
2251799813686016 <default> C88_SimpleParentProcess v1/v1.0.0
2251799813686018 <default> C88_SimpleUserTaskWithIncident_Process v1/v1.0.0
2251799813686017 <default> C88_SimpleUserTask_Process v1/v1.0.0
found: 6
```
Start a single process instance of `C88_SimpleUserTask_Process` by its BPMN process ID and wait until it is active:
```bash
$ ./c8volt pi -b C88_SimpleUserTask_Process
INFO waiting for process instance of 2251799813686017 with key 2251799813687872 to be started by workflow engine...
INFO process instance 2251799813687872 succesfully created (start registered at 2025-11-08T19:37:21.745Z and confirmed at 2025-11-08T19:37:25Z) using process definition id 2251799813686017, C88_SimpleUserTask_Process, v1, tenant: <default>
```
Start the 1st version of the `C88_SimpleParentProcess` by its BPMN process ID and wait until it is active:
```bash
$ ./c8volt run pi -b C88_SimpleUserTask_Process --pd-version=1
INFO waiting for process instance of 2251799813686017 with key 2251799813688140 to be started by workflow engine...
INFO process instance 2251799813688140 succesfully created (start registered at 2025-11-08T19:39:08.557Z and confirmed at 2025-11-08T19:39:12Z) using process definition id 2251799813686017, C88_SimpleUserTask_Process, v1, tenant: <default>
```
Start a 1st version of `C88_SimpleParentProcess` by its process definition ID without waiting for it to be active:
```bash
$ ./c8volt run run pi --pd-id=2251799813686016 --no-wait
INFO process instance creation with the key 2251799813688590 requested at 2025-11-08T19:42:18Z (run not confirmed, as no-wait is set) using process definition id 2251799813686016, C88_SimpleParentProcess, v1, tenant: <default>
```
Start 5 process instances of latest version of `C88_DoubleUserTask_Process` concurrently using 3 workers and wait until they are active:
```bash
$ ./c8volt run pi -b C88_SimpleParentProcess -n 5 -w 3
INFO creating 5 process instances using 3 workers
INFO waiting for process instance of 2251799813687133 with key 2251799813688787 to be started by workflow engine...
INFO waiting for process instance of 2251799813687133 with key 2251799813688797 to be started by workflow engine...
INFO waiting for process instance of 2251799813687133 with key 2251799813688807 to be started by workflow engine...
INFO process instance 2251799813688807 succesfully created (start registered at 2025-11-08T19:43:36.275Z and confirmed at 2025-11-08T19:43:43Z) using process definition id 2251799813687133, C88_SimpleParentProcess, v2, tenant: <default>
INFO process instance 2251799813688787 succesfully created (start registered at 2025-11-08T19:43:36.253Z and confirmed at 2025-11-08T19:43:43Z) using process definition id 2251799813687133, C88_SimpleParentProcess, v2, tenant: <default>
INFO waiting for process instance of 2251799813687133 with key 2251799813688852 to be started by workflow engine...
INFO waiting for process instance of 2251799813687133 with key 2251799813688862 to be started by workflow engine...
INFO process instance 2251799813688797 succesfully created (start registered at 2025-11-08T19:43:36.267Z and confirmed at 2025-11-08T19:43:43Z) using process definition id 2251799813687133, C88_SimpleParentProcess, v2, tenant: <default>
INFO process instance 2251799813688862 succesfully created (start registered at 2025-11-08T19:43:43.843Z and confirmed at 2025-11-08T19:43:51Z) using process definition id 2251799813687133, C88_SimpleParentProcess, v2, tenant: <default>
INFO process instance 2251799813688852 succesfully created (start registered at 2025-11-08T19:43:43.837Z and confirmed at 2025-11-08T19:43:51Z) using process definition id 2251799813687133, C88_SimpleParentProcess, v2, tenant: <default>
INFO creation of 5 process instances completed
```

### Searching Process Instances

**c8volt** provides powerful searching capabilities for process instances.
It not only can find process instances by standard criteria such as process definition ID, BPMN process ID, version, state, tenant, but also traverse their parent-child relationships.
It offers different output formats suitable for scripting, such as one-liner or keys-only. The latter allows easy piping of results to other commands.

#### Searching Process Instances in Action

Get all process instances of `C88_SimpleUserTask_Process`:
```bash
$ ./c8volt get pi -b C88_SimpleUserTask_Process
2251799813686596 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686587 i:false
2251799813686605 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686597 i:false
2251799813687261 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:33:12.933+0000 p:2251799813687256 i:false
2251799813687872 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:37:21.745+0000 p:<root> i:false
2251799813688140 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:39:08.557+0000 p:<root> i:false
2251799813688595 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:42:18.756+0000 p:2251799813688590 i:false
2251799813688792 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:36.253+0000 p:2251799813688787 i:false
2251799813688802 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:36.267+0000 p:2251799813688797 i:false
2251799813688812 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:36.275+0000 p:2251799813688807 i:false
2251799813688857 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:43.837+0000 p:2251799813688852 i:false
2251799813688867 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:43.843+0000 p:2251799813688862 i:false
```
Get all process instances of `C88_SimpleParentProcess` that are children of other process instances:
```bash
$ ./c8volt get pi -b C88_SimpleUserTask_Process --children-only
2251799813686596 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686587 i:false
2251799813686605 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686597 i:false
2251799813687261 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:33:12.933+0000 p:2251799813687256 i:false
2251799813688595 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:42:18.756+0000 p:2251799813688590 i:false
2251799813688792 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:36.253+0000 p:2251799813688787 i:false
2251799813688802 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:36.267+0000 p:2251799813688797 i:false
2251799813688812 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:36.275+0000 p:2251799813688807 i:false
2251799813688857 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:43.837+0000 p:2251799813688852 i:false
2251799813688867 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:43.843+0000 p:2251799813688862 i:false
found: 9
```
Find the root of a child process instance by its key:
```bash
$ ./c8volt walk pi --key=2251799813686596
2251799813686596 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116Z p:2251799813686587 i:false ← 
2251799813686587 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116Z p:<root> i:false
```
Or even the whole family tree of this child process instance:
```bash
$ ./c8volt walk pi --key=2251799813686596 --mode=family
2251799813686587 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116Z p:<root> i:false ⇄ 
2251799813686596 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686587 i:false ⇄ 
2251799813686597 <default> C88_SimpleParentProcess v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686587 i:false ⇄ 
2251799813686605 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686597 i:false
```

### Cancellation of Process Instances

Standard cancellation of process instances in Camunda 8 is asynchronous and does not guarantee that the instance is actually cancelled when the API call returns successfully.
**c8volt** provides a waiting mechanism that polls the process instance state until it reaches the desired `CANCELED` state or a timeout occurs.
It allows also forced cancellation a process instance even it is a child, by traversing the parent chain and cancelling its root ancestor.

#### Cancellation in Action

Assuming you have following process instance tree (look at the sections above how to achieve that with c8volt's embedded deployment and run features):
```bash
$ ./c8volt get pi -b C88_MultipleSubProcessesParentProcess
2251799813686587 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:<root> i:false
found: 1
$ ./c8volt walk pi --key 2251799813686587 --mode=family
2251799813686587 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116Z p:<root> i:false ⇄ 
2251799813686596 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686587 i:false ⇄ 
2251799813686597 <default> C88_SimpleParentProcess v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686587 i:false ⇄ 
2251799813686605 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:28:52.116+0000 p:2251799813686597 i:false
```
Cancel the process instance `2251799813686597` (which is a child of `2251799813686587` and parent of `2251799813686605`):
```bash
$ ./c8volt cancel pi --key 2251799813686597
You are about to cancel 1 process instance(s)? [y/N]: y
INFO cancelling process instances requested for 1 unique key(s) using 1 worker(s)
INFO cannot cancel, process instance with key 2251799813686597 is a child process of a root parent with key 2251799813686587
INFO use --force flag to cancel the root process instance with key 2251799813686587 and all its child processes
INFO cancelling 1 process instances completed: 0 succeeded or already cancelled/teminated, 1 failed
```
You cannot cancel a child process instance directly. It is as standard behavior of Camunda 8.
However **c8volt** provides a flag `--force` to automatically find and cancel the root process instance, thus all its child processes:
```bash
$ ./c8volt cancel pi --key 2251799813686597 --force
You are about to cancel 1 process instance(s)? [y/N]: y
INFO cancelling process instances requested for 1 unique key(s) using 1 worker(s)
INFO cannot cancel, process instance with key 2251799813686597 is a child process of a root parent with key 2251799813686587
INFO force flag is set, cancelling root process instance with key 2251799813686587 and all its child processes
INFO waiting for process instance with key 2251799813686587 to be cancelled by workflow engine...
INFO process instance 2251799813686587 currently in state ACTIVE; waiting...
INFO process instance 2251799813686587 currently in state ACTIVE; waiting...
INFO process instance 2251799813686587 currently in state ACTIVE; waiting...
INFO process instance with key 2251799813686587 was successfully (confirmed) cancelled
INFO cancelling 1 process instances completed: 1 succeeded or already cancelled/teminated, 0 failed
```
Confirm that the whole process instance family is cancelled/terminated running again the `walk` command:
```bash
$ ./c8volt walk pi --key 2251799813686587 --mode=family
2251799813686587 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 TERMINATED s:2025-11-08T19:28:52.116Z e:2025-11-08T20:31:03.312Z p:<root> i:false ⇄ 
2251799813686596 <default> C88_SimpleUserTask_Process v1/v1.0.0 CANCELED s:2025-11-08T19:28:52.116+0000 e:2025-11-08T20:31:03.312+0000 p:2251799813686587 i:false ⇄ 
2251799813686597 <default> C88_SimpleParentProcess v1/v1.0.0 CANCELED s:2025-11-08T19:28:52.116+0000 e:2025-11-08T20:31:03.312+0000 p:2251799813686587 i:false ⇄ 
2251799813686605 <default> C88_SimpleUserTask_Process v1/v1.0.0 CANCELED s:2025-11-08T19:28:52.116+0000 e:2025-11-08T20:31:03.312+0000 p:2251799813686597 i:false
```
Every command that lists process instances or definitions has also a `--keys-only` flag that outputs only the process instance keys, one per line:
```bash
$ ./c8volt walk pi --key 2251799813686587 --mode=family --keys-only
2251799813686587
2251799813686596
2251799813686597
2251799813686605
```
This output type can be easily piped to other commands, e.g., to delete all these process instances.

### Deletion

Deletion of process instances and process definitions in Camunda 8 is not straightforward.
Due to distribution of state across multiple components and asynchronous nature of operations, additional steps are required to ensure that the resource is really deleted.
It is also possible to create an inconsistent state e.g. by deleting a process instance that is a parent of other active process instances as Camunda 8 API does
not prevent that or cascades the deletion to child instances as in case of cancellation.

#### Deletion of Process Instances in Action

Standard deletion of process instances in Camunda 8 is asynchronous and does not guarantee that the instance is actually deleted when the API call returns successfully.
Additionally, Camunda 8 API does not allow deleting a process instance that is still active. It must be cancelled first.
```bash
$ ./c8volt get pi -b C88_SimpleUserTask_Process
2251799813687261 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:33:12.933+0000 p:2251799813687256 i:false
2251799813687872 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:37:21.745+0000 p:<root> i:false
2251799813688140 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:39:08.557+0000 p:<root> i:false
2251799813688595 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:42:18.756+0000 p:2251799813688590 i:false
2251799813688792 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:36.253+0000 p:2251799813688787 i:false
2251799813688802 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:36.267+0000 p:2251799813688797 i:false
2251799813688812 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:36.275+0000 p:2251799813688807 i:false
2251799813688857 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:43.837+0000 p:2251799813688852 i:false
2251799813688867 <default> C88_SimpleUserTask_Process v1/v1.0.0 ACTIVE s:2025-11-08T19:43:43.843+0000 p:2251799813688862 i:false
found: 9
```
Try to delete the process instance `2251799813687872` first:
```bash
$ ./c8volt delete pi --key 2251799813687872
You are about to delete 1 process instance(s)? [y/N]: y
INFO deleting process instances requested for 1 unique key(s) using 1 worker(s)
INFO cannot delete, process instance 2251799813687872 is not in one of terminated states; use --force flag to cancel and then delete the process instance
INFO deleting 1 process instances completed: 0 succeeded, 1 failed
```
Retry with `--force` flag to first cancel and then delete the process instance:
```bash
$ ./c8volt delete pi --key 2251799813687872 --force
delete pi --key 2251799813687872 --force
You are about to delete 1 process instance(s)? [y/N]: y
INFO deleting process instances requested for 1 unique key(s) using 1 worker(s)
INFO process instance with key 2251799813687872 not in one of terminated states; cancelling it first
INFO waiting for process instance with key 2251799813687872 to be cancelled by workflow engine...
INFO process instance 2251799813687872 currently in state ACTIVE; waiting...
INFO process instance 2251799813687872 currently in state ACTIVE; waiting...
INFO process instance 2251799813687872 currently in state ACTIVE; waiting...
INFO process instance with key 2251799813687872 was successfully (confirmed) cancelled
INFO waiting for process instance with key 2251799813687872 to be cancelled by workflow engine...
INFO retrying deletion of process instance with key 2251799813687872
INFO process instance with key 2251799813687872 was successfully deleted
INFO deleting 1 process instances completed: 1 succeeded, 0 failed
```

#### Deletion of Process Definitions in Action

When deleting a process definition in Camunda 8, you must ensure that there are no active process instances running of that definition.
If there are any, you must cancel them first.
**c8volt** helps you with that by providing the `--force` flag that automatically finds and cancels all active process instances.

Try to delete the 4th version of `C88_MultipleSubProcessesParentProcess` process definition that has active process instances:
```bash
$ ./c8volt delete pd -b C88_MultipleSubProcessesParentProcess --pd-version 4
You are about to delete 1 process definition(s)? [y/N]: y
INFO deleting process definitions requested for 1 unique key(s) using 1 worker(s)
INFO deleting 1 process definitions completed: 0 succeeded, 1 failed
ERROR deleting process definiton(s): cannot delete process definition 2251799813743350 with active process instances; user --force to cancel them automatically
```
Retry with `--force` flag to first cancel (confirmed!) all active process instances and then delete the process definition:
```bash
$ ./c8volt delete pd -b C88_MultipleSubProcessesParentProcess --pd-version 4 --force
You are about to delete 1 process definition(s)? [y/N]: y
INFO deleting process definitions requested for 1 unique key(s) using 1 worker(s)
INFO cancelling active process instance(s) for process definition 2251799813743350 before deletion
INFO cancelling process instances requested for 3 unique key(s) using 3 worker(s)
INFO waiting for process instance with key 2251799813744466 to be cancelled by workflow engine...
INFO waiting for process instance with key 2251799813749275 to be cancelled by workflow engine...
INFO process instance 2251799813744466 currently in state ACTIVE; waiting...
INFO process instance 2251799813749275 currently in state ACTIVE; waiting...
INFO waiting for process instance with key 2251799813743626 to be cancelled by workflow engine...
INFO process instance 2251799813743626 currently in state ACTIVE; waiting...
INFO process instance 2251799813744466 currently in state ACTIVE; waiting...
INFO process instance 2251799813743626 currently in state ACTIVE; waiting...
INFO process instance with key 2251799813749275 was successfully (confirmed) cancelled
INFO process instance with key 2251799813744466 was successfully (confirmed) cancelled
INFO process instance with key 2251799813743626 was successfully (confirmed) cancelled
INFO cancelling 3 process instance(s) completed: 3 succeeded or already cancelled/teminated, 0 failed
INFO deleting 1 process definitions completed: 1 succeeded, 0 failed
```

### Scripting Support

#### Error Codes

In case of errors, c8volt returns specific exit codes that can be used in scripts to handle different error scenarios.
If you do not want to deal with specific error codes, you can use the `--no-err-codes` flag to make c8volt always return exit code 0.

#### Command Pipelining

Even thought **c8volt** offers many switches to filter resources in cancel and delete commands, you can also use command pipelining to achieve complex scenarios.
These commands recognize stdin input and read resource keys from it, one per line. The input can be provided by e.g. `echo` or `cat` commands, or by piping output of other c8volt commands with `--keys-only` flag.

Here is an example of common scenario for deleting a whole process instance tree, which is not possible, in opposite to cancellation, with standard Camunda 8 API.
Delete this process instance tree...
```bash
$ ./c8volt walk pi --key 2251799813686587 --mode=family
2251799813686587 <default> C88_MultipleSubProcessesParentProcess v1/v1.0.0 TERMINATED s:2025-11-08T19:28:52.116Z e:2025-11-08T20:31:03.312Z p:<root> i:false ⇄ 
2251799813686596 <default> C88_SimpleUserTask_Process v1/v1.0.0 CANCELED s:2025-11-08T19:28:52.116+0000 e:2025-11-08T20:31:03.312+0000 p:2251799813686587 i:false ⇄ 
2251799813686597 <default> C88_SimpleParentProcess v1/v1.0.0 CANCELED s:2025-11-08T19:28:52.116+0000 e:2025-11-08T20:31:03.312+0000 p:2251799813686587 i:false ⇄ 
2251799813686605 <default> C88_SimpleUserTask_Process v1/v1.0.0 CANCELED s:2025-11-08T19:28:52.116+0000 e:2025-11-08T20:31:03.312+0000 p:2251799813686597 i:false
```
By piping the output of the above `walk` command with `--keys-only` flag to the `delete` command:
```bash
./bin/c8volt walk pi --key 2251799813686587 --mode=family --keys-only | ./bin/c8volt delete pi
INFO deleting process instances requested for 4 unique key(s) using 4 worker(s)
INFO process instance with key 2251799813686605 was successfully deleted
INFO process instance with key 2251799813686596 was successfully deleted
INFO process instance with key 2251799813686587 was successfully deleted
INFO process instance with key 2251799813686597 was successfully deleted
INFO deleting 4 process instances completed: 4 succeeded, 0 failed
```

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
    client_id: "*******" # use environment variable C8VOLT_AUTH_OAUTH2_CLIENT_ID in PROD
    client_secret: "*******" # use environment variable C8VOLT_AUTH_OAUTH2_CLIENT_SECRET in PROD
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
See the [LICENSE](https://github.com/grafvonb/c8volt/blob/main/LICENSE) file for the full text.