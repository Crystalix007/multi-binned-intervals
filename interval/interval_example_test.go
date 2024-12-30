package interval_test

import (
	"fmt"
	"slices"

	"github.com/crystalix007/multi-binned-intervals/interval"
)

func Example() {
	intervals := interval.New[string]()

	intervals.Add(interval.Interval{1, 5}, "first")
	intervals.Add(interval.Interval{7, 10}, "second")
	intervals.Add(interval.Interval{1, 2}, "third")

	intersections, ok := intervals.AllIntersections(5, 8)

	fmt.Printf("Found intersecting values: %t\n", ok)

	if ok {
		// Order is non-determinate.
		slices.Sort(intersections)

		fmt.Printf("Values: %v", intersections)
	}

	// Output:
	// Found intersecting values: true
	// Values: [first second]
}
