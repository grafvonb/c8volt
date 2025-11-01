package processinstance

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/processinstance/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/processinstance/v88"
	"github.com/grafvonb/c8volt/toolx"
)

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger) (API, error) {
	v := cfg.APIs.Version
	switch v {
	case toolx.V87:
		return v87.New(cfg, httpClient, log)
	case toolx.V88:
		return v88.New(cfg, httpClient, log)
	default:
		return nil, fmt.Errorf("%w: %q (supported: %v)", services.ErrUnknownAPIVersion, v, toolx.SupportedCamundaVersionsString())
	}
}
