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

func buildYear() int {
	if t, err := time.Parse(time.RFC3339, date); err == nil {
		return t.Year()
	}
	return time.Now().Year()
}

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
		cmd.Printf("c8volt %s (%s, %s) | camunda: %s | (c) %d Adam Bogdan Boczek | https://boczek.info\n", version, commit, date, toolx.SupportedCamundaVersionsString(), buildYear())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
