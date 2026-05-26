// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incidentfilter

import (
	"strings"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	"github.com/grafvonb/c8volt/toolx"
)

var validErrorTypes = []string{
	string(camundav89.IncidentErrorTypeEnumADHOCSUBPROCESSNORETRIES),
	string(camundav89.IncidentErrorTypeEnumCALLEDDECISIONERROR),
	string(camundav89.IncidentErrorTypeEnumCALLEDELEMENTERROR),
	string(camundav89.IncidentErrorTypeEnumCONDITIONERROR),
	string(camundav89.IncidentErrorTypeEnumDECISIONEVALUATIONERROR),
	string(camundav89.IncidentErrorTypeEnumEXECUTIONLISTENERNORETRIES),
	string(camundav89.IncidentErrorTypeEnumEXTRACTVALUEERROR),
	string(camundav89.IncidentErrorTypeEnumFORMNOTFOUND),
	string(camundav89.IncidentErrorTypeEnumIOMAPPINGERROR),
	string(camundav89.IncidentErrorTypeEnumJOBNORETRIES),
	string(camundav89.IncidentErrorTypeEnumMESSAGESIZEEXCEEDED),
	string(camundav89.IncidentErrorTypeEnumRESOURCENOTFOUND),
	string(camundav89.IncidentErrorTypeEnumTASKLISTENERNORETRIES),
	string(camundav89.IncidentErrorTypeEnumUNHANDLEDERROREVENT),
	string(camundav89.IncidentErrorTypeEnumUNKNOWN),
	string(camundav89.IncidentErrorTypeEnumUNSPECIFIED),
}

var validStates = []string{
	"active",
	"pending",
	"resolved",
	"migrated",
	"unknown",
	"all",
}

// ValidErrorTypes returns incident error type values from the generated Camunda enum.
func ValidErrorTypes() []string {
	out := make([]string, len(validErrorTypes))
	copy(out, validErrorTypes)
	return out
}

func ValidErrorTypesString() string {
	return strings.Join(validErrorTypes, ", ")
}

func ValidStatesString() string {
	return strings.Join(validStates, ", ")
}

func NormalizeState(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", true
	}
	return toolx.CanonicalEnumString(value, validStates)
}

func NormalizeErrorType(value string) (string, bool) {
	if strings.TrimSpace(value) == "" {
		return "", true
	}
	return toolx.CanonicalEnumString(value, validErrorTypes)
}

func ErrorTypeMatches(want string, got string) bool {
	normalized, ok := NormalizeErrorType(want)
	if !ok || normalized == "" {
		return ok
	}
	return strings.EqualFold(strings.TrimSpace(got), normalized)
}

func ErrorMessageContains(want string, got string) bool {
	needle := strings.TrimSpace(want)
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(got), strings.ToLower(needle))
}
