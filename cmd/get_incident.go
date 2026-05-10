// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/spf13/cobra"
)

var (
	flagGetIncidentKeys               []string
	flagGetIncidentMessageLimit       int
	flagGetIncidentNoErrorMessage     bool
	flagGetIncidentState              string
	flagGetIncidentErrorType          string
	flagGetIncidentErrorMessage       string
	flagGetIncidentPIKey              string
	flagGetIncidentRootKey            string
	flagGetIncidentPDKey              string
	flagGetIncidentBpmnProcessID      string
	flagGetIncidentFlowNodeID         string
	flagGetIncidentFNIKey             string
	flagGetIncidentCreationTimeAfter  string
	flagGetIncidentCreationTimeBefore string
	flagGetIncidentSize               int32
	flagGetIncidentLimit              int32
	flagGetIncidentTotal              bool
)

var getIncidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "List or fetch incidents",
	Long: "Get Camunda incidents by key or by search criteria.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. Each unique incident key is fetched once and rendered through the shared get output modes.\n\n" +
		"When no keys are supplied, incidents are searched by state, error type, error message, process context, flow-node context, and creation time. Search mode defaults to active incidents and follows the shared get paging and limit conventions.\n\n" +
		"Use --json for the stable incident payload, --error-message-limit to shorten long error messages, or --with-no-error-message to omit them.",
	Example: `  ./c8volt get incident --key 2251799813685249
  ./c8volt get inc --key 2251799813685249 --key 2251799813685250
  printf '%s\n' 2251799813685249 2251799813685250 | ./c8volt get incident -
  ./c8volt get pi --with-incidents --keys-only | ./c8volt get inc -
  ./c8volt get incident
  ./c8volt get incident --state resolved --error-type io_mapping_error
  ./c8volt get incident --error-message "no retries"
  ./c8volt get incident --creation-time-after 2026-05-08T00:00:00Z --creation-time-before 2026-05-09T00:00:00Z
  ./c8volt get incident --pi-key 2251799813685249 --flow-node-id task-a
  ./c8volt --json get incident --key 2251799813685249
  ./c8volt --keys-only get incident --key 2251799813685249`,
	Aliases: []string{"incidents", "inc"},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := validateOptionalDashArg(args); err != nil {
			return silenceUsageForError(cmd, err)
		}
		return silenceUsageForError(cmd, validateGetIncidentFlagValues(cmd))
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--workers must be positive integer"))
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagGetIncidentKeys, stdinKeys, log, cfg).Unique()
		keyedMode := len(flagGetIncidentKeys) > 0 || len(args) == 1 && args[0] == "-"
		if keyedMode {
			if len(keys) == 0 {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no incident keys provided or found to fetch")))
			}
			if hasGetIncidentSearchModeFlags(cmd) {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, mutuallyExclusiveFlagsf("--key cannot be combined with search filters"))
			}
			if ok, firstBadKey, _ := validateKeys(keys); !ok {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("incident key %q is not a valid key", firstBadKey))
			}

			log.Debug(fmt.Sprintf("fetching incidents for key(s) [%s], render mode: %s", keys, pickMode()))
			incidents, err := cli.GetIncidents(cmd.Context(), keys, flagWorkers, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get incidents: %w", err))
			}
			if flagGetIncidentTotal {
				if err := processInstanceTotalView(cmd, int64(len(incidents.Items))); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render incident total: %w", err))
				}
				return
			}
			if err := listIncidentsView(cmd, incidents, flagGetIncidentMessageLimit, flagGetIncidentNoErrorMessage); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render incidents: %w", err))
			}
			return
		}

		filter := populateGetIncidentSearchFilter()
		log.Debug(fmt.Sprintf("searching incidents, render mode: %s", pickMode()))
		if flagGetIncidentTotal {
			total, err := searchIncidentsTotal(cmd, cli, cfg, filter)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get incidents total: %w", err))
			}
			if err := processInstanceTotalView(cmd, total); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render incident total: %w", err))
			}
			return
		}
		incidents, renderedIncrementally, err := searchIncidentsWithPaging(cmd, cli, cfg, filter)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get incidents: %w", err))
		}
		if renderedIncrementally {
			return
		}
		if err := listIncidentsView(cmd, incidents, flagGetIncidentMessageLimit, flagGetIncidentNoErrorMessage); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render incidents: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getIncidentCmd)

	fs := getIncidentCmd.Flags()
	fs.StringSliceVarP(&flagGetIncidentKeys, "key", "k", nil, "incident key(s) to fetch; repeat or combine with stdin '-'")
	fs.StringVarP(&flagGetIncidentState, "state", "s", "active", "incident state scope for search: active, pending, resolved, migrated, unknown, all")
	fs.StringVar(&flagGetIncidentErrorType, "error-type", "", "case-insensitive incident error type filter for search")
	fs.StringVar(&flagGetIncidentErrorMessage, "error-message", "", "case-insensitive incident error message substring filter for search")
	fs.StringVarP(&flagGetIncidentBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter incidents")
	fs.StringVar(&flagGetIncidentPDKey, "pd-key", "", "process definition key to filter incidents")
	fs.StringVar(&flagGetIncidentPIKey, "pi-key", "", "process instance key to filter incidents")
	fs.StringVar(&flagGetIncidentRootKey, "root-key", "", "root process instance key to filter incidents")
	fs.StringVar(&flagGetIncidentFlowNodeID, "flow-node-id", "", "flow node ID to filter incidents")
	fs.StringVar(&flagGetIncidentFNIKey, "fni-key", "", "flow node instance key to filter incidents")
	fs.StringVar(&flagGetIncidentCreationTimeAfter, "creation-time-after", "", "only include incidents with creation time >= RFC3339 timestamp or YYYY-MM-DD")
	fs.StringVar(&flagGetIncidentCreationTimeBefore, "creation-time-before", "", "only include incidents with creation time <= RFC3339 timestamp or YYYY-MM-DD")
	fs.Int32VarP(&flagGetIncidentSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of incidents to fetch per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetIncidentLimit, "limit", "l", 0, "maximum number of matching incidents to return across all pages")
	fs.BoolVar(&flagGetIncidentTotal, "total", false, "return only the exact numeric total of matching incidents")
	fs.IntVar(&flagGetIncidentMessageLimit, "error-message-limit", 0, "maximum characters to show for incident messages; 0 keeps full messages")
	fs.BoolVar(&flagGetIncidentNoErrorMessage, "with-no-error-message", false, "omit error messages from incident output")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when fetching multiple incidents (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new incident lookups after the first error")

	useInvalidInputFlagErrors(getIncidentCmd)
	setCommandMutation(getIncidentCmd, CommandMutationReadOnly)
	setContractSupport(getIncidentCmd, ContractSupportFull)
	setAutomationSupport(getIncidentCmd, AutomationSupportFull, "supports shared machine output, stdin key pipelines, and unattended paging")
}

func validateGetIncidentFlagValues(cmd *cobra.Command) error {
	if flagGetIncidentSize <= 0 || flagGetIncidentSize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagGetIncidentSize, consts.MaxPISearchSize)
	}
	if flagGetIncidentLimit < 0 || (flagGetIncidentLimit == 0 && isGetIncidentLimitFlagChanged(cmd)) {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if err := validateGetIncidentStateFlag(flagGetIncidentState); err != nil {
		return err
	}
	if err := validateGetIncidentErrorTypeFlag(flagGetIncidentErrorType); err != nil {
		return err
	}
	if err := validateGetIncidentCreationTimeFlag("--creation-time-after", flagGetIncidentCreationTimeAfter); err != nil {
		return err
	}
	if err := validateGetIncidentCreationTimeFlag("--creation-time-before", flagGetIncidentCreationTimeBefore); err != nil {
		return err
	}
	if flagGetIncidentTotal {
		switch pickMode() {
		case RenderModeJSON:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --json")
		case RenderModeKeysOnly:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --keys-only")
		}
	}
	if len(flagGetIncidentKeys) > 0 && hasGetIncidentSearchModeFlags(cmd) {
		return mutuallyExclusiveFlagsf("--key cannot be combined with search filters")
	}
	for flag, value := range map[string]string{
		"--pi-key":   flagGetIncidentPIKey,
		"--root-key": flagGetIncidentRootKey,
		"--pd-key":   flagGetIncidentPDKey,
		"--fni-key":  flagGetIncidentFNIKey,
	} {
		if value == "" {
			continue
		}
		if ok, firstBadKey, _ := validateKeys([]string{value}); !ok {
			return invalidFlagValuef("%s value %q is not a valid key", flag, firstBadKey)
		}
	}
	if flagGetIncidentMessageLimit < 0 {
		return invalidFlagValuef("--error-message-limit must be non-negative")
	}
	if flagGetIncidentNoErrorMessage && cmd != nil && cmd.Flags().Changed("error-message-limit") {
		return mutuallyExclusiveFlagsf("--with-no-error-message cannot be combined with --error-message-limit")
	}
	if pickMode() == RenderModeJSON && cmd != nil && cmd.Flags().Changed("error-message-limit") {
		return mutuallyExclusiveFlagsf("--error-message-limit cannot be combined with --json")
	}
	if pickMode() == RenderModeKeysOnly && cmd != nil && cmd.Flags().Changed("error-message-limit") {
		return mutuallyExclusiveFlagsf("--error-message-limit cannot be combined with --keys-only")
	}
	if pickMode() == RenderModeJSON && flagGetIncidentNoErrorMessage {
		return mutuallyExclusiveFlagsf("--with-no-error-message cannot be combined with --json")
	}
	if pickMode() == RenderModeKeysOnly && flagGetIncidentNoErrorMessage {
		return mutuallyExclusiveFlagsf("--with-no-error-message cannot be combined with --keys-only")
	}
	if ok, firstBadKey, _ := validateKeys(flagGetIncidentKeys); !ok {
		return invalidFlagValuef("incident key %q is not a valid key", firstBadKey)
	}
	return nil
}

