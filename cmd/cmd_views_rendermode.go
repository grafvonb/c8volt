// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

// RenderMode selects how view commands render command results.
type RenderMode int

const (
	RenderModeJSON RenderMode = iota
	RenderModeOneLine
	RenderModeKeysOnly
	RenderModeTree
)

// String returns the CLI-facing name for the render mode.
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

// pickMode resolves the active view mode from the current output flags.
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

// machineReadableModeEnabled reports whether the selected mode is intended for structured consumers.
func machineReadableModeEnabled(mode RenderMode) bool {
	return mode == RenderModeJSON
}

// renderOutputLine writes command output with one trailing newline.
func renderOutputLine(cmd *cobra.Command, format string, args ...any) {
	cmd.Println(strings.TrimRight(fmt.Sprintf(format, args...), "\n"))
}

// renderTerminalRepaint emits the ANSI controls used by watch views to replace the visible result area.
func renderTerminalRepaint(cmd *cobra.Command) error {
	if cmd == nil {
		return nil
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), "\x1b[H\x1b[2J")
	return err
}

// renderHumanLine writes command output through the activity-aware renderer.
func renderHumanLine(cmd *cobra.Command, format string, args ...any) {
	renderHumanLogLine(cmd, false, format, args...)
}

// renderHumanWarningLine writes warnings through the activity-aware renderer.
func renderHumanWarningLine(cmd *cobra.Command, format string, args ...any) {
	renderHumanLogLine(cmd, true, "%s", normalizeWarningText(fmt.Sprintf(format, args...)))
}

func normalizeWarningText(msg string) string {
	msg = strings.TrimSpace(msg)
	lower := strings.ToLower(msg)
	for _, prefix := range []string{"warning:", "warning -", "warning"} {
		if lower == prefix {
			return ""
		}
		if strings.HasPrefix(lower, prefix+" ") {
			return strings.TrimSpace(msg[len(prefix):])
		}
	}
	return msg
}

// renderHumanLogLine routes output through the logger when command context provides one.
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

// itemView renders a single item using shared JSON, key-only, and one-line view conventions.
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

// listOrJSON renders a collection in the selected mode while keeping JSON output on the full response payload.
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

// listOrJSONFlat keeps machine modes unchanged while letting list views align columns from the whole result set.
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
