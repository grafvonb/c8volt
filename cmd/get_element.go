// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

var (
	flagGetElementKey           string
	flagGetElementProcessKey    string
	flagGetElementID            string
	flagGetElementState         string
	flagGetElementType          string
	flagGetElementProcessDefKey string
	flagGetElementBpmnProcessID string
	flagGetElementBatchSize     int32
	flagGetElementLimit         int32
	flagGetElementTotal         bool
)

var getElementCmd = &cobra.Command{
	Use:   "element",
	Short: "Inspect runtime element instances",
	Long: "Inspect Camunda runtime element instances.\n\n" +
		"Use --key with an elementInstanceKey to inspect one runtime BPMN element execution record directly. Element lookup is supported for Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error.",
	Example: `  ./c8volt get element --key <element-instance-key>
  ./c8volt --json get element --key <element-instance-key>`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateGetElementFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if strings.TrimSpace(flagGetElementKey) == "" {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("element search is not implemented yet; use --key for direct lookup")))
			return
		}
		item, err := cli.GetElement(cmd.Context(), strings.TrimSpace(flagGetElementKey), collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get element: %w", err))
		}
		if err := elementView(cmd, item); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render element: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getElementCmd)

	fs := getElementCmd.Flags()
	fs.StringVar(&flagGetElementKey, "key", "", "element instance key for exact lookup")
	fs.StringVar(&flagGetElementProcessKey, "pi-key", "", "process instance key to filter in search mode")
	fs.StringVar(&flagGetElementID, "element-id", "", "BPMN element ID to filter in search mode")
	fs.StringVar(&flagGetElementState, "state", "", "runtime element state to filter in search mode; case-insensitive")
	fs.StringVar(&flagGetElementType, "type", "", "runtime element type to filter in search mode; case-insensitive")
	fs.StringVar(&flagGetElementProcessDefKey, "pd-key", "", "process definition key to filter in search mode")
	fs.StringVar(&flagGetElementBpmnProcessID, "bpmn-process-id", "", "BPMN process ID to filter in search mode")
	fs.Int32VarP(&flagGetElementBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of elements to fetch per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetElementLimit, "limit", "l", 0, "maximum number of elements to return in search mode")
	fs.BoolVar(&flagGetElementTotal, "total", false, "return only the numeric total of matching elements")

	useInvalidInputFlagErrors(getElementCmd)
	setCommandMutation(getElementCmd, CommandMutationReadOnly)
}

func validateGetElementFlags(cmd *cobra.Command) error {
	searchFlags := changedGetElementSearchFilterFlags(cmd)
	if strings.TrimSpace(flagGetElementKey) != "" && len(searchFlags) > 0 {
		return mutuallyExclusiveFlagsf("--key cannot be combined with element search filters: %s", strings.Join(searchFlags, ", "))
	}
	if err := validateGetElementFlagValues(cmd); err != nil {
		return err
	}
	return nil
}

func validateGetElementFlagValues(cmd *cobra.Command) error {
	if cmd == nil {
		return nil
	}
	if strings.TrimSpace(flagGetElementKey) != "" {
		if ok, firstBadKey, _ := validateKeys([]string{flagGetElementKey}); !ok {
			return invalidFlagValuef("--key value %q is not a valid key", firstBadKey)
		}
	}
	if flagGetElementProcessKey != "" {
		if ok, firstBadKey, _ := validateKeys([]string{flagGetElementProcessKey}); !ok {
			return invalidFlagValuef("--pi-key value %q is not a valid key", firstBadKey)
		}
	}
	if flagGetElementProcessDefKey != "" {
		if ok, firstBadKey, _ := validateKeys([]string{flagGetElementProcessDefKey}); !ok {
			return invalidFlagValuef("--pd-key value %q is not a valid key", firstBadKey)
		}
	}
	if flagGetElementBatchSize <= 0 || flagGetElementBatchSize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagGetElementBatchSize, consts.MaxPISearchSize)
	}
	if cmd.Flags().Changed("limit") && flagGetElementLimit <= 0 {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if flagGetElementTotal {
		switch pickMode() {
		case RenderModeJSON:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --json")
		case RenderModeKeysOnly:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --keys-only")
		}
	}
	return nil
}

func changedGetElementSearchFilterFlags(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}
	names := []string{
		"pi-key",
		"element-id",
		"state",
		"type",
		"pd-key",
		"bpmn-process-id",
	}
	changed := make([]string, 0, len(names))
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			changed = append(changed, "--"+name)
		}
	}
	return changed
}
