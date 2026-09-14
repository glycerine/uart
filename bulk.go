package uart

// BulkItem is one key/value entry for sorted ART construction.
type BulkItem struct {
	Key   Key
	Value any
}

// NewArtTreeFromSortedNoCopy builds an ART from items sorted by Key.
//
// The keys and values are not copied. Callers must not modify key bytes while
// the returned tree is in use. Supplying unsorted or duplicate keys can produce
// an invalid tree; this constructor is intentionally lean for bulk memtable
// build paths that already have sorted input.
func NewArtTreeFromSortedNoCopy(items []BulkItem) *Tree {
	t := NewArtTree()
	if len(items) == 0 {
		return t
	}
	t.root = t.buildSortedNoCopy(items, 0)
	t.size = int64(len(items))
	t.treeVersion = 1
	return t
}

func (t *Tree) buildSortedNoCopy(items []BulkItem, depth int) *bnode {
	if len(items) == 1 {
		lf := t.newLeaf(items[0].Key, items[0].Value)
		return t.newBnodeLeaf(lf)
	}

	prefixLen := commonPrefixLenAt(items[0].Key, items[len(items)-1].Key, depth)
	compressed := Key(nil)
	if prefixLen > 0 {
		compressed = items[0].Key[depth : depth+prefixLen]
		depth += prefixLen
	}

	var groupKeys [256]byte
	var groupStarts [257]int
	var groups int
	for start := 0; start < len(items); {
		groupStarts[groups] = start
		groupKeys[groups] = items[start].Key.At(depth)
		end := start + 1
		for end < len(items) && items[end].Key.At(depth) == groupKeys[groups] {
			end++
		}
		groups++
		groupStarts[groups] = end
		start = end
	}

	n := t.newInner(t.newNodeForFanout(groups), len(items))
	n.compressed = compressed

	for g := 0; g < groups; g++ {
		keyb := groupKeys[g]
		start := groupStarts[g]
		end := groupStarts[g+1]
		child := t.buildSortedNoCopy(items[start:end], depth+1)
		if child.isLeaf {
			child.leaf.keybyte = keyb
		} else {
			child.inner.keybyte = keyb
		}
		n.Node.addChild(keyb, child)
	}

	return t.newBnodeInner(n)
}

func commonPrefixLenAt(a, b Key, depth int) int {
	if depth >= len(a) || depth >= len(b) {
		return 0
	}
	return commonPrefixLen(a[depth:], b[depth:])
}

func (t *Tree) newNodeForFanout(n int) inode {
	switch {
	case n <= 4:
		return t.newNode4()
	case n <= 16:
		return t.newNode16()
	case n <= 48:
		return t.newNode48()
	default:
		return t.newNode256()
	}
}
