package interval

import (
	"iter"
	"maps"
	"math"
	"slices"
)

// branchingFactorPower is the power that 2 is raised to in order to get the
// branching factor of the hierarchical interval tree.
//
// i.e. 4 -> 2^4 = 16
const branchingFactorPower = 4

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
	root   node
	values []Value
}

// New creates a new interval tree.
func New[Value any]() Tree[Value] {
	return &tree[Value]{
		root: newHierarchicalNode(),
	}
}

// Add inserts a new interval into the interval tree.
func (t *tree[Value]) Add(interval Interval, value Value) {
	valuesIndex := len(t.values)
	t.values = append(t.values, value)

	// Add the interval, using the new values index.
	t.root.Add(interval, valuesIndex)
}

// AllIntersections returns all values in the interval tree that intersect with
// the given interval.
func (t *tree[Value]) AllIntersections(start uint64, end uint64) ([]Value, bool) {
	indices := t.root.AllIntersections(start, end)

	if len(indices) == 0 {
		return nil, false
	}

	values := make([]Value, 0, len(indices))

	for index := range indices {
		values = append(values, t.values[index])
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
	Add(interval Interval, valuesIndex int)
	AllIntersections(start, end uint64) valueIndices
}

// hierarchicalNode is a node that has several children nodes, bucketed by the
// index.
type hierarchicalNode struct {
	children []node
}

// newHierarchicalNode creates a new hierarchical node.
func newHierarchicalNode() *hierarchicalNode {
	node := hierarchicalNode{
		children: make([]node, 1<<branchingFactorPower),
	}

	for i := range node.children {
		node.children[i] = &leafNode{}
	}

	return &node
}

// Ensure that hierarchicalNode implements the [node] interface.
var _ node = &hierarchicalNode{}

// leafNode is a node that stores the intervals directly.
type leafNode struct {
	indices   []int
	intervals []Interval
}

var _ node = &leafNode{}

// Add inserts a new interval into the interval tree.
func (h *hierarchicalNode) Add(interval Interval, valuesIndex int) {
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
		End:   math.MaxInt64,
	}

	for i := startBucketIndex; i <= endBucketIndex; i++ {
		if i > startBucketIndex {
			newInterval.Start = 0
		}

		if i == endBucketIndex {
			newInterval.End = interval.End << branchingFactorPower
		}

		h.children[i].Add(newInterval, valuesIndex)
	}
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
		bucketOffsetEnd   uint64 = math.MaxInt64
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

		intersections := h.children[i].AllIntersections(bucketOffsetStart, bucketOffsetEnd)

		if len(intersections) > 0 {
			matchingIndices.Merge(intersections)
		}
	}

	return matchingIndices
}

// Add inserts a new interval into the interval tree.
func (l *leafNode) Add(interval Interval, valuesIndex int) {
	l.intervals = append(l.intervals, interval)
	l.indices = append(l.indices, valuesIndex)
}

// AllIntersections returns all values in the interval tree that intersect with
// the given interval.
//
// This node is a leaf node, so it requires a linear scan of the values.
func (l leafNode) AllIntersections(start uint64, end uint64) valueIndices {
	// Optimize for the case where we're looking for all intervals in this
	// bucket.
	if start == 0 && end == math.MaxInt64 {
		valueIndices := make(valueIndices, len(l.intervals))

		for i := range l.intervals {
			valueIndices[l.indices[i]] = struct{}{}
		}

		return valueIndices
	}

	matchingIndices := make(valueIndices, len(l.intervals))

	for i, interval := range l.intervals {
		if end < interval.Start || start > interval.End {
			continue
		}

		matchingIndices[l.indices[i]] = struct{}{}
	}

	return matchingIndices
}