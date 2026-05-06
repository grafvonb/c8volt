// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"strings"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
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
	Request                processDefinitionSelectorValidationRequest
	MatchesByBpmnProcessID map[string]process.ProcessDefinitions
	MissingBpmnProcessIDs  []string
}

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

func (r processDefinitionSelectorValidationRequest) filterForBpmnProcessID(bpmnProcessID string) process.ProcessDefinitionFilter {
	return process.ProcessDefinitionFilter{
		BpmnProcessId:     bpmnProcessID,
		ProcessVersion:    r.ProcessVersion,
		ProcessVersionTag: r.ProcessVersionTag,
	}
}

func validateProcessDefinitionSelectors(ctx context.Context, cli process.API, req processDefinitionSelectorValidationRequest, opts ...options.FacadeOption) (processDefinitionSelectorValidationResult, error) {
	ids := normalizeSelectorBpmnProcessIDs(req.BpmnProcessIds)
	result := processDefinitionSelectorValidationResult{
		Request: processDefinitionSelectorValidationRequest{
			BpmnProcessIds:    ids,
			ProcessVersion:    req.ProcessVersion,
			ProcessVersionTag: req.ProcessVersionTag,
			Mode:              req.Mode,
		},
		MatchesByBpmnProcessID: make(map[string]process.ProcessDefinitions, len(ids)),
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
	}

	return result, nil
}

func processDefinitionSelectorNoPromptError(result processDefinitionSelectorValidationResult) error {
	if result.Valid() {
		return nil
	}
	return localPreconditionError(fmt.Errorf("%s", formatMissingProcessDefinitionSelectors(result)))
}

func formatMissingProcessDefinitionSelectors(result processDefinitionSelectorValidationResult) string {
	var b strings.Builder
	if len(result.MissingBpmnProcessIDs) == 1 {
		b.WriteString("no visible process definition matches the provided selector:\n")
	} else {
		b.WriteString("no visible process definitions match the provided selector(s):\n")
	}
	for _, id := range result.MissingBpmnProcessIDs {
		fmt.Fprintf(&b, "  bpmnProcessId: %s\n", id)
	}
	if result.Request.ProcessVersion != 0 {
		fmt.Fprintf(&b, "  processVersion: %d\n", result.Request.ProcessVersion)
	}
	if result.Request.ProcessVersionTag != "" {
		fmt.Fprintf(&b, "  processVersionTag: %s\n", result.Request.ProcessVersionTag)
	}
	b.WriteString("\n")
	if len(result.MissingBpmnProcessIDs) == 1 {
		b.WriteString("It may not exist, the version/tag/tenant may not match, or your credentials may not have access.")
	} else {
		b.WriteString("They may not exist, the version/tag/tenant may not match, or your credentials may not have access.")
	}
	return b.String()
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
