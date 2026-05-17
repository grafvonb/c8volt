// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/toolx"
)

func embeddedFilesForCamundaVersion(files []string, version toolx.CamundaVersion) []string {
	prefix := version.FilePrefix()
	var out []string
	for _, f := range files {
		if strings.Contains(f, prefix) {
			out = append(out, f)
		}
	}
	return out
}
