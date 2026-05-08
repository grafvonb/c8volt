// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/internal/services/incident"
	"github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/stretchr/testify/require"
)

func TestJobServiceBoundaryDoesNotLeakIntoProcessInstanceOrIncidentAPIs(t *testing.T) {
	assertNoJobOperations(t, reflect.TypeOf((*processinstance.API)(nil)).Elem())
	assertNoJobOperations(t, reflect.TypeOf((*incident.API)(nil)).Elem())
}

func assertNoJobOperations(t *testing.T, api reflect.Type) {
	t.Helper()

	for i := 0; i < api.NumMethod(); i++ {
		method := api.Method(i)
		name := strings.ToLower(method.Name)
		signature := strings.ToLower(method.Type.String())
		require.NotContains(t, name, "job")
		require.NotContains(t, signature, ".job")
		require.NotContains(t, name, "confirmation")
	}
}
