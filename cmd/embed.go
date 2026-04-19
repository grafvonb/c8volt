package cmd

import (
	"github.com/spf13/cobra"
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Inspect, export, or deploy embedded BPMN resources",
	Long: `Inspect, export, or deploy embedded BPMN resources.

Use this command family when the workflow starts from BPMN assets already embedded in
the c8volt binary. Choose ` + "`embed list`" + ` to discover packaged resources,
` + "`embed export`" + ` to write them to disk, and ` + "`embed deploy`" + ` when you want
to deploy an embedded process definition to Camunda.`,
	Example: `  ./c8volt embed list
  ./c8volt embed export --name invoice.bpmn --output-dir ./tmp
  ./c8volt embed deploy --name invoice.bpmn`,
	Aliases: []string{"em", "emb"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"embedd", "embd", "embedded", "embeded"},
}

func init() {
	rootCmd.AddCommand(embedCmd)
}
