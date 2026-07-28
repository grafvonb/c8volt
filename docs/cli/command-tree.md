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

- [`c8volt`]({{ "/cli/c8volt/" | relative_url }}) - Operate Camunda 8 workflows from the command line
  - [`c8volt cancel`]({{ "/cli/c8volt_cancel" | relative_url }}) - Cancel running process instances
    - [`c8volt cancel process-instance`]({{ "/cli/c8volt_cancel_process-instance" | relative_url }}) - Cancel process instances by key or filters
  - [`c8volt capabilities`]({{ "/cli/c8volt_capabilities" | relative_url }}) - Describe commands for scripts and agents
  - [`c8volt config`]({{ "/cli/c8volt_config" | relative_url }}) - Inspect and validate c8volt configuration
    - [`c8volt config show`]({{ "/cli/c8volt_config_show" | relative_url }}) - Show effective configuration
    - [`c8volt config template`]({{ "/cli/c8volt_config_template" | relative_url }}) - Print a blank configuration template
    - [`c8volt config test-connection`]({{ "/cli/c8volt_config_test-connection" | relative_url }}) - Test configured Camunda connection
    - [`c8volt config validate`]({{ "/cli/c8volt_config_validate" | relative_url }}) - Validate effective configuration
  - [`c8volt delete`]({{ "/cli/c8volt_delete" | relative_url }}) - Delete process instances or definitions
    - [`c8volt delete process-definition`]({{ "/cli/c8volt_delete_process-definition" | relative_url }}) - Delete process definition resources
    - [`c8volt delete process-instance`]({{ "/cli/c8volt_delete_process-instance" | relative_url }}) - Delete process instances by key or filters
  - [`c8volt deploy`]({{ "/cli/c8volt_deploy" | relative_url }}) - Deploy BPMN resources to Camunda
    - [`c8volt deploy process-definition`]({{ "/cli/c8volt_deploy_process-definition" | relative_url }}) - Deploy BPMN process definition files
  - [`c8volt embed`]({{ "/cli/c8volt_embed" | relative_url }}) - Use bundled BPMN fixtures
    - [`c8volt embed deploy`]({{ "/cli/c8volt_embed_deploy" | relative_url }}) - Deploy bundled BPMN fixtures
    - [`c8volt embed export`]({{ "/cli/c8volt_embed_export" | relative_url }}) - Export bundled BPMN fixtures to local files
    - [`c8volt embed list`]({{ "/cli/c8volt_embed_list" | relative_url }}) - List bundled BPMN fixture files
  - [`c8volt expect`]({{ "/cli/c8volt_expect" | relative_url }}) - Wait for process instances to satisfy expectations
    - [`c8volt expect process-instance`]({{ "/cli/c8volt_expect_process-instance" | relative_url }}) - Wait for process instances to satisfy expectations
  - [`c8volt get`]({{ "/cli/c8volt_get" | relative_url }}) - Inspect cluster, process, job, element, incident, tenant, and resource state
    - [`c8volt get cluster`]({{ "/cli/c8volt_get_cluster" | relative_url }}) - Inspect cluster-wide topology, version, and license information
      - [`c8volt get cluster license`]({{ "/cli/c8volt_get_cluster_license" | relative_url }}) - Show connected cluster license
      - [`c8volt get cluster topology`]({{ "/cli/c8volt_get_cluster_topology" | relative_url }}) - Show connected cluster topology as a tree
      - [`c8volt get cluster version`]({{ "/cli/c8volt_get_cluster_version" | relative_url }}) - Show connected cluster version
    - [`c8volt get element`]({{ "/cli/c8volt_get_element" | relative_url }}) - List or fetch runtime element instances
    - [`c8volt get incident`]({{ "/cli/c8volt_get_incident" | relative_url }}) - List or fetch incidents
    - [`c8volt get job`]({{ "/cli/c8volt_get_job" | relative_url }}) - Inspect or search jobs
    - [`c8volt get process-definition`]({{ "/cli/c8volt_get_process-definition" | relative_url }}) - List or fetch deployed process definitions
    - [`c8volt get process-instance`]({{ "/cli/c8volt_get_process-instance" | relative_url }}) - List or fetch process instances
    - [`c8volt get resource`]({{ "/cli/c8volt_get_resource" | relative_url }}) - Get a resource by ID
    - [`c8volt get tenant`]({{ "/cli/c8volt_get_tenant" | relative_url }}) - List tenants
  - [`c8volt ops`]({{ "/cli/c8volt_ops" | relative_url }}) - Discover high-level operational workflows
    - [`c8volt ops analyse`]({{ "/cli/c8volt_ops_analyse" | relative_url }}) - Discover read-only operational analyses
      - [`c8volt ops analyse slow-process-instances`]({{ "/cli/c8volt_ops_analyse_slow-process-instances" | relative_url }}) - Analyse slow process-instance timings
    - [`c8volt ops execute`]({{ "/cli/c8volt_ops_execute" | relative_url }}) - Discover predefined operational playbooks
      - [`c8volt ops execute retention-policy`]({{ "/cli/c8volt_ops_execute_retention-policy" | relative_url }}) - Execute process-instance retention cleanup
      - [`c8volt ops execute smoke-test`]({{ "/cli/c8volt_ops_execute_smoke-test" | relative_url }}) - Execute a cluster smoke test workflow
    - [`c8volt ops purge`]({{ "/cli/c8volt_ops_purge" | relative_url }}) - Discover destructive operational cleanup workflows
      - [`c8volt ops purge all-process-definitions`]({{ "/cli/c8volt_ops_purge_all-process-definitions" | relative_url }}) - Purge all selected process definitions
      - [`c8volt ops purge orphan-process-instances`]({{ "/cli/c8volt_ops_purge_orphan-process-instances" | relative_url }}) - Purge orphan child process instances
      - [`c8volt ops purge process-instances-with-incidents`]({{ "/cli/c8volt_ops_purge_process-instances-with-incidents" | relative_url }}) - Purge process instances selected by incidents
    - [`c8volt ops repair`]({{ "/cli/c8volt_ops_repair" | relative_url }}) - Discover repair and remediation workflows
      - [`c8volt ops repair incident`]({{ "/cli/c8volt_ops_repair_incident" | relative_url }}) - Repair incidents by key or filter
      - [`c8volt ops repair process-instance`]({{ "/cli/c8volt_ops_repair_process-instance" | relative_url }}) - Repair incidents selected by process instances
  - [`c8volt resolve`]({{ "/cli/c8volt_resolve" | relative_url }}) - Resolve operational incidents
    - [`c8volt resolve incident`]({{ "/cli/c8volt_resolve_incident" | relative_url }}) - Resolve incidents by key
    - [`c8volt resolve process-instance`]({{ "/cli/c8volt_resolve_process-instance" | relative_url }}) - Resolve process-instance incidents by key
  - [`c8volt run`]({{ "/cli/c8volt_run" | relative_url }}) - Start process instances
    - [`c8volt run process-instance`]({{ "/cli/c8volt_run_process-instance" | relative_url }}) - Start process instances and confirm creation
  - [`c8volt update`]({{ "/cli/c8volt_update" | relative_url }}) - Update existing resources
    - [`c8volt update job`]({{ "/cli/c8volt_update_job" | relative_url }}) - Update a job by key
    - [`c8volt update process-instance`]({{ "/cli/c8volt_update_process-instance" | relative_url }}) - Update process-instance variables by key
  - [`c8volt version`]({{ "/cli/c8volt_version" | relative_url }}) - Print version information
  - [`c8volt walk`]({{ "/cli/c8volt_walk" | relative_url }}) - Inspect process-instance relationships
    - [`c8volt walk process-instance`]({{ "/cli/c8volt_walk_process-instance" | relative_url }}) - Inspect the parent/child tree of process instances
