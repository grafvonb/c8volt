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
}

func CurrentBuildInfo() BuildInfo {
	return BuildInfo{
		Version:                  version,
		Commit:                   commit,
		Date:                     date,
		SupportedCamundaVersions: toolx.SupportedCamundaVersionsString(),
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
		"Default output stays compact for human use. Use --json when automation needs the shared result envelope and version metadata fields.",
	Run: func(cmd *cobra.Command, args []string) {
		info := CurrentBuildInfo()
		if flagViewAsJson {
			out := map[string]string{
				"version":                  info.Version,
				"commit":                   info.Commit,
				"date":                     info.Date,
				"supportedCamundaVersions": info.SupportedCamundaVersions,
			}
			if err := renderJSONPayload(cmd, RenderModeJSON, out); err != nil {
				handleCommandError(cmd, nil, flagNoErrCodes, err)
			}
			return
		}
		cmd.Printf("c8volt %s (%s, %s) | camunda: %s | (c) %d Adam Bogdan Boczek | https://boczek.info\n", info.Version, info.Commit, info.Date, info.SupportedCamundaVersions, buildYear())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	setCommandMutation(versionCmd, CommandMutationReadOnly)
	setContractSupport(versionCmd, ContractSupportFull)
}
