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

func TestFormatDocsBuildInfoRelease(t *testing.T) {
	info := cmd.BuildInfo{
		Version:                  "v2.1.0",
		Commit:                   "abcdef123456",
		Date:                     "2026-04-11T09:10:11Z",
		SupportedCamundaVersions: "8.7, 8.8",
	}

	got := formatDocsBuildInfo(info)

	for _, want := range []string{
		"Generated from release `v2.1.0`",
		"commit `abcdef123456`",
		"built `2026-04-11T09:10:11Z`",
		"Supported Camunda 8 versions: 8.7, 8.8",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected build info to contain %q, got %q", want, got)
		}
	}
}

func TestFormatDocsBuildInfoNonRelease(t *testing.T) {
	info := cmd.BuildInfo{
		Version:                  "v2.1.0-8-gabcdef123456-dirty",
		Commit:                   "abcdef123456",
		Date:                     "2026-04-11T09:10:11Z",
		SupportedCamundaVersions: "8.7, 8.8",
	}

	got := formatDocsBuildInfo(info)

	for _, want := range []string{
		"Generated from build `c8volt v2.1.0-8-gabcdef123456-dirty`",
		"commit `abcdef123456`",
		"built `2026-04-11T09:10:11Z`",
		"Supported Camunda 8 versions: 8.7, 8.8",
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

// TestGeneratedProcessInstanceDocsDocumentHasUserTasksLookup protects generated command docs for the task-key lookup surface.
func TestGeneratedProcessInstanceDocsDocumentHasUserTasksLookup(t *testing.T) {
	out := t.TempDir()
	root := cmd.Root()
	root.DisableAutoGenTag = true

	prep := func(filename string) string {
		base := filepath.Base(filename)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		title := strings.ReplaceAll(name, "_", " ")
		return "---\ntitle: \"" + title + "\"\nnav_exclude: true\n---\n\n"
	}
	link := func(name string) string { return docsLinkName(name) }
	if err := doc.GenMarkdownTreeCustom(root, out, prep, link); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(out, "c8volt_get_process-instance.md"))
	if err != nil {
		t.Fatalf("read generated process-instance docs: %v", err)
	}
	got := string(b)

	for _, want := range []string{
		"--has-user-tasks strings",
		"user task key(s) whose owning process instances should be fetched",
		"./c8volt get pi --has-user-tasks \u003cuser-task-key\u003e",
		"Use --has-user-tasks to fetch process instances by their owning user-task keys.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected generated docs to contain %q, got %q", want, got)
		}
	}

	for _, obsolete := range []string{
		"Camunda v2 user-task search first",
		"Tasklist V1 lookup for legacy user-task compatibility",
		"Camunda 8.7 remains unsupported",
		"There is no Tasklist or Operate fallback",
	} {
		if strings.Contains(got, obsolete) {
			t.Fatalf("expected generated docs to omit %q, got %q", obsolete, got)
		}
	}
}

// TestGeneratedGetIncidentDocsDocumentLookupSearchAndOutput protects generated docs for direct incident lookup and search.
func TestGeneratedGetIncidentDocsDocumentLookupSearchAndOutput(t *testing.T) {
	out := t.TempDir()
	root := cmd.Root()
	root.DisableAutoGenTag = true

	prep := func(filename string) string {
		base := filepath.Base(filename)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		title := strings.ReplaceAll(name, "_", " ")
		return "---\ntitle: \"" + title + "\"\nnav_exclude: true\n---\n\n"
	}
	link := func(name string) string { return docsLinkName(name) }
	if err := doc.GenMarkdownTreeCustom(root, out, prep, link); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	getDoc := readGeneratedDocForTest(t, out, "c8volt_get.md")
	for _, want := range []string{
		"Inspect cluster, process, job, incident, tenant, and resource state without changing it.",
		"./c8volt get incident --key <incident-key>",
		"./c8volt get incident --state active --error-type job_no_retries --pi-keys-only",
		"[c8volt get incident](c8volt_get_incident)",
	} {
		if !strings.Contains(getDoc, want) {
			t.Fatalf("expected generated get docs to contain %q, got %q", want, getDoc)
		}
	}

	incidentDoc := readGeneratedDocForTest(t, out, "c8volt_get_incident.md")
	for _, want := range []string{
		"List or fetch incidents",
		"Get Camunda incidents by key or by search criteria.",
		"./c8volt get inc --key <incident-key> --key <another-incident-key>",
		"./c8volt get incident --state resolved --error-type io_mapping_error",
		"./c8volt get incident --state active --error-type job_no_retries --pi-keys-only",
		"./c8volt get incident --state active --error-type job_no_retries --pi-keys-only | ./c8volt cancel pi --dry-run -",
		"--error-message string",
		"case-insensitive incident error message substring filter for search",
		"--pi-keys-only",
		"return only process instance keys for matching incidents",
		"--creation-time-after string",
		"only include incidents with creation time >= RFC3339 timestamp or YYYY-MM-DD",
		"--total",
		"return only the exact numeric total of matching incidents",
	} {
		if !strings.Contains(incidentDoc, want) {
			t.Fatalf("expected generated get incident docs to contain %q, got %q", want, incidentDoc)
		}
	}

	for _, unwanted := range []string{
		"\n      --with-incidents",
		"\n      --incidents-only",
		"\n      --direct-incidents-only",
		"\n      --no-incidents-only",
	} {
		if strings.Contains(incidentDoc, unwanted) {
			t.Fatalf("expected generated get incident docs to omit %q, got %q", unwanted, incidentDoc)
		}
	}
}

