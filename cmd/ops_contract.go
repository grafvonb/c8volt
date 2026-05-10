// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
)

// OpsWorkflowStepStatus is the shared status vocabulary for future ops workflow report steps.
type OpsWorkflowStepStatus string

const (
	// OpsWorkflowStepStatusPlanned means the step was selected but not submitted.
	OpsWorkflowStepStatusPlanned OpsWorkflowStepStatus = "planned"
	// OpsWorkflowStepStatusSkipped means the step intentionally did not run.
	OpsWorkflowStepStatusSkipped OpsWorkflowStepStatus = "skipped"
	// OpsWorkflowStepStatusSubmitted means the operation was submitted but not confirmed.
	OpsWorkflowStepStatusSubmitted OpsWorkflowStepStatus = "submitted"
	// OpsWorkflowStepStatusConfirmed means the requested outcome was verified.
	OpsWorkflowStepStatusConfirmed OpsWorkflowStepStatus = "confirmed"
	// OpsWorkflowStepStatusConfirmationFailed means a submitted operation could not be verified.
	OpsWorkflowStepStatusConfirmationFailed OpsWorkflowStepStatus = "confirmation_failed"
	// OpsWorkflowStepStatusBlocked means the step could not proceed because a precondition was not met.
	OpsWorkflowStepStatusBlocked OpsWorkflowStepStatus = "blocked"
	// OpsWorkflowStepStatusFailed means step execution failed.
	OpsWorkflowStepStatusFailed OpsWorkflowStepStatus = "failed"
)

// String returns the stable report token for the step status.
func (s OpsWorkflowStepStatus) String() string {
	return string(s)
}

// IsValid reports whether the status belongs to the shared ops workflow vocabulary.
func (s OpsWorkflowStepStatus) IsValid() bool {
	switch s {
	case OpsWorkflowStepStatusPlanned,
		OpsWorkflowStepStatusSkipped,
		OpsWorkflowStepStatusSubmitted,
		OpsWorkflowStepStatusConfirmed,
		OpsWorkflowStepStatusConfirmationFailed,
		OpsWorkflowStepStatusBlocked,
		OpsWorkflowStepStatusFailed:
		return true
	default:
		return false
	}
}

// opsWorkflowStepStatuses returns statuses in the canonical report ordering used by tests and docs.
func opsWorkflowStepStatuses() []OpsWorkflowStepStatus {
	return []OpsWorkflowStepStatus{
		OpsWorkflowStepStatusPlanned,
		OpsWorkflowStepStatusSkipped,
		OpsWorkflowStepStatusSubmitted,
		OpsWorkflowStepStatusConfirmed,
		OpsWorkflowStepStatusConfirmationFailed,
		OpsWorkflowStepStatusBlocked,
		OpsWorkflowStepStatusFailed,
	}
}

// OpsWorkflowReportFormat is the output format contract for future ops workflow report files.
type OpsWorkflowReportFormat string

const (
	// OpsWorkflowReportFormatMarkdown renders a human-readable Markdown report.
	OpsWorkflowReportFormatMarkdown OpsWorkflowReportFormat = "markdown"
	// OpsWorkflowReportFormatJSON renders a deterministic JSON report.
	OpsWorkflowReportFormatJSON OpsWorkflowReportFormat = "json"
)

// String returns the stable report-format token.
func (f OpsWorkflowReportFormat) String() string {
	return string(f)
}

// IsValid reports whether the format is supported by the shared ops report contract.
func (f OpsWorkflowReportFormat) IsValid() bool {
	switch f {
	case OpsWorkflowReportFormatMarkdown, OpsWorkflowReportFormatJSON:
		return true
	default:
		return false
	}
}

// OpsWorkflowReport is the workflow-neutral report shape future ops commands can render or encode.
type OpsWorkflowReport struct {
	Workflow string                  `json:"workflow"`
	DryRun   bool                    `json:"dryRun,omitempty"`
	Steps    []OpsWorkflowReportStep `json:"steps,omitempty"`
}

// OpsWorkflowReportStep captures one workflow step without embedding resource-specific API details.
type OpsWorkflowReportStep struct {
	Name    string                `json:"name"`
	Target  string                `json:"target,omitempty"`
	Status  OpsWorkflowStepStatus `json:"status"`
	Message string                `json:"message,omitempty"`
}

// opsWorkflowReportFormatForPath infers a report format from the output path unless a valid format was requested.
func opsWorkflowReportFormatForPath(reportPath string, requested OpsWorkflowReportFormat) (OpsWorkflowReportFormat, error) {
	if requested != "" {
		if !requested.IsValid() {
			return "", fmt.Errorf("unsupported ops workflow report format %q", requested)
		}
		return requested, nil
	}

	switch strings.ToLower(filepath.Ext(reportPath)) {
	case ".json":
		return OpsWorkflowReportFormatJSON, nil
	case ".md", ".markdown", "":
		return OpsWorkflowReportFormatMarkdown, nil
	default:
		return "", fmt.Errorf("unsupported ops workflow report file extension %q", filepath.Ext(reportPath))
	}
}
