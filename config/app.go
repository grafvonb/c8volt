package config

import (
	"errors"
	"fmt"

	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx"
)

type App struct {
	CamundaVersion          toolx.CamundaVersion `mapstructure:"camunda_version" json:"camunda_version" yaml:"camunda_version"`
	Tenant                  string               `mapstructure:"tenant" json:"tenant" yaml:"tenant"`
	ProcessInstancePageSize int32                `mapstructure:"process_instance_page_size" json:"process_instance_page_size" yaml:"process_instance_page_size"`
	Backoff                 BackoffConfig        `mapstructure:"backoff" json:"backoff" yaml:"backoff"`
	NoErrCodes              bool                 `mapstructure:"no_err_codes" json:"-" yaml:"-"`
}

const DefaultTenant = "<default>"

func (a *App) ViewTenant() string {
	if a.Tenant == "" {
		return DefaultTenant
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
	if a.Tenant == "" && a.CamundaVersion == toolx.V87 {
		a.Tenant = DefaultTenant
	}
	if a.ProcessInstancePageSize <= 0 {
		a.ProcessInstancePageSize = consts.MaxPISearchSize
	}
	return errors.Join(errs...)
}

func (a *App) normalizeWithConfiguredKeys(isConfigured func(string) bool) error {
	var errs []error
	switch a.CamundaVersion {
	case "":
		if !isConfigured("app.camunda_version") {
			a.CamundaVersion = toolx.CurrentCamundaVersion
		}
	default:
		v, err := toolx.NormalizeCamundaVersion(string(a.CamundaVersion))
		if err != nil {
			errs = append(errs, fmt.Errorf("version: %w", err))
		} else {
			a.CamundaVersion = v
		}
	}
	if err := a.Backoff.normalizeWithConfiguredKeys(isConfigured); err != nil {
		errs = append(errs, fmt.Errorf("backoff: %w", err))
	}
	if a.Tenant == "" && a.CamundaVersion == toolx.V87 && !isConfigured("app.tenant") {
		a.Tenant = DefaultTenant
	}
	if a.ProcessInstancePageSize <= 0 && !isConfigured("app.process_instance_page_size") {
		a.ProcessInstancePageSize = consts.MaxPISearchSize
	}
	return errors.Join(errs...)
}

func (a *App) Validate() error {
	var errs []error
	if a.ProcessInstancePageSize <= 0 {
		errs = append(errs, errors.New("process_instance_page_size must be greater than 0"))
	}
	if err := a.Backoff.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("backoff: %w", err))
	}
	return errors.Join(errs...)
}