// TestGeneratedResolveDocsDocumentResolveWorkflows protects generated docs for the incident recovery command family.
func TestGeneratedResolveDocsDocumentResolveWorkflows(t *testing.T) {
	out := t.TempDir()
	root := cmd.Root()
	root.DisableAutoGenTag = true

	prep := func(filename string) string {
		base := filepath.Base(filename)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		title := strings.ReplaceAll(name, "_", " ")
		return "---\ntitle: \"" + title + "\"\nnav_exclude: true\n---\n\n"
	}
	link := func(name string) string { return docsLinkName(name) }
	if err := doc.GenMarkdownTreeCustom(root, out, prep, link); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	resolveDoc := readGeneratedDocForTest(t, out, "c8volt_resolve.md")
	for _, want := range []string{
		"Resolve operational incidents.",
		"./c8volt resolve incident --key <incident-key>",
		"[c8volt resolve incident](c8volt_resolve_incident)",
		"[c8volt resolve process-instance](c8volt_resolve_process-instance)",
	} {
		if !strings.Contains(resolveDoc, want) {
			t.Fatalf("expected generated resolve docs to contain %q, got %q", want, resolveDoc)
		}
	}

	incidentDoc := readGeneratedDocForTest(t, out, "c8volt_resolve_incident.md")
	for _, want := range []string{
		"Resolve incidents by key.",
		"Each unique incident key is submitted for resolution and reported independently.",
		"./c8volt resolve inc --key <incident-key> --key <another-incident-key>",
		"--dry-run",
		"preview incident resolutions without submitting mutation",
		"--no-wait",
		"return after the resolution request is accepted without incident confirmation",
	} {
		if !strings.Contains(incidentDoc, want) {
			t.Fatalf("expected generated resolve incident docs to contain %q, got %q", want, incidentDoc)
		}
	}

	processInstanceDoc := readGeneratedDocForTest(t, out, "c8volt_resolve_process-instance.md")
	for _, want := range []string{
		"Resolve process-instance incidents by key.",
		"discovers active incidents at command start",
		"./c8volt resolve pi --key 2251799813685250 --key 2251799813685260",
		"--dry-run",
		"preview process-instance incident resolutions without submitting mutation",
		"--no-wait",
		"return after resolution requests are accepted without incident confirmation",
	} {
		if !strings.Contains(processInstanceDoc, want) {
			t.Fatalf("expected generated resolve process-instance docs to contain %q, got %q", want, processInstanceDoc)
		}
	}
}

