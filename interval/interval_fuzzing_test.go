package interval_test

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/crystalix007/multi-binned-intervals/interval"
)

func FuzzTree(f *testing.F) {
	f.Fuzz(func(
		t *testing.T,
		intervalCount uint16,
		intervalEndpoint1 uint64,
		intervalEndpoint2 uint64,
	) {
		intervals := make([]interval.Interval, intervalCount)
		values := make([]uint64, intervalCount)

		intervalBegin, intervalEnd := sortInterval(intervalEndpoint1, intervalEndpoint2)

		for i := range intervalCount {
			values[i] = rand.Uint64()

			valueIntervalEndpoint1 := rand.Uint64()
			valueIntervalEndpoint2 := rand.Uint64()

			valueIntervalBegin, valueIntervalEnd := sortInterval(valueIntervalEndpoint1, valueIntervalEndpoint2)

			intervals[i] = interval.Interval{
				Start: valueIntervalBegin,
				End:   valueIntervalEnd,
			}
		}

		tree := interval.New[uint64]()

		for i := range intervalCount {
			tree.Add(intervals[i], values[i])
		}

		treeIntersectionValues, foundIntersections := tree.AllIntersections(intervalBegin, intervalEnd)

		expectedIntersections := getExpectedIntersections(intervalBegin, intervalEnd, intervals)

		require.Equal(t, len(expectedIntersections) > 0, foundIntersections)

		expectedIntersectionValues := make([]uint64, len(expectedIntersections))

		for i, intersection := range expectedIntersections {
			expectedIntersectionValues[i] = values[intersection]
		}

		require.ElementsMatch(t, expectedIntersectionValues, treeIntersectionValues)
	})
}

func getExpectedIntersections(intervalBegin, intervalEnd uint64, intervals []interval.Interval) []uint64 {
	intersections := make([]uint64, 0, len(intervals))

	for i, interval := range intervals {
		if intervalEnd < interval.Start || interval.End < intervalBegin {
			continue
		}

		intersections = append(intersections, uint64(i))
	}

	return intersections
}

func sortInterval(endpoint1, endpoint2 uint64) (uint64, uint64) {
	if endpoint1 < endpoint2 {
		return endpoint1, endpoint2
	}

	return endpoint2, endpoint1
}
