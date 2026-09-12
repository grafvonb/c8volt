# Command help review

Reviewed every declared command description, including grouping commands and the hidden variable command (57 descriptions). Runtime result formatting, logging, and machine contracts remain unchanged.

Help now describes targets, selection, safeguards, lifecycle behavior, version requirements, and flags. Presentation details belong outside command descriptions. The all-command regression test checks every registered command, including hidden commands.

## Reviewed descriptions

- cmd/cancel.go: Cancel running process instances.
- cmd/cancel_processinstance.go: Cancel process instances by key or search filters.
- cmd/capabilities.go: Describe the public c8volt command contract for scripts, CI jobs, and agents.
- cmd/config.go: Inspect and validate c8volt configuration.
- cmd/config_show.go: Show effective configuration with sensitive values sanitized.
- cmd/config_template.go: Print a blank configuration template.
- cmd/config_test_connection.go: Test configured Camunda connection.
- cmd/config_validate.go: Validate effective configuration.
- cmd/delete.go: Delete process instances or process definitions.
- cmd/delete_processdefinition.go: Delete process definition resources from Camunda.
- cmd/delete_processinstance.go: Delete process instances by key or search filters, with optional cancellation when deletion encounters nonterminal instances.
- cmd/deploy.go: Deploy BPMN resources to Camunda.
- cmd/deploy_processdefinition.go: Deploy BPMN process definition files and report the deployed definitions.
- cmd/embed.go: Use bundled BPMN fixtures.
- cmd/embed_deploy.go: Deploy bundled BPMN fixtures.
- cmd/embed_export.go: Export bundled BPMN fixtures to local files.
- cmd/embed_list.go: List bundled BPMN fixture files.
- cmd/expect.go: Wait for process instances to satisfy state or incident expectations.
- cmd/expect_processinstance.go: Wait for process instances to satisfy requested state and incident expectations.
- cmd/get.go: Inspect cluster, process, job, element, incident, tenant, and resource state without changing it.
- cmd/get_cluster.go: Inspect cluster-wide topology, version, and license information.
- cmd/get_cluster_license.go: Show connected cluster license.
- cmd/get_cluster_topology.go: Show connected cluster topology as a sorted tree.
- cmd/get_cluster_version.go: Show connected cluster version.
- cmd/get_element.go: List or fetch Camunda runtime element instances.
- cmd/get_incident.go: Get Camunda incidents by key or by search criteria.
- cmd/get_job.go: Inspect or search Camunda jobs.
- cmd/get_processdefinition.go: List or fetch deployed process definitions.
- cmd/get_processinstance.go: Get process instances by key or by search criteria.
- cmd/get_resource.go: Get a single resource by ID.
- cmd/get_tenant.go: List tenants visible to the configured environment.
- cmd/get_variable.go: Get a variable by name from a process instance.
- cmd/ops.go: Discover high-level operational workflows.
- cmd/ops_analyse_slow_process_instances.go: Discover read-only operational analyses.
- cmd/ops_analyse_slow_process_instances.go: Analyse slow process-instance timings.
- cmd/ops_execute.go: Discover predefined operational playbooks.
- cmd/ops_execute_retention_policy.go: Execute process-instance retention cleanup.
- cmd/ops_execute_smoketest.go: Execute a cluster smoke test workflow.
- cmd/ops_purge.go: Discover destructive operational cleanup workflows.
- cmd/ops_purge_all_processdefinitions.go: Purge all selected process definitions.
- cmd/ops_purge_orphan_processinstances.go: Purge orphan child process instances.
- cmd/ops_purge_processinstances_with_incidents.go: Purge process instances selected by incidents.
- cmd/ops_repair.go: Discover repair and remediation workflows.
- cmd/ops_repair_incident.go: Repair incidents by key or filter.
- cmd/ops_repair_processinstance.go: Repair incidents selected by process instances.
- cmd/resolve.go: Resolve operational incidents.
- cmd/resolve_incident.go: Resolve incidents by key.
- cmd/resolve_processinstance.go: Resolve process-instance incidents by key.
- cmd/root.go: c8volt: Camunda 8 Operations CLI.
- cmd/run.go: Start process instances.
- cmd/run_processinstance.go: Start process instances and confirm creation.
- cmd/update.go: Update existing resources.
- cmd/update_job.go: Update a Camunda job by key.
- cmd/update_processinstance.go: Update process-instance variables by key.
- cmd/version.go: Print version information.
- cmd/walk.go: Inspect process-instance relationships.
- cmd/walk_processinstance.go: Inspect the parent/child tree of process instances.
