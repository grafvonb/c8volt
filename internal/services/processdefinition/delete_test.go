// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processdefinition

import (
	"bytes"
	"log/slog"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestFormatPartialCancellationImpactWarning_HidesMissingAncestorKeysUntilVerbose verifies quiet warnings hide key detail.
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

// TestProcessDefinitionDeleteLogSubjectUsesBPMNProcessIDVersionAndKey verifies full process-definition labels.
func TestProcessDefinitionDeleteLogSubjectUsesBPMNProcessIDVersionAndKey(t *testing.T) {
	t.Parallel()

	got := processDefinitionDeleteLogSubject(d.DeleteProcessDefinitionPlanItem{
		Key:               "2251799813685255",
		BpmnProcessId:     "invoice",
		ProcessVersion:    5,
		ProcessVersionTag: "v1.0.0",
		TenantId:          "<default>",
	})

	assert.Equal(t, "pd 2251799813685255 invoice v5/v1.0.0 <default>", got)
}

// TestProcessDefinitionDeleteLogSubjectOmitsMissingVersion verifies labels stay compact without version metadata.
func TestProcessDefinitionDeleteLogSubjectOmitsMissingVersion(t *testing.T) {
	t.Parallel()

	got := processDefinitionDeleteLogSubject(d.DeleteProcessDefinitionPlanItem{
		Key:           "2251799813685255",
		BpmnProcessId: "invoice",
		TenantId:      "tenant-a",
	})

	assert.Equal(t, "pd 2251799813685255 invoice tenant-a", got)
}

// TestProcessDefinitionDeleteLogSubjectFallsBackToKeyOnly verifies key-only labels when BPMN metadata is absent.
func TestProcessDefinitionDeleteLogSubjectFallsBackToKeyOnly(t *testing.T) {
	t.Parallel()

	got := processDefinitionDeleteLogSubject(d.DeleteProcessDefinitionPlanItem{Key: "2251799813685255"})

	assert.Equal(t, "pd 2251799813685255", got)
}

// TestLogProcessDefinitionDeleteResultUsesSequentialLifecycleTerms verifies accepted and confirmed delete wording.
func TestLogProcessDefinitionDeleteResultUsesSequentialLifecycleTerms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		resp d.ResourceDeleteResponse
		want string
	}{
		{
			name: "confirmed batch after wait",
			resp: d.ResourceDeleteResponse{BatchOperationKey: "batch-1", BatchState: "COMPLETED"},
			want: "pd 1 invoice v3 tenant; delete confirmed; batch batch-1, state COMPLETED",
		},
		{
			name: "accepted batch without confirmation",
			resp: d.ResourceDeleteResponse{BatchOperationKey: "batch-1"},
			want: "pd 1 invoice v3 tenant; delete accepted; batch batch-1",
		},
		{
			name: "direct status without batch",
			resp: d.ResourceDeleteResponse{Status: "204 No Content"},
			want: "pd 1 invoice v3 tenant; delete done; status 204 No Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := slog.New(slog.NewTextHandler(&buf, nil))

			logProcessDefinitionDeleteResult(log, "pd 1 invoice v3 tenant", tt.resp)

			assert.Contains(t, buf.String(), tt.want)
		})
	}
}
