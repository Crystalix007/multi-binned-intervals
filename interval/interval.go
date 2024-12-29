package interval

import (
	"iter"
	"maps"
	"math"
	"slices"
)

const (
	// branchingFactorPower is the power that 2 is raised to in order to get the
	// branching factor of the hierarchical interval tree.
	//
	// i.e. 4 -> 2^4 = 16
	branchingFactorPower uint64 = 4

	// hierarchicalFanout is the number of child elements in a single
	// hierarchical node.
	hierarchicalFanout uint64 = 1 << branchingFactorPower

	// offsetMask is the mask used to extract the offset from the interval
	// start and end.
	offsetMask uint64 = 1<<branchingFactorPower - 1

	// bucketMask is the mask used to extract the bucket from the interval start
	// and end.
	bucketMask uint64 = ^offsetMask

	// maxLeafFanout is the maximum number of intervals that a leaf node can
	// store.
	//
	// This is a performance tweakable that can be adjusted based on the
	// expected number of intervals.
	maxLeafFanout = 16
)

// Interval represents the closed interval [Start, End].
type Interval struct {
	Start uint64
	End   uint64
}

// Tree is a node in a hierarchical interval tree.
type Tree[Value any] interface {
	Add(interval Interval, value Value)
	AllIntersections(start, end uint64) ([]Value, bool)
}

// tree is a hierarchical interval tree.
//
// It stores the intervals in a tree structure, and the values in a separate
// slice.
type tree[Value any] struct {
	Root   node
	Values []Value
}

// New creates a new interval tree.
func New[Value any]() Tree[Value] {
	return &tree[Value]{
		Root: newHierarchicalNode(),
	}
}

// Add inserts a new interval into the interval tree.
func (t *tree[Value]) Add(interval Interval, value Value) {
	valuesIndex := len(t.Values)
	t.Values = append(t.Values, value)

	// Add the interval, using the new values index.
	t.Root.Add(interval, valuesIndex)
}

// AllIntersections returns all values in the interval tree that intersect with
// the given interval.
func (t *tree[Value]) AllIntersections(start uint64, end uint64) ([]Value, bool) {
	indices := t.Root.AllIntersections(start, end)

	if len(indices) == 0 {
		return nil, false
	}

	values := make([]Value, 0, len(indices))

	for index := range indices {
		values = append(values, t.Values[index])
	}

	return values, true
}

// valueIndices is a set of value indices.
type valueIndices map[int]struct{}

// Merge merges the other value indices into this set.
func (v *valueIndices) Merge(other valueIndices) {
	for index := range other {
		(*v)[index] = struct{}{}
	}
}

// All returns an iterator over the value indices.
func (v valueIndices) All() iter.Seq[int] {
	return func(yield func(int) bool) {
		for index := range v {
			if !yield(index) {
				return
			}
		}

		return
	}
}

// Sorted returns the value indices in sorted order.
func (v valueIndices) Sorted() []int {
	indices := slices.Collect(maps.Keys(v))

	slices.Sort(indices)

	return indices
}

// node is the interface that all node types in the interval tree implement.
type node interface {
	Add(interval Interval, valuesIndex int) node
	AllIntersections(start, end uint64) valueIndices
}

// hierarchicalNode is a node that has several children nodes, bucketed by the
// index.
type hierarchicalNode struct {
	Children []node
}

// newHierarchicalNode creates a new hierarchical node.
func newHierarchicalNode() *hierarchicalNode {
	node := hierarchicalNode{
		Children: make([]node, hierarchicalFanout),
	}

	return &node
}

// Ensure that hierarchicalNode implements the [node] interface.
var _ node = &hierarchicalNode{}

// leafNode is a node that stores the intervals directly.
type leafNode struct {
	Indices   []int
	Intervals []Interval
}

var _ node = &leafNode{}

// Add inserts a new interval into the interval tree.
func (h *hierarchicalNode) Add(interval Interval, valuesIndex int) node {
	// Indices are split into two parts:
	//
	// MSB Bits:  0123    4567 ...
	//           Bucket  Offset...
	//
	// Therefore we shift down by 64 - 4 to get the bucket index.
	startBucketIndex := interval.Start >> (64 - branchingFactorPower)
	endBucketIndex := interval.End >> (64 - branchingFactorPower)

	newInterval := Interval{
		Start: interval.Start << branchingFactorPower,
		End:   math.MaxUint64,
	}

	for i := startBucketIndex; i <= endBucketIndex; i++ {
		if i > startBucketIndex {
			newInterval.Start = 0
		}

		if i == endBucketIndex {
			newInterval.End = interval.End << branchingFactorPower
		}

		if h.Children[i] == nil {
			h.Children[i] = &leafNode{}
		}

		h.Children[i] = h.Children[i].Add(newInterval, valuesIndex)
	}

	return h
}

