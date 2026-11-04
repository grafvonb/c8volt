// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/cmd"
	"github.com/spf13/cobra/doc"
)

// TestFormatDocsBuildInfoRelease verifies generated docs disclose release provenance and Camunda support metadata.
func TestFormatDocsBuildInfoRelease(t *testing.T) {
	info := cmd.BuildInfo{
		Version:                  "v2.1.0",
		Commit:                   "abcdef123456",
		Date:                     "2026-04-11T09:10:11Z",
		SupportedCamundaVersions: "8.8, 8.9, 8.10",
		Camunda810Baseline:       "8.10.0-alpha4",
		Camunda810BaselineStatus: "prerelease",
	}

	got := formatDocsBuildInfo(info)

	for _, want := range []string{
		"Generated from release `v2.1.0`",
		"commit `abcdef123456`",
		"built `2026-04-11T09:10:11Z`",
		"Supported Camunda 8 versions: 8.8, 8.9, 8.10",
		"Camunda 8.10 baseline: 8.10.0-alpha4 (prerelease)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected build info to contain %q, got %q", want, got)
		}
	}
}

// TestFormatDocsBuildInfoNonRelease verifies non-release docs still disclose compatibility metadata.
func TestFormatDocsBuildInfoNonRelease(t *testing.T) {
	info := cmd.BuildInfo{
		Version:                  "v2.1.0-8-gabcdef123456-dirty",
		Commit:                   "abcdef123456",
		Date:                     "2026-04-11T09:10:11Z",
		SupportedCamundaVersions: "8.8, 8.9, 8.10",
		Camunda810Baseline:       "8.10.0-alpha4",
		Camunda810BaselineStatus: "prerelease",
	}

	got := formatDocsBuildInfo(info)

	for _, want := range []string{
		"Generated from build `c8volt v2.1.0-8-gabcdef123456-dirty`",
		"commit `abcdef123456`",
		"built `2026-04-11T09:10:11Z`",
		"Supported Camunda 8 versions: 8.8, 8.9, 8.10",
		"Camunda 8.10 baseline: 8.10.0-alpha4 (prerelease)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected build info to contain %q, got %q", want, got)
		}
	}
}

// TestRewriteDocsIndexLinks verifies README-relative links become valid generated docs links.
func TestRewriteDocsIndexLinks(t *testing.T) {
	body := strings.Join([]string{
		`<img src="./docs/logo/c8volt.png" />`,
		`Screencast: ![demo](docs/assets/screencasts/fast-start.gif)`,
		`Asset: <img src="./docs/assets/example.png" />`,
		`CLI: [reference](./docs/cli/index.md)`,
		`CLI page: [get process-instance](docs/cli/c8volt_get_process-instance.md)`,
		`CLI page with dot: [get element](./docs/cli/c8volt_get_element.md)`,
		`Ops index: [playbooks](docs/ops/index.md)`,
		`Ops page: [Execute Smoke Test](docs/ops/execute-smoke-test.md)`,
		`Ops page with dot: [Execute Retention Policy](./docs/ops/execute-retention-policy.md)`,
		`Docs: [LICENSE](./LICENSE), [COPYRIGHT](./COPYRIGHT), [NOTICE.md](./NOTICE.md)`,
		`Project: [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](./SECURITY.md), [TRADEMARKS.md](TRADEMARKS.md), [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)`,
		`Lowercase target: [trademarks.md](trademarks.md)`,
	}, "\n")

	got := rewriteDocsIndexLinks(body)

	for _, want := range []string{
		`<img src="./logo/c8volt.png" />`,
		`Screencast: ![demo](./assets/screencasts/fast-start.gif)`,
		`Asset: <img src="./assets/example.png" />`,
		`CLI: [reference](./cli/)`,
		`CLI page: [get process-instance](./cli/c8volt_get_process-instance)`,
		`CLI page with dot: [get element](./cli/c8volt_get_element)`,
		`Ops index: [playbooks](./ops/)`,
		`Ops page: [Execute Smoke Test](./ops/execute-smoke-test/)`,
		`Ops page with dot: [Execute Retention Policy](./ops/execute-retention-policy/)`,
		`[LICENSE](https://github.com/grafvonb/c8volt/blob/main/LICENSE)`,
		`[COPYRIGHT](https://github.com/grafvonb/c8volt/blob/main/COPYRIGHT)`,
		`[NOTICE.md](https://github.com/grafvonb/c8volt/blob/main/NOTICE.md)`,
		`[CONTRIBUTING.md](https://github.com/grafvonb/c8volt/blob/main/CONTRIBUTING.md)`,
		`[SECURITY.md](https://github.com/grafvonb/c8volt/blob/main/SECURITY.md)`,
		`[TRADEMARKS.md](https://github.com/grafvonb/c8volt/blob/main/TRADEMARKS.md)`,
		`[CODE_OF_CONDUCT.md](https://github.com/grafvonb/c8volt/blob/main/CODE_OF_CONDUCT.md)`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected rewritten body to contain %q, got %q", want, got)
		}
	}
}

