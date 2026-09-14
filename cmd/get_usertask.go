// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/domain"
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
)

var getUserTaskCmd = &cobra.Command{
	Use:   "user-task [-]",
	Short: "List or fetch user tasks",
	Long: `Get native Camunda user tasks by key or search criteria.

Provide repeated or comma-separated --key values, or newline-separated keys on stdin. A trailing '-' explicitly selects stdin; nonterminal stdin is also detected without it. Each unique key is fetched once in first-input order.

Keyed reads require Camunda 8.8 or newer and use backend authorization without discovery-tenant filtering. Search flags are reserved for paginated discovery and cannot be combined with keys.`,
	Example: `  ./c8volt get user-task --key <user-task-key>
  ./c8volt get ut -k <user-task-key>,<another-user-task-key>
  printf '%s\n' "$USER_TASK_KEY" | ./c8volt get user-tasks
  printf '%s\n' "$USER_TASK_KEY" | ./c8volt get uts -
  ./c8volt --json get user-task --key <user-task-key>
  ./c8volt --keys-only get user-task --key <user-task-key>`,
	Aliases: []string{"user-tasks", "ut", "uts"},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := validateOptionalDashArg(args); err != nil {
			return silenceUsageForError(cmd, err)
		}
		return silenceUsageForError(cmd, validateGetUserTaskFlags(cmd))
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		stdinKeys, err := readUserTaskKeys(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(cmd, flagGetUserTaskKeys, stdinKeys, log, cfg).Unique()
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("%w: user task search is not implemented", domain.ErrUnsupported))
		}
		if hasGetUserTaskSearchFlags(cmd) {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, mutuallyExclusiveFlagsf("--key cannot be combined with search filters, --limit, or --total"))
		}
		if ok, firstBadKey, _ := validateKeys(keys); !ok {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("user task key %q is not a valid key", firstBadKey))
		}

		result, err := cli.GetUserTasks(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get user tasks: %w", err))
		}
		if err := userTasksView(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render user tasks: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getUserTaskCmd)

	flags := getUserTaskCmd.Flags()
	flags.StringSliceVarP(&flagGetUserTaskKeys, "key", "k", nil, "user task key(s) to fetch; repeat, comma-separate, or combine with stdin")
	flags.StringVar(&flagGetUserTaskPIKey, "pi-key", "", "process instance key to filter user tasks")
	flags.StringVar(&flagGetUserTaskPDKey, "pd-key", "", "process definition key to filter user tasks")
	flags.StringVarP(&flagGetUserTaskBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter user tasks")
	flags.StringVar(&flagGetUserTaskElementID, "element-id", "", "BPMN task element ID to filter user tasks")
	flags.StringVarP(&flagGetUserTaskState, "state", "s", "all", "user task state to filter; case-insensitive, or all")
	flags.StringVar(&flagGetUserTaskAssignee, "assignee", "", "exact assignee to filter user tasks")
	flags.StringVar(&flagGetUserTaskCandidateUser, "candidate-user", "", "exact candidate user membership to filter user tasks")
	flags.StringVar(&flagGetUserTaskCandidateGroup, "candidate-group", "", "exact candidate group membership to filter user tasks")
	flags.Int32VarP(&flagGetUserTaskBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of user tasks to request per page (maximum %d)", consts.MaxPISearchSize))
	flags.Int32VarP(&flagGetUserTaskLimit, "limit", "l", 0, "maximum number of matching user tasks to return across all pages")
	flags.BoolVar(&flagGetUserTaskTotal, "total", false, "return only the exact numeric total of matching user tasks")
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
	if flagGetUserTaskBatchSize <= 0 || flagGetUserTaskBatchSize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagGetUserTaskBatchSize, consts.MaxPISearchSize)
	}
	if cmd.Flags().Changed("limit") && flagGetUserTaskLimit <= 0 {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return invalidFlagValuef("--workers must be positive integer")
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
	if len(flagGetUserTaskKeys) > 0 && hasGetUserTaskSearchFlags(cmd) {
		return mutuallyExclusiveFlagsf("--key cannot be combined with search filters, --limit, or --total")
	}
	return nil
}

// hasGetUserTaskSearchFlags distinguishes explicitly supplied selectors from
// defaults, including an explicit --state all.
func hasGetUserTaskSearchFlags(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	for _, name := range []string{"pi-key", "pd-key", "bpmn-process-id", "element-id", "state", "assignee", "candidate-user", "candidate-group", "limit", "total"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}
