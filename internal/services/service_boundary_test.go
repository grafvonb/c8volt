// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package services_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/internal/services/incident"
	"github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/stretchr/testify/require"
)

// TestNonJobServiceAPIsDoNotExposeJobOperations protects the package boundary:
// process-instance and incident services should compose with jobs through callers,
// not grow job-specific primitives on their shared APIs.
func TestNonJobServiceAPIsDoNotExposeJobOperations(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		api  any
	}{
		{name: "incident", api: (*incident.API)(nil)},
		{name: "processinstance", api: (*processinstance.API)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			apiType := reflect.TypeOf(tc.api).Elem()
			for i := 0; i < apiType.NumMethod(); i++ {
				method := apiType.Method(i)
				require.NotContains(t, strings.ToLower(method.Name), "job")
			}
		})
	}
}
