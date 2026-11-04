// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var (
	flagGetUserTaskKeys           []string
	flagGetUserTaskPIKey          string
	flagGetUserTaskPDKey          string
	flagGetUserTaskBpmnProcessID  string
	flagGetUserTaskElementID      string
	flagGetUserTaskState          string
	flagGetUserTaskAssignee       string
	flagGetUserTaskCandidateUser  string
	flagGetUserTaskCandidateGroup string
	flagGetUserTaskBatchSize      int32
	flagGetUserTaskLimit          int32
	flagGetUserTaskTotal          bool
	flagGetUserTaskWithVars       bool
	flagGetUserTaskVarValueLimit  int
)

var getUserTaskCmd = &cobra.Command{
	Use:   "user-task [-]",
	Short: "Fetch or search native user tasks",
	Long: `Inspect native user tasks by key or search by state, assignee, candidate group,
process instance, or variables. Search filters combine with AND.

Use --bid for one exact task business ID (only c8.10+). Variable filters match
local variables; --with-vars retrieves effective variables, including inherited
values. Use --limit to bound search results or --total to count matches.

Repeat --key or read newline-separated keys from stdin with '-'. Explicit keys
cannot be combined with search filters.`,
	Example: `  c8volt get user-task --key <user-task-key> --with-vars
  c8volt get user-task --state created --assignee alice --limit 10
  c8volt get user-task --candidate-group accounting --total`,
	Aliases: []string{"user-tasks", "ut", "uts"},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := validateOptionalDashArg(args); err != nil {
			return silenceUsageForError(cmd, err)
		}
		// BID validation runs with command context so its failures honor the shared envelope and configured logger.
		if searchBusinessIDFlag(cmd) != nil {
			return nil
		}
		return silenceUsageForError(cmd, validateGetUserTaskFlags(cmd))
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if searchBusinessIDFlag(cmd) != nil {
			if err := validateGetUserTaskFlags(cmd); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
		}

		stdinKeys, err := readUserTaskKeys(args)
		if err != nil {
			handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(cmd, flagGetUserTaskKeys, stdinKeys, log, cfg).Unique()
		if len(keys) == 0 {
			request, err := newGetUserTaskSearchRequest(cmd)
			if err != nil {
				handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
			}
			if flagGetUserTaskTotal {
				total, err := cli.SearchUserTasksTotal(cmd.Context(), request, collectOptions()...)
				if err != nil {
					handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get user tasks total: %w", err))
				}
				if err := userTaskTotalView(cmd, total); err != nil {
					handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render user tasks total: %w", err))
				}
				return
			}
			result, renderedIncrementally, err := searchUserTasksWithPaging(cmd, cli, request)
			if err != nil {
				handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get user tasks: %w", err))
			}
			if renderedIncrementally {
				return
			}
			if err := renderSelectedUserTasks(cmd, cli, result); err != nil {
				handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err, len(result.Items))
			}
			return
		}
		if hasGetUserTaskSearchFlags(cmd) {
			handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, mutuallyExclusiveFlagsf("--key cannot be combined with search filters, --limit, or --total"))
		}
		if ok, firstBadKey, _ := validateKeys(keys); !ok {
			handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("user task key %q is not a valid key", firstBadKey))
		}

		result, err := cli.GetUserTasks(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get user tasks: %w", err))
		}
		if err := renderSelectedUserTasks(cmd, cli, result); err != nil {
			handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
		}
	},
}

