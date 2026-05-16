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

// Verifies deployed process definitions use the compact pd-first grammar and align the scannable columns.
func TestListProcessDefinitionDeploymentsView_AlignsCompactPDGrammar(t *testing.T) {
	cmd := &cobra.Command{Use: "deploy"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := listProcessDefinitionDeploymentsView(cmd, []resource.ProcessDefinitionDeployment{
		{
			Key:               "deployment-a",
			DefinitionKey:     "1",
			DefinitionId:      "Short",
			DefinitionVersion: 1,
			ResourceName:      "processdefinitions/Short.bpmn",
			TenantId:          "<default>",
		},
		{
			Key:               "deployment-b",
			DefinitionKey:     "22",
			DefinitionId:      "MuchLongerProcess",
			DefinitionVersion: 12,
			ResourceName:      "processdefinitions/MuchLongerProcess.bpmn",
			TenantId:          "tenant",
		},
	})

	require.NoError(t, err)
	require.Equal(t, ""+
		"pd 1  Short             v1  <default>; deploy deployment-a; resource processdefinitions/Short.bpmn\n"+
		"pd 22 MuchLongerProcess v12 tenant;    deploy deployment-b; resource processdefinitions/MuchLongerProcess.bpmn\n"+
		"found: 2\n", buf.String())
}

func TestOneLinePDDeploy_UsesCompactPDGrammarWithoutPadding(t *testing.T) {
	line := oneLinePDDeploy(resource.ProcessDefinitionDeployment{
		Key:               "deployment-a",
		DefinitionKey:     "1",
		DefinitionId:      "Short",
		DefinitionVersion: 1,
		ResourceName:      "processdefinitions/Short.bpmn",
		TenantId:          "<default>",
	})

	require.Equal(t, "pd 1 Short v1 <default>; deploy deployment-a; resource processdefinitions/Short.bpmn", line)
}
