package walker

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type PIWalker interface {
	GetProcessInstanceByKey(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstance, error)
}

func Ancestry(ctx context.Context, s PIWalker, startKey string, opts ...services.CallOption) (rootKey string, path []string, chain map[string]d.ProcessInstance, err error) {
	// visited keeps track of visited nodes to detect cycles
	// well-know pattern to have fast lookups, no duplicates, clear semantic and low memory usage with visited[cur] = struct{}{} below
	_ = services.ApplyCallOptions(opts)

	visited := make(map[string]struct{})
	chain = make(map[string]d.ProcessInstance)

	cur := startKey
	for {
		// check for context cancellation
		select {
		case <-ctx.Done():
			return "", nil, chain, ctx.Err()
		default:
		}

		if _, seen := visited[cur]; seen {
			return "", nil, chain, fmt.Errorf("%w for this key %s", services.ErrCycleDetected, cur)
		}
		visited[cur] = struct{}{}

		it, getErr := s.GetProcessInstanceByKey(ctx, cur, opts...)
		if getErr != nil {
			return "", nil, chain, fmt.Errorf("get %s: %w", cur, getErr)
		}
		chain[cur] = it
		path = append(path, cur)

		// no parent => cur is root
		if it.ParentKey == "" {
			rootKey = cur
			return
		}

		cur = it.ParentKey
	}
}

func Descendants(ctx context.Context, s PIWalker, rootKey string, opts ...services.CallOption) (desc []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error) {
	_ = services.ApplyCallOptions(opts)

	visited := make(map[string]struct{})
	edges = make(map[string][]string)
	chain = make(map[string]d.ProcessInstance)

	// depth-first search (DFS) to explore the tree
	var dfs func(string) error
	dfs = func(parent string) error {
		// check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, seen := visited[parent]; seen {
			// already expanded this subtree
			return nil
		}
		visited[parent] = struct{}{}

		desc = append(desc, parent)
		if _, ok := chain[parent]; !ok {
			it, getErr := s.GetProcessInstanceByKey(ctx, parent, opts...)
			if getErr != nil {
				return fmt.Errorf("get %s: %w", parent, getErr)
			}
			chain[parent] = it
		}

		children, e := s.GetDirectChildrenOfProcessInstance(ctx, parent, opts...)
		if e != nil {
			return fmt.Errorf("list children of %s: %w", parent, e)
		}

		// keep an entry even if no children (useful for tree rendering)
		if _, ok := edges[parent]; !ok {
			edges[parent] = nil
		}

		for i := range children {
			it := children[i]
			k := it.Key

			edges[parent] = append(edges[parent], k)
			chain[k] = it

			if dfsErr := dfs(k); dfsErr != nil {
				return dfsErr
			}
		}
		return nil
	}

	if err = dfs(rootKey); err != nil {
		return nil, nil, nil, err
	}
	return desc, edges, chain, nil
}

func Family(ctx context.Context, s PIWalker, startKey string, opts ...services.CallOption) (fam []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error) {
	rootKey, _, _, err := Ancestry(ctx, s, startKey, opts...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ancestry fetch: %w", err)
	}
	return Descendants(ctx, s, rootKey, opts...)
}
