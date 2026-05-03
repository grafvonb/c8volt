// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

type RenderMode int

const (
	RenderModeJSON RenderMode = iota
	RenderModeOneLine
	RenderModeKeysOnly
	RenderModeTree
)

func (m RenderMode) String() string {
	switch m {
	case RenderModeJSON:
		return "json"
	case RenderModeOneLine:
		return "one-line"
	case RenderModeKeysOnly:
		return "keys-only"
	case RenderModeTree:
		return "tree"
	default:
		return fmt.Sprintf("unknown(%d)", m)
	}
}

func pickMode() RenderMode {
	switch {
	case flagViewAsJson:
		return RenderModeJSON
	case flagViewKeysOnly:
		return RenderModeKeysOnly
	default:
		return RenderModeOneLine
	}
}

func machineReadableModeEnabled(mode RenderMode) bool {
	return mode == RenderModeJSON
}

// renderOutputLine writes command output with one trailing newline.
func renderOutputLine(cmd *cobra.Command, format string, args ...any) {
	cmd.Println(strings.TrimRight(fmt.Sprintf(format, args...), "\n"))
}

// renderHumanLine writes human-readable command output through the activity-aware renderer.
func renderHumanLine(cmd *cobra.Command, format string, args ...any) {
	renderHumanLogLine(cmd, false, format, args...)
}

// renderHumanWarningLine writes human-readable warnings through the activity-aware renderer.
func renderHumanWarningLine(cmd *cobra.Command, format string, args ...any) {
	renderHumanLogLine(cmd, true, format, args...)
}

// renderHumanLogLine routes human output through the logger when command context provides one.
func renderHumanLogLine(cmd *cobra.Command, warn bool, format string, args ...any) {
	msg := strings.TrimRight(fmt.Sprintf(format, args...), "\n")
	log, err := logging.FromContext(cmd.Context())
	if err == nil {
		for _, line := range strings.Split(msg, "\n") {
			if warn {
				log.Warn(line)
			} else {
				log.Info(line)
			}
		}
		return
	}
	cmd.Println(msg)
}

func itemView[Item any](cmd *cobra.Command, item Item, mode RenderMode, oneLine func(Item) string, keyOf func(Item) string) error {
	switch mode {
	case RenderModeJSON:
		return renderJSONPayload(cmd, mode, item)
	case RenderModeKeysOnly:
		renderOutputLine(cmd, "%s", keyOf(item))
	default:
		renderOutputLine(cmd, "%s", strings.TrimSpace(oneLine(item)))
	}
	return nil
}

func listOrJSON[Resp any, Item any](cmd *cobra.Command, resp Resp, items []Item, mode RenderMode, oneLine func(Item) string, keyOf func(Item) string) error {
	switch mode {
	case RenderModeJSON:
		return renderJSONPayload(cmd, mode, resp)
	case RenderModeKeysOnly:
		for _, it := range items {
			renderOutputLine(cmd, "%s", keyOf(it))
		}
	default: // RenderModeOneLine
		for _, it := range items {
			renderOutputLine(cmd, "%s", strings.TrimSpace(oneLine(it)))
		}
		renderOutputLine(cmd, "found: %d", len(items))
	}
	return nil
}

type flatRow []string

// listOrJSONFlat keeps machine modes unchanged while letting human list views align columns from the whole result set.
func listOrJSONFlat[Resp any, Item any](cmd *cobra.Command, resp Resp, items []Item, mode RenderMode, rowOf func(Item) flatRow, keyOf func(Item) string) error {
	switch mode {
	case RenderModeJSON:
		return renderJSONPayload(cmd, mode, resp)
	case RenderModeKeysOnly:
		for _, it := range items {
			renderOutputLine(cmd, "%s", keyOf(it))
		}
	default: // RenderModeOneLine
		rows := make([]flatRow, 0, len(items))
		for _, it := range items {
			rows = append(rows, rowOf(it))
		}
		for _, line := range formatFlatRows(rows) {
			renderOutputLine(cmd, "%s", line)
		}
		renderOutputLine(cmd, "found: %d", len(items))
	}
	return nil
}

// formatFlatRows preserves row order and field order while padding only to widths observed in the current list.
func formatFlatRows(rows []flatRow) []string {
	widths := flatColumnWidths(rows)
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		parts := make([]string, 0, len(row))
		for i, col := range row {
			if widths[i] == 0 {
				continue
			}
			part := col
			if hasVisibleColumnAfter(widths, i) {
				part += strings.Repeat(" ", widths[i]-len(col))
			}
			parts = append(parts, part)
		}
		out = append(out, strings.TrimRight(strings.Join(parts, " "), " "))
	}
	return out
}

// hasVisibleColumnAfter prevents all-empty trailing optional columns from creating scan-hostile gaps.
func hasVisibleColumnAfter(widths []int, index int) bool {
	for i := index + 1; i < len(widths); i++ {
		if widths[i] > 0 {
			return true
		}
	}
	return false
}

// flatColumnWidths derives display widths without truncating long values such as BPMN process IDs.
func flatColumnWidths(rows []flatRow) []int {
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	widths := make([]int, maxCols)
	for _, row := range rows {
		for i, col := range row {
			if len(col) > widths[i] {
				widths[i] = len(col)
			}
		}
	}
	return widths
}

// compactFlatRow is used by single-row and walk views where alignment padding would obscure the path shape.
func compactFlatRow(row flatRow) string {
	parts := make([]string, 0, len(row))
	for _, col := range row {
		if col != "" {
			parts = append(parts, col)
		}
	}
	return strings.Join(parts, " ")
}
