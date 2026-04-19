package common

import (
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireSingleProcessInstance(t *testing.T) {
	t.Parallel()

	t.Run("returns single match", func(t *testing.T) {
		pi, err := RequireSingleProcessInstance([]d.ProcessInstance{{Key: "123", TenantId: "tenant"}}, "123")

		require.NoError(t, err)
		assert.Equal(t, "123", pi.Key)
	})

	t.Run("maps empty search to not found", func(t *testing.T) {
		_, err := RequireSingleProcessInstance(nil, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrNotFound)
	})

	t.Run("rejects duplicate matches as malformed", func(t *testing.T) {
		_, err := RequireSingleProcessInstance([]d.ProcessInstance{{Key: "123"}, {Key: "123"}}, "123")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrMalformedResponse)
	})
}

func TestProcessInstanceFilterHasTenantSafeLookupFields(t *testing.T) {
	t.Parallel()

	assert.False(t, ProcessInstanceFilterHasTenantSafeLookupFields(d.ProcessInstanceFilter{}))
	assert.True(t, ProcessInstanceFilterHasTenantSafeLookupFields(d.ProcessInstanceFilter{Key: "123"}))
	assert.True(t, ProcessInstanceFilterHasTenantSafeLookupFields(d.ProcessInstanceFilter{ParentKey: "456"}))
	assert.True(t, ProcessInstanceFilterHasTenantSafeLookupFields(d.ProcessInstanceFilter{State: d.StateActive}))
}
