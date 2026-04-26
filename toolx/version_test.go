// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupportedCamundaVersionsIncludeV89(t *testing.T) {
	t.Parallel()

	require.Equal(t, []CamundaVersion{V87, V88, V89}, SupportedCamundaVersions())
	require.Equal(t, "8.7, 8.8, 8.9", SupportedCamundaVersionsString())
}

func TestImplementedCamundaVersionsStayOnRuntimeImplementedSet(t *testing.T) {
	t.Parallel()

	require.Equal(t, []CamundaVersion{V87, V88, V89}, ImplementedCamundaVersions())
	require.Equal(t, "8.7, 8.8, 8.9", ImplementedCamundaVersionsString())
}
