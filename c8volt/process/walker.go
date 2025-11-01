package process

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/options"
)

type Walker interface {
	Ancestry(ctx context.Context, startKey string, opts ...options.FacadeOption) (rootKey string, path []string, chain map[string]ProcessInstance, err error)
	Descendants(ctx context.Context, rootKey string, opts ...options.FacadeOption) (desc []string, edges map[string][]string, chain map[string]ProcessInstance, err error)
	Family(ctx context.Context, startKey string, opts ...options.FacadeOption) (fam []string, edges map[string][]string, chain map[string]ProcessInstance, err error)
}

func AsWalker(client API) (Walker, bool) {
	w, ok := client.(Walker)
	return w, ok
}

func (c *client) Ancestry(ctx context.Context, startKey string, opts ...options.FacadeOption) (string, []string, map[string]ProcessInstance, error) {
	rootKey, path, dchain, err := c.piApi.Ancestry(ctx, startKey, options.MapFacadeOptionsToCallOptions(opts)...)
	return rootKey, path, fromDomainProcessInstanceMap(dchain), err
}

func (c *client) Descendants(ctx context.Context, rootKey string, opts ...options.FacadeOption) ([]string, map[string][]string, map[string]ProcessInstance, error) {
	desc, edges, dchain, err := c.piApi.Descendants(ctx, rootKey, options.MapFacadeOptionsToCallOptions(opts)...)
	return desc, edges, fromDomainProcessInstanceMap(dchain), err
}

func (c *client) Family(ctx context.Context, startKey string, opts ...options.FacadeOption) ([]string, map[string][]string, map[string]ProcessInstance, error) {
	fam, edges, dchain, err := c.piApi.Family(ctx, startKey, options.MapFacadeOptionsToCallOptions(opts)...)
	return fam, edges, fromDomainProcessInstanceMap(dchain), err
}
