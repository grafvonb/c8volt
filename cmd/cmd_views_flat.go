// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
)

// flatRow represents a single display row before optional column alignment is applied.
type flatRow []string

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

// zeroAsMinus keeps optional count columns compact in flat-row renderers.
func zeroAsMinus(v int64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", v)
}
