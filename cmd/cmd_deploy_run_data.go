// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
)

func buildRunProcessInstanceDatasFromDeployments(
	deployments []resource.ProcessDefinitionDeployment,
	units []resource.DeploymentUnitData,
	tenantID string,
) ([]process.ProcessInstanceData, error) {
	datas := make([]process.ProcessInstanceData, 0, len(deployments))
	for _, pdd := range deployments {
		switch {
		case pdd.DefinitionKey != "":
			datas = append(datas, process.ProcessInstanceData{
				ProcessDefinitionSpecificId: pdd.DefinitionKey,
				TenantId:                    tenantID,
			})
		case pdd.DefinitionId != "":
			datas = append(datas, process.ProcessInstanceData{
				BpmnProcessId: pdd.DefinitionId,
				TenantId:      tenantID,
			})
		}
	}
	if len(datas) > 0 {
		return datas, nil
	}

	for _, unit := range units {
		ids, err := extractBPMNProcessIDs(unit.Data)
		if err != nil {
			return nil, localPreconditionError(fmt.Errorf("extracting BPMN process ID(s) from %q for --run: %w", unit.Name, err))
		}
		for _, id := range ids {
			datas = append(datas, process.ProcessInstanceData{
				BpmnProcessId: id,
				TenantId:      tenantID,
			})
		}
	}

	if len(datas) == 0 {
		return nil, localPreconditionError(fmt.Errorf("cannot determine process definition identifier(s) for --run from deployment response or BPMN resources"))
	}

	return datas, nil
}

func extractBPMNProcessIDs(data []byte) ([]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	seen := map[string]struct{}{}
	var ids []string

	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "process" {
			continue
		}

		var id string
		for _, attr := range start.Attr {
			if attr.Name.Local == "id" {
				id = attr.Value
				break
			}
		}
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	return ids, nil
}
