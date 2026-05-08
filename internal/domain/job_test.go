// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJobUpdateRequestHasUpdates(t *testing.T) {
	retries := int32(3)
	timeout := int64(300000)

	tests := []struct {
		name string
		req  JobUpdateRequest
		want bool
	}{
		{name: "none", req: JobUpdateRequest{}, want: false},
		{name: "retries", req: JobUpdateRequest{Retries: &retries}, want: true},
		{name: "timeout", req: JobUpdateRequest{TimeoutMillis: &timeout}, want: true},
		{name: "both", req: JobUpdateRequest{Retries: &retries, TimeoutMillis: &timeout}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.req.HasUpdates())
		})
	}
}
