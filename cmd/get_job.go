// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

var (
	flagGetJobKey            string
	flagGetJobState          string
	flagGetJobType           string
	flagGetJobProcessKey     string
	flagGetJobElementKey     string
	flagGetJobElementID      string
	flagGetJobWorker         string
	flagGetJobRetries        int32
	flagGetJobKind           string
	flagGetJobListenerEvent  string
	flagGetJobLimit          int32
	flagGetErrorMessageLimit int
)

var getJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Inspect or search jobs",
	Long: "Inspect or search Camunda jobs.\n\n" +
		"Use --key with the jobKey exposed by incident-aware process-instance output to inspect a matching runtime job directly. Search mode will use list filters such as --state, --type, --pi-key, --element-instance-key, --element-id, --worker, --retries, --kind, --listener-event-type, and --limit. Use --json for the stable job payload, or --error-message-limit to shorten long error messages. Job lookup and search are supported for Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error.",
	Example: `  ./c8volt get job --key <job-key>
  ./c8volt get job --state FAILED --limit 50
  ./c8volt --json get job --key <job-key>`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateGetJobFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if strings.TrimSpace(flagGetJobKey) != "" {
			item, err := cli.GetJob(cmd.Context(), flagGetJobKey, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get job: %w", err))
			}
			if err := jobView(cmd, item); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job: %w", err))
			}
			return
		}
		result, err := cli.SearchJobs(cmd.Context(), newGetJobSearchRequest(cmd), collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get jobs: %w", err))
		}
		if err := jobsView(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render jobs: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getJobCmd)

	fs := getJobCmd.Flags()
	fs.StringVar(&flagGetJobKey, "key", "", "job key for exact lookup; omit to list or search jobs")
	fs.StringVar(&flagGetJobState, "state", "", "Camunda job state to filter in search mode")
	fs.StringVar(&flagGetJobType, "type", "", "job type to filter in search mode")
	fs.StringVar(&flagGetJobProcessKey, "pi-key", "", "process instance key to filter in search mode")
	fs.StringVar(&flagGetJobElementKey, "element-instance-key", "", "element instance key to filter in search mode")
	fs.StringVar(&flagGetJobElementID, "element-id", "", "BPMN element ID to filter in search mode")
	fs.StringVar(&flagGetJobWorker, "worker", "", "worker name to filter in search mode")
	fs.Int32Var(&flagGetJobRetries, "retries", 0, "exact retry count to filter in search mode")
	fs.StringVar(&flagGetJobKind, "kind", "", "Camunda job kind to filter in search mode")
	fs.StringVar(&flagGetJobListenerEvent, "listener-event-type", "", "listener event type to filter in search mode")
	fs.Int32Var(&flagGetJobLimit, "limit", 0, "maximum number of jobs to return in search mode")
	fs.IntVar(&flagGetErrorMessageLimit, "error-message-limit", 0, "maximum characters to show for error messages; 0 keeps full messages")

	useInvalidInputFlagErrors(getJobCmd)
	setCommandMutation(getJobCmd, CommandMutationReadOnly)
	setContractSupport(getJobCmd, ContractSupportFull)
	setAutomationSupport(getJobCmd, AutomationSupportFull, "supports shared machine output and unattended job reads")
}

func validateGetJobFlags(cmd *cobra.Command) error {
	searchFlags := changedGetJobSearchFlags(cmd)
	if strings.TrimSpace(flagGetJobKey) != "" && len(searchFlags) > 0 {
		return mutuallyExclusiveFlagsf("--key cannot be combined with job search filters: %s", strings.Join(searchFlags, ", "))
	}
	if err := validateGetJobSearchFlags(cmd); err != nil {
		return err
	}
	if flagGetErrorMessageLimit < 0 {
		return invalidFlagValuef("--error-message-limit must be non-negative")
	}
	if pickMode() == RenderModeJSON && cmd != nil && cmd.Flags().Changed("error-message-limit") {
		return mutuallyExclusiveFlagsf("--error-message-limit cannot be combined with --json")
	}
	return nil
}

