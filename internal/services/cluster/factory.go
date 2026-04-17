package cluster

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/cluster/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/cluster/v88"
	"github.com/grafvonb/c8volt/toolx"
)

type constructor func(*config.Config, *http.Client, *slog.Logger) (API, error)

var constructors = map[toolx.CamundaVersion]constructor{
	toolx.V87: func(cfg *config.Config, httpClient *http.Client, log *slog.Logger) (API, error) {
		return v87.New(cfg, httpClient, log)
	},
	toolx.V88: func(cfg *config.Config, httpClient *http.Client, log *slog.Logger) (API, error) {
		return v88.New(cfg, httpClient, log)
	},
}

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger) (API, error) {
	v := cfg.App.CamundaVersion
	build, ok := constructors[v]
	if !ok {
		return nil, fmt.Errorf("%w: %q (supported: %v)", services.ErrUnknownAPIVersion, v, toolx.ImplementedCamundaVersionsString())
	}
	return build(cfg, httpClient, log)
}