func TestStripDocsIndexExcludedBlocks(t *testing.T) {
	body := strings.Join([]string{
		"# c8volt Camunda 8 CLI",
		"<!-- docs-index-exclude-start -->",
		"**Full documentation:** [c8volt.info](https://c8volt.info)",
		"<!-- docs-index-exclude-end -->",
		"> **done is done**",
	}, "\n")

	got := stripDocsIndexExcludedBlocks(body)

	if strings.Contains(got, "Full documentation") || strings.Contains(got, "docs-index-exclude") {
		t.Fatalf("expected README-only docs block to be stripped, got %q", got)
	}
	if !strings.Contains(got, "# c8volt Camunda 8 CLI") || !strings.Contains(got, "> **done is done**") {
		t.Fatalf("expected surrounding README content to remain, got %q", got)
	}
}

func TestCLIMarkdownPreludeOmitsOpsBreadcrumb(t *testing.T) {
	rootPrelude := cliMarkdownPrelude("c8volt")
	for _, want := range []string{
		`title: "c8volt"`,
		"permalink: /cli/c8volt/",
		`parent: "CLI Reference"`,
		"nav_order: 1",
		"nav_exclude: false",
	} {
		if !strings.Contains(rootPrelude, want) {
			t.Fatalf("expected root CLI page prelude to contain %q, got %q", want, rootPrelude)
		}
	}

	opsPrelude := cliMarkdownPrelude("c8volt_ops_repair_incident")
	if strings.Contains(opsPrelude, "CLI Reference") {
		t.Fatalf("expected ops CLI page prelude to omit CLI reference breadcrumb, got %q", opsPrelude)
	}
	if !strings.Contains(opsPrelude, `title: "c8volt ops repair incident"`) {
		t.Fatalf("expected ops CLI page prelude to preserve title, got %q", opsPrelude)
	}

	regularPrelude := cliMarkdownPrelude("c8volt_get_process-instance")
	if !strings.Contains(regularPrelude, "CLI Reference") {
		t.Fatalf("expected non-ops CLI page prelude to keep CLI reference breadcrumb, got %q", regularPrelude)
	}
}

// TestCLICommandTreeDocumentsGeneratedCommandSurface includes every public command, including exact-instance suspension and resumption.
func TestCLICommandTreeDocumentsGeneratedCommandSurface(t *testing.T) {
	root := cmd.Root()
	root.DisableAutoGenTag = true

	tree := commandTreeMarkdown(root, 0)

	for _, want := range []string{
		"- [`c8volt`]({{ \"/cli/c8volt/\" | relative_url }})",
		"  - [`c8volt get`]({{ \"/cli/c8volt_get\" | relative_url }})",
		"    - [`c8volt get process-instance`]({{ \"/cli/c8volt_get_process-instance\" | relative_url }})",
		"    - [`c8volt get element`]({{ \"/cli/c8volt_get_element\" | relative_url }})",
		"    - [`c8volt get user-task`]({{ \"/cli/c8volt_get_user-task\" | relative_url }})",
		"  - [`c8volt resume`]({{ \"/cli/c8volt_resume\" | relative_url }})",
		"    - [`c8volt resume process-instance`]({{ \"/cli/c8volt_resume_process-instance\" | relative_url }})",
		"  - [`c8volt suspend`]({{ \"/cli/c8volt_suspend\" | relative_url }})",
		"    - [`c8volt suspend process-instance`]({{ \"/cli/c8volt_suspend_process-instance\" | relative_url }})",
		"    - [`c8volt update element`]({{ \"/cli/c8volt_update_element\" | relative_url }})",
		"    - [`c8volt update user-task`]({{ \"/cli/c8volt_update_user-task\" | relative_url }})",
		"    - [`c8volt ops analyse`]({{ \"/cli/c8volt_ops_analyse\" | relative_url }})",
		"      - [`c8volt ops analyse slow-process-instances`]({{ \"/cli/c8volt_ops_analyse_slow-process-instances\" | relative_url }})",
	} {
		if !strings.Contains(tree, want) {
			t.Fatalf("expected command tree to contain %q, got:\n%s", want, tree)
		}
	}

	if got := strings.Count(tree, "- [`") - 1; got != 62 {
		t.Fatalf("expected command tree to contain 62 non-root commands, got %d", got)
	}
}