// TestGeneratedOpsDocsDocumentGroupingCommands protects generated docs for the ops command foundation.
func TestGeneratedOpsDocsDocumentGroupingCommands(t *testing.T) {
	out := t.TempDir()
	root := cmd.Root()
	root.DisableAutoGenTag = true

	prep := func(filename string) string {
		base := filepath.Base(filename)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		title := strings.ReplaceAll(name, "_", " ")
		return "---\ntitle: \"" + title + "\"\nnav_exclude: true\n---\n\n"
	}
	link := func(name string) string { return docsLinkName(name) }
	if err := doc.GenMarkdownTreeCustom(root, out, prep, link); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	opsDoc := readGeneratedDocForTest(t, out, "c8volt_ops.md")
	for _, want := range []string{
		"Discover high-level operational workflows",
		"./c8volt ops --help",
		"[c8volt ops execute](c8volt_ops_execute)",
		"[c8volt ops repair](c8volt_ops_repair)",
	} {
		if !strings.Contains(opsDoc, want) {
			t.Fatalf("expected generated ops docs to contain %q, got %q", want, opsDoc)
		}
	}

	executeDoc := readGeneratedDocForTest(t, out, "c8volt_ops_execute.md")
	for _, want := range []string{
		"Discover predefined operational playbooks",
		"lists playbooks that discover target sets",
		"existing c8volt resource actions",
		"./c8volt ops execute --help",
		"./c8volt ops execute retention-policy --retention-days 90 --dry-run",
		"[c8volt ops execute retention-policy](c8volt_ops_execute_retention-policy)",
	} {
		if !strings.Contains(executeDoc, want) {
			t.Fatalf("expected generated ops execute docs to contain %q, got %q", want, executeDoc)
		}
	}

	retentionDoc := readGeneratedDocForTest(t, out, "c8volt_ops_execute_retention-policy.md")
	for _, want := range []string{
		"Execute process-instance retention cleanup",
		"--retention-days int",
		"--report-file string",
		"./c8volt ops execute retention-policy --retention-days 90 --state completed --bpmn-process-id order-process --dry-run",
		"[c8volt ops execute](c8volt_ops_execute)",
	} {
		if !strings.Contains(retentionDoc, want) {
			t.Fatalf("expected generated ops execute retention-policy docs to contain %q, got %q", want, retentionDoc)
		}
	}

	repairDoc := readGeneratedDocForTest(t, out, "c8volt_ops_repair.md")
	for _, want := range []string{
		"Discover repair and remediation workflows",
		"Target-specific subcommands will define their own target semantics",
		"./c8volt ops repair --help",
		"incident",
	} {
		if !strings.Contains(repairDoc, want) {
			t.Fatalf("expected generated ops repair docs to contain %q, got %q", want, repairDoc)
		}
	}
	repairIncidentDoc := readGeneratedDocForTest(t, out, "c8volt_ops_repair_incident.md")
	for _, want := range []string{
		"Repair incidents by key",
		"--key strings",
		"--retries int32",
		"--job-timeout string",
		"[c8volt ops repair](c8volt_ops_repair)",
	} {
		if !strings.Contains(repairIncidentDoc, want) {
			t.Fatalf("expected generated ops repair incident docs to contain %q, got %q", want, repairIncidentDoc)
		}
	}

	for _, unwanted := range []string{
		"orphan-cleanup",
		"smoke-test",
		"--key string",
		"--key strings",
		"repair process-instance",
	} {
		if strings.Contains(opsDoc, unwanted) {
			t.Fatalf("expected generated ops docs to omit %q", unwanted)
		}
	}
	if strings.Contains(repairDoc, "--key strings") || strings.Contains(repairDoc, "repair process-instance") {
		t.Fatalf("expected generated ops repair grouping docs to omit target flags and unavailable process-instance repair")
	}
}

// TestGeneratedConfigDocsDocumentSplitDiagnostics protects generated command docs for config diagnostics.
func TestGeneratedConfigDocsDocumentSplitDiagnostics(t *testing.T) {
	out := t.TempDir()
	root := cmd.Root()
	root.DisableAutoGenTag = true

	prep := func(filename string) string {
		base := filepath.Base(filename)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		title := strings.ReplaceAll(name, "_", " ")
		return "---\ntitle: \"" + title + "\"\nnav_exclude: true\n---\n\n"
	}
	link := func(name string) string { return docsLinkName(name) }
	if err := doc.GenMarkdownTreeCustom(root, out, prep, link); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	configDoc := readGeneratedDocForTest(t, out, "c8volt_config.md")
	for _, want := range []string{
		"`config validate`",
		"`config template`",
		"`config test-connection`",
		"./c8volt --config ./config.yaml config validate",
		"./c8volt config template",
		"./c8volt --config ./config.yaml config test-connection",
		"[c8volt config show](c8volt_config_show)",
		"[c8volt config validate](c8volt_config_validate)",
		"[c8volt config template](c8volt_config_template)",
		"[c8volt config test-connection](c8volt_config_test-connection)",
	} {
		if !strings.Contains(configDoc, want) {
			t.Fatalf("expected generated config docs to contain %q, got %q", want, configDoc)
		}
	}

	showDoc := readGeneratedDocForTest(t, out, "c8volt_config_show.md")
	for _, want := range []string{
		"compatibility shortcuts",
		"--validate",
		"compatibility shortcut: validate the effective configuration",
		"--template",
		"compatibility shortcut: print a blank configuration template",
	} {
		if !strings.Contains(showDoc, want) {
			t.Fatalf("expected generated config show docs to contain %q, got %q", want, showDoc)
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
