// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatNumericZoneTime(t *testing.T) {
	require.Equal(t, "2026-04-23T01:07:49.000+00:00", FormatNumericZoneTime(time.Date(2026, 4, 23, 1, 7, 49, 0, time.UTC)))
	require.Equal(t, "2026-04-23T01:07:49.007+00:00", FormatNumericZoneTime(time.Date(2026, 4, 23, 1, 7, 49, 7_000_000, time.UTC)))
}

func TestFormatTime(t *testing.T) {
	value := time.Date(2026, 4, 23, 1, 7, 49, 7_000_000, time.UTC)

	require.Equal(t, "2026-04-23T01:07:49.007", FormatTime(value, false))
	require.Equal(t, "2026-04-23T01:07:49.007+00:00", FormatTime(value, true))
}

func TestFormatNumericZoneTimestamp(t *testing.T) {
	require.Equal(t, "2026-05-14T05:35:40.226+00:00", FormatNumericZoneTimestamp("2026-05-14T05:35:40.226Z"))
	require.Equal(t, "2026-05-14T05:35:40.210+00:00", FormatNumericZoneTimestamp("2026-05-14T05:35:40.21Z"))
	require.Equal(t, "2026-05-14T05:35:41.000+00:00", FormatNumericZoneTimestamp("2026-05-14T05:35:41Z"))
	require.Equal(t, "not-a-date", FormatNumericZoneTimestamp("not-a-date"))
	require.Equal(t, "", FormatNumericZoneTimestamp(""))
}

func TestFormatTimestamp(t *testing.T) {
	require.Equal(t, "2026-05-14T05:35:40.226", FormatTimestamp("2026-05-14T05:35:40.226Z", false))
	require.Equal(t, "2026-05-14T05:35:40.210", FormatTimestamp("2026-05-14T05:35:40.21Z", false))
	require.Equal(t, "2026-05-14T05:35:41.000", FormatTimestamp("2026-05-14T05:35:41Z", false))
	require.Equal(t, "2026-05-14T05:35:41.000+00:00", FormatTimestamp("2026-05-14T05:35:41Z", true))
	require.Equal(t, "not-a-date", FormatTimestamp("not-a-date", false))
	require.Equal(t, "", FormatTimestamp("", false))
}
