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

func WithConfig(c *config.Config) Option   { return func(x *cfg) { x.cfg = c } }
func WithHTTPClient(h *http.Client) Option { return func(x *cfg) { x.http = h } }
func WithLogger(l *slog.Logger) Option     { return func(x *cfg) { x.log = l } }

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
		ClusterAPI: cluster.New(cAPI),
		ProcessAPI: process.New(pdAPI, piAPI),
		TaskAPI:    task.New(pdAPI, piAPI),
		capsFunc: func(context.Context) (Capabilities, error) {
			return Capabilities{
				APIVersion: string(c.cfg.APIs.Version),
				Features:   map[Feature]bool{},
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
