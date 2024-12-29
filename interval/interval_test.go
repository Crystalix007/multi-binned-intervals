package interval_test

import (
	"math"
	"slices"
	"testing"

	"github.com/crystalix007/multi-binned-intervals/interval"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestLeafNode_ShouldSplit_unsplittable(t *testing.T) {
	t.Parallel()

	leaf := interval.LeafNode{
		Indices: []int{0},
		Intervals: []interval.Interval{
			{Start: 0, End: 0},
		},
	}

	// Add more intervals than the predicate to split, but with indices that
	// cannot be split efficiently.
	for i := range interval.MaxLeafFanout {
		leaf.Add(interval.Interval{Start: uint64(1), End: uint64(1)}, 1+i)
	}

	require.False(t, leaf.ShouldSplit())
}

func TestLeafNode_ShouldSplit_splittable(t *testing.T) {
	t.Parallel()

	// Create a leaf node with 16 intervals that can be split across different
	// buckets (i.e. are different in more than their last 4 bits).
	var leaf interval.LeafNode

	for i := range interval.MaxLeafFanout {
		leaf.Add(interval.Interval{Start: uint64(i), End: uint64(i + 1)}, i)
	}

	require.True(t, leaf.ShouldSplit())
}

func TestLeafNode_Add_splitting(t *testing.T) {
	t.Parallel()

	var leaf interval.Node = &interval.LeafNode{}

	// Create a leaf node with enough nodes to be considered for branching.
	subBucketSize := 1 << (64 - interval.BranchingFactorPower)

	for i := range interval.MaxLeafFanout {
		leaf = leaf.Add(interval.Interval{Start: uint64(i * subBucketSize), End: uint64(i * subBucketSize)}, i)
	}

	var expectedLeafNode interval.LeafNode

	for i := range interval.MaxLeafFanout {
		expectedLeafNode.Indices = append(expectedLeafNode.Indices, i)
		expectedLeafNode.Intervals = append(expectedLeafNode.Intervals, interval.Interval{Start: uint64(i * subBucketSize), End: uint64(i * subBucketSize)})
	}

	require.Equal(t, &expectedLeafNode, leaf)

	// Add one more interval to trigger the split.
	leaf = leaf.Add(interval.Interval{Start: 1, End: 1}, interval.MaxLeafFanout)

	expectedHierarchicalNode := interval.HierarchicalNode{
		Children: make([]interval.Node, interval.HierarchicalFanout),
	}

	// The leaf node should have been converted into a hierarchical node, but
	// the new leaf nodes should be the first child with the same elements.
	for i := range interval.MaxLeafFanout {
		expectedNewLeafNode := interval.LeafNode{
			Indices:   []int{i},
			Intervals: []interval.Interval{{Start: 0, End: 0}},
		}

		expectedHierarchicalNode.Children[i] = &expectedNewLeafNode
	}

	// Now we also need to add the final interval.
	firstLeafNode := expectedHierarchicalNode.Children[0].(*interval.LeafNode)
	firstLeafNode.Indices = append(firstLeafNode.Indices, interval.MaxLeafFanout)

	// Has to be a new interval, because the previous one was split, and this
	// has thus been left-shifted by [interval.BranchingFactorPower] bits.
	firstLeafNode.Intervals = append(firstLeafNode.Intervals, interval.Interval{Start: interval.HierarchicalFanout, End: interval.HierarchicalFanout})

	require.Equal(t, &expectedHierarchicalNode, leaf)
}

func TestTree(t *testing.T) {
	t.Parallel()

	// Add some values.
	tree := interval.New[string]()

	tree.Add(interval.Interval{Start: 0, End: 10}, "a")
	tree.Add(interval.Interval{Start: 3000, End: (math.MaxUint64 / 16) * 2}, "b")
	tree.Add(interval.Interval{Start: math.MaxUint64 - 16, End: math.MaxUint64}, "c")

	// Check the intersections.

	t.Run("EqualToInterval", func(t *testing.T) {
		t.Parallel()

		intersections, ok := tree.AllIntersections(0, 10)

		require.True(t, ok)
		require.Equal(t, []string{"a"}, intersections)
	})

	t.Run("OverlappingInterval", func(t *testing.T) {
		t.Parallel()

		intersections, ok := tree.AllIntersections(5, 15)

		require.True(t, ok)
		require.Equal(t, []string{"a"}, intersections)
	})

	t.Run("NoIntersectingInterval", func(t *testing.T) {
		t.Parallel()

		intersections, ok := tree.AllIntersections(11, 20)

		require.False(t, ok)
		require.Nil(t, intersections)
	})

	t.Run("MaxInt64", func(t *testing.T) {
		t.Parallel()

		intersections, ok := tree.AllIntersections(math.MaxUint64, math.MaxUint64)

		require.True(t, ok)
		require.Equal(t, []string{"c"}, intersections)
	})

	t.Run("StraddlingBuckets", func(t *testing.T) {
		t.Parallel()

		intersections, ok := tree.AllIntersections(7, (math.MaxUint64/16)*10)

		require.True(t, ok)
		require.ElementsMatch(t, []string{"a", "b"}, intersections)
	})
}

func TestTree_resizing(t *testing.T) {
	t.Parallel()

	tree := interval.New[int]()

	for i := 0; i < 100; i++ {
		tree.Add(interval.Interval{Start: uint64(i), End: uint64(i + 1)}, i)
	}

	intersections, ok := tree.AllIntersections(0, 1600)

	require.True(t, ok)

	slices.Sort(intersections)
	spew.Dump(intersections)

	require.Len(t, intersections, 100)
}
