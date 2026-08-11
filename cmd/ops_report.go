// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// validateOpsWorkflowReportFlags enforces shared report flag dependency and format validation.
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

// opsWorkflowReportWriteModeForConfirmedMutation allows confirmed mutations to replace earlier planning reports.
func opsWorkflowReportWriteModeForConfirmedMutation(confirmed bool) OpsWorkflowReportWriteMode {
	if confirmed {
		return OpsWorkflowReportOverwriteExisting
	}
	return OpsWorkflowReportPreserveExisting
}

// validateOpsWorkflowReportPathForPlanning prevents a dry-run or pre-confirmation report from overwriting an existing file.
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

// writeOpsWorkflowReportFile writes report bytes with the selected overwrite policy.
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
