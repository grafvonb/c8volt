package config

import (
	"errors"
	"fmt"

	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

type App struct {
	CamundaVersion toolx.CamundaVersion `mapstructure:"camunda_version" json:"camunda_version" yaml:"camunda_version"`
	Tenant         string               `mapstructure:"tenant" json:"tenant" yaml:"tenant"`
	Backoff        common.BackoffConfig `mapstructure:"backoff" json:"backoff" yaml:"backoff"`
	NoErrCodes     bool                 `mapstructure:"no_err_codes" json:"-" yaml:"-"`
}

func (a *App) ViewTenant() string {
	if a.Tenant == "" {
		return "default"
	}
	return a.Tenant
}

func (a *App) Normalize() error {
	var errs []error
	switch a.CamundaVersion {
	case "":
		a.CamundaVersion = toolx.CurrentCamundaVersion
	default:
		v, err := toolx.NormalizeCamundaVersion(string(a.CamundaVersion))
		if err != nil {
			errs = append(errs, fmt.Errorf("version: %w", err))
		} else {
			a.CamundaVersion = v
		}
	}
	if err := a.Backoff.Normalize(); err != nil {
		errs = append(errs, fmt.Errorf("backoff: %w", err))
	}
	return errors.Join(errs...)
}
