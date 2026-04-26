// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/grafvonb/c8volt/cmd"
	"github.com/spf13/cobra/doc"
)

// main generates CLI documentation in the specified format and output directory.
// https://cobra.dev/docs/how-to-guides/clis-for-llms/
func main() {
	out := flag.String("out", "./docs/cli", "output directory")
	format := flag.String("format", "markdown", "markdown|man|rest")
	front := flag.Bool("frontmatter", false, "prepend simple YAML front matter to markdown")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	root := cmd.Root()
	root.DisableAutoGenTag = true // stable, reproducible files (no timestamp footer)

	switch *format {
	case "markdown":
		if *front {
			prep := func(filename string) string {
				base := filepath.Base(filename)
				name := strings.TrimSuffix(base, filepath.Ext(base))
				title := strings.ReplaceAll(name, "_", " ")
				return fmt.Sprintf("---\ntitle: %q\nslug: %q\ndescription: \"CLI reference for %s\"\n---\n\n", title, name, title)
			}
			link := func(name string) string { return docsLinkName(name) }
			if err := doc.GenMarkdownTreeCustom(root, *out, prep, link); err != nil {
				log.Fatal(err)
			}
		} else {
			prep := func(filename string) string {
				base := filepath.Base(filename)
				name := strings.TrimSuffix(base, filepath.Ext(base))
				title := strings.ReplaceAll(name, "_", " ")
				return fmt.Sprintf("---\ntitle: %q\nnav_exclude: true\n---\n\n[CLI Reference]({{ \"/cli/\" | relative_url }})\n", title)
			}
			link := func(name string) string { return docsLinkName(name) }
			if err := doc.GenMarkdownTreeCustom(root, *out, prep, link); err != nil {
				log.Fatal(err)
			}
			if err := syncDocsIndexFromReadme("README.md", "./docs/index.md"); err != nil {
				log.Fatal(err)
			}
		}
	case "man":
		hdr := &doc.GenManHeader{Title: strings.ToUpper(root.Name()), Section: "1"}
		if err := doc.GenManTree(root, hdr, *out); err != nil {
			log.Fatal(err)
		}
	case "rest":
		if err := doc.GenReSTTree(root, *out); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown format: %s", *format)
	}
}

func syncDocsIndexFromReadme(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}

	body := string(b)
	body = rewriteDocsIndexLinks(body)

	const frontMatter = `---
title: "c8volt"
permalink: /
nav_order: 1
has_toc: true
---

`

	buildInfo := formatDocsBuildInfo(cmd.CurrentBuildInfo())

	if err := os.WriteFile(dst, []byte(frontMatter+buildInfo+body), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func rewriteDocsIndexLinks(body string) string {
	body = strings.ReplaceAll(body, "./docs/logo/", "./logo/")
	body = strings.ReplaceAll(body, "](./docs/cli/index.md)", "](./cli/)")

	body = rewriteGovernanceLinks(body)

	return body
}

var (
	governanceLinkTargetPattern = regexp.MustCompile(`\]\((\./)?([A-Za-z0-9_.-]+)\)`)
	governanceDocs              = map[string]string{
		"code_of_conduct.md": "CODE_OF_CONDUCT.md",
		"contributing.md":    "CONTRIBUTING.md",
		"copyright":          "COPYRIGHT",
		"license":            "LICENSE",
		"notice.md":          "NOTICE.md",
		"security.md":        "SECURITY.md",
		"trademarks.md":      "TRADEMARKS.md",
	}
)

func rewriteGovernanceLinks(body string) string {
	return governanceLinkTargetPattern.ReplaceAllStringFunc(body, func(linkTarget string) string {
		matches := governanceLinkTargetPattern.FindStringSubmatch(linkTarget)
		if len(matches) != 3 {
			return linkTarget
		}

		doc, ok := governanceDocs[strings.ToLower(matches[2])]
		if !ok {
			return linkTarget
		}

		return "](https://github.com/grafvonb/c8volt/blob/main/" + doc + ")"
	})
}

func docsLinkName(name string) string {
	lower := strings.ToLower(name)
	return strings.TrimSuffix(lower, filepath.Ext(lower))
}

func formatDocsBuildInfo(info cmd.BuildInfo) string {
	if isTaggedReleaseVersion(info.Version) {
		return fmt.Sprintf("> Generated from release `%s`, commit `%s`, built `%s` | Supported Camunda 8 versions: %s\n\n", info.Version, info.Commit, info.Date, info.SupportedCamundaVersions)
	}

	return fmt.Sprintf("> Generated from build `c8volt %s`, commit `%s`, built `%s` | Supported Camunda 8 versions: %s\n\n", info.Version, info.Commit, info.Date, info.SupportedCamundaVersions)
}

func isTaggedReleaseVersion(version string) bool {
	return strings.HasPrefix(version, "v") && !strings.Contains(version, "-") && version != "dev"
}
