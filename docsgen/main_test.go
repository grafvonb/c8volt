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
		`CLI: [reference](./docs/cli/index.md)`,
		`Docs: [LICENSE](./LICENSE), [COPYRIGHT](./COPYRIGHT), [NOTICE.md](./NOTICE.md)`,
		`Project: [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](./SECURITY.md), [TRADEMARKS.md](TRADEMARKS.md), [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)`,
		`Lowercase target: [trademarks.md](trademarks.md)`,
	}, "\n")

	got := rewriteDocsIndexLinks(body)

	for _, want := range []string{
		`<img src="./logo/c8volt.png" />`,
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
