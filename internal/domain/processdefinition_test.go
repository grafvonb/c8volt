// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProcessDefinitionFilterString_RendersOnlyActiveFields verifies debug output only includes configured filters.
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

// TestProcessDefinitionStatisticsFilterString_RendersOnlyActiveFields verifies statistics filters use the shared debug format.
func TestProcessDefinitionStatisticsFilterString_RendersOnlyActiveFields(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessDefinitionStatisticsFilter{}))
	require.Equal(t, `{tenantId="tenant-a"}`, fmt.Sprintf("%+v", ProcessDefinitionStatisticsFilter{TenantId: "tenant-a"}))
}
