// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"fmt"
	"strings"
)

// CascadeProcessIDs names the version-matched process identities used by the
// parent/call-activity real-state scenarios.
type CascadeProcessIDs struct {
	Parent string `json:"parent"`
	Mid    string `json:"mid"`
	Leaf   string `json:"leaf"`
}

// RealStateMinor accepts the Camunda minors supported by live real-state
// validation. Camunda 8.8 remains baseline/volume-only.
func RealStateMinor(value string) (string, error) {
	minor, err := CanonicalMinor(value)
	if err != nil {
		return "", err
	}
	if minor == "8.8" {
		return "", fmt.Errorf("Camunda %s does not support real-state release validation", minor)
	}
	return minor, nil
}

// EmbeddedFixturePrefix maps a canonical supported minor to its checked-in
// embedded BPMN filename and process-ID prefix.
func EmbeddedFixturePrefix(value string) (string, error) {
	minor, err := CanonicalMinor(value)
	if err != nil {
		return "", err
	}
	switch minor {
	case "8.8":
		return "C88_", nil
	case "8.9":
		return "C89_", nil
	case "8.10":
		return "C810_", nil
	default:
		return "", fmt.Errorf("unsupported embedded fixture version %q", value)
	}
}

// SelectEmbeddedFixture finds one version-matched fixture path and reports a
// missing fixture explicitly instead of silently falling back to another minor.
func SelectEmbeddedFixture(value string, basename string, files []string) (string, error) {
	prefix, err := EmbeddedFixturePrefix(value)
	if err != nil {
		return "", err
	}
	wantSuffix := prefix + strings.TrimSpace(basename)
	for _, file := range files {
		if strings.HasSuffix(file, wantSuffix) {
			return file, nil
		}
	}
	return "", fmt.Errorf("no embedded %s fixture found for Camunda version %q", basename, value)
}

// RealStateCascadeProcessIDs derives the exact parent, mid, and leaf identities
// used by the selected minor's checked-in cascade fixture closure.
func RealStateCascadeProcessIDs(value string) (CascadeProcessIDs, error) {
	if _, err := RealStateMinor(value); err != nil {
		return CascadeProcessIDs{}, err
	}
	prefix, err := EmbeddedFixturePrefix(value)
	if err != nil {
		return CascadeProcessIDs{}, err
	}
	return CascadeProcessIDs{
		Parent: prefix + "MultipleSubProcessesParent",
		Mid:    prefix + "SimpleParent",
		Leaf:   prefix + "SimpleUserTask",
	}, nil
}
