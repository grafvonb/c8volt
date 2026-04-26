// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/stretchr/testify/require"
)

func TestBuildRunProcessInstanceDatasFromDeployments_UsesDefinitionKey(t *testing.T) {
	t.Parallel()

	pdds := []resource.ProcessDefinitionDeployment{{DefinitionKey: "2251799813685255"}}

	datas, err := buildRunProcessInstanceDatasFromDeployments(pdds, nil, "tenant-a")

	require.NoError(t, err)
	require.Len(t, datas, 1)
	require.Equal(t, "2251799813685255", datas[0].ProcessDefinitionSpecificId)
	require.Equal(t, "tenant-a", datas[0].TenantId)
}

func TestBuildRunProcessInstanceDatasFromDeployments_FallsBackToDefinitionID(t *testing.T) {
	t.Parallel()

	pdds := []resource.ProcessDefinitionDeployment{{DefinitionId: "order-process"}}

	datas, err := buildRunProcessInstanceDatasFromDeployments(pdds, nil, "tenant-a")

	require.NoError(t, err)
	require.Len(t, datas, 1)
	require.Equal(t, "order-process", datas[0].BpmnProcessId)
	require.Equal(t, "tenant-a", datas[0].TenantId)
}

func TestBuildRunProcessInstanceDatasFromDeployments_FallsBackToBPMNResourceParsing(t *testing.T) {
	t.Parallel()

	units := []resource.DeploymentUnitData{{
		Name: "order.bpmn",
		Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`),
	}}

	datas, err := buildRunProcessInstanceDatasFromDeployments(nil, units, "<default>")

	require.NoError(t, err)
	require.Len(t, datas, 1)
	require.Equal(t, "order-process", datas[0].BpmnProcessId)
	require.Equal(t, "<default>", datas[0].TenantId)
}

func TestBuildRunProcessInstanceDatasFromDeployments_ReturnsErrorWhenNoIdentifiersFound(t *testing.T) {
	t.Parallel()

	units := []resource.DeploymentUnitData{{
		Name: "empty.bpmn",
		Data: []byte(`<?xml version="1.0"?><bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"></bpmn:definitions>`),
	}}

	_, err := buildRunProcessInstanceDatasFromDeployments(nil, units, "<default>")

	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot determine process definition identifier")
}
