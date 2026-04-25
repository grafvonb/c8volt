package toolx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDurationStringValue_ValidatesDurationAndStoresOriginalText(t *testing.T) {
	var value string
	flagValue := NewDurationStringValue("30s", &value)

	require.Equal(t, "30s", flagValue.String())
	require.Equal(t, "duration", flagValue.Type())
	require.NoError(t, flagValue.Set("1m30s"))
	require.Equal(t, "1m30s", value)
	require.Equal(t, "1m30s", flagValue.String())
	require.Error(t, flagValue.Set("eventually"))
	require.Equal(t, "1m30s", value)
}
