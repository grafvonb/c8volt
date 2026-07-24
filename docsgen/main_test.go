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

	if got := strings.Count(body, "\n| `"); got != 55 {
		t.Fatalf("expected assessment artifact to contain 55 command-node rows, got %d", got)
	}
}

// TestCLIDebtRefactorUserFacingDocsDocumentPagingContracts keeps README and
// docs examples aligned with generated command help for paging and automation.
func TestCLIDebtRefactorUserFacingDocsDocumentPagingContracts(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		fragments []string
	}{
		{
			name: "README",
			path: filepath.Join("..", "README.md"),
			fragments: []string{
				"Discovery pages through the full matching scope by default; `--batch-size` only tunes page size, while `--limit` is the explicit way to cap the frozen scope.",
				"Use `--batch-size` or `-n` to control how many process instances each backend page may fetch.",
				"Use `--limit` or `-l` to cap the total number of matched process instances returned or processed across all pages.",
				"machine-readable command contract is available from `c8volt capabilities --json`",
			},
		},
		{
			name: "use cases",
			path: filepath.Join("..", "docs", "use-cases.md"),
			fragments: []string{
				"Implemented ops workflows page discovery through the full matching scope by default.",
				"`--batch-size` controls discovery page size, `--limit` freezes a smaller scope, and `--automation` or `--auto-confirm` makes unattended execution explicit.",
			},
		},
		{
			name: "Camunda CLI options",
			path: filepath.Join("..", "docs", "camunda-cli.md"),
			fragments: []string{
				"paged discovery where `--batch-size` controls page size and `--limit` caps returned or frozen scope",
				"machine-readable capability discovery with `c8volt capabilities --json`",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read %s: %v", tt.path, err)
			}
			body := string(bodyBytes)
			for _, want := range tt.fragments {
				if !strings.Contains(body, want) {
					t.Fatalf("expected %s to contain %q", tt.path, want)
				}
			}
		})
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
		"Inspect cluster, process, job, element, incident, tenant, and resource state without changing it.",
		"./c8volt get incident --key <incident-key>",
		"./c8volt get incident --state active --error-type io_mapping_error --pi-keys-only",
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
		"./c8volt get incident --state active --error-type io_mapping_error --pi-keys-only",
		"./c8volt get incident --state active --error-type io_mapping_error --pi-keys-only | ./c8volt cancel pi --dry-run -",
		"--error-message string",
		"case-insensitive incident error message substring filter for search",
		"--pi-keys-only",
		"return only process instance keys for matching incidents",
		"--creation-time-after string",
		"only include incidents with creation time >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD",
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

// TestGeneratedGetElementDocsDocumentLookupSearchAndOutput protects generated docs for runtime element lookup and search.
func TestGeneratedGetElementDocsDocumentLookupSearchAndOutput(t *testing.T) {
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
		"Inspect cluster, process, job, element, incident, tenant, and resource state without changing it.",
		"./c8volt get ei --pi-key <process-instance-key> --limit 10",
		"[c8volt get element](c8volt_get_element)",
	} {
		if !strings.Contains(getDoc, want) {
			t.Fatalf("expected generated get docs to contain %q, got %q", want, getDoc)
		}
	}

	elementDoc := readGeneratedDocForTest(t, out, "c8volt_get_element.md")
	for _, want := range []string{
		"List or fetch runtime element instances",
		"List or fetch Camunda runtime element instances.",
		"Use --key when you know an element instance key.",
		"Search mode follows the shared get paging and limit conventions.",
		"Compact human rows include dur:<duration>",
		"Use --with-listeners to include runtime listener jobs under matching element rows.",
		"Use --json for the stable element payload and --keys-only when piping element instance keys.",
		"./c8volt get ei -k <element-instance-key>",
		"./c8volt get ei -k <element-instance-key> --with-listeners",
		"./c8volt get ei --pi-key <process-instance-key> --limit 10",
		"./c8volt get ei --pi-key <process-instance-key> --with-listeners",
		"./c8volt get element --pi-key <process-instance-key> --total",
		"./c8volt --json get element --key <element-instance-key> --with-listeners",
		"--element-id string",
		"BPMN element ID to filter in search mode",
		"--pi-key string",
		"process instance key to filter in search mode",
		"--batch-size int32",
		"number of elements to request per page; does not cap total returned rows",
		"--total",
		"return only the numeric total of matching elements",
		"--with-listeners",
		"include runtime listener jobs under matching element rows",
	} {
		if !strings.Contains(elementDoc, want) {
			t.Fatalf("expected generated get element docs to contain %q, got %q", want, elementDoc)
		}
	}
}

