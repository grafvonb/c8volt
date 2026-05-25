// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/spf13/cobra"
)

var (
	flagUpdateJobKey             string
	flagUpdateJobRetries         int32
	flagUpdateJobTimeoutRaw      string
	flagUpdateJobFail            bool
	flagUpdateJobRetryBackoffRaw string
	flagUpdateJobMessage         string
	flagUpdateJobBPMNError       string
	flagUpdateJobComplete        bool
	flagUpdateJobVariables       string
)

var updateJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Update a job by key",
	Long: "Update a Camunda job by key.\n\n" +
		"The command supports retries, timeout updates, and worker outcome modes for Camunda 8.8 and 8.9. It builds a pre-mutation plan, supports --dry-run previews, and asks for confirmation before material interactive mutations. Retry updates are confirmed by reading the job by key by default; timeout updates and worker outcomes report accepted submission without deadline or outcome confirmation. JSON mutations require --dry-run, --auto-confirm, or --automation, and --json cannot be combined with --verbose. Camunda 8.7 returns an unsupported-version error before mutation.",
	Example: `  ./c8volt update job --key <job-key> --retries 3 --dry-run
  ./c8volt update job --key <job-key> --retries 3 --auto-confirm
  ./c8volt update job --key <job-key> --timeout 5m --auto-confirm
  ./c8volt update job --key <job-key> --fail --retries 0 --message "worker unavailable" --dry-run
  ./c8volt update job --key <job-key> --throw-bpmn-error PAYMENT_DECLINED --message "card declined" --dry-run
  ./c8volt update job --key <job-key> --complete --vars '{"approved":true}' --dry-run
  ./c8volt --json update job --key <job-key> --retries 3 --dry-run`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		request, err := parseUpdateJobRequest(cmd)
		if err != nil {
			failBeforeCli(cmd, err)
		}
		if err := validateUpdateJobJSONGuardrails(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		plan, err := planUpdateJob(cmd.Context(), cli, request)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan job update: %w", err))
		}
		request.UpdatePlan = &plan
		if err := validateUpdateJobPlanPreconditions(plan, request); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if flagDryRun {
			if err := jobUpdatePlanView(cmd, plan, "dry run"); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job update dry-run result: %w", err))
			}
			return
		}
		if !plan.HasMaterialChange() {
			if err := jobUpdatePlanView(cmd, plan, "plan"); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job update plan: %w", err))
			}
			return
		}
		if !shouldImplicitlyConfirm(cmd) {
			if err := jobUpdatePlanView(cmd, plan, "plan"); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job update plan: %w", err))
			}
			prompt := fmt.Sprintf("You are about to update job %s. Do you want to proceed?", request.Key)
			if err := confirmCmdOrAbortFn(false, prompt); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
		}
		if request.WorkerOutcome != nil {
			request.WorkerOutcome.OutcomePlan = &plan
			result, err := cli.SubmitJobWorkerOutcome(cmd.Context(), *request.WorkerOutcome, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("update job: %w", err))
			}
			if err := jobWorkerOutcomeResultView(cmd, result); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job update result: %w", err))
			}
			return
		}
		result, err := cli.UpdateJob(cmd.Context(), request, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("update job: %w", err))
		}
		if err := jobUpdateResultView(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job update result: %w", err))
		}
	},
}

