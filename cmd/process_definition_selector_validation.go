// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type processDefinitionSelectorValidationMode int

const (
	processDefinitionSelectorValidationAny processDefinitionSelectorValidationMode = iota
	processDefinitionSelectorValidationLatest
)

type processDefinitionSelectorValidationRequest struct {
	BpmnProcessIds    []string
	ProcessVersion    int32
	ProcessVersionTag string
	Mode              processDefinitionSelectorValidationMode
}

type processDefinitionSelectorValidationResult struct {
	Request                    processDefinitionSelectorValidationRequest
	MatchesByBpmnProcessID     map[string]process.ProcessDefinitions
	NearMatchesByBpmnProcessID map[string]process.ProcessDefinitions
	MissingBpmnProcessIDs      []string
}

type processDefinitionSelectorMissingError struct {
	message string
}

func (e processDefinitionSelectorMissingError) Error() string {
	return e.message
}

func (e processDefinitionSelectorMissingError) Unwrap() error {
	return ferrors.ErrLocalPrecondition
}

var processDefinitionSelectorInteractiveTerminalFn = processDefinitionSelectorInteractiveTerminal
var confirmProcessDefinitionSelectorListVisibleFn = confirmCmdOrAbortDefaultYes

func (r processDefinitionSelectorValidationResult) Valid() bool {
	return len(r.MissingBpmnProcessIDs) == 0
}

func newPIProcessDefinitionSelectorValidationRequest() processDefinitionSelectorValidationRequest {
	return processDefinitionSelectorValidationRequest{
		BpmnProcessIds:    normalizeSelectorBpmnProcessIDs([]string{flagGetPIBpmnProcessID}),
		ProcessVersion:    flagGetPIProcessVersion,
		ProcessVersionTag: flagGetPIProcessVersionTag,
	}
}

func newRunPIProcessDefinitionSelectorValidationRequest() processDefinitionSelectorValidationRequest {
	mode := processDefinitionSelectorValidationLatest
	if flagRunPIProcessDefinitionVersion != 0 {
		mode = processDefinitionSelectorValidationAny
	}
	return processDefinitionSelectorValidationRequest{
		BpmnProcessIds: normalizeSelectorBpmnProcessIDs(flagRunPIProcessDefinitionBpmnProcessIds),
		ProcessVersion: flagRunPIProcessDefinitionVersion,
		Mode:           mode,
	}
}

func (r processDefinitionSelectorValidationRequest) filterForBpmnProcessID(bpmnProcessID string) process.ProcessDefinitionFilter {
	return process.ProcessDefinitionFilter{
		BpmnProcessId:     bpmnProcessID,
		ProcessVersion:    r.ProcessVersion,
		ProcessVersionTag: r.ProcessVersionTag,
	}
}

// validateProcessDefinitionSelectors proves BPMN selectors are visible before PI commands turn typos into empty searches or no-op mutations.
func validateProcessDefinitionSelectors(ctx context.Context, cli process.API, req processDefinitionSelectorValidationRequest, opts ...options.FacadeOption) (processDefinitionSelectorValidationResult, error) {
	ids := normalizeSelectorBpmnProcessIDs(req.BpmnProcessIds)
	result := processDefinitionSelectorValidationResult{
		Request: processDefinitionSelectorValidationRequest{
			BpmnProcessIds:    ids,
			ProcessVersion:    req.ProcessVersion,
			ProcessVersionTag: req.ProcessVersionTag,
			Mode:              req.Mode,
		},
		MatchesByBpmnProcessID:     make(map[string]process.ProcessDefinitions, len(ids)),
		NearMatchesByBpmnProcessID: make(map[string]process.ProcessDefinitions, len(ids)),
	}

	for _, id := range ids {
		filter := result.Request.filterForBpmnProcessID(id)
		var (
			matches process.ProcessDefinitions
			err     error
		)
		if req.Mode == processDefinitionSelectorValidationLatest {
			matches, err = cli.SearchProcessDefinitionsLatest(ctx, filter, opts...)
		} else {
			matches, err = cli.SearchProcessDefinitions(ctx, filter, opts...)
		}
		if err != nil {
			return result, fmt.Errorf("validate process definition selector %q: %w", id, err)
		}
		if processDefinitionSelectorHasMatches(matches) {
			result.MatchesByBpmnProcessID[id] = matches
			continue
		}
		result.MissingBpmnProcessIDs = append(result.MissingBpmnProcessIDs, id)
		if !result.Request.HasVersionSelector() {
			continue
		}
		nearMatchOpts := append(append([]options.FacadeOption(nil), opts...), options.WithIgnoreTenant())
		nearMatches, err := cli.SearchProcessDefinitions(ctx, process.ProcessDefinitionFilter{BpmnProcessId: id}, nearMatchOpts...)
		if err != nil {
			return result, fmt.Errorf("validate process definition selector %q without version/tag: %w", id, err)
		}
		if processDefinitionSelectorHasMatches(nearMatches) {
			result.NearMatchesByBpmnProcessID[id] = nearMatches
		}
	}

	return result, nil
}

