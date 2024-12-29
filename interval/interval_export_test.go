package interval

const (
	// MaxLeafFanout is the maximum number of intervals that a leaf node can
	// store. Re-exported [maxLeafFanout] for testing purposes.
	MaxLeafFanout = maxLeafFanout

	// BranchingFactorPower is the power of 2 that determines the branching
	// factor of the hierarchical interval tree. Re-exported
	// [branchingFactorPower].
	BranchingFactorPower = branchingFactorPower

	// HierarchicalFanout is the maximum number of children that a hierarchical
	// node can store. Re-exported [hierarchicalFanout] for testing purposes.
	HierarchicalFanout = hierarchicalFanout
)

// Node reexports the internal [node] type.
type Node = node

// ValueIndices reexports the internal [valueIndices] type.
type ValueIndices = valueIndices

// HierarchicalNode reexports the internal [hierarchicalNode] type.
type HierarchicalNode = hierarchicalNode

// LeafNode reexports the internal [leafNode] type.
type LeafNode = leafNode

// ShouldSplit reexports the internal [shouldSplit] method.
func (l *LeafNode) ShouldSplit() bool {
	return l.shouldSplit()
}