func TestDocsReferenceLinkNameUsesStableSitePaths(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
	}{
		{name: "c8volt.md", want: `{{ "/cli/c8volt/" | relative_url }}`},
		{name: "c8volt_get_process-instance.md", want: `{{ "/cli/c8volt_get_process-instance" | relative_url }}`},
	} {
		if got := docsReferenceLinkName(tt.name); got != tt.want {
			t.Fatalf("expected link for %s to be %q, got %q", tt.name, tt.want, got)
		}
	}
}

func TestCLIDebtRefactorAssessmentArtifactDocumentsBaseline(t *testing.T) {
	bodyBytes, err := os.ReadFile(filepath.Join("..", "specs", "254-cli-debt-refactor", "assessment.md"))
	if err != nil {
		t.Fatalf("read assessment artifact: %v", err)
	}
	body := string(bodyBytes)

	for _, want := range []string{
		"## Command Node Assessment",
		"| Path | Aliases | Family | Mutation | Contract | Automation | Output Modes | Paging | Mutates | Activity | Durable Progress | Machine Constraints | Ownership | Execution Style | Risk |",
		"## High-Risk Workflows And Duplicated Mechanics",
		"## Intentional Differences And Non-Goals",
		"## Performance Characterization Plan",
		"`get process-instance`",
		"`delete process-instance`",
		"`ops purge process-instances-with-incidents`",
		"`walk process-instance`",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected assessment artifact to contain %q", want)
		}
	}

	if got := strings.Count(body, "\n| `"); got != 57 {
		t.Fatalf("expected assessment artifact to contain 57 command-node rows, got %d", got)
	}
}

// TestCLIDebtRefactorUserFacingDocsDocumentPagingContracts keeps README and
// docs examples aligned with generated command help for paging and automation.
func TestCLIDebtRefactorUserFacingDocsDocumentPagingContracts(t *testing.T) {
	for path, fragments := range map[string][]string{
		"../README.md":           {"# c8volt v5", "Camunda 8.10", "--set-retries", "--with-stats", "c8volt capabilities --json"},
		"../docs/ops/index.md":   {"--batch-size", "--limit", "--auto-confirm", "--automation", "--report-file"},
		"../docs/use-cases.md":   {"Current usage belongs in the", "command contract"},
		"../docs/camunda-cli.md": {"c8volt capabilities --json", "8.10"},
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range fragments {
			if !strings.Contains(string(data), fragment) {
				t.Errorf("%s: missing %q", path, fragment)
			}
		}
	}
}

// TestGeneratedProcessInstanceDocsDocumentHasUserTasksLookup protects generated command docs for the task-key lookup surface.
func TestGeneratedProcessInstanceDocsDocumentHasUserTasksLookup(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_get_process-instance.md")
	for _, want := range []string{"--has-user-tasks strings", "user task key(s) whose owning process instances should be fetched"} {
		if !strings.Contains(body0, want) {
			t.Errorf("get process-instance: missing %q", want)
		}
	}
}

// TestGeneratedGetIncidentDocsDocumentLookupSearchAndOutput protects generated docs for direct incident lookup and search.
func TestGeneratedGetIncidentDocsDocumentLookupSearchAndOutput(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_get_incident.md")
	for _, want := range []string{"Search defaults to active incidents", "--error-message string", "--pi-keys-only", "--creation-time-after string", "--total", "c8volt get incident --key <incident-key>"} {
		if !strings.Contains(body0, want) {
			t.Errorf("get incident: missing %q", want)
		}
	}
}

// TestGeneratedGetElementDocsDocumentLookupSearchAndOutput protects generated docs for runtime element lookup and search.
func TestGeneratedGetElementDocsDocumentLookupSearchAndOutput(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_get_element.md")
	for _, want := range []string{"--pi-key string", "--element-id string", "--with-listeners", "--with-vars", "--total", "Parent-scope variables are excluded", "worker waiting time"} {
		if !strings.Contains(body0, want) {
			t.Errorf("get element: missing %q", want)
		}
	}
}