func (r processDefinitionSelectorValidationRequest) HasVersionSelector() bool {
	return r.ProcessVersion != 0 || r.ProcessVersionTag != ""
}

func processDefinitionSelectorNoPromptError(result processDefinitionSelectorValidationResult) error {
	if result.Valid() {
		return nil
	}
	return newProcessDefinitionSelectorMissingError(result)
}

func processDefinitionSelectorValidationError(result processDefinitionSelectorValidationResult) error {
	if result.Valid() {
		return nil
	}

	return newProcessDefinitionSelectorMissingError(result)
}

// handleProcessDefinitionSelectorValidationError keeps machine modes non-interactive while giving TTY users one recovery listing.
func handleProcessDefinitionSelectorValidationError(cmd *cobra.Command, log *slog.Logger, noErrCodes bool, cli process.API, result processDefinitionSelectorValidationResult) {
	err := processDefinitionSelectorValidationError(result)
	if err == nil {
		return
	}
	if !processDefinitionSelectorPromptAllowed(cmd) {
		handleCommandError(cmd, log, noErrCodes, err)
	}
	if log == nil {
		log = slog.Default()
	}
	log.Error(err.Error())
	if recoveryErr := processDefinitionSelectorRecovery(cmd, cli, result); recoveryErr != nil {
		handleCommandError(cmd, log, noErrCodes, recoveryErr)
	}
	os.Exit(ferrors.ResolveExitCode(noErrCodes, err))
}

// processDefinitionSelectorRecovery reuses existing process-definition list rendering instead of introducing a second diagnostic format.
func processDefinitionSelectorRecovery(cmd *cobra.Command, cli process.API, result processDefinitionSelectorValidationResult) error {
	if result.HasNearMatches() {
		if err := confirmProcessDefinitionSelectorListVisibleFn(false, "List matching process definitions?"); err != nil {
			return nil
		}
		if err := listNearMatchProcessDefinitionsForSelectorValidation(cmd, result); err != nil {
			return localPreconditionError(fmt.Errorf("%s; list matching process definitions: %w", formatMissingProcessDefinitionSelectorsSummary(result), err))
		}
		return nil
	}

	pds, err := visibleProcessDefinitionsForSelectorValidation(cmd, cli)
	if err != nil {
		return localPreconditionError(fmt.Errorf("%s; list visible process definitions: %w", formatMissingProcessDefinitionSelectorsSummary(result), err))
	}
	if !processDefinitionSelectorHasMatches(pds) {
		return nil
	}
	if err := confirmProcessDefinitionSelectorListVisibleFn(false, "List visible process definitions?"); err != nil {
		return nil
	}
	if err := listProcessDefinitionsView(cmd, pds); err != nil {
		return localPreconditionError(fmt.Errorf("%s; list visible process definitions: render process definitions: %w", formatMissingProcessDefinitionSelectorsSummary(result), err))
	}
	return nil
}

// processDefinitionSelectorPromptAllowed is stricter than normal confirmation because selector diagnostics must never surprise pipelines with listings.
func processDefinitionSelectorPromptAllowed(cmd *cobra.Command) bool {
	if cmd == nil || flagCmdAutoConfirm || flagViewAsJson || flagViewKeysOnly || automationModeEnabled(cmd) {
		return false
	}
	return processDefinitionSelectorInteractiveTerminalFn()
}

func (r processDefinitionSelectorValidationResult) HasNearMatches() bool {
	for _, id := range r.MissingBpmnProcessIDs {
		if processDefinitionSelectorHasMatches(r.NearMatchesByBpmnProcessID[id]) {
			return true
		}
	}
	return false
}

func processDefinitionSelectorInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func visibleProcessDefinitionsForSelectorValidation(cmd *cobra.Command, cli process.API) (process.ProcessDefinitions, error) {
	pds, err := cli.SearchProcessDefinitions(cmd.Context(), process.ProcessDefinitionFilter{}, collectOptions()...)
	if err != nil {
		return process.ProcessDefinitions{}, fmt.Errorf("search process definitions: %w", err)
	}
	return pds, nil
}

func listVisibleProcessDefinitionsForSelectorValidation(cmd *cobra.Command, cli process.API) error {
	pds, err := visibleProcessDefinitionsForSelectorValidation(cmd, cli)
	if err != nil {
		return err
	}
	if err := listProcessDefinitionsView(cmd, pds); err != nil {
		return fmt.Errorf("render process definitions: %w", err)
	}
	return nil
}

func listNearMatchProcessDefinitionsForSelectorValidation(cmd *cobra.Command, result processDefinitionSelectorValidationResult) error {
	var items []process.ProcessDefinition
	for _, id := range result.MissingBpmnProcessIDs {
		matches := result.NearMatchesByBpmnProcessID[id]
		items = append(items, matches.Items...)
	}
	if err := listProcessDefinitionsView(cmd, process.ProcessDefinitions{
		Total: int32(len(items)),
		Items: items,
	}); err != nil {
		return fmt.Errorf("render process definitions: %w", err)
	}
	return nil
}

func formatMissingProcessDefinitionSelectors(result processDefinitionSelectorValidationResult) string {
	return fmt.Sprintf("%s: %s", formatMissingProcessDefinitionSelectorsSummary(result), formatMissingProcessDefinitionSelectorList(result))
}

func formatMissingProcessDefinitionSelectorList(result processDefinitionSelectorValidationResult) string {
	rows := formatMissingProcessDefinitionSelectorRows(result)
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, fmt.Sprintf("[%s]", row))
	}
	return strings.Join(out, ", ")
}

func formatMissingProcessDefinitionSelectorRows(result processDefinitionSelectorValidationResult) []string {
	rows := make([]string, 0, len(result.MissingBpmnProcessIDs))
	for _, id := range result.MissingBpmnProcessIDs {
		row := flatRow{id}
		if version := formatProcessDefinitionSelectorVersion(result.Request); version != "" {
			row = append(row, version)
		}
		rows = append(rows, compactFlatRow(row))
	}
	return rows
}

func newProcessDefinitionSelectorMissingError(result processDefinitionSelectorValidationResult) error {
	return processDefinitionSelectorMissingError{message: formatMissingProcessDefinitionSelectors(result)}
}

func formatProcessDefinitionSelectorVersion(req processDefinitionSelectorValidationRequest) string {
	switch {
	case req.ProcessVersion != 0 && req.ProcessVersionTag != "":
		return fmt.Sprintf("v%d/%s", req.ProcessVersion, req.ProcessVersionTag)
	case req.ProcessVersion != 0:
		return fmt.Sprintf("v%d", req.ProcessVersion)
	case req.ProcessVersionTag != "":
		return "*/" + req.ProcessVersionTag
	default:
		return ""
	}
}

func formatMissingProcessDefinitionSelectorsSummary(result processDefinitionSelectorValidationResult) string {
	if len(result.MissingBpmnProcessIDs) == 1 {
		return "no visible process definition matches the provided selector"
	}
	return "no visible process definitions match the provided selector(s)"
}

func processDefinitionSelectorHasMatches(matches process.ProcessDefinitions) bool {
	return matches.Total > 0 || len(matches.Items) > 0
}

func normalizeSelectorBpmnProcessIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// confirmCmdOrAbortDefaultYes is scoped to selector recovery; command launch behavior still uses the shared confirmation helpers.
func confirmCmdOrAbortDefaultYes(autoConfirm bool, prompt string) error {
	if autoConfirm || !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}
	fmt.Print(formatConfirmationPrompt(prompt, "[Y/n]"))
	in := bufio.NewScanner(os.Stdin)
	if !in.Scan() {
		return localPreconditionError(ErrCmdAborted)
	}
	switch strings.ToLower(strings.TrimSpace(in.Text())) {
	case "", "y", "yes":
		return nil
	default:
		return localPreconditionError(ErrCmdAborted)
	}
}
