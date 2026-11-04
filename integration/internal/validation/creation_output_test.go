// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHumanCreationKeys(t *testing.T) {
	plain := "12:00:00.001 INFO creation target: default tenant\n12:00:00.002 INFO 2251799813685250 <default> process v1 ACTIVE\n12:00:00.003 INFO found: 1\n"
	structured := "{\"level\":\"INFO\",\"msg\":\"2251799813685250 <default> process v1 ACTIVE\"}\n{\"level\":\"INFO\",\"msg\":\"found: 1\"}\n"
	for _, log := range []string{plain, structured} {
		keys, err := HumanCreationKeys("", log, 1)
		require.NoError(t, err)
		require.Equal(t, []string{"2251799813685250"}, keys)
	}
	for _, tc := range []struct{ stdout, stderr string }{
		{plain, ""}, // Reverting the old stdout assertion must fail.
		{" ", plain},
		{"", "2251799813685250 <default> process v1 ACTIVE\nfound: 1\n"}, // Direct prints are not logger records.
		{"", "12:00:00 INFO 2251799813685250 p ACTIVE\n"},
		{"", "12:00:00 INFO found: 1\n"},
		{"", plain + "12:00:00 INFO found: 1\n"},
		{"", plain + "12:00:00 INFO 2251799813685250 p ACTIVE\n"},
		{"", "12:00:00 ERROR 2251799813685250 p ACTIVE\n12:00:00 INFO found: 1\n"},
	} {
		_, err := HumanCreationKeys(tc.stdout, tc.stderr, 1)
		require.Error(t, err)
	}
}
