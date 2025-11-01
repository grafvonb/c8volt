package cmd

import (
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var (
	version = "dev" // set by ldflags
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		if flagViewAsJson {
			out := map[string]string{
				"version":                  version,
				"commit":                   commit,
				"date":                     date,
				"supportedCamundaVersions": toolx.SupportedCamundaVersionsString(),
			}
			cmd.Println(toolx.ToJSONString(out))
			return
		}
		cmd.Printf("c8volt version %s, commit %s, built at %s. Supported Camunda versions: %s\n", version, commit, date, toolx.SupportedCamundaVersionsString())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
