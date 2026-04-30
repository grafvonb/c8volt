// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessDefinitionFilterString_RendersOnlyActiveFields(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessDefinitionFilter{}))

	got := fmt.Sprintf("%+v", ProcessDefinitionFilter{
		Key:               "2251799813685960",
		BpmnProcessId:     "EnquiryProcess",
		ProcessVersion:    2,
		ProcessVersionTag: "2.0.0",
	})

	require.Equal(t, `{key="2251799813685960", bpmnProcessId="EnquiryProcess", processVersion=2, processVersionTag="2.0.0"}`, got)
}

func TestProcessInstanceFilterString_RendersOptionalBooleansConsistently(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessInstanceFilter{}))
	require.NotContains(t, fmt.Sprintf("%+v", ProcessInstanceFilter{}), "<nil>")

	hasParent := true
	hasIncident := false
	got := fmt.Sprintf("%+v", ProcessInstanceFilter{
		BpmnProcessId: "order",
		HasParent:     &hasParent,
		HasIncident:   &hasIncident,
	})

	require.Equal(t, `{bpmnProcessId="order", hasParent=true, hasIncident=false}`, got)
}
