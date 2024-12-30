# Multi-Binned Interval Tree

An alternative to regular interval trees, that bins values into multiple bins to
optimise intersection lookup.

## Usage

```go
import "github.com/crystalix007/multi-binned-intervals/interval"

func main() {
    intervals := interval.New[string]()

    intervals.Add(interval.Interval{1, 5}, "first")
    intervals.Add(interval.Interval{7, 10}, "second")
    intervals.Add(interval.Interval{1, 2}, "third")

    intersections, ok := intervals.AllIntersections(5, 8)

    fmt.Printf("Found intersecting values: %b\n", ok)

    if ok {
        // Order is non-determinate.
        slices.Sort(intersections)

        fmt.Printf("Values: %v", intersections)
    }

    // Output:
    // Found intersecting values: true
    // Values: [first, second]
}
```
