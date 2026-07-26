---
title: "Command Tree"
permalink: /cli/command-tree/
parent: "CLI Reference"
nav_order: 3
nav_exclude: false
has_toc: true
---

# CLI Command Tree

This generated tree lists the root reference plus the 55 available c8volt commands. Each entry links to the generated reference page for the same command metadata used by the binary.

- [`c8volt`](./c8volt) - Operate Camunda 8 workflows from the command line
  - [`c8volt cancel`](./c8volt_cancel) - Cancel running process instances
    - [`c8volt cancel process-instance`](./c8volt_cancel_process-instance) - Cancel process instances by key or filters
  - [`c8volt capabilities`](./c8volt_capabilities) - Describe commands for scripts and agents
  - [`c8volt config`](./c8volt_config) - Inspect and validate c8volt configuration
    - [`c8volt config show`](./c8volt_config_show) - Show effective configuration
    - [`c8volt config template`](./c8volt_config_template) - Print a blank configuration template
    - [`c8volt config test-connection`](./c8volt_config_test-connection) - Test configured Camunda connection
    - [`c8volt config validate`](./c8volt_config_validate) - Validate effective configuration
  - [`c8volt delete`](./c8volt_delete) - Delete process instances or definitions
    - [`c8volt delete process-definition`](./c8volt_delete_process-definition) - Delete process definition resources
    - [`c8volt delete process-instance`](./c8volt_delete_process-instance) - Delete process instances by key or filters
  - [`c8volt deploy`](./c8volt_deploy) - Deploy BPMN resources to Camunda
    - [`c8volt deploy process-definition`](./c8volt_deploy_process-definition) - Deploy BPMN process definition files
  - [`c8volt embed`](./c8volt_embed) - Use bundled BPMN fixtures
    - [`c8volt embed deploy`](./c8volt_embed_deploy) - Deploy bundled BPMN fixtures
    - [`c8volt embed export`](./c8volt_embed_export) - Export bundled BPMN fixtures to local files
    - [`c8volt embed list`](./c8volt_embed_list) - List bundled BPMN fixture files
  - [`c8volt expect`](./c8volt_expect) - Wait for process instances to satisfy expectations
    - [`c8volt expect process-instance`](./c8volt_expect_process-instance) - Wait for process instances to satisfy expectations
  - [`c8volt get`](./c8volt_get) - Inspect cluster, process, job, element, incident, tenant, and resource state
    - [`c8volt get cluster`](./c8volt_get_cluster) - Inspect cluster-wide topology, version, and license information
      - [`c8volt get cluster license`](./c8volt_get_cluster_license) - Show connected cluster license
      - [`c8volt get cluster topology`](./c8volt_get_cluster_topology) - Show connected cluster topology as a tree
      - [`c8volt get cluster version`](./c8volt_get_cluster_version) - Show connected cluster version
    - [`c8volt get element`](./c8volt_get_element) - List or fetch runtime element instances
    - [`c8volt get incident`](./c8volt_get_incident) - List or fetch incidents
    - [`c8volt get job`](./c8volt_get_job) - Inspect or search jobs
    - [`c8volt get process-definition`](./c8volt_get_process-definition) - List or fetch deployed process definitions
    - [`c8volt get process-instance`](./c8volt_get_process-instance) - List or fetch process instances
    - [`c8volt get resource`](./c8volt_get_resource) - Get a resource by ID
    - [`c8volt get tenant`](./c8volt_get_tenant) - List tenants
  - [`c8volt ops`](./c8volt_ops) - Discover high-level operational workflows
    - [`c8volt ops analyse`](./c8volt_ops_analyse) - Discover read-only operational analyses
      - [`c8volt ops analyse slow-process-instances`](./c8volt_ops_analyse_slow-process-instances) - Analyse slow process-instance timings
    - [`c8volt ops execute`](./c8volt_ops_execute) - Discover predefined operational playbooks
      - [`c8volt ops execute retention-policy`](./c8volt_ops_execute_retention-policy) - Execute process-instance retention cleanup
      - [`c8volt ops execute smoke-test`](./c8volt_ops_execute_smoke-test) - Execute a cluster smoke test workflow
    - [`c8volt ops purge`](./c8volt_ops_purge) - Discover destructive operational cleanup workflows
      - [`c8volt ops purge all-process-definitions`](./c8volt_ops_purge_all-process-definitions) - Purge all selected process definitions
      - [`c8volt ops purge orphan-process-instances`](./c8volt_ops_purge_orphan-process-instances) - Purge orphan child process instances
      - [`c8volt ops purge process-instances-with-incidents`](./c8volt_ops_purge_process-instances-with-incidents) - Purge process instances selected by incidents
    - [`c8volt ops repair`](./c8volt_ops_repair) - Discover repair and remediation workflows
      - [`c8volt ops repair incident`](./c8volt_ops_repair_incident) - Repair incidents by key or filter
      - [`c8volt ops repair process-instance`](./c8volt_ops_repair_process-instance) - Repair incidents selected by process instances
  - [`c8volt resolve`](./c8volt_resolve) - Resolve operational incidents
    - [`c8volt resolve incident`](./c8volt_resolve_incident) - Resolve incidents by key
    - [`c8volt resolve process-instance`](./c8volt_resolve_process-instance) - Resolve process-instance incidents by key
  - [`c8volt run`](./c8volt_run) - Start process instances
    - [`c8volt run process-instance`](./c8volt_run_process-instance) - Start process instances and confirm creation
  - [`c8volt update`](./c8volt_update) - Update existing resources
    - [`c8volt update job`](./c8volt_update_job) - Update a job by key
    - [`c8volt update process-instance`](./c8volt_update_process-instance) - Update process-instance variables by key
  - [`c8volt version`](./c8volt_version) - Print version information
  - [`c8volt walk`](./c8volt_walk) - Inspect process-instance relationships
    - [`c8volt walk process-instance`](./c8volt_walk_process-instance) - Inspect the parent/child tree of process instances
