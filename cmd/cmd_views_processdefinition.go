// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// processDefinitionView renders a single process definition through the shared get-output contract.
func processDefinitionView(cmd *cobra.Command, item process.ProcessDefinition) error {
	return itemView(cmd, item, pickMode(), oneLinePD, func(it process.ProcessDefinition) string { return it.Key })
}

// listProcessDefinitionsView renders process-definition collections in human, JSON, or keys-only mode.
func listProcessDefinitionsView(cmd *cobra.Command, resp process.ProcessDefinitions) error {
	return listOrJSONFlat(cmd, resp, resp.Items, pickMode(), flatRowPD, func(it process.ProcessDefinition) string { return it.Key })
}

// processDefinitionDisplay owns the unstyled values that determine both an
// ordinary row and watch-mode change identity.
type processDefinitionDisplay struct {
	key                    string
	tenantID               string
	bpmnProcessID          string
	version                string
	statisticsPresent      bool
	active                 string
	completed              string
	canceled               string
	incidentCountSupported bool
	incidents              string
}

// processDefinitionChangedFields preserves value-level change identity for
// watch styling without making plain marker rendering depend on ANSI text.
type processDefinitionChangedFields struct {
	tenantID               bool
	bpmnProcessID          bool
	version                bool
	statisticsPresent      bool
	active                 bool
	completed              bool
	canceled               bool
	incidentCountSupported bool
	incidents              bool
}

// any reports whether an existing definition changed any displayed value.
func (f processDefinitionChangedFields) any() bool {
	return f.tenantID || f.bpmnProcessID || f.version || f.statisticsPresent ||
		f.active || f.completed || f.canceled || f.incidentCountSupported || f.incidents
}

// processDefinitionWatchRow combines one current display projection with its
// comparison marker and value-level change identity.
type processDefinitionWatchRow struct {
	display processDefinitionDisplay
	marker  byte
	changed processDefinitionChangedFields
}

// processDefinitionWatchCandidate is a complete refresh body plus the owned
// baseline that may be committed only after every output write succeeds.
type processDefinitionWatchCandidate struct {
	rows      []processDefinitionWatchRow
	updated   int
	new       int
	removed   int
	nextByKey map[string]processDefinitionDisplay
}

const (
	processDefinitionWatchBoldStart = "\x1b[1m"
	processDefinitionWatchBoldReset = "\x1b[0m"
)

// processDefinitionWatchView renders one complete refresh and returns the
// first write failure so unseen candidates never become the baseline.
func processDefinitionWatchView(cmd *cobra.Command, candidate processDefinitionWatchCandidate, bold bool) error {
	if cmd == nil {
		return nil
	}
	w := cmd.OutOrStdout()
	if err := writeProcessDefinitionWatchLine(w, "Changes: %d updated, %d new, %d removed   (* changed, + new)", candidate.updated, candidate.new, candidate.removed); err != nil {
		return err
	}
	for i, line := range formatProcessDefinitionWatchRows(candidate.rows, bold) {
		if err := writeProcessDefinitionWatchLine(w, "%c %s", candidate.rows[i].marker, line); err != nil {
			return err
		}
	}
	return writeProcessDefinitionWatchLine(w, "found: %d", len(candidate.rows))
}

// formatProcessDefinitionWatchRows computes alignment from plain values before
// adding ANSI controls around the changed value spans of updated rows.
func formatProcessDefinitionWatchRows(rows []processDefinitionWatchRow, bold bool) []string {
	plainRows := make([]flatRow, 0, len(rows))
	for _, row := range rows {
		plainRows = append(plainRows, row.display.flatRow())
	}
	widths := flatColumnWidths(plainRows)
	formatted := make([]string, 0, len(rows))
	for rowIndex, row := range rows {
		plainRow := plainRows[rowIndex]
		parts := make([]string, 0, len(plainRow))
		for columnIndex, plainColumn := range plainRow {
			if widths[columnIndex] == 0 {
				continue
			}
			part := formatProcessDefinitionWatchColumn(row, columnIndex, bold && row.marker == '*')
			if hasVisibleColumnAfter(widths, columnIndex) {
				part += strings.Repeat(" ", widths[columnIndex]-len(plainColumn))
			}
			parts = append(parts, part)
		}
		formatted = append(formatted, strings.TrimRight(strings.Join(parts, " "), " "))
	}
	return formatted
}

// formatProcessDefinitionWatchColumn decorates only semantic values; column
// labels, brackets, separators, and alignment padding remain unstyled.
func formatProcessDefinitionWatchColumn(row processDefinitionWatchRow, columnIndex int, bold bool) string {
	display := row.display
	switch columnIndex {
	case 0:
		return display.key
	case 1:
		return formatProcessDefinitionWatchValue(display.tenantID, bold && row.changed.tenantID)
	case 2:
		return formatProcessDefinitionWatchValue(display.bpmnProcessID, bold && row.changed.bpmnProcessID)
	case 3:
		return formatProcessDefinitionWatchValue(display.version, bold && row.changed.version)
	case 4:
		return formatProcessDefinitionWatchStatistics(display, row.changed, bold)
	default:
		return ""
	}
}

