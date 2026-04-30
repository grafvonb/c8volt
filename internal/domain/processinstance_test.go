// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

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
		TenantId:          "tenant-a",
		ProcessVersion:    2,
		ProcessVersionTag: "2.0.0",
		IsLatestVersion:   true,
	})

	require.Equal(t, `{bpmnProcessId="EnquiryProcess", key="2251799813685960", tenantId="tenant-a", processVersion=2, processVersionTag="2.0.0", isLatestVersion=true}`, got)
}

func TestProcessDefinitionStatisticsFilterString_RendersOnlyActiveFields(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessDefinitionStatisticsFilter{}))
	require.Equal(t, `{tenantId="tenant-a"}`, fmt.Sprintf("%+v", ProcessDefinitionStatisticsFilter{TenantId: "tenant-a"}))
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
