// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

type opsHumanNoticeLevel string

const (
	opsHumanNoticeLevelWarning opsHumanNoticeLevel = "warning"

	opsHumanNoticeOrphanParentMissing = "orphan_parent_missing"
)

type opsHumanNotice struct {
	Level opsHumanNoticeLevel
	Code  string
	Text  string
}

type opsHumanNoticeFilter func(opsHumanNotice) bool

func renderOpsHumanNotices(cmd *cobra.Command, notices []opsHumanNotice, filter opsHumanNoticeFilter) {
	for _, notice := range notices {
		if notice.Text == "" {
			continue
		}
		if filter != nil && !filter(notice) {
			continue
		}
		switch notice.Level {
		case opsHumanNoticeLevelWarning:
			renderHumanWarningLine(cmd, "%s", notice.Text)
		default:
			renderHumanLine(cmd, "%s", notice.Text)
		}
	}
}
