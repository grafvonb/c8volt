// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestReportBusinessIDConversion ensures report snapshots own every observed
// pointer, including nested cleanup previews and independent absent/empty items.
func TestReportBusinessIDConversion(t *testing.T) {
	items := []d.ProcessInstance{{Key: "root", BusinessId: toolx.Ptr("new")}, {Key: "child", BusinessId: toolx.Ptr("old")}, {Key: "nil"}, {Key: "empty", BusinessId: toolx.Ptr("")}}
	preview := d.DryRunPIKeyExpansion{SelectedFinalState: items, RequiresCancelBeforeDelete: items}
	retention := fromDomainRetentionAuditReport(d.RetentionAuditReport{DeletePlan: d.RetentionDeletePlan{FinalStateItems: items, NonFinalAffectedItems: items, SkippedNonFinalRoots: items}})
	incident := fromDomainIncidentPurgeReport(d.IncidentPurgeReport{DeletePlan: d.IncidentPurgeDeletePlan{FinalStateItems: items, NonFinalAffectedItems: items}})
	orphan := fromDomainOrphanPurgeReport(d.OrphanPurgeReport{DeletionPlan: d.DeletionPlan{DryRunPreview: preview}})
	definitions := fromDomainAllProcessDefinitionsPurgeReport(d.AllProcessDefinitionsPurgeReport{DeletePlan: d.AllProcessDefinitionsPurgeDeletePlan{Items: []d.DeleteProcessDefinitionPlanItem{{CancellationPlan: preview}}}})
	*items[0].BusinessId = "changed root"
	*items[1].BusinessId = "changed child"
	*items[3].BusinessId = "changed empty"
	for _, got := range [][]process.ProcessInstance{retention.DeletePlan.FinalStateItems, retention.DeletePlan.NonFinalAffectedItems, retention.DeletePlan.SkippedNonFinalRoots, incident.DeletePlan.FinalStateItems, incident.DeletePlan.NonFinalAffectedItems, orphan.DeletionPlan.DryRunPreview.SelectedFinalState, orphan.DeletionPlan.DryRunPreview.RequiresCancelBeforeDelete, definitions.DeletePlan.Items[0].CancellationPlan.SelectedFinalState, definitions.DeletePlan.Items[0].CancellationPlan.RequiresCancelBeforeDelete} {
		require.Equal(t, "new", *got[0].BusinessId)
		require.Equal(t, "old", *got[1].BusinessId)
		require.Nil(t, got[2].BusinessId)
		require.Equal(t, "", *got[3].BusinessId)
	}
}