// TestGeneratedPagedWorkflowDocsDocumentContracts protects generated CLI docs
// for page-size, total-limit, frozen-scope, automation, and progress wording.
func TestGeneratedPagedWorkflowDocsDocumentContracts(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_get_job.md")
	for _, want := range []string{"--batch-size int32", "--limit int32"} {
		if !strings.Contains(body0, want) {
			t.Errorf("get job: missing %q", want)
		}
	}
	body1 := readGeneratedDocForTest(t, out, "c8volt_get_element.md")
	for _, want := range []string{"--batch-size int32", "--limit int32"} {
		if !strings.Contains(body1, want) {
			t.Errorf("get element: missing %q", want)
		}
	}
	body2 := readGeneratedDocForTest(t, out, "c8volt_get_incident.md")
	for _, want := range []string{"--batch-size int32", "--limit int32"} {
		if !strings.Contains(body2, want) {
			t.Errorf("get incident: missing %q", want)
		}
	}
	body3 := readGeneratedDocForTest(t, out, "c8volt_get_process-instance.md")
	for _, want := range []string{"--batch-size int32", "--limit int32"} {
		if !strings.Contains(body3, want) {
			t.Errorf("get process-instance: missing %q", want)
		}
	}
	body4 := readGeneratedDocForTest(t, out, "c8volt_cancel_process-instance.md")
	for _, want := range []string{"--batch-size int32", "--limit int32"} {
		if !strings.Contains(body4, want) {
			t.Errorf("cancel process-instance: missing %q", want)
		}
	}
	body5 := readGeneratedDocForTest(t, out, "c8volt_delete_process-instance.md")
	for _, want := range []string{"--batch-size int32", "--limit int32"} {
		if !strings.Contains(body5, want) {
			t.Errorf("delete process-instance: missing %q", want)
		}
	}
	body6 := readGeneratedDocForTest(t, out, "c8volt_ops_execute_retention-policy.md")
	for _, want := range []string{"--batch-size int32", "--limit int32"} {
		if !strings.Contains(body6, want) {
			t.Errorf("ops execute retention-policy: missing %q", want)
		}
	}
}

// TestGeneratedRunProcessInstanceDocsDocumentPipeline protects generated docs for keys-only run composition.
func TestGeneratedRunProcessInstanceDocsDocumentPipeline(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_run_process-instance.md")
	for _, want := range []string{"--keys-only", "--set-vars string", "--set-bid string", "Creation confirmation does not mean the process has completed"} {
		if !strings.Contains(body0, want) {
			t.Errorf("run process-instance: missing %q", want)
		}
	}
}

// TestGeneratedResolveDocsDocumentResolveWorkflows protects generated docs for the incident recovery command family.
func TestGeneratedResolveDocsDocumentResolveWorkflows(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_resolve_incident.md")
	for _, want := range []string{"--key strings", "--no-wait", "--json"} {
		if !strings.Contains(body0, want) {
			t.Errorf("resolve incident: missing %q", want)
		}
	}
	body1 := readGeneratedDocForTest(t, out, "c8volt_resolve_process-instance.md")
	for _, want := range []string{"--dry-run", "job retries", "incident scope before confirmation"} {
		if !strings.Contains(body1, want) {
			t.Errorf("resolve process-instance: missing %q", want)
		}
	}
}

// TestGeneratedOpsDocsDocumentGroupingCommands protects generated docs for the ops command foundation.
func TestGeneratedOpsDocsDocumentGroupingCommands(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_ops.md")
	for _, want := range []string{"repair", "retention", "--dry-run"} {
		if !strings.Contains(body0, want) {
			t.Errorf("ops: missing %q", want)
		}
	}
	body1 := readGeneratedDocForTest(t, out, "c8volt_ops_analyse.md")
	for _, want := range []string{"slow-process-instances"} {
		if !strings.Contains(body1, want) {
			t.Errorf("ops analyse: missing %q", want)
		}
	}
	body2 := readGeneratedDocForTest(t, out, "c8volt_ops_execute.md")
	for _, want := range []string{"retention-policy", "smoke-test"} {
		if !strings.Contains(body2, want) {
			t.Errorf("ops execute: missing %q", want)
		}
	}
	body3 := readGeneratedDocForTest(t, out, "c8volt_ops_purge.md")
	for _, want := range []string{"all-process-definitions", "orphan-process-instances"} {
		if !strings.Contains(body3, want) {
			t.Errorf("ops purge: missing %q", want)
		}
	}
	body4 := readGeneratedDocForTest(t, out, "c8volt_ops_repair.md")
	for _, want := range []string{"incident", "process-instance"} {
		if !strings.Contains(body4, want) {
			t.Errorf("ops repair: missing %q", want)
		}
	}
}

