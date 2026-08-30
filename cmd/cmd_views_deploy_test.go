// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Verifies deployed process definitions render as compact operation logs instead of search result rows.
func TestListProcessDefinitionDeploymentsView_RendersDeploymentLogs(t *testing.T) {
	prevNoWait := flagNoWait
	prevVerbose := flagVerbose
	flagNoWait = false
	flagVerbose = false
	t.Cleanup(func() {
		flagNoWait = prevNoWait
		flagVerbose = prevVerbose
	})

	cmd := &cobra.Command{Use: "deploy"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := listProcessDefinitionDeploymentsView(cmd, []resource.ProcessDefinitionDeployment{
		{
			Key:               "deployment-a",
			DefinitionKey:     "22",
			DefinitionId:      "MuchLongerProcess",
			DefinitionVersion: 12,
			ResourceName:      "processdefinitions/MuchLongerProcess.bpmn",
			TenantId:          "<default>",
		},
		{
			Key:               "deployment-b",
			DefinitionKey:     "1",
			DefinitionId:      "Short",
			DefinitionVersion: 1,
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
		"pd 333 AnotherProcess    v2  <default>; deployed\n"+
		"pd 22  MuchLongerProcess v12 <default>; deployed\n"+
		"pd 1   Short             v1  <default>; deployed\n"+
		"pd deploy done; deployed 3, tenant <default>\n", buf.String())
}

func TestListProcessDefinitionDeploymentsView_RendersVerboseResourceLines(t *testing.T) {
	prevNoWait := flagNoWait
	prevVerbose := flagVerbose
	flagNoWait = true
	flagVerbose = true
	t.Cleanup(func() {
		flagNoWait = prevNoWait
		flagVerbose = prevVerbose
	})

	cmd := &cobra.Command{Use: "deploy"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	err := listProcessDefinitionDeploymentsView(cmd, []resource.ProcessDefinitionDeployment{{
		Key:               "deployment-a",
		DefinitionKey:     "1",
		DefinitionId:      "Short",
		DefinitionVersion: 1,
		ResourceName:      "processdefinitions/Short.bpmn",
		TenantId:          "<default>",
	}})

	require.NoError(t, err)
	require.Equal(t, ""+
		"pd 1 Short v1 <default>; submitted\n"+
		"  resource: processdefinitions/Short.bpmn\n"+
		"pd deploy done; submitted 1, tenant <default>, deployment deployment-a\n", buf.String())
}

func TestOneLinePDDeploy_UsesMutationLogGrammarWithoutPadding(t *testing.T) {
	prevNoWait := flagNoWait
	flagNoWait = false
	t.Cleanup(func() {
		flagNoWait = prevNoWait
	})

	line := oneLinePDDeploy(resource.ProcessDefinitionDeployment{
		Key:               "deployment-a",
		DefinitionKey:     "1",
		DefinitionId:      "Short",
		DefinitionVersion: 1,
		ResourceName:      "processdefinitions/Short.bpmn",
		TenantId:          "<default>",
	})

	require.Equal(t, "pd 1 Short v1 <default>; deployed", line)
}

func TestListProcessDefinitionDeploymentsView_JSONEnvelopeIncludesAttachedTenantContext(t *testing.T) {
	resetTenantContextRenderFlags(t)
	prevNoWait := flagNoWait
	flagNoWait = true
	t.Cleanup(func() {
		flagNoWait = prevNoWait
	})

	cmd := &cobra.Command{Use: "deploy"}
	setCommandMutation(cmd, CommandMutationStateChanging)
	setContractSupport(cmd, ContractSupportFull)
	attachTenantContext(cmd, withTenantContextEvidence(newCreationTenantContext("tenant-a"), []string{"tenant-a"}, 0))
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	flagViewAsJson = true

	err := listProcessDefinitionDeploymentsView(cmd, []resource.ProcessDefinitionDeployment{{
		Key:               "deployment-a",
		DefinitionKey:     "1",
		DefinitionId:      "Short",
		DefinitionVersion: 1,
		ResourceName:      "processdefinitions/Short.bpmn",
		TenantId:          "tenant-a",
	}})

	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	tenantContext := requireJSONObject(t, got["tenantContext"])
	require.Equal(t, "creation", tenantContext["mode"])
	require.Equal(t, "tenant-a", tenantContext["targetTenantId"])
	require.Equal(t, []any{"tenant-a"}, tenantContext["resolvedTenantIds"])
	requireJSONItems(t, got["payload"], 1)
}

func TestListProcessDefinitionDeploymentsView_KeysOnlySuppressesTenantContext(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd := &cobra.Command{Use: "deploy"}
	attachTenantContext(cmd, newCreationTenantContext("tenant-a"))
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	flagViewKeysOnly = true

	err := listProcessDefinitionDeploymentsView(cmd, []resource.ProcessDefinitionDeployment{{
		DefinitionKey: "1",
		TenantId:      "tenant-a",
	}})

	require.NoError(t, err)
	require.Equal(t, "1\n", buf.String())
}
