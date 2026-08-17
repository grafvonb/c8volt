// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package embedded

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	bpmnNamespace    = "http://www.omg.org/spec/BPMN/20100524/MODEL"
	bpmndiNamespace  = "http://www.omg.org/spec/BPMN/20100524/DI"
	zeebeNamespace   = "http://camunda.org/schema/zeebe/1.0"
	modelerNamespace = "http://camunda.org/schema/modeler/1.0"
)

var c810ProductionDefinitions = []string{
	"processdefinitions/C810_DoubleUserTask.bpmn",
	"processdefinitions/C810_MultipleSubProcessesParent.bpmn",
	"processdefinitions/C810_NoOpCompletion.bpmn",
	"processdefinitions/C810_SimpleParent.bpmn",
	"processdefinitions/C810_SimpleParentWithIncidentSubprocess.bpmn",
	"processdefinitions/C810_SimpleServiceTask.bpmn",
	"processdefinitions/C810_SimpleUserTask.bpmn",
	"processdefinitions/C810_SimpleUserTaskWithIncident.bpmn",
}

func TestC810ProductionDefinitions(t *testing.T) {
	t.Parallel()

	files, err := List()
	require.NoError(t, err)

	var got []string
	for _, file := range files {
		if strings.HasPrefix(filepath.Base(file), "C810_") {
			got = append(got, file)
		}
	}
	sort.Strings(got)
	require.Equal(t, c810ProductionDefinitions, got)

	c810Family := make(map[string]struct{}, len(c810ProductionDefinitions))
	for _, file := range c810ProductionDefinitions {
		id := strings.TrimSuffix(filepath.Base(file), ".bpmn")
		c810Family[id] = struct{}{}
	}

	for _, file := range c810ProductionDefinitions {
		t.Run(strings.TrimSuffix(filepath.Base(file), ".bpmn"), func(t *testing.T) {
			t.Parallel()

			data := readEmbeddedBPMN(t, file)
			require.NotContains(t, string(data), "C89")

			definition := parseBPMNDefinition(t, data)
			processID := strings.TrimSuffix(filepath.Base(file), ".bpmn")

			require.Equal(t, "Camunda Cloud", definition.ExecutionPlatform)
			require.Equal(t, "8.10.0", definition.ExecutionPlatformVersion)
			require.Equal(t, []string{processID}, definition.ProcessIDs)
			require.Len(t, definition.ProcessNames, 1)
			require.True(t, strings.HasPrefix(definition.ProcessNames[0], "C810 "), "process name %q must use C810 identity", definition.ProcessNames[0])
			require.NotContains(t, definition.ProcessNames[0], "C89")
			require.Equal(t, []string{"v2.0.0"}, definition.VersionTags)
			require.Equal(t, []string{processID}, definition.PlaneOwners)

			for _, calledProcess := range definition.CalledProcesses {
				require.True(t, strings.HasPrefix(calledProcess, "C810_"), "called process %q must use C810 identity", calledProcess)
				require.NotContains(t, calledProcess, "C89")
				require.Contains(t, c810Family, calledProcess, "called process %q must resolve inside the C810 family", calledProcess)
			}

			sourcePath := strings.Replace(file, "C810_", "C89_", 1)
			sourceData := readEmbeddedBPMN(t, sourcePath)
			require.Equal(t, string(sourceData), normalizeC810Definition(data))
		})
	}
}

type bpmnDefinition struct {
	ExecutionPlatform        string
	ExecutionPlatformVersion string
	ProcessIDs               []string
	ProcessNames             []string
	PlaneOwners              []string
	VersionTags              []string
	CalledProcesses          []string
}

func parseBPMNDefinition(t *testing.T, data []byte) bpmnDefinition {
	t.Helper()

	var definition bpmnDefinition
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		switch {
		case start.Name.Space == bpmnNamespace && start.Name.Local == "definitions":
			definition.ExecutionPlatform = requiredAttr(t, start.Attr, modelerNamespace, "executionPlatform")
			definition.ExecutionPlatformVersion = requiredAttr(t, start.Attr, modelerNamespace, "executionPlatformVersion")
		case start.Name.Space == bpmnNamespace && start.Name.Local == "process":
			definition.ProcessIDs = append(definition.ProcessIDs, requiredAttr(t, start.Attr, "", "id"))
			definition.ProcessNames = append(definition.ProcessNames, requiredAttr(t, start.Attr, "", "name"))
		case start.Name.Space == bpmndiNamespace && start.Name.Local == "BPMNPlane":
			definition.PlaneOwners = append(definition.PlaneOwners, requiredAttr(t, start.Attr, "", "bpmnElement"))
		case start.Name.Space == zeebeNamespace && start.Name.Local == "versionTag":
			definition.VersionTags = append(definition.VersionTags, requiredAttr(t, start.Attr, "", "value"))
		case start.Name.Space == zeebeNamespace && start.Name.Local == "calledElement":
			definition.CalledProcesses = append(definition.CalledProcesses, requiredAttr(t, start.Attr, "", "processId"))
		}
	}

	return definition
}

func requiredAttr(t *testing.T, attrs []xml.Attr, namespace, name string) string {
	t.Helper()

	for _, attr := range attrs {
		if attr.Name.Local == name && attr.Name.Space == namespace {
			return attr.Value
		}
	}
	require.Failf(t, "missing XML attribute", "attribute %s in namespace %s not found", name, namespace)
	return ""
}

func normalizeC810Definition(data []byte) string {
	normalized := strings.ReplaceAll(string(data), "8.10.0", "8.9.0")
	normalized = strings.ReplaceAll(normalized, "C810", "C89")
	return normalized
}

func readEmbeddedBPMN(t *testing.T, path string) []byte {
	t.Helper()

	data, err := fs.ReadFile(FS, path)
	require.NoError(t, err)
	return data
}
