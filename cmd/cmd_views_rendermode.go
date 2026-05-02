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
