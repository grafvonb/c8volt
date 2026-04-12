package config

import (
	"testing"

	"github.com/grafvonb/c8volt/consts"
	"github.com/stretchr/testify/require"
)

func TestAppNormalize_DefaultsProcessInstancePageSize(t *testing.T) {
	t.Parallel()

	app := &App{}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, int32(consts.MaxPISearchSize), app.ProcessInstancePageSize)
}

func TestAppNormalize_PreservesPositiveProcessInstancePageSize(t *testing.T) {
	t.Parallel()

	app := &App{ProcessInstancePageSize: 250}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, int32(250), app.ProcessInstancePageSize)
}
