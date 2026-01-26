package process

import (
	"context"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	types "github.com/grafvonb/c8volt/typex"
)

func (c *client) DryRunCancelOrDeleteGetPIKeys(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (roots types.Keys, collected types.Keys, err error) {
	for _, key := range keys {
		root, _, _, err := c.Ancestry(ctx, key, opts...)
		if err != nil {
			return nil, nil, ferr.FromDomain(err)
		}
		roots = append(roots, root)
	}
	for _, root := range roots {
		fam, _, _, err := c.Descendants(ctx, root, opts...)
		if err != nil {
			return nil, nil, ferr.FromDomain(err)
		}
		collected = append(collected, fam...)
	}
	return roots, collected, nil
}
