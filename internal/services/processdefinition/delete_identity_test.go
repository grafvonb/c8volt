// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processdefinition

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestDeleteProcessDefinitionsOwnsResultIdentity keeps deduplication and identity
// in services even when cancellation leaves every worker slot unattempted.
func TestDeleteProcessDefinitionsOwnsResultIdentity(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	got, _ := DeleteProcessDefinitionReports(ctx, testResourceDeleteAPI{}, nil, nil, log,
		typex.Keys{"pd-b", "pd-a", "pd-b", "pd-c"}, 1, services.WithNoStateCheck(), services.WithNoWait())
	require.Len(t, got, 3)
	for i, key := range []string{"pd-b", "pd-a", "pd-c"} {
		require.Equal(t, key, got[i].Key)
		require.False(t, got[i].Ok)
	}
}
