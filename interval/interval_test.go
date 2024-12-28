package interval_test

import (
	"math"
	"testing"

	"github.com/crystalix007/multi-binned-intervals/interval"
	"github.com/stretchr/testify/require"
)

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