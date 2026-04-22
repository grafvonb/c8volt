package walker

import (
	"context"
	"strings"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPIWalker struct {
	getProcessInstance               func(ctx context.Context, key string) (d.ProcessInstance, error)
	getDirectChildrenProcessInstance func(ctx context.Context, key string) ([]d.ProcessInstance, error)
	searchProcessInstances           func(ctx context.Context, filter d.ProcessInstanceFilter, size int32) ([]d.ProcessInstance, error)
}

type stubTraversalAPI struct {
	stubPIWalker
}

func (s stubPIWalker) GetProcessInstance(ctx context.Context, key string, _ ...services.CallOption) (d.ProcessInstance, error) {
	return s.getProcessInstance(ctx, key)
}

func (s stubPIWalker) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstance, error) {
	return s.getDirectChildrenProcessInstance(ctx, key)
}

func (s stubPIWalker) SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
	if s.searchProcessInstances == nil {
		return nil, d.ErrUnsupported
	}
	return s.searchProcessInstances(ctx, filter, size)
}

func (s stubTraversalAPI) Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error) {
	return Ancestry(ctx, s.stubPIWalker, startKey, opts...)
}

func (s stubTraversalAPI) Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	return Descendants(ctx, s.stubPIWalker, rootKey, opts...)
}

func (s stubTraversalAPI) Family(ctx context.Context, startKey string, opts ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	return Family(ctx, s.stubPIWalker, startKey, opts...)
}

func TestAncestry(t *testing.T) {
	t.Run("returns root path and chain for a valid ancestry", func(t *testing.T) {
		t.Parallel()

		w := treeWalker()

		rootKey, path, chain, err := Ancestry(context.Background(), w, "grandchild")

		require.NoError(t, err)
		assert.Equal(t, "root", rootKey)
		assert.Equal(t, []string{"grandchild", "child-a", "root"}, path)
		assert.Equal(t, "child-a", chain["grandchild"].ParentKey)
		assert.Equal(t, "", chain["root"].ParentKey)
	})

	t.Run("detects cycles while climbing parent links", func(t *testing.T) {
		t.Parallel()

		w := stubPIWalker{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				switch key {
				case "a":
					return d.ProcessInstance{Key: "a", ParentKey: "b"}, nil
				case "b":
					return d.ProcessInstance{Key: "b", ParentKey: "a"}, nil
				default:
					return d.ProcessInstance{}, d.ErrNotFound
				}
			},
			getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
				t.Fatalf("unexpected GetDirectChildrenOfProcessInstance call")
				return nil, nil
			},
		}

		rootKey, path, chain, err := Ancestry(context.Background(), w, "a")

		require.Error(t, err)
		assert.ErrorIs(t, err, services.ErrCycleDetected)
		assert.Empty(t, rootKey)
		assert.Nil(t, path)
		assert.Contains(t, chain, "a")
		assert.Contains(t, chain, "b")
	})

	t.Run("reports orphaned parent chains", func(t *testing.T) {
		t.Parallel()

		w := stubPIWalker{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				if key == "child" {
					return d.ProcessInstance{Key: "child", ParentKey: "missing"}, nil
				}
				return d.ProcessInstance{}, d.ErrNotFound
			},
			getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
				t.Fatalf("unexpected GetDirectChildrenOfProcessInstance call")
				return nil, nil
			},
		}

		rootKey, path, chain, err := Ancestry(context.Background(), w, "child")

		require.Error(t, err)
		assert.ErrorIs(t, err, services.ErrOrphanedInstance)
		assert.Equal(t, "missing", rootKey)
		assert.Equal(t, []string{"child"}, path)
		assert.Equal(t, d.ProcessInstance{Key: "child", ParentKey: "missing"}, chain["child"])
	})

	t.Run("keeps ancestry breadcrumb without restating the same key detail", func(t *testing.T) {
		t.Parallel()

		w := stubPIWalker{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				return d.ProcessInstance{}, d.ErrNotFound
			},
			getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
				t.Fatalf("unexpected GetDirectChildrenOfProcessInstance call")
				return nil, nil
			},
		}

		_, _, _, err := Ancestry(context.Background(), w, "child")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ancestry")
		assert.NotContains(t, err.Error(), "ancestry get")
		assert.True(t, strings.HasPrefix(err.Error(), "ancestry:"))
		assert.NotContains(t, err.Error(), "get child")
	})
}

func TestDescendants(t *testing.T) {
	t.Run("returns descendants edges and chain in depth-first order", func(t *testing.T) {
		t.Parallel()

		desc, edges, chain, err := Descendants(context.Background(), treeWalker(), "root")

		require.NoError(t, err)
		assert.Equal(t, []string{"root", "child-a", "grandchild", "child-b"}, desc)
		assert.Equal(t, []string{"child-a", "child-b"}, edges["root"])
		assert.Equal(t, []string{"grandchild"}, edges["child-a"])
		assert.Nil(t, edges["grandchild"])
		assert.Nil(t, edges["child-b"])
		assert.Equal(t, "root", chain["child-a"].ParentKey)
		assert.Equal(t, "child-a", chain["grandchild"].ParentKey)
	})

	t.Run("honors cancelled context before traversal starts", func(t *testing.T) {
		t.Parallel()

		called := false
		w := stubPIWalker{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				called = true
				return d.ProcessInstance{}, nil
			},
			getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
				called = true
				return nil, nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		desc, edges, chain, err := Descendants(ctx, w, "root")

		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.False(t, called)
		assert.Nil(t, desc)
		assert.Nil(t, edges)
		assert.Nil(t, chain)
	})

	t.Run("keeps descendants breadcrumb without restating the same key detail", func(t *testing.T) {
		t.Parallel()

		w := stubPIWalker{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				return d.ProcessInstance{}, nil
			},
			getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
				return nil, d.ErrNotFound
			},
		}

		_, _, _, err := Descendants(context.Background(), w, "root")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "descendants children")
		assert.NotContains(t, err.Error(), "descendants list children")
		assert.True(t, strings.HasPrefix(err.Error(), "descendants children:"))
		assert.NotContains(t, err.Error(), "list children of root")
	})
}

