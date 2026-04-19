package cmd

import (
	"github.com/spf13/cobra"
)

var walkCmd = &cobra.Command{
	Use:   "walk",
	Short: "Inspect parent and child relationships for verification follow-up",
	Long: `Inspect parent and child relationships for verification follow-up.

Use this read-only command family when you need to understand how process instances are
related after a run, cancel, or delete operation. Child commands explain which traversal
shape they return, when tree rendering is available, and which output modes remain
human-first versus structured.`,
	Example: `  ./c8volt walk process-instance --help
  ./c8volt walk process-instance --key 2251799813711967 --family
  ./c8volt --json walk process-instance --key 2251799813711967 --children`,
	Aliases: []string{"w", "traverse"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"walkk", "travers"},
}

func init() {
	rootCmd.AddCommand(walkCmd)

	setCommandMutation(walkCmd, CommandMutationReadOnly)
}
