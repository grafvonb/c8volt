// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx"
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
	Short: "Inspect or search runtime element instances",
	Long: "Inspect or search Camunda runtime element instances.\n\n" +
		"Use --key with an elementInstanceKey to inspect one runtime BPMN element execution record directly. The `ei` alias follows the compact element-instance tag used in human output. Search mode uses filters such as --pi-key, --element-id, --state, --type, --pd-key, and --bpmn-process-id. Search mode pages through matching runtime elements by default with the standard paging controls. --batch-size tunes per-page discovery requests only, --limit intentionally caps total returned elements, and --total returns only the matching count. Use --json for the stable element payload. Element lookup and search are supported for Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error.",
	Example: `  ./c8volt get ei -k <element-instance-key>
  ./c8volt get ei --pi-key <process-instance-key> --limit 10
  ./c8volt get element --pi-key <process-instance-key> --total
  ./c8volt --json get ei --pi-key <process-instance-key> --limit 5
  ./c8volt --json get element --key <element-instance-key>`,
	Aliases: []string{"ei"},
	Args:    cobra.NoArgs,
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
		if strings.TrimSpace(flagGetElementKey) != "" {
			item, err := cli.GetElement(cmd.Context(), strings.TrimSpace(flagGetElementKey), collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get element: %w", err))
			}
			if err := elementView(cmd, item); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render element: %w", err))
			}
			return
		}
		searchRequest := newGetElementSearchRequest(cmd)
		if flagGetElementTotal {
			total, err := searchElementsTotal(cmd, cli, searchRequest)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get elements total: %w", err))
			}
			if err := processInstanceTotalView(cmd, total); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render elements total: %w", err))
			}
			return
		}
		result, renderedIncrementally, err := searchElementsWithPaging(cmd, cli, searchRequest)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get elements: %w", err))
		}
		if renderedIncrementally {
			return
		}
		if err := elementsView(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render elements: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getElementCmd)

	fs := getElementCmd.Flags()
	fs.StringVarP(&flagGetElementKey, "key", "k", "", "element instance key for exact lookup; omit to list or search runtime elements")
	fs.StringVar(&flagGetElementProcessKey, "pi-key", "", "process instance key to filter in search mode")
	fs.StringVar(&flagGetElementID, "element-id", "", "BPMN element ID to filter in search mode")
	fs.StringVarP(&flagGetElementState, "state", "s", "", "runtime element state to filter in search mode; case-insensitive")
	fs.StringVar(&flagGetElementType, "type", "", "runtime element type to filter in search mode; case-insensitive")
	fs.StringVar(&flagGetElementProcessDefKey, "pd-key", "", "process definition key to filter in search mode")
	fs.StringVarP(&flagGetElementBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter in search mode")
	fs.Int32VarP(&flagGetElementBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of elements to fetch per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetElementLimit, "limit", "l", 0, "maximum number of elements to return in search mode")
	fs.BoolVar(&flagGetElementTotal, "total", false, "return only the numeric total of matching elements")

	useInvalidInputFlagErrors(getElementCmd)
	setCommandMutation(getElementCmd, CommandMutationReadOnly)
	setContractSupport(getElementCmd, ContractSupportFull)
	setAutomationSupport(getElementCmd, AutomationSupportFull, "supports shared machine output and unattended element reads")
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
	if flagGetElementState != "" && !validElementState(flagGetElementState) {
		return invalidFlagValuef("invalid value for --state: %q, valid values are: %s", flagGetElementState, strings.Join(validElementStates, ", "))
	}
	if flagGetElementType != "" && !validElementType(flagGetElementType) {
		return invalidFlagValuef("invalid value for --type: %q, valid values are: %s", flagGetElementType, strings.Join(validElementTypes, ", "))
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

func newGetElementSearchRequest(cmd *cobra.Command) element.SearchRequest {
	_ = cmd
	return element.SearchRequest{
		ProcessInstanceKey:   strings.TrimSpace(flagGetElementProcessKey),
		ElementId:            strings.TrimSpace(flagGetElementID),
		State:                normalizedElementState(flagGetElementState),
		Type:                 normalizedElementType(flagGetElementType),
		ProcessDefinitionKey: strings.TrimSpace(flagGetElementProcessDefKey),
		BpmnProcessId:        strings.TrimSpace(flagGetElementBpmnProcessID),
		BatchSize:            flagGetElementBatchSize,
		Limit:                flagGetElementLimit,
	}
}

var validElementStates = []string{
	"ACTIVE",
	"COMPLETED",
	"TERMINATED",
}

var validElementTypes = []string{
	"AD_HOC_SUB_PROCESS",
	"AD_HOC_SUB_PROCESS_INNER_INSTANCE",
	"BOUNDARY_EVENT",
	"BUSINESS_RULE_TASK",
	"CALL_ACTIVITY",
	"END_EVENT",
	"EVENT_BASED_GATEWAY",
	"EVENT_SUB_PROCESS",
	"EXCLUSIVE_GATEWAY",
	"INCLUSIVE_GATEWAY",
	"INTERMEDIATE_CATCH_EVENT",
	"INTERMEDIATE_THROW_EVENT",
	"MANUAL_TASK",
	"MULTI_INSTANCE_BODY",
	"PARALLEL_GATEWAY",
	"PROCESS",
	"RECEIVE_TASK",
	"SCRIPT_TASK",
	"SEND_TASK",
	"SEQUENCE_FLOW",
	"SERVICE_TASK",
	"START_EVENT",
	"SUB_PROCESS",
	"TASK",
	"UNKNOWN",
	"UNSPECIFIED",
	"USER_TASK",
}

func validElementState(value string) bool {
	return toolx.ValidEnumString(value, validElementStates)
}

func validElementType(value string) bool {
	return toolx.ValidEnumString(value, validElementTypes)
}

func normalizedElementState(value string) string {
	return normalizedElementEnum(value, validElementStates)
}

func normalizedElementType(value string) string {
	return normalizedElementEnum(value, validElementTypes)
}

func normalizedElementEnum(value string, valid []string) string {
	canonical, _ := toolx.CanonicalEnumString(value, valid)
	return canonical
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