// validateGetJobSearchFlags rejects invalid list/search values before the command
// creates a client or sends a Camunda search request.
func validateGetJobSearchFlags(cmd *cobra.Command) error {
	if cmd == nil {
		return nil
	}
	if cmd.Flags().Changed("limit") && flagGetJobLimit <= 0 {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if cmd.Flags().Changed("retries") && flagGetJobRetries < 0 {
		return invalidFlagValuef("--retries must be non-negative")
	}
	if flagGetJobProcessKey != "" {
		if ok, firstBadKey, _ := validateKeys([]string{flagGetJobProcessKey}); !ok {
			return invalidFlagValuef("--pi-key value %q is not a valid key", firstBadKey)
		}
	}
	if flagGetJobElementKey != "" {
		if ok, firstBadKey, _ := validateKeys([]string{flagGetJobElementKey}); !ok {
			return invalidFlagValuef("--element-instance-key value %q is not a valid key", firstBadKey)
		}
	}
	if flagGetJobState != "" && !validJobState(flagGetJobState) {
		return invalidFlagValuef("invalid value for --state: %q, valid values are: %s", flagGetJobState, strings.Join(validJobStates, ", "))
	}
	if flagGetJobKind != "" && !validJobKind(flagGetJobKind) {
		return invalidFlagValuef("invalid value for --kind: %q, valid values are: %s", flagGetJobKind, strings.Join(validJobKinds, ", "))
	}
	if flagGetJobListenerEvent != "" && !validJobListenerEventType(flagGetJobListenerEvent) {
		return invalidFlagValuef("invalid value for --listener-event-type: %q, valid values are: %s", flagGetJobListenerEvent, strings.Join(validJobListenerEventTypes, ", "))
	}
	return nil
}

// newGetJobSearchRequest maps validated command flags into the public facade
// request while preserving zero retries only when the operator supplied it.
func newGetJobSearchRequest(cmd *cobra.Command) job.SearchRequest {
	req := job.SearchRequest{
		State:              flagGetJobState,
		Type:               flagGetJobType,
		ProcessInstanceKey: flagGetJobProcessKey,
		ElementInstanceKey: flagGetJobElementKey,
		ElementId:          flagGetJobElementID,
		Worker:             flagGetJobWorker,
		Kind:               flagGetJobKind,
		ListenerEventType:  flagGetJobListenerEvent,
		Limit:              effectiveGetJobLimit(),
	}
	if cmd != nil && cmd.Flags().Changed("retries") {
		retries := flagGetJobRetries
		req.Retries = &retries
	}
	return req
}

// effectiveGetJobLimit keeps list/search bounded even when the caller does not
// pass --limit.
func effectiveGetJobLimit() int32 {
	if flagGetJobLimit > 0 {
		return flagGetJobLimit
	}
	return consts.MaxPISearchSize
}

// validJobStates is the explicit Camunda v8.8/v8.9 job state allowlist for
// local search validation.
var validJobStates = []string{
	"CANCELED",
	"COMPLETED",
	"CREATED",
	"ERROR_THROWN",
	"FAILED",
	"MIGRATED",
	"RETRIES_UPDATED",
	"TIMED_OUT",
}

// validJobKinds is the explicit Camunda v8.8/v8.9 job kind allowlist for local
// search validation.
var validJobKinds = []string{
	"AD_HOC_SUB_PROCESS",
	"BPMN_ELEMENT",
	"EXECUTION_LISTENER",
	"TASK_LISTENER",
}

// validJobListenerEventTypes is the explicit Camunda v8.8/v8.9 listener event
// allowlist for local search validation.
var validJobListenerEventTypes = []string{
	"ASSIGNING",
	"CANCELING",
	"COMPLETING",
	"CREATING",
	"END",
	"START",
	"UNSPECIFIED",
	"UPDATING",
}

// validJobState reports whether value is an upstream job state accepted by the
// generated Camunda job search endpoint.
func validJobState(value string) bool {
	return stringInList(value, validJobStates)
}

// validJobKind reports whether value is an upstream job kind accepted by the
// generated Camunda job search endpoint.
func validJobKind(value string) bool {
	return stringInList(value, validJobKinds)
}

// validJobListenerEventType reports whether value is an upstream listener event
// type accepted by the generated Camunda job search endpoint.
func validJobListenerEventType(value string) bool {
	return stringInList(value, validJobListenerEventTypes)
}

// stringInList keeps enum-style validation strict and case-sensitive.
func stringInList(value string, valid []string) bool {
	for _, candidate := range valid {
		if value == candidate {
			return true
		}
	}
	return false
}

func changedGetJobSearchFlags(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}
	names := []string{
		"state",
		"type",
		"pi-key",
		"element-instance-key",
		"element-id",
		"worker",
		"retries",
		"kind",
		"listener-event-type",
		"limit",
	}
	changed := make([]string, 0, len(names))
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			changed = append(changed, "--"+name)
		}
	}
	return changed
}
