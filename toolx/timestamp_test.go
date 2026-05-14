// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatNumericZoneTime(t *testing.T) {
	require.Equal(t, "2026-04-23T01:07:49+00:00", FormatNumericZoneTime(time.Date(2026, 4, 23, 1, 7, 49, 0, time.UTC)))
}

func TestFormatNumericZoneTimestamp(t *testing.T) {
	require.Equal(t, "2026-05-14T05:35:40.226+00:00", FormatNumericZoneTimestamp("2026-05-14T05:35:40.226Z"))
	require.Equal(t, "2026-05-14T05:35:41+00:00", FormatNumericZoneTimestamp("2026-05-14T05:35:41Z"))
	require.Equal(t, "not-a-date", FormatNumericZoneTimestamp("not-a-date"))
	require.Equal(t, "", FormatNumericZoneTimestamp(""))
}