// init registers user-task discovery and display flags and declares command output and automation
// capabilities.
func init() {
	getCmd.AddCommand(getUserTaskCmd)

	flags := getUserTaskCmd.Flags()
	flags.String("bid", "", "exact stored task business ID (only c8.10+); commas, spaces and wildcards are literal")
	flags.StringSliceVarP(&flagGetUserTaskKeys, "key", "k", nil, "user task key(s) to fetch; repeat, comma-separate, or combine with stdin")
	flags.StringVar(&flagGetUserTaskPIKey, "pi-key", "", "process instance key to filter in search mode")
	flags.StringVar(&flagGetUserTaskPDKey, "pd-key", "", "process definition key to filter in search mode")
	flags.StringVarP(&flagGetUserTaskBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter in search mode")
	flags.StringVar(&flagGetUserTaskElementID, "element-id", "", "BPMN task element ID to filter in search mode")
	flags.StringVarP(&flagGetUserTaskState, "state", "s", "all", "user task state to filter in search mode; case-insensitive; all disables the predicate")
	flags.StringVar(&flagGetUserTaskAssignee, "assignee", "", "exact assignee to filter in search mode")
	flags.StringVar(&flagGetUserTaskCandidateUser, "candidate-user", "", "exact candidate-user membership to filter in search mode")
	flags.StringVar(&flagGetUserTaskCandidateGroup, "candidate-group", "", "exact candidate-group membership to filter in search mode")
	flags.Int32VarP(&flagGetUserTaskBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of user tasks to request per page; does not cap total results (maximum %d)", consts.MaxPISearchSize))
	flags.Int32VarP(&flagGetUserTaskLimit, "limit", "l", 0, "maximum number of matching user tasks to return across all pages; omit for unlimited")
	flags.BoolVar(&flagGetUserTaskTotal, "total", false, "return only the exact numeric total of matching user tasks")
	flags.BoolVar(&flagGetUserTaskWithVars, "with-vars", false, "include effective variables for selected user tasks")
	flags.IntVar(&flagGetUserTaskVarValueLimit, "var-value-limit", 0, "maximum characters to show for variable values when --with-vars is set; 0 disables truncation")
	flags.StringArrayVar(&flagGetUserTaskVarExists, "var-exists", nil, "require local variable name(s) to exist; repeat or separate names with commas")
	flags.StringArrayVar(&flagGetUserTaskVars, "var", nil, "require local variable equality or advanced clause(s); repeat or separate clauses with commas")
	flags.StringArrayVar(&flagGetUserTaskVarLikes, "var-like", nil, "require local variable value pattern clause(s); repeat or separate clauses with commas")
	flags.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when fetching multiple user tasks")
	flags.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued user task reads as workers when --workers is unset")
	flags.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new user task reads after the first error")

	useInvalidInputFlagErrors(getUserTaskCmd)
	setCommandMutation(getUserTaskCmd, CommandMutationReadOnly)
	setContractSupport(getUserTaskCmd, ContractSupportFull)
	setAutomationSupport(getUserTaskCmd, AutomationSupportFull, "supports shared machine output, stdin key pipelines, and unattended reads")
}

// validateGetUserTaskFlags rejects local key, bound, worker, and output
// conflicts before any native task request can be issued.
func validateGetUserTaskFlags(cmd *cobra.Command) error {
	if err := validateSearchBusinessIDFlag(cmd); err != nil {
		return err
	}
	if flagGetUserTaskBatchSize <= 0 || flagGetUserTaskBatchSize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagGetUserTaskBatchSize, consts.MaxPISearchSize)
	}
	if cmd.Flags().Changed("limit") && flagGetUserTaskLimit <= 0 {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return invalidFlagValuef("--workers must be positive integer")
	}
	if flagGetUserTaskVarValueLimit < 0 {
		return invalidFlagValuef("invalid value for --var-value-limit: %d, expected non-negative integer", flagGetUserTaskVarValueLimit)
	}
	if cmd.Flags().Changed("var-value-limit") && !flagGetUserTaskWithVars {
		return missingDependentFlagsf("--var-value-limit requires --with-vars")
	}
	for _, candidate := range []struct {
		flag   string
		values []string
	}{
		{flag: "key", values: flagGetUserTaskKeys},
		{flag: "pi-key", values: []string{flagGetUserTaskPIKey}},
		{flag: "pd-key", values: []string{flagGetUserTaskPDKey}},
	} {
		for _, value := range candidate.values {
			if value == "" {
				continue
			}
			if ok, bad, _ := validateKeys([]string{value}); !ok {
				return invalidFlagValuef("--%s value %q is not a valid key", candidate.flag, bad)
			}
		}
	}
	if flagGetUserTaskTotal {
		if cmd.Flags().Changed("limit") {
			return mutuallyExclusiveFlagsf("--total cannot be combined with --limit")
		}
		switch pickMode() {
		case RenderModeJSON:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --json")
		case RenderModeKeysOnly:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --keys-only")
		}
	}
	if !validUserTaskState(flagGetUserTaskState) {
		return invalidFlagValuef("invalid value for --state: %q, valid values are: all, %s", flagGetUserTaskState, strings.Join(validUserTaskStates, ", "))
	}
	if _, err := parseUserTaskVariableFilters(); err != nil {
		return err
	}
	if len(flagGetUserTaskKeys) > 0 && hasGetUserTaskSearchFlags(cmd) {
		return mutuallyExclusiveFlagsf("--key cannot be combined with search filters, --limit, or --total")
	}
	return nil
}

// newGetUserTaskSearchRequest maps validated command flags into an AND-combined
// native search request without changing case-sensitive identity predicates.
func newGetUserTaskSearchRequest(cmds ...*cobra.Command) (task.SearchRequest, error) {
	variableFilters, err := parseUserTaskVariableFilters()
	if err != nil {
		return task.SearchRequest{}, err
	}
	return task.SearchRequest{
		BusinessId:           searchBusinessIDFlag(cmds...),
		ProcessInstanceKey:   strings.TrimSpace(flagGetUserTaskPIKey),
		ProcessDefinitionKey: strings.TrimSpace(flagGetUserTaskPDKey),
		BpmnProcessId:        strings.TrimSpace(flagGetUserTaskBpmnProcessID),
		ElementId:            strings.TrimSpace(flagGetUserTaskElementID),
		State:                normalizedUserTaskState(flagGetUserTaskState),
		Assignee:             strings.TrimSpace(flagGetUserTaskAssignee),
		CandidateUser:        strings.TrimSpace(flagGetUserTaskCandidateUser),
		CandidateGroup:       strings.TrimSpace(flagGetUserTaskCandidateGroup),
		VariableFilters:      variableFilters,
		BatchSize:            flagGetUserTaskBatchSize,
		Limit:                flagGetUserTaskLimit,
	}, nil
}

var validUserTaskStates = []string{
	"ASSIGNING", "CANCELED", "CANCELING", "COMPLETED", "COMPLETING",
	"CREATED", "CREATING", "FAILED", "UPDATING",
}

// validUserTaskState accepts the unrestricted all sentinel or one supported state.
func validUserTaskState(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "all") || toolx.ValidEnumString(strings.TrimSpace(value), validUserTaskStates)
}

// normalizedUserTaskState converts supported names to backend form and omits all.
func normalizedUserTaskState(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "all") {
		return ""
	}
	canonical, _ := toolx.CanonicalEnumString(value, validUserTaskStates)
	return canonical
}

// hasGetUserTaskSearchFlags distinguishes explicitly supplied selectors from
// defaults, including an explicit --state all.
func hasGetUserTaskSearchFlags(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	for _, name := range []string{"bid", "pi-key", "pd-key", "bpmn-process-id", "element-id", "state", "assignee", "candidate-user", "candidate-group", "var-exists", "var", "var-like", "limit", "total"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}
