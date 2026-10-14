// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

// Package assets embeds release-validation policy files so callers do not
// depend on their current working directory.
package assets

import _ "embed"

var (
	// ReleaseCoverageJSON is the checked-in release coverage catalog.
	//go:embed release-coverage.json
	ReleaseCoverageJSON []byte

	// ReleaseGapsJSON is the checked-in accepted prerequisite-gap catalog.
	//go:embed release-gaps.json
	ReleaseGapsJSON []byte
)
