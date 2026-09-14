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
)

var getUserTaskCmd = &cobra.Command{
	Use:   "user-task [-]",
	Short: "Fetch or search native user tasks",
	Long: `Get native Camunda user tasks by key or search criteria.

Provide repeated or comma-separated --key values, or newline-separated keys on stdin. A trailing '-' explicitly selects stdin; nonterminal stdin is also detected without it. Each unique key is fetched once in first-input order.

Every requested key must resolve or the command fails without a partial result. Keyed reads require Camunda 8.8 or newer and use backend authorization without discovery-tenant filtering, so --tenant does not hide an authorized task and the task's actual tenant is returned.

Without keys, search by process, element, state, assignment, candidate, and effective tenant scope. Predicates are combined with AND. Supported states are ASSIGNING, CANCELED, CANCELING, COMPLETED, COMPLETING, CREATED, CREATING, FAILED, and UPDATING; state matching is case-insensitive, and all applies no state predicate. --batch-size controls each discovery request, --limit caps returned tasks across all pages, and --total emits the exact matching count.

Interactive searches offer another page separately from command results when more matches remain. Use --auto-confirm or --automation for unattended paging. --quiet suppresses human results but preserves explicitly requested JSON, keys-only, and numeric total output.

Human rows show task key, tenant, element ID, and state, followed by optional name: and assignee: details, then BPMN process ID and related pi:, ei:, and pd: keys. Empty optional fields are omitted.

Use --json for one collection envelope or --keys-only for one task key per line. Keys cannot be combined with search filters, --limit, or --total; --total also conflicts with --limit, --json, and --keys-only. Search and keyed reads require Camunda 8.8, 8.9, or 8.10; Camunda 8.7 is unsupported. Task mutations, variables, forms, audit history, date filters, custom sorting, and watch mode are not provided by this command.`,
	Example: `  ./c8volt get user-task --key <user-task-key>
  ./c8volt get ut -k <user-task-key>,<another-user-task-key>
  ./c8volt get user-task --state created --assignee alice --limit 25
  ./c8volt get user-task --candidate-group accounting --total
  ./c8volt --automation --keys-only get user-task --batch-size 100
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
			request := newGetUserTaskSearchRequest()
			if flagGetUserTaskTotal {
				total, err := cli.SearchUserTasksTotal(cmd.Context(), request, collectOptions()...)
				if err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get user tasks total: %w", err))
				}
				if err := userTaskTotalView(cmd, total); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render user tasks total: %w", err))
				}
				return
			}
			result, renderedIncrementally, err := searchUserTasksWithPaging(cmd, cli, request)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get user tasks: %w", err))
			}
			if renderedIncrementally {
				return
			}
			if err := userTasksView(cmd, result); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render user tasks: %w", err))
			}
			return
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
	if !validUserTaskState(flagGetUserTaskState) {
		return invalidFlagValuef("invalid value for --state: %q, valid values are: all, %s", flagGetUserTaskState, strings.Join(validUserTaskStates, ", "))
	}
	if len(flagGetUserTaskKeys) > 0 && hasGetUserTaskSearchFlags(cmd) {
		return mutuallyExclusiveFlagsf("--key cannot be combined with search filters, --limit, or --total")
	}
	return nil
}

// newGetUserTaskSearchRequest maps validated command flags into an AND-combined
// native search request without changing case-sensitive identity predicates.
func newGetUserTaskSearchRequest() task.SearchRequest {
	return task.SearchRequest{
		ProcessInstanceKey:   strings.TrimSpace(flagGetUserTaskPIKey),
		ProcessDefinitionKey: strings.TrimSpace(flagGetUserTaskPDKey),
		BpmnProcessId:        strings.TrimSpace(flagGetUserTaskBpmnProcessID),
		ElementId:            strings.TrimSpace(flagGetUserTaskElementID),
		State:                normalizedUserTaskState(flagGetUserTaskState),
		Assignee:             strings.TrimSpace(flagGetUserTaskAssignee),
		CandidateUser:        strings.TrimSpace(flagGetUserTaskCandidateUser),
		CandidateGroup:       strings.TrimSpace(flagGetUserTaskCandidateGroup),
		BatchSize:            flagGetUserTaskBatchSize,
		Limit:                flagGetUserTaskLimit,
	}
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
	for _, name := range []string{"pi-key", "pd-key", "bpmn-process-id", "element-id", "state", "assignee", "candidate-user", "candidate-group", "limit", "total"} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}
