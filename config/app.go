package config

import "github.com/grafvonb/c8volt/internal/services/common"

type App struct {
	Tenant  string               `mapstructure:"tenant" json:"tenant" yaml:"tenant"`
	Backoff common.BackoffConfig `mapstructure:"backoff" json:"backoff" yaml:"backoff"`
}

func (a *App) ViewTenant() string {
	if a.Tenant == "" {
		return "default"
	}
	return a.Tenant
}
