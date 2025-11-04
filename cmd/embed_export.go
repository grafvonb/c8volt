package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/embedded"
	"github.com/spf13/cobra"
)

var (
	flagEmbedExportFileNames []string // may contain exact names or globs
	flagEmbedExportOut       string
	flagEmbedExportWithForce bool
	flagEmbedExportAll       bool
)

var embedExportCmd = &cobra.Command{
	Use:        "export",
	Short:      "Export embedded (virtual) resources to local files. Can be used to deploy updated versions of embedded resources using 'c8volt deploy'.",
	Aliases:    []string{"exp", "extract"},
	SuggestFor: []string{"exprot", "exrpot", "exract", "extrat"},
	Run: func(cmd *cobra.Command, args []string) {
		_, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		all, err := embedded.List()
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		// index for case-insensitive exact lookups: lower -> canonical
		index := make(map[string]string, len(all))
		for _, a := range all {
			index[strings.ToLower(a)] = a
		}
		var toExport []string
		switch {
		case flagEmbedExportAll:
			toExport = append(toExport, all...)
		case len(flagEmbedExportFileNames) == 0:
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("either --all or at least one --file is required"))
		default:
			seen := make(map[string]struct{})
			matchFound := false
			for _, f := range flagEmbedExportFileNames {
				if embedExportContainsGlob(f) {
					pat := strings.ToLower(f) // case-insensitive
					for _, cand := range all {
						ok, err := path.Match(pat, strings.ToLower(cand))
						if err != nil {
							ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("invalid pattern %q: %w", f, err))
						}
						if ok {
							if _, dup := seen[cand]; !dup {
								seen[cand] = struct{}{}
								toExport = append(toExport, cand)
								matchFound = true
							}
						}
					}
				} else {
					// exact, case-insensitive
					if canon, ok := index[strings.ToLower(f)]; ok {
						if _, dup := seen[canon]; !dup {
							seen[canon] = struct{}{}
							toExport = append(toExport, canon)
						}
					} else {
						ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("embedded file %q not found (case-insensitive); run 'embed list' to see all files", f))
					}
				}
			}
			if embedExportAllAreGlobs(flagEmbedExportFileNames) && !matchFound {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("pattern(s) %v matched no embedded files; have you forgotten to quote them in the shell, or provide a pattern for folder like '*/*.bpmn'?", flagEmbedExportFileNames))
			}
		}
		sort.Strings(toExport)
		outBase := flagEmbedExportOut
		if outBase == "" {
			outBase = "."
		}
		var exported int
		for _, vpath := range toExport {
			data, err := fs.ReadFile(embedded.FS, vpath)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("read embedded %q: %w", vpath, err))
			}
			dst := filepath.Clean(filepath.Join(outBase, vpath))
			dir := filepath.Dir(dst)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("create dir %q: %w", dir, err))
			}
			if !flagEmbedExportWithForce {
				if info, err := os.Stat(dst); err == nil && info.Mode().IsRegular() {
					ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("destination file %q exists; use --force to overwrite", dst))
				}
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("write %q: %w", dst, err))
			}
			log.Debug(fmt.Sprintf("exported %q > %q", vpath, dst))
			exported++
		}
		log.Info(fmt.Sprintf("exported %d embedded resource(s) to %q", exported, filepath.Clean(outBase)))
	},
}

func init() {
	embedCmd.AddCommand(embedExportCmd)
	embedExportCmd.Flags().StringSliceVarP(&flagEmbedExportFileNames, "file", "f", nil, "embedded file(s) or a glob pattern to export (repeatable, quote patterns in the shell like zsh)")
	embedExportCmd.Flags().StringVarP(&flagEmbedExportOut, "out", "o", ".", "output base directory")
	embedExportCmd.Flags().BoolVar(&flagEmbedExportWithForce, "force", false, "overwrite if destination file exists")
	embedExportCmd.Flags().BoolVar(&flagEmbedExportAll, "all", false, "export all embedded files")
}

func embedExportContainsGlob(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func embedExportAllAreGlobs(items []string) bool {
	if len(items) == 0 {
		return false
	}
	for _, it := range items {
		if !embedExportContainsGlob(it) {
			return false
		}
	}
	return true
}
