// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/tenant/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/tenant/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/tenant/v89"
	"github.com/grafvonb/c8volt/toolx"
)

// New selects the tenant service implementation for the configured Camunda version.
func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger) (API, error) {
	v := cfg.App.CamundaVersion
	switch v {
	case toolx.V89:
		return v89.New(cfg, httpClient, log)
	case toolx.V88:
		return v88.New(cfg, httpClient, log)
	case toolx.V87:
		return v87.New(cfg, httpClient, log)
	default:
		return nil, fmt.Errorf("%w: %q (supported: %v)", services.ErrUnknownAPIVersion, v, toolx.ImplementedCamundaVersionsString())
	}
}