func TestFamily(t *testing.T) {
	t.Run("walks up to the root and then returns the whole family tree", func(t *testing.T) {
		t.Parallel()

		family, edges, chain, err := Family(context.Background(), treeWalker(), "grandchild")

		require.NoError(t, err)
		assert.Equal(t, []string{"root", "child-a", "grandchild", "child-b"}, family)
		assert.Equal(t, []string{"child-a", "child-b"}, edges["root"])
		assert.Equal(t, "child-a", chain["grandchild"].ParentKey)
	})

	t.Run("keeps family breadcrumb without the old ancestry fetch wording", func(t *testing.T) {
		t.Parallel()

		w := stubPIWalker{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				return d.ProcessInstance{}, d.ErrNotFound
			},
			getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
				t.Fatalf("unexpected GetDirectChildrenOfProcessInstance call")
				return nil, nil
			},
		}

		_, _, _, err := Family(context.Background(), w, "child")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "family")
		assert.NotContains(t, err.Error(), "family ancestry")
		assert.True(t, strings.HasPrefix(err.Error(), "family:"))
		assert.NotContains(t, err.Error(), "ancestry fetch")
	})
}

func TestTraversalResults(t *testing.T) {
	t.Run("ancestry result returns partial warning when parent is missing", func(t *testing.T) {
		t.Parallel()

		w := stubPIWalker{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				if key == "child" {
					return d.ProcessInstance{Key: "child", ParentKey: "missing"}, nil
				}
				return d.ProcessInstance{}, d.ErrNotFound
			},
			getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
				t.Fatalf("unexpected GetDirectChildrenOfProcessInstance call")
				return nil, nil
			},
		}

		result, err := pitraversal.BuildAncestryResult(context.Background(), stubTraversalAPI{stubPIWalker: w}, "child")

		require.NoError(t, err)
		assert.Equal(t, pitraversal.OutcomePartial, result.Outcome)
		assert.Equal(t, "child", result.RootKey)
		assert.Equal(t, []string{"child"}, result.Keys)
		assert.Equal(t, "one or more parent process instances were not found", result.Warning)
		require.Len(t, result.MissingAncestors, 1)
		assert.Equal(t, pitraversal.MissingAncestor{Key: "missing", StartKey: "child"}, result.MissingAncestors[0])
	})

	t.Run("family result keeps resolved descendants when ancestry is partial", func(t *testing.T) {
		t.Parallel()

		items := map[string]d.ProcessInstance{
			"child":      {Key: "child", ParentKey: "missing"},
			"grandchild": {Key: "grandchild", ParentKey: "child"},
		}

		w := stubPIWalker{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				it, ok := items[key]
				if !ok {
					return d.ProcessInstance{}, d.ErrNotFound
				}
				return it, nil
			},
			getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
				switch key {
				case "child":
					return []d.ProcessInstance{items["grandchild"]}, nil
				case "grandchild":
					return nil, nil
				default:
					return nil, d.ErrNotFound
				}
			},
		}

		result, err := pitraversal.BuildFamilyResult(context.Background(), stubTraversalAPI{stubPIWalker: w}, "child")

		require.NoError(t, err)
		assert.Equal(t, pitraversal.OutcomePartial, result.Outcome)
		assert.Equal(t, "child", result.RootKey)
		assert.Equal(t, []string{"child", "grandchild"}, result.Keys)
		assert.Equal(t, []string{"grandchild"}, result.Edges["child"])
		assert.Nil(t, result.Edges["grandchild"])
		require.Len(t, result.MissingAncestors, 1)
		assert.Equal(t, pitraversal.MissingAncestor{Key: "missing", StartKey: "child"}, result.MissingAncestors[0])
	})

	t.Run("family result fails when nothing can be resolved", func(t *testing.T) {
		t.Parallel()

		w := stubPIWalker{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				return d.ProcessInstance{}, d.ErrNotFound
			},
			getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
				t.Fatalf("unexpected GetDirectChildrenOfProcessInstance call")
				return nil, nil
			},
		}

		_, err := pitraversal.BuildFamilyResult(context.Background(), stubTraversalAPI{stubPIWalker: w}, "missing")

		require.Error(t, err)
		assert.ErrorIs(t, err, d.ErrNotFound)
	})
}

func treeWalker() stubPIWalker {
	items := map[string]d.ProcessInstance{
		"root":       {Key: "root"},
		"child-a":    {Key: "child-a", ParentKey: "root"},
		"child-b":    {Key: "child-b", ParentKey: "root"},
		"grandchild": {Key: "grandchild", ParentKey: "child-a"},
	}
	children := map[string][]d.ProcessInstance{
		"root":       {items["child-a"], items["child-b"]},
		"child-a":    {items["grandchild"]},
		"child-b":    nil,
		"grandchild": nil,
	}

	return stubPIWalker{
		getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
			it, ok := items[key]
			if !ok {
				return d.ProcessInstance{}, d.ErrNotFound
			}
			return it, nil
		},
		getDirectChildrenProcessInstance: func(ctx context.Context, key string) ([]d.ProcessInstance, error) {
			kids, ok := children[key]
			if !ok {
				return nil, d.ErrNotFound
			}
			return kids, nil
		},
	}
}
