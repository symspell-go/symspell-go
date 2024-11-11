package symspell

// SuggestionStage is used to temporarily stage dictionary data during the adding of many words.
type SuggestionStage struct {
	deletes map[int]Entry
	nodes   ChunkArrayNode
}

// Entry represents a delete entry.
type Entry struct {
	count int
	first int
}

// Node represents a node in the chunk array.
type Node struct {
	suggestion string
	next       int
}

// NewSuggestionStage creates a new instance of SuggestionStage.
func NewSuggestionStage(initialCapacity int) *SuggestionStage {
	return &SuggestionStage{
		deletes: make(map[int]Entry, initialCapacity),
		nodes:   NewChunkArrayNode(initialCapacity * 2),
	}
}

// DeleteCount returns the count of unique delete words.
func (ss *SuggestionStage) DeleteCount() int {
	return len(ss.deletes)
}

// NodeCount returns the total count of all suggestions for all deletes.
func (ss *SuggestionStage) NodeCount() int {
	return ss.nodes.Count()
}

// Clear clears all the data from the SuggestionStaging.
func (ss *SuggestionStage) Clear() {
	ss.deletes = make(map[int]Entry)
	ss.nodes.Clear()
}

// Add adds a delete hash and suggestion to the staging.
func (ss *SuggestionStage) Add(deleteHash int, suggestion string) {
	entry, found := ss.deletes[deleteHash]
	if !found {
		entry = Entry{count: 0, first: -1}
	}
	next := entry.first
	entry.count++
	entry.first = ss.nodes.Count()
	ss.deletes[deleteHash] = entry
	ss.nodes.Add(Node{suggestion: suggestion, next: next})
}

// CommitTo commits staged dictionary additions to the main deletes map.
func (ss *SuggestionStage) CommitTo(permanentDeletes map[int]map[string]struct{}) {
	//for key, entry := range ss.deletes {
	//	var i int
	//	suggestions, found := permanentDeletes[key]
	//	if found {
	//		i = len(suggestions)
	//		newSuggestions := make([]string, len(suggestions)+entry.count)
	//		copy(newSuggestions, suggestions)
	//		permanentDeletes[key] = newSuggestions
	//		suggestions = newSuggestions
	//	} else {
	//		i = 0
	//		suggestions = make([]string, entry.count)
	//		permanentDeletes[key] = suggestions
	//	}
	//	next := entry.first
	//	for next >= 0 {
	//		node := ss.nodes.Get(next)
	//		suggestions[i] = node.suggestion
	//		next = node.next
	//		i++
	//	}
	//}
}

// ChunkArrayNode is a growable list of Node elements optimized for adding.
type ChunkArrayNode struct {
	values [][]Node
	count  int
}

// Constants for ChunkArrayNode.
const (
	ChunkSize = 4096
	DivShift  = 12
)

// NewChunkArrayNode creates a new ChunkArrayNode.
func NewChunkArrayNode(initialCapacity int) ChunkArrayNode {
	chunks := (initialCapacity + ChunkSize - 1) / ChunkSize
	values := make([][]Node, chunks)
	for i := range values {
		values[i] = make([]Node, ChunkSize)
	}
	return ChunkArrayNode{
		values: values,
		count:  0,
	}
}

// Add adds a Node to the ChunkArrayNode.
func (ca *ChunkArrayNode) Add(value Node) int {
	if ca.count == ca.capacity() {
		newValues := make([][]Node, len(ca.values)+1)
		copy(newValues, ca.values)
		newValues[len(ca.values)] = make([]Node, ChunkSize)
		ca.values = newValues
	}
	row := ca.row(ca.count)
	col := ca.col(ca.count)
	ca.values[row][col] = value
	ca.count++
	return ca.count - 1
}

// Count returns the number of Nodes in the ChunkArrayNode.
func (ca *ChunkArrayNode) Count() int {
	return ca.count
}

// Get retrieves a Node by index.
func (ca *ChunkArrayNode) Get(index int) Node {
	row := ca.row(index)
	col := ca.col(index)
	return ca.values[row][col]
}

// Clear resets the ChunkArrayNode.
func (ca *ChunkArrayNode) Clear() {
	ca.count = 0
}

func (ca *ChunkArrayNode) capacity() int {
	return len(ca.values) * ChunkSize
}

func (ca *ChunkArrayNode) row(index int) int {
	return index >> DivShift
}

func (ca *ChunkArrayNode) col(index int) int {
	return index & (ChunkSize - 1)
}
