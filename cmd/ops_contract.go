// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// OpsWorkflowReportWriteMode controls whether an ops report may replace an existing file.
type OpsWorkflowReportWriteMode int

const (
	// OpsWorkflowReportPreserveExisting creates a report only when the path is unused.
	OpsWorkflowReportPreserveExisting OpsWorkflowReportWriteMode = iota
	// OpsWorkflowReportOverwriteExisting allows a confirmed operation to replace a previous report.
	OpsWorkflowReportOverwriteExisting
)

// OpsWorkflowReport is the workflow-neutral report shape ops commands can render or encode.
type OpsWorkflowReport struct {
	SchemaVersion   string                  `json:"schemaVersion,omitempty"`
	CommandName     string                  `json:"commandName,omitempty"`
	Workflow        string                  `json:"workflow,omitempty"`
	StartedAt       time.Time               `json:"startedAt,omitempty"`
	FinishedAt      time.Time               `json:"finishedAt,omitempty"`
	Duration        string                  `json:"duration,omitempty"`
	DryRun          bool                    `json:"dryRun,omitempty"`
	C8voltVersion   string                  `json:"c8voltVersion,omitempty"`
	CamundaVersion  string                  `json:"camundaVersion,omitempty"`
	ProfileIdentity string                  `json:"profileIdentity,omitempty"`
	Steps           []OpsWorkflowReportStep `json:"steps,omitempty"`
	Errors          []string                `json:"errors,omitempty"`
	Outcome         string                  `json:"outcome,omitempty"`
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
		return OpsWorkflowReportFormatMarkdown, nil
	}
}

func validateOpsWorkflowReportFlags(reportPath string, requested OpsWorkflowReportFormat) error {
	if requested != "" && reportPath == "" {
		return missingDependentFlagsf("--report-format requires --report-file")
	}
	if reportPath == "" {
		return nil
	}
	_, err := opsWorkflowReportFormatForPath(reportPath, requested)
	if err != nil {
		return invalidFlagValuef("%v", err)
	}
	return nil
}

func opsWorkflowReportWriteModeForConfirmedMutation(confirmed bool) OpsWorkflowReportWriteMode {
	if confirmed {
		return OpsWorkflowReportOverwriteExisting
	}
	return OpsWorkflowReportPreserveExisting
}

func validateOpsWorkflowReportPathForPlanning(path string, mode OpsWorkflowReportWriteMode) error {
	if path == "" || mode == OpsWorkflowReportOverwriteExisting {
		return nil
	}
	_, err := os.Stat(path)
	if err == nil {
		return localPreconditionError(fmt.Errorf("report file already exists: %s", path))
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return localPreconditionError(err)
}

func writeOpsWorkflowReportFile(path string, data []byte, mode OpsWorkflowReportWriteMode) error {
	flags := os.O_WRONLY | os.O_CREATE
	if mode == OpsWorkflowReportOverwriteExisting {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("report file already exists: %s", path)
		}
		return err
	}
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	return nil
}
