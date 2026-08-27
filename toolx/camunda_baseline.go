// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

// CamundaBaseline describes the active generated-source baseline behind a
// stable operator-facing Camunda version identity.
type CamundaBaseline struct {
	Version    CamundaVersion
	Status     string
	Tag        string
	Commit     string
	Repository string
	SourceSpec string
}

var v810Baseline = CamundaBaseline{
	Version:    V810,
	Status:     "prerelease",
	Tag:        "8.10.0-alpha4",
	Commit:     "4a76f06c9df8ad0a6c64fe88b92c618c2ebf2ef6",
	Repository: "https://github.com/camunda/camunda.git",
	SourceSpec: "zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml",
}

// V810Baseline returns the single active Camunda 8.10 generated-client source baseline.
func V810Baseline() CamundaBaseline {
	return v810Baseline
}
