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
	Status:     "final",
	Tag:        "8.10.3",
	Commit:     "3961b79c3fdaf43c48691ed77672d6db42d0abcc",
	Repository: "https://github.com/camunda/camunda.git",
	SourceSpec: "zeebe/gateway-protocol/src/main/proto/v2/rest-api.yaml",
}

// V810Baseline returns the single active Camunda 8.10 generated-client source baseline.
func V810Baseline() CamundaBaseline {
	return v810Baseline
}
