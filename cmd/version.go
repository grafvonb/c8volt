// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"time"

	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var (
	version = "dev" // set by ldflags
	commit  = "none"
	date    = "unknown"
)

type BuildInfo struct {
	Version                  string
	Commit                   string
	Date                     string
	SupportedCamundaVersions string
	Camunda810Baseline       string
	Camunda810BaselineStatus string
}

func CurrentBuildInfo() BuildInfo {
	v810Baseline := toolx.V810Baseline()
	return BuildInfo{
		Version:                  version,
		Commit:                   commit,
		Date:                     date,
		SupportedCamundaVersions: toolx.SupportedCamundaVersionsString(),
		Camunda810Baseline:       v810Baseline.Tag,
		Camunda810BaselineStatus: v810Baseline.Status,
	}
}

func buildYear() int {
	if t, err := time.Parse(time.RFC3339, date); err == nil {
		return t.Year()
	}
	return time.Now().Year()
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: "Print version information.\n\n" +
		"Use --json for version metadata, including supported Camunda versions and the active Camunda 8.10 baseline.",
	Example: `  ./c8volt version
  ./c8volt version --json`,
	Run: func(cmd *cobra.Command, args []string) {
		info := CurrentBuildInfo()
		if flagViewAsJson {
			out := map[string]string{
				"version":                  info.Version,
				"commit":                   info.Commit,
				"date":                     info.Date,
				"supportedCamundaVersions": info.SupportedCamundaVersions,
				"camunda810Baseline":       info.Camunda810Baseline,
				"camunda810BaselineStatus": info.Camunda810BaselineStatus,
			}
			if err := renderJSONPayload(cmd, RenderModeJSON, out); err != nil {
				handleCommandError(cmd, nil, flagNoErrCodes, err)
			}
			return
		}
		renderHumanLine(cmd, "c8volt %s (%s, %s) | https://c8volt.info\nSupported Camunda versions: %s\nCamunda 8.10 baseline: %s (%s)\n(c) %d Adam Bogdan Boczek | https://boczek.info", info.Version, info.Commit, info.Date, info.SupportedCamundaVersions, info.Camunda810Baseline, info.Camunda810BaselineStatus, buildYear())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	setCommandMutation(versionCmd, CommandMutationReadOnly)
	setContractSupport(versionCmd, ContractSupportFull)
}
