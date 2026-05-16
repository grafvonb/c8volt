// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestFormatPartialCancellationImpactWarning_HidesMissingAncestorKeysUntilVerbose(t *testing.T) {
	t.Parallel()

	plan := d.DryRunPIKeyExpansion{
		MissingAncestors: []d.MissingAncestor{
			{Key: "missing-1", StartKey: "child-1"},
			{Key: "missing-2", StartKey: "child-2"},
		},
		Warning: "one or more parent process instances were not found",
	}

	quiet := formatPartialCancellationImpactWarning("pd-1", plan, false)
	verbose := formatPartialCancellationImpactWarning("pd-1", plan, true)

	assert.Contains(t, quiet, "2 missing ancestor key(s)")
	assert.Contains(t, quiet, "use --verbose to list keys")
	assert.NotContains(t, quiet, "missing-1")
	assert.NotContains(t, quiet, "missing-2")
	assert.Contains(t, verbose, "missing ancestor keys: missing-1, missing-2")
}

func TestProcessDefinitionDeleteLogSubjectUsesBPMNProcessIDVersionAndKey(t *testing.T) {
	t.Parallel()

	got := processDefinitionDeleteLogSubject(d.DeleteProcessDefinitionPlanItem{
		Key:               "2251799813685255",
		BpmnProcessId:     "invoice",
		ProcessVersion:    5,
		ProcessVersionTag: "v1.0.0",
	})

	assert.Equal(t, "invoice v5/v1.0.0 (2251799813685255)", got)
}

func TestProcessDefinitionDeleteLogSubjectOmitsMissingVersion(t *testing.T) {
	t.Parallel()

	got := processDefinitionDeleteLogSubject(d.DeleteProcessDefinitionPlanItem{
		Key:           "2251799813685255",
		BpmnProcessId: "invoice",
	})

	assert.Equal(t, "invoice (2251799813685255)", got)
}

func TestProcessDefinitionDeleteLogSubjectFallsBackToKeyOnly(t *testing.T) {
	t.Parallel()

	got := processDefinitionDeleteLogSubject(d.DeleteProcessDefinitionPlanItem{Key: "2251799813685255"})

	assert.Equal(t, "pd 2251799813685255", got)
}