func init() {
	updateCmd.AddCommand(updateJobCmd)

	fs := updateJobCmd.Flags()
	fs.StringVar(&flagUpdateJobKey, "key", "", "job key to update")
	fs.Int32Var(&flagUpdateJobRetries, "retries", 0, "retry count to set, or remaining retries for --fail")
	fs.StringVar(&flagUpdateJobTimeoutRaw, "timeout", "", "timeout duration to submit for the job, for example 60s, 5m, or 1h")
	fs.BoolVar(&flagUpdateJobFail, "fail", false, "report a technical job failure")
	fs.StringVar(&flagUpdateJobRetryBackoffRaw, "retry-backoff", "", "duration before a failed job becomes retryable, for example 60s, 5m, or 1h")
	fs.StringVar(&flagUpdateJobMessage, "message", "", "operator message for worker outcome modes")
	fs.StringVar(&flagUpdateJobBPMNError, "throw-bpmn-error", "", "BPMN error code to throw for the job")
	fs.BoolVar(&flagUpdateJobComplete, "complete", false, "complete the job through the worker outcome API")
	fs.StringVar(&flagUpdateJobVariables, "vars", "", "JSON object with variables for BPMN error or completion outcomes")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview job updates without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after the update request is accepted without retry confirmation")

	useInvalidInputFlagErrors(updateJobCmd)
	setCommandMutation(updateJobCmd, CommandMutationStateChanging)
	setContractSupport(updateJobCmd, ContractSupportFull)
	setAutomationSupport(updateJobCmd, AutomationSupportFull, "supports shared machine output, non-mutating dry-run previews, and accepted results")
	setOutputModes(updateJobCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
	setFlagContractRequired(updateJobCmd, "key")
}

func parseUpdateJobRequest(cmd *cobra.Command) (job.UpdateRequest, error) {
	if strings.TrimSpace(flagUpdateJobKey) == "" {
		return job.UpdateRequest{}, invalidFlagValuef("job update requires a non-empty --key")
	}
	if cmd.Flags().Changed("throw-bpmn-error") {
		return parseUpdateJobBPMNErrorRequest(cmd)
	}
	if cmd.Flags().Changed("fail") {
		return parseUpdateJobTechnicalFailureRequest(cmd)
	}
	if cmd.Flags().Changed("complete") {
		return parseUpdateJobCompletionRequest(cmd)
	}
	if workerFlags := changedUpdateJobWorkerOutcomeFlags(cmd); len(workerFlags) > 0 {
		return job.UpdateRequest{}, invalidFlagValuef("job worker outcome flags are reserved for the BPMN error and completion implementations: %s", strings.Join(workerFlags, ", "))
	}
	retriesChanged := cmd.Flags().Changed("retries")
	timeoutChanged := cmd.Flags().Changed("timeout")
	if !retriesChanged && !timeoutChanged {
		return job.UpdateRequest{}, invalidFlagValuef("update job requires --retries, --timeout, or both")
	}
	request := job.UpdateRequest{
		Key:         flagUpdateJobKey,
		NoWait:      flagNoWait,
		AutoConfirm: flagCmdAutoConfirm,
		Automation:  updateJobAutomationEnabled(cmd),
		DryRun:      flagDryRun,
	}
	if retriesChanged {
		if flagUpdateJobRetries < 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --retries: %d, expected non-negative integer", flagUpdateJobRetries)
		}
		retries := flagUpdateJobRetries
		request.Retries = &retries
		request.ConfirmRetries = !flagNoWait
	}
	if timeoutChanged {
		timeout, err := time.ParseDuration(flagUpdateJobTimeoutRaw)
		if err != nil || timeout <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --timeout: %q, expected positive duration such as 60s, 5m, or 1h", flagUpdateJobTimeoutRaw)
		}
		timeoutMillis := timeout.Milliseconds()
		if timeoutMillis <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --timeout: %q, duration must be at least 1ms", flagUpdateJobTimeoutRaw)
		}
		request.Timeout = &timeout
		request.TimeoutRaw = flagUpdateJobTimeoutRaw
		request.TimeoutMillis = &timeoutMillis
	}
	return request, nil
}

func parseUpdateJobBPMNErrorRequest(cmd *cobra.Command) (job.UpdateRequest, error) {
	if cmd.Flags().Changed("fail") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --fail")
	}
	if cmd.Flags().Changed("complete") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --complete")
	}
	if cmd.Flags().Changed("retries") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --retries")
	}
	if cmd.Flags().Changed("timeout") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --timeout")
	}
	if cmd.Flags().Changed("retry-backoff") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--throw-bpmn-error cannot be combined with --retry-backoff")
	}
	errorCode := strings.TrimSpace(flagUpdateJobBPMNError)
	if errorCode == "" {
		return job.UpdateRequest{}, invalidFlagValuef("BPMN error requires a non-empty --throw-bpmn-error")
	}
	var variables map[string]any
	if cmd.Flags().Changed("vars") {
		parsed, err := parseUpdateJobVariables(flagUpdateJobVariables)
		if err != nil {
			return job.UpdateRequest{}, err
		}
		variables = parsed
	}
	outcome := job.WorkerOutcomeRequest{
		Key:         flagUpdateJobKey,
		Mode:        job.WorkerOutcomeBPMNError,
		Message:     flagUpdateJobMessage,
		Variables:   variables,
		ErrorCode:   errorCode,
		NoWait:      flagNoWait,
		AutoConfirm: flagCmdAutoConfirm,
		Automation:  updateJobAutomationEnabled(cmd),
		DryRun:      flagDryRun,
	}
	return job.UpdateRequest{
		Key:           flagUpdateJobKey,
		NoWait:        flagNoWait,
		AutoConfirm:   flagCmdAutoConfirm,
		Automation:    updateJobAutomationEnabled(cmd),
		DryRun:        flagDryRun,
		WorkerOutcome: &outcome,
	}, nil
}