// TestGeneratedPagedWorkflowDocsDocumentContracts protects generated CLI docs
// for page-size, total-limit, frozen-scope, automation, and progress wording.
func TestGeneratedPagedWorkflowDocsDocumentContracts(t *testing.T) {
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

	tests := []struct {
		name      string
		file      string
		fragments []string
	}{
		{
			name: "get job",
			file: "c8volt_get_job.md",
			fragments: []string{
				"--batch-size controls each backend page request",
				"--limit caps total returned jobs across all pages",
				"JSON, keys-only, quiet, and automation output remain free of prompts and progress text",
				"number of jobs to request per page; does not cap total returned rows",
				"maximum number of matching jobs to return across all pages; omit to continue through all matches",
			},
		},
		{
			name: "get element",
			file: "c8volt_get_element.md",
			fragments: []string{
				"--batch-size controls each backend page request",
				"--limit caps returned element rows across all pages",
				"JSON, keys-only, quiet, and automation output remain free of prompts and progress text",
				"number of elements to request per page; does not cap total returned rows",
				"maximum number of matching elements to return across all pages; omit to continue through all matches",
			},
		},
		{
			name: "get incident",
			file: "c8volt_get_incident.md",
			fragments: []string{
				"--batch-size controls each backend page request",
				"--limit caps total returned incidents across all pages",
				"JSON, keys-only, pi-keys-only, quiet, and automation output remain free of prompts and progress text",
				"number of incidents to request per page; does not cap total returned rows",
				"maximum number of matching incidents to return across all pages; omit to continue through all matches",
			},
		},
		{
			name: "get process instance",
			file: "c8volt_get_process-instance.md",
			fragments: []string{
				"--batch-size controls each backend page request",
				"--limit caps total returned process instances across all pages",
				"JSON, keys-only, quiet, and automation output remain free of prompts and progress text",
				"number of process instances to request per page; does not cap total returned rows",
				"maximum number of matching process instances to return across all pages; omit to continue through all matches",
			},
		},
		{
			name: "cancel process instance",
			file: "c8volt_cancel_process-instance.md",
			fragments: []string{
				"--batch-size controls each discovery page request",
				"--limit caps the selected process-instance scope across all pages",
				"--workers, --fail-fast, and --no-worker-limit bound independent planning or cancellation work",
				"number of process instances to inspect per discovery page; does not cap total selected scope",
				"maximum number of matching process instances to select for cancellation across all pages; omit to continue through all matches",
			},
		},
		{
			name: "delete process instance",
			file: "c8volt_delete_process-instance.md",
			fragments: []string{
				"freezes every selected page-level delete plan before one confirmation and mutation",
				"--batch-size controls each discovery page request",
				"--limit caps the frozen delete scope across all pages",
				"--workers, --fail-fast, and --no-worker-limit bound independent planning, cancellation, or deletion work",
				"number of process instances to inspect per discovery page; does not cap total frozen scope",
				"maximum number of matching process instances to freeze for deletion across all pages; omit to continue through all matches",
			},
		},
		{
			name: "retention policy",
			file: "c8volt_ops_execute_retention-policy.md",
			fragments: []string{
				"Discovery pages through all matching retention candidates by default.",
				"--batch-size controls each discovery page request",
				"--limit caps the frozen retention scope",
				"--workers, --fail-fast, and --no-worker-limit bound independent delete planning or deletion work",
				"number of process instances to inspect per discovery page; does not cap total frozen scope",
				"maximum number of matching process instances to freeze for retention cleanup; omit to discover all matches",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readGeneratedDocForTest(t, out, tt.file)
			for _, want := range tt.fragments {
				if !strings.Contains(got, want) {
					t.Fatalf("expected generated docs to contain %q, got %q", want, got)
				}
			}
		})
	}
}

