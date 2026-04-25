package c8volt

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/config"
	csvc "github.com/grafvonb/c8volt/internal/services/cluster"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"

	"github.com/grafvonb/c8volt/c8volt/cluster"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/task"
)

type Option func(*cfg)

// WithConfig supplies the resolved c8volt configuration used to choose service versions and base URLs.
// A nil config is allowed and falls back to an empty Config, which the internal services may reject if required fields are absent.
func WithConfig(c *config.Config) Option { return func(x *cfg) { x.cfg = c } }

// WithHTTPClient supplies the HTTP client shared by all internal services.
// A nil client falls back to a default client with a 30 second timeout.
func WithHTTPClient(h *http.Client) Option { return func(x *cfg) { x.http = h } }

// WithLogger supplies the logger shared by facade and internal services.
// A nil logger falls back to slog.Default.
func WithLogger(l *slog.Logger) Option { return func(x *cfg) { x.log = l } }

// New wires the public facade over version-specific internal services.
// Options provide config, HTTP transport, and logging; service factories still validate required configuration such as base URLs.
func New(opts ...Option) (API, error) {
	c := cfg{
		http: &http.Client{Timeout: 30 * time.Second},
		log:  slog.Default(),
	}
	for _, o := range opts {
		o(&c)
	}
	if c.cfg == nil {
		c.cfg = &config.Config{}
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: 30 * time.Second}
	}
	if c.log == nil {
		c.log = slog.Default()
	}

	// wire internals
	cAPI, err := csvc.New(c.cfg, c.http, c.log)
	if err != nil {
		return nil, err
	}
	pdAPI, err := pdsvc.New(c.cfg, c.http, c.log)
	if err != nil {
		return nil, err
	}
	piAPI, err := pisvc.New(c.cfg, c.http, c.log)
	if err != nil {
		return nil, err
	}
	rAPI, err := rsvc.New(c.cfg, c.http, c.log)
	if err != nil {
		return nil, err
	}

	cl := client{
		ClusterAPI: cluster.New(cAPI, c.log),
		ProcessAPI: process.New(pdAPI, piAPI, c.log),
		TaskAPI:    task.New(pdAPI, piAPI, c.log),
		capsFunc: func(context.Context) (Capabilities, error) {
			return Capabilities{
				CamundaVersion: string(c.cfg.App.CamundaVersion),
				Features:       map[Feature]bool{},
			}, nil
		},
	}
	cl.ResourceAPI = resource.New(rAPI, cl.ProcessAPI, c.log)
	return &cl, nil
}

type cfg struct {
	cfg  *config.Config
	http *http.Client
	log  *slog.Logger
}

type ClusterAPI = cluster.API
type ProcessAPI = process.API
type TaskAPI = task.API
type ResourceAPI = resource.API

var _ API = (*client)(nil)

type client struct {
	ClusterAPI
	ProcessAPI
	TaskAPI
	ResourceAPI

	capsFunc func(context.Context) (Capabilities, error)
}

func (c *client) Capabilities(ctx context.Context) (Capabilities, error) { return c.capsFunc(ctx) }