func parseUpdateJobCompletionRequest(cmd *cobra.Command) (job.UpdateRequest, error) {
	if cmd.Flags().Changed("fail") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --fail")
	}
	if cmd.Flags().Changed("throw-bpmn-error") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --throw-bpmn-error")
	}
	if cmd.Flags().Changed("retries") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --retries")
	}
	if cmd.Flags().Changed("timeout") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --timeout")
	}
	if cmd.Flags().Changed("retry-backoff") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --retry-backoff")
	}
	if cmd.Flags().Changed("message") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--complete cannot be combined with --message")
	}
	var variables map[string]any
	if cmd.Flags().Changed("vars") {
		parsed, err := parseUpdateJobVariables(flagUpdateJobVariables)
		if err != nil {
			return job.UpdateRequest{}, err
		}
		variables = parsed
	}
	outcome := job.WorkerOutcomeRequest{
		Key:         flagUpdateJobKey,
		Mode:        job.WorkerOutcomeCompletion,
		Variables:   variables,
		NoWait:      flagNoWait,
		AutoConfirm: flagCmdAutoConfirm,
		Automation:  updateJobAutomationEnabled(cmd),
		DryRun:      flagDryRun,
	}
	return job.UpdateRequest{
		Key:           flagUpdateJobKey,
		NoWait:        flagNoWait,
		AutoConfirm:   flagCmdAutoConfirm,
		Automation:    updateJobAutomationEnabled(cmd),
		DryRun:        flagDryRun,
		WorkerOutcome: &outcome,
	}, nil
}

func parseUpdateJobTechnicalFailureRequest(cmd *cobra.Command) (job.UpdateRequest, error) {
	if cmd.Flags().Changed("throw-bpmn-error") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--fail cannot be combined with --throw-bpmn-error")
	}
	if cmd.Flags().Changed("complete") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--fail cannot be combined with --complete")
	}
	if cmd.Flags().Changed("timeout") {
		return job.UpdateRequest{}, mutuallyExclusiveFlagsf("--fail cannot be combined with --timeout")
	}
	if cmd.Flags().Changed("vars") {
		return job.UpdateRequest{}, invalidFlagValuef("--vars is reserved for BPMN error and completion implementations")
	}
	if !cmd.Flags().Changed("retries") {
		return job.UpdateRequest{}, invalidFlagValuef("technical job failure requires --retries")
	}
	if flagUpdateJobRetries < 0 {
		return job.UpdateRequest{}, invalidFlagValuef("invalid value for --retries: %d, expected non-negative integer", flagUpdateJobRetries)
	}
	retries := flagUpdateJobRetries
	outcome := job.WorkerOutcomeRequest{
		Key:             flagUpdateJobKey,
		Mode:            job.WorkerOutcomeTechnicalFailure,
		Message:         flagUpdateJobMessage,
		Retries:         &retries,
		RetryBackoffRaw: flagUpdateJobRetryBackoffRaw,
		NoWait:          flagNoWait,
		AutoConfirm:     flagCmdAutoConfirm,
		Automation:      updateJobAutomationEnabled(cmd),
		DryRun:          flagDryRun,
	}
	if cmd.Flags().Changed("retry-backoff") {
		retryBackoff, err := time.ParseDuration(flagUpdateJobRetryBackoffRaw)
		if err != nil || retryBackoff <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --retry-backoff: %q, expected positive duration such as 60s, 5m, or 1h", flagUpdateJobRetryBackoffRaw)
		}
		retryBackoffMillis := retryBackoff.Milliseconds()
		if retryBackoffMillis <= 0 {
			return job.UpdateRequest{}, invalidFlagValuef("invalid value for --retry-backoff: %q, duration must be at least 1ms", flagUpdateJobRetryBackoffRaw)
		}
		outcome.RetryBackoff = &retryBackoff
		outcome.RetryBackoffMillis = &retryBackoffMillis
	}
	return job.UpdateRequest{
		Key:           flagUpdateJobKey,
		NoWait:        flagNoWait,
		AutoConfirm:   flagCmdAutoConfirm,
		Automation:    updateJobAutomationEnabled(cmd),
		DryRun:        flagDryRun,
		WorkerOutcome: &outcome,
	}, nil
}

func parseUpdateJobVariables(raw string) (map[string]any, error) {
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, invalidFlagValuef("--vars must be a valid JSON object: %v", err)
	}
	variables, ok := decoded.(map[string]any)
	if !ok || variables == nil {
		return nil, invalidFlagValuef("--vars must be a JSON object")
	}
	return variables, nil
}

func changedUpdateJobWorkerOutcomeFlags(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}
	names := []string{
		"fail",
		"retry-backoff",
		"message",
		"throw-bpmn-error",
		"complete",
		"vars",
	}
	changed := make([]string, 0, len(names))
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			changed = append(changed, "--"+name)
		}
	}
	return changed
}

