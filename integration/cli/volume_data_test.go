// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

type volumeDataset struct {
	Marker                        string          `json:"marker"`
	Profile                       string          `json:"profile"`
	CamundaVersion                string          `json:"camundaVersion,omitempty"`
	RequestedCount                int             `json:"requestedCount"`
	PositiveFixturePath           string          `json:"positiveFixturePath,omitempty"`
	PositiveBpmnProcessID         string          `json:"positiveBpmnProcessId,omitempty"`
	PositiveProcessDefinitionKeys []string        `json:"positiveProcessDefinitionKeys,omitempty"`
	PositiveProcessInstanceKeys   []string        `json:"positiveProcessInstanceKeys,omitempty"`
	NegativeFixturePath           string          `json:"negativeFixturePath,omitempty"`
	NegativeBpmnProcessID         string          `json:"negativeBpmnProcessId,omitempty"`
	NegativeProcessDefinitionKeys []string        `json:"negativeProcessDefinitionKeys,omitempty"`
	NegativeProcessInstanceKeys   []string        `json:"negativeProcessInstanceKeys,omitempty"`
	PositiveSelectors             []string        `json:"positiveSelectors,omitempty"`
	NegativeSelectors             []string        `json:"negativeSelectors,omitempty"`
	RetainedResources             []string        `json:"retainedResources,omitempty"`
	CleanupRecords                []cleanupRecord `json:"cleanupRecords,omitempty"`
}

func (d volumeDataset) allProcessInstanceKeys() []string {
	keys := make([]string, 0, len(d.PositiveProcessInstanceKeys)+len(d.NegativeProcessInstanceKeys))
	keys = append(keys, d.PositiveProcessInstanceKeys...)
	keys = append(keys, d.NegativeProcessInstanceKeys...)
	return keys
}

func classifyVolumeOwnership(seedKeys []string, observedKeys []string) []string {
	if len(seedKeys) == 0 {
		if len(observedKeys) == 0 {
			return nil
		}
		return []string{volumeDataPreexisting}
	}
	if len(observedKeys) == 0 {
		return []string{volumeDataSeeded}
	}
	seeded := map[string]struct{}{}
	for _, key := range seedKeys {
		if key != "" {
			seeded[key] = struct{}{}
		}
	}
	hasSeeded := false
	hasPreexisting := false
	for _, key := range observedKeys {
		if _, ok := seeded[key]; ok {
			hasSeeded = true
		} else if key != "" {
			hasPreexisting = true
		}
	}
	var out []string
	if hasSeeded {
		out = append(out, volumeDataSeeded)
	}
	if hasPreexisting {
		out = append(out, volumeDataPreexisting)
	}
	return out
}

func TestVolumeOwnershipClassification(t *testing.T) {
	cases := []struct {
		name string
		seed []string
		got  []string
		want []string
	}{
		{name: "seeded only", seed: []string{"1"}, got: []string{"1"}, want: []string{volumeDataSeeded}},
		{name: "dirty mixed", seed: []string{"1"}, got: []string{"1", "2"}, want: []string{volumeDataSeeded, volumeDataPreexisting}},
		{name: "preexisting only", seed: nil, got: []string{"2"}, want: []string{volumeDataPreexisting}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertStringSlicesEqual(t, classifyVolumeOwnership(tc.seed, tc.got), tc.want)
		})
	}
}