// formatProcessDefinitionWatchStatistics keeps each statistic independently
// stylable while leaving its compact labels and punctuation outside bold spans.
func formatProcessDefinitionWatchStatistics(display processDefinitionDisplay, changed processDefinitionChangedFields, bold bool) string {
	if !display.statisticsPresent {
		return ""
	}
	incidentTag := ""
	if display.incidentCountSupported {
		incidentTag = fmt.Sprintf(" inc:%s", formatProcessDefinitionWatchValue(display.incidents, bold && changed.incidents))
	}
	return fmt.Sprintf("[ac:%s cp:%s cx:%s%s]",
		formatProcessDefinitionWatchValue(display.active, bold && changed.active),
		formatProcessDefinitionWatchValue(display.completed, bold && changed.completed),
		formatProcessDefinitionWatchValue(display.canceled, bold && changed.canceled),
		incidentTag,
	)
}

// formatProcessDefinitionWatchValue wraps a nonempty changed value in the
// minimal bold span so visible width and neighboring text remain unchanged.
func formatProcessDefinitionWatchValue(value string, bold bool) string {
	if !bold || value == "" {
		return value
	}
	return processDefinitionWatchBoldStart + value + processDefinitionWatchBoldReset
}

// writeProcessDefinitionWatchLine keeps watch rendering fully error-aware.
func writeProcessDefinitionWatchLine(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format+"\n", args...)
	return err
}

// formatProcessDefinitionFlatRows returns aligned human rows for process-definition lists.
func formatProcessDefinitionFlatRows(items []process.ProcessDefinition) []string {
	rows := make([]flatRow, 0, len(items))
	for _, it := range items {
		rows = append(rows, flatRowPD(it))
	}
	return formatFlatRows(rows)
}

// oneLinePD formats a compact process-definition row for single-item human output.
func oneLinePD(it process.ProcessDefinition) string {
	return compactFlatRow(flatRowPD(it))
}

// flatRowPD mirrors the process-definition human order while allowing statistics to remain an optional tail column.
func flatRowPD(it process.ProcessDefinition) flatRow {
	return newProcessDefinitionDisplay(it).flatRow()
}

// newProcessDefinitionDisplay copies only values visible in process-definition
// rows, preventing later fixture or model mutation from changing the baseline.
func newProcessDefinitionDisplay(it process.ProcessDefinition) processDefinitionDisplay {
	vTag := ""
	if it.ProcessVersionTag != "" {
		vTag = "/" + it.ProcessVersionTag
	}
	display := processDefinitionDisplay{
		key:           it.Key,
		tenantID:      it.TenantId,
		bpmnProcessID: it.BpmnProcessId,
		version:       fmt.Sprintf("v%d%s", it.ProcessVersion, vTag),
	}
	if it.Statistics != nil {
		stats := it.Statistics
		display.statisticsPresent = true
		display.active = zeroAsMinus(stats.Active)
		display.completed = zeroAsMinus(stats.Completed)
		display.canceled = zeroAsMinus(stats.Canceled)
		display.incidentCountSupported = stats.IncidentCountSupported
		if stats.IncidentCountSupported {
			display.incidents = zeroAsMinus(stats.Incidents)
		}
	}
	return display
}

// flatRow reconstructs the established ordinary row bytes from owned display values.
func (d processDefinitionDisplay) flatRow() flatRow {
	row := flatRow{d.key, d.tenantID, d.bpmnProcessID, d.version}
	if d.statisticsPresent {
		incidentTag := ""
		if d.incidentCountSupported {
			incidentTag = fmt.Sprintf(" inc:%s", d.incidents)
		}
		row = append(row, fmt.Sprintf("[ac:%s cp:%s cx:%s%s]", d.active, d.completed, d.canceled, incidentTag))
	}
	return row
}

// changedFields compares semantic displayed values before alignment or styling.
func (d processDefinitionDisplay) changedFields(previous processDefinitionDisplay) processDefinitionChangedFields {
	return processDefinitionChangedFields{
		tenantID:               d.tenantID != previous.tenantID,
		bpmnProcessID:          d.bpmnProcessID != previous.bpmnProcessID,
		version:                d.version != previous.version,
		statisticsPresent:      d.statisticsPresent != previous.statisticsPresent,
		active:                 d.active != previous.active,
		completed:              d.completed != previous.completed,
		canceled:               d.canceled != previous.canceled,
		incidentCountSupported: d.incidentCountSupported != previous.incidentCountSupported,
		incidents:              d.incidents != previous.incidents,
	}
}
