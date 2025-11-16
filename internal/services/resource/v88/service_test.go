package v88

import (
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/domain"
	testx2 "github.com/grafvonb/c8volt/testx"

	"github.com/stretchr/testify/require"
)

func Test_Internal_Deployment_v88_Deploy_OK(t *testing.T) {
	ctx := testx2.ITCtx(t, 20*time.Second)
	cfg := testx2.TestConfig(t)
	log := testx2.Logger(t)

	fs := testx2.NewFakeServer(t)
	httpClient := fs.FS.Client()
	cfg.APIs.Camunda.BaseURL = fs.BaseURL + "/v2"

	svc, err := New(cfg, httpClient, log)
	require.NoError(t, err)

	s := "<xml>content</xml>"
	d, err := svc.Deploy(ctx, "", []domain.DeploymentUnitData{
		{Data: []byte(s)},
	})
	require.NoError(t, err)
	require.NotEmpty(t, d)

	t.Logf("success: got deployment")
	testx2.LogJson(t, d)
}