// TestGeneratedOpsPagedDiscoveryDocsDocumentHelp verifies generated ops docs preserve the complete-discovery flag contract.
func TestGeneratedOpsPagedDiscoveryDocsDocumentHelp(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_ops_purge_process-instances-with-incidents.md")
	for _, want := range []string{"--batch-size int32", "--limit int32", "--dry-run", "--report-file"} {
		if !strings.Contains(body0, want) {
			t.Errorf("ops purge process-instances-with-incidents: missing %q", want)
		}
	}
	body1 := readGeneratedDocForTest(t, out, "c8volt_ops_repair_incident.md")
	for _, want := range []string{"--batch-size int32", "--limit int32", "--dry-run", "--report-file"} {
		if !strings.Contains(body1, want) {
			t.Errorf("ops repair incident: missing %q", want)
		}
	}
	body2 := readGeneratedDocForTest(t, out, "c8volt_ops_repair_process-instance.md")
	for _, want := range []string{"--batch-size int32", "--limit int32", "--dry-run", "--report-file"} {
		if !strings.Contains(body2, want) {
			t.Errorf("ops repair process-instance: missing %q", want)
		}
	}
	body3 := readGeneratedDocForTest(t, out, "c8volt_ops_purge_all-process-definitions.md")
	for _, want := range []string{"--batch-size int32", "--limit int32", "--dry-run", "--report-file"} {
		if !strings.Contains(body3, want) {
			t.Errorf("ops purge all-process-definitions: missing %q", want)
		}
	}
}

// TestGeneratedConfigDocsDocumentSplitDiagnostics protects generated command docs for config diagnostics.
func TestGeneratedConfigDocsDocumentSplitDiagnostics(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_config.md")
	for _, want := range []string{"show", "template", "validate", "test-connection"} {
		if !strings.Contains(body0, want) {
			t.Errorf("config: missing %q", want)
		}
	}
	body1 := readGeneratedDocForTest(t, out, "c8volt_config_show.md")
	for _, want := range []string{"secrets redacted"} {
		if !strings.Contains(body1, want) {
			t.Errorf("config show: missing %q", want)
		}
	}
	body2 := readGeneratedDocForTest(t, out, "c8volt_config_validate.md")
	for _, want := range []string{"without contacting Camunda"} {
		if !strings.Contains(body2, want) {
			t.Errorf("config validate: missing %q", want)
		}
	}
	body3 := readGeneratedDocForTest(t, out, "c8volt_config_template.md")
	for _, want := range []string{"app.camunda_version"} {
		if !strings.Contains(body3, want) {
			t.Errorf("config template: missing %q", want)
		}
	}
	body4 := readGeneratedDocForTest(t, out, "c8volt_config_test-connection.md")
	for _, want := range []string{"gateway version", "does not fail"} {
		if !strings.Contains(body4, want) {
			t.Errorf("config test-connection: missing %q", want)
		}
	}
}

// TestGeneratedGetProcessInstanceDocsDocumentVariableSearch verifies generated
// CLI markdown carries the same variable-search contract as command help.
func TestGeneratedGetProcessInstanceDocsDocumentVariableSearch(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_get_process-instance.md")
	for _, want := range []string{"--var stringArray", "--var-like stringArray", "--var-exists stringArray", "filters combine with AND"} {
		if !strings.Contains(body0, want) {
			t.Errorf("get process-instance: missing %q", want)
		}
	}
}