func validateUpdateJobJSONGuardrails(cmd *cobra.Command) error {
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for update job")
	}
	if flagDryRun || pickMode() != RenderModeJSON || flagCmdAutoConfirm || flagCmdAutomation || updateJobAutomationEnabled(cmd) {
		return nil
	}
	return missingDependentFlagsf("--json update job requires --dry-run, --auto-confirm, or --automation")
}

func updateJobAutomationEnabled(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Context() == nil {
		return false
	}
	return automationModeEnabled(cmd)
}

func planUpdateJob(ctx context.Context, cli c8volt.API, request job.UpdateRequest) (job.UpdatePlan, error) {
	current, err := cli.GetJob(ctx, request.Key, collectOptions()...)
	if err != nil {
		return job.UpdatePlan{}, err
	}
	return buildUpdateJobPlan(current, request), nil
}

func buildUpdateJobPlan(current job.Job, request job.UpdateRequest) job.UpdatePlan {
	plan := job.UpdatePlan{
		Key:               request.Key,
		Current:           current,
		Mode:              job.MutationModeUpdate,
		RetryStatus:       job.RetryChangeNotRequested,
		DryRun:            request.DryRun,
		MutationSubmitted: false,
	}
	if request.WorkerOutcome != nil {
		return buildWorkerOutcomeUpdatePlan(current, request)
	}
	if request.Retries != nil {
		retries := *request.Retries
		plan.RequestedRetries = &retries
		status := job.RetryChangeChanged
		before := strconv.FormatInt(int64(current.Retries), 10)
		if current.Retries == retries {
			status = job.RetryChangeUnchanged
		}
		plan.RetryStatus = status
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "retries",
			Before: before,
			After:  strconv.FormatInt(int64(retries), 10),
			Status: string(status),
		})
		if status == job.RetryChangeChanged {
			plan.MaterialChange = true
		}
	}
	if request.TimeoutMillis != nil {
		timeoutMillis := *request.TimeoutMillis
		plan.RequestedTimeout = request.TimeoutRaw
		plan.TimeoutMillis = &timeoutMillis
		plan.MaterialChange = true
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "timeout",
			After:  request.TimeoutRaw,
			Status: "submit",
		})
	}
	return plan
}

// buildWorkerOutcomeUpdatePlan renders worker outcomes through the same dry-run
// and confirmation plan shape used by existing retry and timeout updates.
func buildWorkerOutcomeUpdatePlan(current job.Job, request job.UpdateRequest) job.UpdatePlan {
	outcome := request.WorkerOutcome
	plan := job.UpdatePlan{
		Key:               request.Key,
		Current:           current,
		Mode:              job.MutationMode(outcome.Mode),
		RequestedRetries:  outcome.Retries,
		RetryStatus:       job.RetryChangeNotRequested,
		Message:           outcome.Message,
		RetryBackoff:      outcome.RetryBackoffRaw,
		RetryBackoffMS:    outcome.RetryBackoffMillis,
		ErrorCode:         outcome.ErrorCode,
		Variables:         outcome.Variables,
		MaterialChange:    true,
		DryRun:            request.DryRun,
		MutationSubmitted: false,
	}
	plan.Items = append(plan.Items, job.UpdatePlanItem{
		Name:   string(outcome.Mode),
		After:  "submit",
		Status: "submit",
	})
	if outcome.Retries != nil {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "retries",
			After:  strconv.FormatInt(int64(*outcome.Retries), 10),
			Status: "submit",
		})
	}
	if outcome.RetryBackoffMillis != nil {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "retryBackoff",
			After:  outcome.RetryBackoffRaw,
			Status: "submit",
		})
	}
	if outcome.ErrorCode != "" {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "errorCode",
			After:  outcome.ErrorCode,
			Status: "submit",
		})
	}
	if outcome.Message != "" {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "message",
			After:  outcome.Message,
			Status: "submit",
		})
	}
	if outcome.Variables != nil {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "variables",
			After:  "submit",
			Status: "submit",
		})
	}
	return plan
}

// validateUpdateJobPlanPreconditions rejects planned updates that Camunda cannot accept for the current job state.
func validateUpdateJobPlanPreconditions(plan job.UpdatePlan, request job.UpdateRequest) error {
	if request.WorkerOutcome != nil {
		return nil
	}
	if request.TimeoutMillis == nil {
		return nil
	}
	if plan.Current.Key == "" {
		return nil
	}
	if strings.EqualFold(plan.Current.State, "CREATED") {
		return nil
	}
	state := plan.Current.State
	if state == "" {
		state = "unknown"
	}
	return localPreconditionError(fmt.Errorf("job timeout can be updated only for active jobs; job %s is %s", request.Key, state))
}
