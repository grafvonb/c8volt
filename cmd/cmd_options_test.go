package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/stretchr/testify/require"
)

func TestCollectOptionsIncludesAllowInconsistent(t *testing.T) {
	resetCollectOptionsFlags()
	t.Cleanup(resetCollectOptionsFlags)

	flagAllowInconsistent = true

	cfg := foptions.ApplyFacadeOptions(collectOptions())
	require.True(t, cfg.AllowInconsistent)
}

func resetCollectOptionsFlags() {
	flagNoWait = false
	flagNoStateCheck = false
	flagForce = false
	flagDeployPDWithRun = false
	flagGetPDWithStat = false
	flagDryRun = false
	flagVerbose = false
	flagFailFast = false
	flagNoWorkerLimit = false
	flagAllowInconsistent = false
}