func TestGeneratedAllTenantsDocsDocumentSyntaxAndRestrictions(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_get_process-instance.md")
	for _, want := range []string{"--all-tenants", "search all accessible tenants; cannot combine with --tenant", "explicit keys use backend authorization"} {
		if !strings.Contains(body0, want) {
			t.Errorf("get process-instance: missing %q", want)
		}
	}
	body1 := readGeneratedDocForTest(t, out, "c8volt_get_resource.md")
	for _, want := range []string{"--all-tenants", "search all accessible tenants; cannot combine with --tenant", "explicit keys use backend authorization"} {
		if !strings.Contains(body1, want) {
			t.Errorf("get resource: missing %q", want)
		}
	}
	body2 := readGeneratedDocForTest(t, out, "c8volt_deploy_process-definition.md")
	for _, want := range []string{"--all-tenants", "search all accessible tenants; cannot combine with --tenant", "explicit keys use backend authorization"} {
		if !strings.Contains(body2, want) {
			t.Errorf("deploy process-definition: missing %q", want)
		}
	}
	body3 := readGeneratedDocForTest(t, out, "c8volt_embed_deploy.md")
	for _, want := range []string{"--all-tenants", "search all accessible tenants; cannot combine with --tenant", "explicit keys use backend authorization"} {
		if !strings.Contains(body3, want) {
			t.Errorf("embed deploy: missing %q", want)
		}
	}
	body4 := readGeneratedDocForTest(t, out, "c8volt_run_process-instance.md")
	for _, want := range []string{"--all-tenants", "search all accessible tenants; cannot combine with --tenant", "explicit keys use backend authorization"} {
		if !strings.Contains(body4, want) {
			t.Errorf("run process-instance: missing %q", want)
		}
	}
	body5 := readGeneratedDocForTest(t, out, "c8volt_ops_execute_smoke-test.md")
	for _, want := range []string{"--all-tenants", "search all accessible tenants; cannot combine with --tenant", "explicit keys use backend authorization"} {
		if !strings.Contains(body5, want) {
			t.Errorf("ops execute smoke-test: missing %q", want)
		}
	}
}

func TestGeneratedWalkProcessInstanceDocsDocumentListeners(t *testing.T) {
	out := generateReferenceForTest(t)
	body0 := readGeneratedDocForTest(t, out, "c8volt_walk_process-instance.md")
	for _, want := range []string{"--with-elements", "--with-listeners", "requires --with-elements", "c8volt walk process-instance --key <process-instance-key> --with-elements --with-listeners"} {
		if !strings.Contains(body0, want) {
			t.Errorf("walk process-instance: missing %q", want)
		}
	}
}

func readGeneratedDocForTest(t *testing.T, out string, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(out, name))
	if err != nil {
		t.Fatalf("read generated docs %s: %v", name, err)
	}
	return string(b)
}

// TestFlagIntentGeneratedReference keeps every renamed consumer and the action/code
// split aligned with the generated reference, without mistaking retained controls for aliases.
func TestFlagIntentGeneratedReference(t *testing.T) {
	for file, mappings := range map[string]map[string]string{
		"update_element":              {"vars": "set-vars", "vars-file": "set-vars-file"},
		"update_process-instance":     {"vars": "set-vars", "vars-file": "set-vars-file"},
		"update_user-task":            {"vars": "set-vars", "vars-file": "set-vars-file", "assign": "set-assignee"},
		"update_job":                  {"vars": "set-vars", "message": "set-error-message", "retry-backoff": "set-retry-backoff"},
		"run_process-instance":        {"vars": "set-vars", "bid": "set-bid"},
		"ops_repair_incident":         {"vars": "set-vars", "vars-file": "set-vars-file", "retries": "set-retries", "job-timeout": "set-job-timeout"},
		"ops_repair_process-instance": {"vars": "set-vars", "vars-file": "set-vars-file", "retries": "set-retries", "job-timeout": "set-job-timeout"},
		"get_process-definition":      {"stat": "with-stats"},
		"embed_list":                  {"details": "with-paths"},
		"get_incident":                {"with-no-error-message": "no-error-message"},
	} {
		data, err := os.ReadFile(filepath.Join("..", "docs", "cli", "c8volt_"+file+".md"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for old, canonical := range mappings {
			if !strings.Contains(text, "--"+canonical+" ") {
				t.Errorf("%s missing --%s", file, canonical)
			}
			if strings.Contains(text, "--"+old+" ") {
				t.Errorf("%s retains --%s", file, old)
			}
		}
		if file == "update_job" {
			for _, name := range []string{"set-retries", "set-timeout", "retries", "timeout", "throw-bpmn-error", "set-error-code"} {
				if !strings.Contains(text, "--"+name+" ") {
					t.Errorf("job reference missing --%s", name)
				}
			}
		}
	}
}

// generateReferenceForTest checks the real local generator rather than keeping
// copied help paragraphs that force every new behavior into operator examples.
func generateReferenceForTest(t *testing.T) string {
	t.Helper()
	out := t.TempDir()
	if err := doc.GenMarkdownTreeCustom(cmd.Root(), out, func(string) string { return "" }, docsReferenceLinkName); err != nil {
		t.Fatal(err)
	}
	return out
}
