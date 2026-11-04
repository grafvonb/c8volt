// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/integration/internal/validation"
	"github.com/stretchr/testify/require"
)

// newLifecycleTreeFixture deploys only the existing three-model dependency
// closure, with unique IDs and rewritten call references. No shared definitions
// or unrelated embedded models are deployed or deleted.
func newLifecycleTreeFixture(t *testing.T, profile integrationProfile) (f *suspendedFixture, root, mid, leaf, sibling string) {
	t.Helper()
	f = newSuspendedFixtureBase(t, profile)
	originals := []string{"C810_SimpleUserTask", "C810_SimpleParent", "C810_MultipleSubProcessesParent"}
	suffix := "_" + sanitizeEvidenceName(suite.marker+"_"+profile.Name+"_lifecycle")
	ids := validation.CascadeProcessIDs{Parent: originals[2] + suffix, Mid: originals[1] + suffix, Leaf: originals[0] + suffix}
	var replacements []string
	for _, id := range originals {
		replacements = append(replacements, id, id+suffix)
	}
	replacer := strings.NewReplacer(replacements...)
	var rootDeployments []seededDeployment
	for _, id := range originals {
		data, err := os.ReadFile(filepath.Join(suite.repoRoot, "embedded", "processdefinitions", id+".bpmn"))
		require.NoError(t, err)
		path := filepath.Join(suite.workDir, id+suffix+".bpmn")
		require.NoError(t, os.WriteFile(path, []byte(replacer.Replace(string(data))), 0600))
		result := f.run(t, "deploy-"+id, "deploy", "pd", "--file", path, "--automation", "--json")
		require.NoError(t, result.Err, result.Stderr)
		var deployments []seededDeployment
		decodeSuspendedLiveEnvelope(t, result.Stdout, "succeeded", &deployments)
		keys := processDefinitionKeys(deployments)
		f.dataset.PositiveProcessDefinitionKeys = append(f.dataset.PositiveProcessDefinitionKeys, keys...)
		require.Len(t, keys, 1)
		rootDeployments = deployments
	}
	f.dataset.PositiveBpmnProcessID = ids.Parent
	instances, record, err := runVolumeProcessInstances(t, f.profile, embeddedFixtureSelection{BpmnProcessID: ids.Parent}, rootDeployments, 1, "lifecycle-tree")
	f.records = append(f.records, record)
	f.dataset.PositiveProcessInstanceKeys = processInstanceKeys(instances)
	require.NoError(t, err)
	require.Len(t, f.dataset.PositiveProcessInstanceKeys, 1)
	root = f.dataset.PositiveProcessInstanceKeys[0]
	family, record, err := waitForRealStateCascadeFamily(t, f.profile, "lifecycle", root, ids)
	record.DataOwnership = []string{"seeded"}
	f.records = append(f.records, record)
	require.NoError(t, err)
	f.dataset.PositiveProcessInstanceKeys = family.Keys
	// Register every member before assertions so failed setup still cleans up.
	for _, key := range family.Keys {
		f.resumeKeys[key] = true
	}
	require.Len(t, family.Keys, 4)
	require.Len(t, family.Items, 4)
	for _, item := range family.Items {
		if item.BpmnProcessID == ids.Mid {
			mid = item.Key
		}
	}
	require.NotEmpty(t, mid)
	for _, item := range family.Items {
		if item.BpmnProcessID != ids.Leaf {
			continue
		}
		observed, err := f.observe(t, item.Key, "ACTIVE", false)
		require.NoError(t, err)
		require.NotNil(t, observed.ParentProcessInstanceKey)
		switch string(*observed.ParentProcessInstanceKey) {
		case mid:
			leaf = item.Key
		case root:
			sibling = item.Key
		default:
			t.Fatalf("unexpected parent of owned leaf %s", item.Key)
		}
	}
	require.NotEmpty(t, leaf)
	require.NotEmpty(t, sibling)
	writeDataEvidence(t, "lifecycle-tree-"+profile.Name+".json", f.dataset)
	return
}
