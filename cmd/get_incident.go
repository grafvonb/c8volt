// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/spf13/cobra"
)

var (
	flagGetIncidentKeys               []string
	flagGetIncidentPIKeysOnly         bool
	flagGetIncidentMessageLimit       int
	flagGetIncidentNoErrorMessage     bool
	flagGetIncidentState              string
	flagGetIncidentErrorType          string
	flagGetIncidentErrorMessage       string
	flagGetIncidentPIKey              string
	flagGetIncidentRootKey            string
	flagGetIncidentPDKey              string
	flagGetIncidentBpmnProcessID      string
	flagGetIncidentElementID          string
	flagGetIncidentElementInstanceKey string
	flagGetIncidentCreationTimeAfter  string
	flagGetIncidentCreationTimeBefore string
	flagGetIncidentCreationTimeNewer  int
	flagGetIncidentCreationTimeOlder  int
	flagGetIncidentSize               int32
	flagGetIncidentLimit              int32
	flagGetIncidentTotal              bool
)

var getIncidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "List or fetch incidents",
	Long: "Get Camunda incidents by key or by search criteria.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. Each unique incident key is fetched once and rendered through the shared get output modes.\n\n" +
		"When no keys are supplied, incidents are searched by state, error type, error message, process context, element context, and creation time. Search mode defaults to active incidents and follows the shared get paging and limit conventions. --batch-size controls each backend page request, --limit caps total returned incidents across all pages, and --total returns only the exact matching count. Verbose paging progress is written away from stdout; JSON, keys-only, pi-keys-only, quiet, and automation output remain free of prompts and progress text.\n\n" +
		"When --bpmn-process-id is supplied in search mode, the BPMN process definition selector is validated before incident totals, keys-only output, process-instance-key output, or paging. Missing or invisible definitions fail explicitly; --json, --automation, --keys-only, --pi-keys-only, and non-TTY runs never prompt for recovery output.\n\n" +
		"Use --json for the stable incident payload, --keys-only for incident keys, --pi-keys-only for process instance keys, --error-message-limit to shorten long error messages, or --with-no-error-message to omit them.",
	Example: `  ./c8volt get incident --key <incident-key>
  ./c8volt get incident --key <incident-key> --key <another-incident-key>
  printf '%s\n' "$INCIDENT_KEY_A" "$INCIDENT_KEY_B" | ./c8volt get incident -
  ./c8volt get incident --state active --keys-only | ./c8volt get incident -
  ./c8volt get incident --state active --limit 5
  ./c8volt get incident --state resolved --error-type io_mapping_error --limit 5
  ./c8volt get incident --state active --error-type io_mapping_error --pi-keys-only
  ./c8volt get incident --state active --error-type io_mapping_error --pi-keys-only | ./c8volt cancel process-instance --dry-run -
  ./c8volt get incident --error-message "intentional" --limit 5
  ./c8volt get incident --creation-time-after 2026-05-01T00:00:00Z --creation-time-before 2026-05-31T00:00:00Z --limit 5
  ./c8volt get incident --pi-key <process-instance-key> --element-id <element-id>
  ./c8volt --json get incident --key <incident-key>
  ./c8volt --keys-only get incident --key <incident-key>`,
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

			log.Debug(fmt.Sprintf("getting incidents; keys [%s], mode %s", keys, pickMode()))
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
			if flagGetIncidentPIKeysOnly {
				if err := renderIncidentProcessInstanceKeys(cmd, incidents.Items); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render incident process instance keys: %w", err))
				}
				return
			}
			if err := listIncidentsView(cmd, incidents, flagGetIncidentMessageLimit, flagGetIncidentNoErrorMessage); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render incidents: %w", err))
			}
			return
		}

		filter := populateGetIncidentSearchFilter()
		if flagGetIncidentBpmnProcessID != "" {
			result, err := validateProcessDefinitionSelectorsForCommand(cmd.Context(), cmd, cli, newIncidentProcessDefinitionSelectorValidationRequest(), collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
			if !result.Valid() {
				handleProcessDefinitionSelectorValidationError(cmd, log, cfg.App.NoErrCodes, cli, result)
			}
		}
		log.Debug(fmt.Sprintf("searching incidents; mode %s", pickMode()))
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
	fs.BoolVar(&flagGetIncidentPIKeysOnly, "pi-keys-only", false, "return only process instance keys for matching incidents")
	fs.StringVarP(&flagGetIncidentState, "state", "s", "active", "incident state scope for search: active, pending, resolved, migrated, unknown, all")
	fs.StringVar(&flagGetIncidentErrorType, "error-type", "", "case-insensitive incident error type filter for search")
	fs.StringVar(&flagGetIncidentErrorMessage, "error-message", "", "case-insensitive incident error message substring filter for search")
	fs.StringVarP(&flagGetIncidentBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to validate and filter incidents")
	fs.StringVar(&flagGetIncidentPDKey, "pd-key", "", "process definition key to filter incidents")
	fs.StringVar(&flagGetIncidentPIKey, "pi-key", "", "process instance key to filter incidents")
	fs.StringVar(&flagGetIncidentRootKey, "root-key", "", "root process instance key to filter incidents")
	fs.StringVar(&flagGetIncidentElementID, "element-id", "", "BPMN element ID to filter incidents")
	fs.StringVar(&flagGetIncidentElementInstanceKey, "element-instance-key", "", "element instance key to filter incidents")
	fs.StringVar(&flagGetIncidentCreationTimeAfter, "creation-time-after", "", "only include incidents with creation time >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.StringVar(&flagGetIncidentCreationTimeBefore, "creation-time-before", "", "only include incidents with creation time <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.IntVar(&flagGetIncidentCreationTimeNewer, "creation-time-newer-days", -1, "only include incidents with creation time N days old or newer (0 means today)")
	fs.IntVar(&flagGetIncidentCreationTimeOlder, "creation-time-older-days", -1, "only include incidents with creation time N days old or older")
	fs.Int32VarP(&flagGetIncidentSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of incidents to request per page; does not cap total returned rows (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetIncidentLimit, "limit", "l", 0, "maximum number of matching incidents to return across all pages; omit to continue through all matches")
	fs.BoolVar(&flagGetIncidentTotal, "total", false, "return only the exact numeric total of matching incidents")
	fs.IntVar(&flagGetIncidentMessageLimit, "error-message-limit", 0, "maximum characters to show for incident messages; 0 keeps full messages")
	fs.BoolVar(&flagGetIncidentNoErrorMessage, "with-no-error-message", false, "omit error messages from incident output")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when fetching multiple incidents (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
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
	if err := validateIncidentCreationTimeFilters(
		"--creation-time-after", flagGetIncidentCreationTimeAfter,
		"--creation-time-before", flagGetIncidentCreationTimeBefore,
		"--creation-time-newer-days", flagGetIncidentCreationTimeNewer,
		"--creation-time-older-days", flagGetIncidentCreationTimeOlder,
	); err != nil {
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
	if flagGetIncidentPIKeysOnly {
		if flagGetIncidentTotal {
			return mutuallyExclusiveFlagsf("--pi-keys-only cannot be combined with --total")
		}
		switch pickMode() {
		case RenderModeJSON:
			return mutuallyExclusiveFlagsf("--pi-keys-only cannot be combined with --json")
		case RenderModeKeysOnly:
			return mutuallyExclusiveFlagsf("--pi-keys-only cannot be combined with --keys-only")
		}
	}
	if len(flagGetIncidentKeys) > 0 && hasGetIncidentSearchModeFlags(cmd) {
		return mutuallyExclusiveFlagsf("--key cannot be combined with search filters")
	}
	for flag, value := range map[string]string{
		"--pi-key":               flagGetIncidentPIKey,
		"--root-key":             flagGetIncidentRootKey,
		"--pd-key":               flagGetIncidentPDKey,
		"--element-instance-key": flagGetIncidentElementInstanceKey,
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
	if flagGetIncidentPIKeysOnly && cmd != nil && cmd.Flags().Changed("error-message-limit") {
		return mutuallyExclusiveFlagsf("--pi-keys-only cannot be combined with --error-message-limit")
	}
	if flagGetIncidentPIKeysOnly && flagGetIncidentNoErrorMessage {
		return mutuallyExclusiveFlagsf("--pi-keys-only cannot be combined with --with-no-error-message")
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
	if strings.TrimSpace(value) == "" {
		return invalidFlagValuef("invalid value for --state: %q, valid values are: %s", value, incidentfilter.ValidStatesString())
	}
	if _, ok := incidentfilter.NormalizeState(value); ok {
		return nil
	}
	return invalidFlagValuef("invalid value for --state: %q, valid values are: %s", value, incidentfilter.ValidStatesString())
}

func validateGetIncidentErrorTypeFlag(value string) error {
	if _, ok := incidentfilter.NormalizeErrorType(value); ok {
		return nil
	}
	return invalidFlagValuef("invalid value for --error-type: %q", value)
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
		"element-id",
		"element-instance-key",
		"creation-time-after",
		"creation-time-before",
		"creation-time-newer-days",
		"creation-time-older-days",
		"batch-size",
		"limit",
	} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

func populateGetIncidentSearchFilter() incident.Filter {
	errorType, _ := incidentfilter.NormalizeErrorType(flagGetIncidentErrorType)
	state, _ := incidentfilter.NormalizeState(flagGetIncidentState)
	creationTimeAfter, _ := pickIncidentCreationTimeLowerBound(flagGetIncidentCreationTimeAfter, flagGetIncidentCreationTimeNewer)
	creationTimeBefore, _ := pickIncidentCreationTimeUpperBound(flagGetIncidentCreationTimeBefore, flagGetIncidentCreationTimeOlder)
	return incident.Filter{
		State:                  state,
		ErrorType:              errorType,
		ErrorMessage:           flagGetIncidentErrorMessage,
		ProcessInstanceKey:     flagGetIncidentPIKey,
		RootProcessInstanceKey: flagGetIncidentRootKey,
		ProcessDefinitionKey:   flagGetIncidentPDKey,
		ProcessDefinitionId:    flagGetIncidentBpmnProcessID,
		ElementId:              flagGetIncidentElementID,
		ElementInstanceKey:     flagGetIncidentElementInstanceKey,
		CreationTimeAfter:      creationTimeAfter,
		CreationTimeBefore:     creationTimeBefore,
	}
}

func resetGetIncidentFlagState() {
	flagGetIncidentKeys = nil
	flagGetIncidentPIKeysOnly = false
	flagGetIncidentMessageLimit = 0
	flagGetIncidentNoErrorMessage = false
	flagGetIncidentState = "active"
	flagGetIncidentErrorType = ""
	flagGetIncidentErrorMessage = ""
	flagGetIncidentPIKey = ""
	flagGetIncidentRootKey = ""
	flagGetIncidentPDKey = ""
	flagGetIncidentBpmnProcessID = ""
	flagGetIncidentElementID = ""
	flagGetIncidentElementInstanceKey = ""
	flagGetIncidentCreationTimeAfter = ""
	flagGetIncidentCreationTimeBefore = ""
	flagGetIncidentCreationTimeNewer = -1
	flagGetIncidentCreationTimeOlder = -1
	flagGetIncidentSize = consts.MaxPISearchSize
	flagGetIncidentLimit = 0
	flagGetIncidentTotal = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
}