// TestGeneratedRunProcessInstanceDocsDocumentPipeline protects generated docs for keys-only run composition.
func TestGeneratedRunProcessInstanceDocsDocumentPipeline(t *testing.T) {
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

	runDoc := readGeneratedDocForTest(t, out, "c8volt_run.md")
	for _, want := range []string{
		"waits until created instances are observable",
		"./c8volt run pi -b <bpmn-process-id> --keys-only | ./c8volt expect pi --state completed -",
		"[c8volt run process-instance](c8volt_run_process-instance)",
	} {
		if !strings.Contains(runDoc, want) {
			t.Fatalf("expected generated run docs to contain %q, got %q", want, runDoc)
		}
	}

	processInstanceDoc := readGeneratedDocForTest(t, out, "c8volt_run_process-instance.md")
	for _, want := range []string{
		"Start process instances and confirm creation",
		"Created instances are confirmed after Camunda observes ACTIVE, COMPLETED, CANCELED, or TERMINATED.",
		"./c8volt run pi -b <bpmn-process-id> --keys-only | ./c8volt expect pi --state completed -",
		"./c8volt run pi -b <long-running-bpmn-process-id> --keys-only | ./c8volt expect pi --state active -",
	} {
		if !strings.Contains(processInstanceDoc, want) {
			t.Fatalf("expected generated run process-instance docs to contain %q, got %q", want, processInstanceDoc)
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
		"./c8volt resolve pi --key <process-instance-key> --key <another-process-instance-key>",
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
		"./c8volt ops execute retention-policy --retention-days 90 --state completed --bpmn-process-id <bpmn-process-id> --dry-run",
		"[c8volt ops execute](c8volt_ops_execute)",
	} {
		if !strings.Contains(retentionDoc, want) {
			t.Fatalf("expected generated ops execute retention-policy docs to contain %q, got %q", want, retentionDoc)
		}
	}

	repairDoc := readGeneratedDocForTest(t, out, "c8volt_ops_repair.md")
	for _, want := range []string{
		"Discover repair and remediation workflows",
		"lists target-specific remediation workflows",
		"./c8volt ops repair --help",
		"incident",
		"process-instance",
	} {
		if !strings.Contains(repairDoc, want) {
			t.Fatalf("expected generated ops repair docs to contain %q, got %q", want, repairDoc)
		}
	}
	repairIncidentDoc := readGeneratedDocForTest(t, out, "c8volt_ops_repair_incident.md")
	for _, want := range []string{
		"Repair incidents by key",
		"--key strings",
		"--element-id string",
		"--element-instance-key string",
		"--retries int32",
		"--job-timeout string",
		"[c8volt ops repair](c8volt_ops_repair)",
	} {
		if !strings.Contains(repairIncidentDoc, want) {
			t.Fatalf("expected generated ops repair incident docs to contain %q, got %q", want, repairIncidentDoc)
		}
	}
	for _, unwanted := range []string{"--flow-node-id", "--fni-key"} {
		if strings.Contains(repairIncidentDoc, unwanted) {
			t.Fatalf("expected generated ops repair incident docs to omit %q", unwanted)
		}
	}
	repairProcessInstanceDoc := readGeneratedDocForTest(t, out, "c8volt_ops_repair_process-instance.md")
	for _, want := range []string{
		"Repair incidents selected by process instances",
		"--key strings",
		"--direct-incidents-only",
		"[c8volt ops repair](c8volt_ops_repair)",
	} {
		if !strings.Contains(repairProcessInstanceDoc, want) {
			t.Fatalf("expected generated ops repair process-instance docs to contain %q, got %q", want, repairProcessInstanceDoc)
		}
	}
	if strings.Contains(repairProcessInstanceDoc, "--incidents-only") {
		t.Fatalf("expected generated ops repair process-instance docs not to contain --incidents-only, got %q", repairProcessInstanceDoc)
	}

	analyseDoc := readGeneratedDocForTest(t, out, "c8volt_ops_analyse_slow-process-instances.md")
	for _, want := range []string{
		"Analyze slow process-instance timings",
		"./c8volt ops analyse slow-process-instances --key 2251799813685249",
		"./c8volt ops analyze slow-process-instances --bpmn-process-id OrderProcess --state all --limit 20",
		"./c8volt ops analyse slow-process-instances --key 2251799813685249 --with-full-timeline",
		"./c8volt ops analyse slow-process-instances --key 2251799813685249 --with-listeners",
		"./c8volt ops analyse spi --bpmn-process-id OrderProcess --dur-longer 1h30m --dur-element-longer 30s",
		"./c8volt get pi --state active --keys-only | ./c8volt ops analyse slow-process-instances -",
		"Default output shows compact slowest element contributors",
		"Detail filters such as --element-id, --type, --element-state, and --dur-element-longer keep only process instances with matching element or transition detail rows",
		"Use --with-full-timeline to inspect complete chronological element and transition detail",
		"Use --with-listeners to include runtime listener jobs under matching element timeline rows",
		"Duration thresholds use Go duration syntax",
		"Calendar units such as 1d are not accepted",
		"--key strings",
		"--bpmn-process-id string",
		"--pd-key string",
		"--state string",
		"--no-incidents-only",
		"--batch-size int32",
		"--limit int32",
		"--element-id string",
		"--type string",
		"--element-state string",
		"--dur-longer string",
		"--dur-element-longer string",
		"--with-full-timeline",
		"--with-listeners",
		"show complete chronological element and transition detail",
		"include runtime listener jobs under matching element timeline rows",
	} {
		if !strings.Contains(analyseDoc, want) {
			t.Fatalf("expected generated ops analyse slow-process-instances docs to contain %q, got %q", want, analyseDoc)
		}
	}
	if strings.Contains(strings.ReplaceAll(analyseDoc, "--no-incidents-only", ""), "--incidents-only") {
		t.Fatalf("expected generated ops analyse slow-process-instances docs not to contain --incidents-only, got %q", analyseDoc)
	}
	if strings.Contains(analyseDoc, "--duration-after") {
		t.Fatalf("expected generated ops analyse slow-process-instances docs not to contain --duration-after, got %q", analyseDoc)
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
	if strings.Contains(repairDoc, "--key strings") {
		t.Fatalf("expected generated ops repair grouping docs to omit target flags")
	}
}

// TestGeneratedOpsPagedDiscoveryDocsDocumentHelp verifies generated ops docs preserve the complete-discovery flag contract.
func TestGeneratedOpsPagedDiscoveryDocsDocumentHelp(t *testing.T) {
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

	tests := []struct {
		name      string
		file      string
		fragments []string
	}{
		{
			name: "incident purge",
			file: "c8volt_ops_purge_process-instances-with-incidents.md",
			fragments: []string{
				"Discovery pages through all matching incidents by default.",
				"--batch-size tunes per-page discovery requests only",
				"--limit intentionally caps the frozen scope",
				"--element-id string",
				"--element-instance-key string",
				"number of incidents to inspect per discovery page; does not cap total frozen scope",
				"maximum number of matching incidents to freeze before candidate process-instance dedupe; omit to discover all matches",
			},
		},
		{
			name: "repair incident",
			file: "c8volt_ops_repair_incident.md",
			fragments: []string{
				"Search mode pages through all matching incidents by default.",
				"--batch-size tunes per-page discovery requests only",
				"--limit intentionally caps the frozen scope",
				"--element-id string",
				"--element-instance-key string",
				"number of incidents to inspect per discovery page; does not cap total frozen scope",
				"maximum number of matching incidents to freeze for repair; omit to discover all matches",
			},
		},
		{
			name: "repair process-instance",
			file: "c8volt_ops_repair_process-instance.md",
			fragments: []string{
				"Search mode pages through all matching incident-bearing process instances by default.",
				"--batch-size tunes per-page discovery requests only",
				"--limit intentionally caps the frozen scope",
				"number of process instances to inspect per discovery page; does not cap total frozen scope",
				"maximum number of matching process instances to freeze for repair; omit to discover all matches",
			},
		},
		{
			name: "all process definitions purge",
			file: "c8volt_ops_purge_all-process-definitions.md",
			fragments: []string{
				"Discovery pages through all matching process definitions by default.",
				"--batch-size tunes per-page discovery requests only",
				"--limit intentionally caps the frozen scope",
				"number of process definitions to inspect per discovery page; does not cap total frozen scope",
				"maximum number of matching process definitions to freeze for purge; omit to discover all matches",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readGeneratedDocForTest(t, out, tt.file)
			for _, want := range tt.fragments {
				if !strings.Contains(got, want) {
					t.Fatalf("expected generated docs to contain %q, got %q", want, got)
				}
			}
			for _, unwanted := range []string{"--flow-node-id", "--fni-key"} {
				if strings.Contains(got, unwanted) {
					t.Fatalf("expected generated docs to omit %q, got %q", unwanted, got)
				}
			}
		})
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

// TestGeneratedGetProcessInstanceDocsDocumentVariableSearch verifies generated
// CLI markdown carries the same variable-search contract as command help.
func TestGeneratedGetProcessInstanceDocsDocumentVariableSearch(t *testing.T) {
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

	piDoc := readGeneratedDocForTest(t, out, "c8volt_get_process-instance.md")
	for _, want := range []string{
		"Use variable-search flags to narrow list/search results natively on Camunda 8.8 and 8.9",
		"--var accepts name=value equality shorthand plus advanced name.$operator=value clauses",
		"--var-like uses native wildcard patterns",
		"Variable scopeKey means the scope where the variable is directly defined.",
		"./c8volt get pi --var-exists payload,email --limit 5",
		"./c8volt get pi --var 'status.$in=[\"approved\",\"pending\"]' --limit 5",
		"--var-exists stringArray",
		"--var-like stringArray",
		"Use --with-elements to include runtime element instances under matching process-instance rows.",
		"Nested human element rows include dur:<duration>",
		"Use --with-listeners with --with-elements to include runtime listener jobs under matching element rows.",
		"./c8volt get pi --key <process-instance-key> --with-elements",
		"./c8volt get pi --key <process-instance-key> --with-elements --with-listeners",
		"--with-elements",
		"include runtime element instances for keyed or list/search process-instance output",
		"--with-listeners",
		"include runtime listener jobs under matching element rows; requires --with-elements",
	} {
		if !strings.Contains(piDoc, want) {
			t.Fatalf("expected generated get process-instance docs to contain %q, got %q", want, piDoc)
		}
	}
}

func TestGeneratedWalkProcessInstanceDocsDocumentListeners(t *testing.T) {
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

	walkDoc := readGeneratedDocForTest(t, out, "c8volt_walk_process-instance.md")
	for _, want := range []string{
		"Inspect the parent/child tree of process instances.",
		"Add --with-incidents, --with-vars, and/or --with-elements",
		"Use --with-listeners with --with-elements to include runtime listener jobs under matching element rows.",
		"./c8volt walk pi --key <process-instance-key> --with-elements",
		"./c8volt walk pi --key <process-instance-key> --with-elements --with-listeners",
		"--with-elements",
		"show runtime element instances for keyed process-instance walks",
		"--with-listeners",
		"show runtime listener jobs under matching element rows; requires --with-elements",
	} {
		if !strings.Contains(walkDoc, want) {
			t.Fatalf("expected generated walk process-instance docs to contain %q, got %q", want, walkDoc)
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
