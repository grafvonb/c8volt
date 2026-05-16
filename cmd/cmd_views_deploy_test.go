// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Verifies deployed process definitions render like get pd rows and align the scannable columns.
func TestListProcessDefinitionDeploymentsView_AlignsLikeProcessDefinitions(t *testing.T) {
	cmd := &cobra.Command{Use: "deploy"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := listProcessDefinitionDeploymentsView(cmd, []resource.ProcessDefinitionDeployment{
		{
			Key:               "deployment-a",
			DefinitionKey:     "22",
			DefinitionId:      "MuchLongerProcess",
			DefinitionVersion: 12,
			VersionTag:        "v1.0.0",
			ResourceName:      "processdefinitions/MuchLongerProcess.bpmn",
			TenantId:          "tenant",
		},
		{
			Key:               "deployment-b",
			DefinitionKey:     "1",
			DefinitionId:      "Short",
			DefinitionVersion: 1,
			VersionTag:        "v1.1.0",
			ResourceName:      "processdefinitions/Short.bpmn",
			TenantId:          "<default>",
		},
		{
			Key:               "deployment-c",
			DefinitionKey:     "333",
			DefinitionId:      "AnotherProcess",
			DefinitionVersion: 2,
			ResourceName:      "processdefinitions/AnotherProcess.bpmn",
			TenantId:          "<default>",
		},
	})

	require.NoError(t, err)
	require.Equal(t, ""+
		"333 <default> AnotherProcess    v2\n"+
		"22  tenant    MuchLongerProcess v12/v1.0.0\n"+
		"1   <default> Short             v1/v1.1.0\n"+
		"found: 3\n", buf.String())
}

func TestOneLinePDDeploy_UsesProcessDefinitionGrammarWithoutPadding(t *testing.T) {
	line := oneLinePDDeploy(resource.ProcessDefinitionDeployment{
		Key:               "deployment-a",
		DefinitionKey:     "1",
		DefinitionId:      "Short",
		DefinitionVersion: 1,
		VersionTag:        "v1.1.0",
		ResourceName:      "processdefinitions/Short.bpmn",
		TenantId:          "<default>",
	})

	require.Equal(t, "1 <default> Short v1/v1.1.0", line)
}
