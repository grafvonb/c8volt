package walker

import (
	"context"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPIWalker struct {
	getProcessInstance               func(ctx context.Context, key string) (d.ProcessInstance, error)
	getDirectChildrenProcessInstance func(ctx context.Context, key string) ([]d.ProcessInstance, error)
}

func (s stubPIWalker) GetProcessInstance(ctx context.Context, key string, _ ...services.CallOption) (d.ProcessInstance, error) {
	return s.getProcessInstance(ctx, key)
}

func (s stubPIWalker) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstance, error) {
	return s.getDirectChildrenProcessInstance(ctx, key)
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
		assert.Nil(t, path)
		assert.Equal(t, d.ProcessInstance{Key: "child", ParentKey: "missing"}, chain["child"])
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