func validateGetIncidentStateFlag(value string) error {
	switch value {
	case "active", "pending", "resolved", "migrated", "unknown", "all":
		return nil
	default:
		return invalidFlagValuef("invalid value for --state: %q, valid values are: active, pending, resolved, migrated, unknown, all", value)
	}
}

func validateGetIncidentErrorTypeFlag(value string) error {
	if _, ok := incidentfilter.NormalizeErrorType(value); ok {
		return nil
	}
	return invalidFlagValuef("invalid value for --error-type: %q", value)
}

func validateGetIncidentCreationTimeFlag(name string, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return nil
	}
	if _, err := time.Parse(time.DateOnly, value); err == nil {
		return nil
	}
	return invalidFlagValuef("invalid value for %s: %q, expected RFC3339 timestamp or YYYY-MM-DD", name, value)
}

func isGetIncidentLimitFlagChanged(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Flags().Changed("limit")
}

func hasGetIncidentSearchModeFlags(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	for _, name := range []string{
		"state",
		"error-type",
		"error-message",
		"pi-key",
		"root-key",
		"pd-key",
		"bpmn-process-id",
		"flow-node-id",
		"fni-key",
		"creation-time-after",
		"creation-time-before",
		"batch-size",
		"limit",
	} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func populateGetIncidentSearchFilter() process.IncidentFilter {
	errorType, _ := incidentfilter.NormalizeErrorType(flagGetIncidentErrorType)
	return process.IncidentFilter{
		State:                  flagGetIncidentState,
		ErrorType:              errorType,
		ErrorMessage:           flagGetIncidentErrorMessage,
		ProcessInstanceKey:     flagGetIncidentPIKey,
		RootProcessInstanceKey: flagGetIncidentRootKey,
		ProcessDefinitionKey:   flagGetIncidentPDKey,
		ProcessDefinitionId:    flagGetIncidentBpmnProcessID,
		FlowNodeId:             flagGetIncidentFlowNodeID,
		FlowNodeInstanceKey:    flagGetIncidentFNIKey,
		CreationTimeAfter:      flagGetIncidentCreationTimeAfter,
		CreationTimeBefore:     flagGetIncidentCreationTimeBefore,
	}
}

func resetGetIncidentFlagState() {
	flagGetIncidentKeys = nil
	flagGetIncidentMessageLimit = 0
	flagGetIncidentNoErrorMessage = false
	flagGetIncidentState = "active"
	flagGetIncidentErrorType = ""
	flagGetIncidentErrorMessage = ""
	flagGetIncidentPIKey = ""
	flagGetIncidentRootKey = ""
	flagGetIncidentPDKey = ""
	flagGetIncidentBpmnProcessID = ""
	flagGetIncidentFlowNodeID = ""
	flagGetIncidentFNIKey = ""
	flagGetIncidentCreationTimeAfter = ""
	flagGetIncidentCreationTimeBefore = ""
	flagGetIncidentSize = consts.MaxPISearchSize
	flagGetIncidentLimit = 0
	flagGetIncidentTotal = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
}