// AllIntersections returns all values in the interval tree that intersect with
// the given interval.
func (h hierarchicalNode) AllIntersections(start uint64, end uint64) valueIndices {
	// Indices are split into two parts:
	//
	// MSB Bits:  0123    4567 ...
	//           Bucket  Offset...
	//
	// Therefore we shift down by 64 - 4 to get the bucket index.
	startBucketIndex := start >> (64 - branchingFactorPower)
	endBucketIndex := end >> (64 - branchingFactorPower)

	matchingIndices := make(valueIndices)

	// The new "offset" indices to search for within the bucket.
	//
	// | Bucket 0 | Bucket 1 | Bucket 2 | ...
	//     ^--------------------^
	//   start                 end
	//
	// This start bucket offset is only valid for the first bucket, as other
	// buckets should be searched from the very beginning.
	var (
		bucketOffsetStart uint64 = start << branchingFactorPower
		bucketOffsetEnd   uint64 = math.MaxUint64
	)

	for i := startBucketIndex; i <= endBucketIndex; i++ {
		if i > startBucketIndex {
			bucketOffsetStart = 0
		}

		// If we're at the last bucket, we need to set the end offset to the
		// end of the interval.
		if i == endBucketIndex {
			bucketOffsetEnd = end << branchingFactorPower
		}

		// If the bucket is nil, then there are no intervals in this bucket.
		if h.Children[i] == nil {
			continue
		}

		intersections := h.Children[i].AllIntersections(bucketOffsetStart, bucketOffsetEnd)

		if len(intersections) > 0 {
			matchingIndices.Merge(intersections)
		}
	}

	return matchingIndices
}

// Add inserts a new interval into the interval tree.
func (l *leafNode) Add(interval Interval, valuesIndex int) node {
	// If this leaf node is full, we need to convert it into a hierarchical
	// node.
	//
	// Only check if len(l.intervals) == maxLeafFanout, or 2 * maxLeafFanout,
	// or 3 * maxLeafFanout, etc. since we don't want to check too frequently,
	// but we also don't want to only check once and then reach unbounded
	// growth.
	if len(l.Intervals) > 0 &&
		len(l.Intervals)%maxLeafFanout == 0 &&
		l.shouldSplit() {
		h := newHierarchicalNode()

		for i, interval := range l.Intervals {
			h.Add(interval, l.Indices[i])
		}

		return h.Add(interval, valuesIndex)
	}

	l.Intervals = append(l.Intervals, interval)
	l.Indices = append(l.Indices, valuesIndex)

	return l
}

// shouldSplit returns whether the leaf node should be split.
//
// This checks if there would be any benefit to splitting the leaf node.
func (l *leafNode) shouldSplit() bool {
	hierarchicalStartOffsetCount := make(map[uint64]int, len(l.Intervals))
	hierarchicalEndOffsetCount := make(map[uint64]int, len(l.Intervals))

	for _, interval := range l.Intervals {
		allButLastOffsetStart := interval.Start & bucketMask
		allButLastOffsetEnd := interval.End & bucketMask

		hierarchicalStartOffsetCount[allButLastOffsetStart]++
		hierarchicalEndOffsetCount[allButLastOffsetEnd]++
	}

	for _, count := range hierarchicalStartOffsetCount {
		// If any bucket contains some intervals, but not all of them, then we
		// should split the leaf node.
		if count != 0 && count != len(l.Intervals) {
			return true
		}
	}

	for _, count := range hierarchicalEndOffsetCount {
		// If any bucket contains some intervals, but not all of them, then we
		// should split the leaf node.
		if count != 0 && count != len(l.Intervals) {
			return true
		}
	}

	return false
}

// AllIntersections returns all values in the interval tree that intersect with
// the given interval.
//
// This node is a leaf node, so it requires a linear scan of the values.
func (l leafNode) AllIntersections(start uint64, end uint64) valueIndices {
	// Optimize for the case where we're looking for all intervals in this
	// bucket.
	if start == 0 && end == math.MaxUint64 {
		valueIndices := make(valueIndices, len(l.Intervals))

		for i := range l.Intervals {
			valueIndices[l.Indices[i]] = struct{}{}
		}

		return valueIndices
	}

	matchingIndices := make(valueIndices, len(l.Intervals))

	for i, interval := range l.Intervals {
		if end < interval.Start || start > interval.End {
			continue
		}

		matchingIndices[l.Indices[i]] = struct{}{}
	}

	return matchingIndices
}
