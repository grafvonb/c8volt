// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedIsolatedHeldDataset gives lifecycle tests their own definition so old
// instances cannot change batch counts or prevent draining completion.
func seedIsolatedHeldDataset(t *testing.T, profile integrationProfile, label string, count int) (volumeDataset, []evidenceRecord, error) {
	t.Helper()
	selection, _, err := discoverEmbeddedFixture(t, profile)
	if err != nil {
		return volumeDataset{}, nil, err
	}
	data, err := os.ReadFile(filepath.Join(suite.repoRoot, "embedded", selection.Path))
	if err != nil {
		return volumeDataset{}, nil, err
	}
	id := selection.BpmnProcessID + "_" + label + "_" + sanitizeEvidenceName(suite.marker+"_"+profile.Name)
	data = []byte(strings.ReplaceAll(string(data), selection.BpmnProcessID, id))
	path := filepath.Join(suite.workDir, id+".bpmn")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return volumeDataset{}, nil, err
	}
	result := runC8VoltForProfile(t, profile.Name, label+"-isolated-deploy", "--automation", "--json", "deploy", "pd", "--file", path)
	if result.Err != nil {
		return volumeDataset{}, nil, fmt.Errorf("deploy isolated held fixture: %w; %s", result.Err, result.Stderr)
	}
	var deployments []seededDeployment
	if err := decodeCommandPayload(result.Stdout, &deployments); err != nil {
		return volumeDataset{}, nil, err
	}
	selection.BpmnProcessID = id
	instances, record, err := runVolumeProcessInstances(t, profile, selection, deployments, count, label)
	return volumeDataset{PositiveBpmnProcessID: id, PositiveProcessDefinitionKeys: processDefinitionKeys(deployments), PositiveProcessInstanceKeys: processInstanceKeys(instances)}, []evidenceRecord{record}, err
}
