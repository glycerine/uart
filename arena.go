package uart

const (
	bnodeArenaChunkLen   = 4096
	leafArenaChunkLen    = 4096
	innerArenaChunkLen   = 2048
	node4ArenaChunkLen   = 2048
	node16ArenaChunkLen  = 1024
	node48ArenaChunkLen  = 128
	node256ArenaChunkLen = 64
)

type bnodeArena struct {
	chunks [][]bnode
	next   int
}

func (a *bnodeArena) alloc() *bnode {
	if len(a.chunks) == 0 || a.next == len(a.chunks[len(a.chunks)-1]) {
		a.chunks = append(a.chunks, make([]bnode, bnodeArenaChunkLen))
		a.next = 0
	}
	chunk := a.chunks[len(a.chunks)-1]
	p := &chunk[a.next]
	a.next++
	return p
}

type leafArena struct {
	chunks [][]Leaf
	next   int
}

func (a *leafArena) alloc() *Leaf {
	if len(a.chunks) == 0 || a.next == len(a.chunks[len(a.chunks)-1]) {
		a.chunks = append(a.chunks, make([]Leaf, leafArenaChunkLen))
		a.next = 0
	}
	chunk := a.chunks[len(a.chunks)-1]
	p := &chunk[a.next]
	a.next++
	return p
}

type innerArena struct {
	chunks [][]inner
	next   int
}

func (a *innerArena) alloc() *inner {
	if len(a.chunks) == 0 || a.next == len(a.chunks[len(a.chunks)-1]) {
		a.chunks = append(a.chunks, make([]inner, innerArenaChunkLen))
		a.next = 0
	}
	chunk := a.chunks[len(a.chunks)-1]
	p := &chunk[a.next]
	a.next++
	return p
}

type node4Arena struct {
	chunks [][]node4
	next   int
}

func (a *node4Arena) alloc() *node4 {
	if len(a.chunks) == 0 || a.next == len(a.chunks[len(a.chunks)-1]) {
		a.chunks = append(a.chunks, make([]node4, node4ArenaChunkLen))
		a.next = 0
	}
	chunk := a.chunks[len(a.chunks)-1]
	p := &chunk[a.next]
	a.next++
	return p
}

type node16Arena struct {
	chunks [][]node16
	next   int
}

func (a *node16Arena) alloc() *node16 {
	if len(a.chunks) == 0 || a.next == len(a.chunks[len(a.chunks)-1]) {
		a.chunks = append(a.chunks, make([]node16, node16ArenaChunkLen))
		a.next = 0
	}
	chunk := a.chunks[len(a.chunks)-1]
	p := &chunk[a.next]
	a.next++
	return p
}

type node48Arena struct {
	chunks [][]node48
	next   int
}

func (a *node48Arena) alloc() *node48 {
	if len(a.chunks) == 0 || a.next == len(a.chunks[len(a.chunks)-1]) {
		a.chunks = append(a.chunks, make([]node48, node48ArenaChunkLen))
		a.next = 0
	}
	chunk := a.chunks[len(a.chunks)-1]
	p := &chunk[a.next]
	a.next++
	return p
}

type node256Arena struct {
	chunks [][]node256
	next   int
}

func (a *node256Arena) alloc() *node256 {
	if len(a.chunks) == 0 || a.next == len(a.chunks[len(a.chunks)-1]) {
		a.chunks = append(a.chunks, make([]node256, node256ArenaChunkLen))
		a.next = 0
	}
	chunk := a.chunks[len(a.chunks)-1]
	p := &chunk[a.next]
	a.next++
	return p
}

func (t *Tree) newLeaf(key Key, value any) *Leaf {
	if t == nil {
		return NewLeaf(key, value, nil)
	}
	if t.orderedLeaves != nil && t.orderedLeafNext < len(t.orderedLeaves) {
		lf := &t.orderedLeaves[t.orderedLeafNext]
		t.orderedLeafNext++
		*lf = Leaf{Key: key, Value: value}
		return lf
	}
	lf := t.leafArena.alloc()
	*lf = Leaf{Key: key, Value: value}
	return lf
}

func (t *Tree) newBnodeLeaf(lf *Leaf) *bnode {
	if t == nil {
		return bnodeLeaf(lf)
	}
	b := t.bnodeArena.alloc()
	*b = bnode{leaf: lf, isLeaf: true}
	return b
}

func (t *Tree) newBnodeInner(n *inner) *bnode {
	if t == nil {
		return bnodeInner(n)
	}
	b := t.bnodeArena.alloc()
	*b = bnode{inner: n}
	return b
}

func (t *Tree) newInner(node inode, subN int) *inner {
	if t == nil {
		return &inner{Node: node, SubN: subN}
	}
	n := t.innerArena.alloc()
	*n = inner{Node: node, SubN: subN}
	return n
}

func (t *Tree) newNode4() *node4 {
	if t == nil {
		return &node4{}
	}
	n := t.node4Arena.alloc()
	*n = node4{}
	return n
}

func (t *Tree) newNode16() *node16 {
	if t == nil {
		return &node16{}
	}
	n := t.node16Arena.alloc()
	*n = node16{}
	return n
}

func (t *Tree) newNode48() *node48 {
	if t == nil {
		return &node48{}
	}
	n := t.node48Arena.alloc()
	*n = node48{}
	return n
}

func (t *Tree) newNode256() *node256 {
	if t == nil {
		return &node256{}
	}
	n := t.node256Arena.alloc()
	*n = node256{}
	return n
}

func growNode(nd inode, t *Tree) inode {
	if t == nil {
		return nd.grow()
	}
	switch n := nd.(type) {
	case *node4:
		nn := t.newNode16()
		nn.lth = n.lth
		copy(nn.keys[:], n.keys[:])
		copy(nn.children[:], n.children[:])
		nn.redoPren()
		return nn
	case *node16:
		nn := t.newNode48()
		nn.lth = n.lth
		copy(nn.children[:], n.children[:])
		for i, child := range n.children {
			if child != nil {
				nn.keys[n.keys[i]] = uint16(i) + 1
			}
		}
		nn.redoPren()
		return nn
	case *node48:
		nn := t.newNode256()
		nn.lth = n.lth
		for b, i := range n.keys {
			if i != 0 {
				nn.children[b] = n.children[i-1]
			}
		}
		nn.redoPren()
		return nn
	default:
		return nd.grow()
	}
}

func (t *Tree) prefixBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	if t != nil && t.SharePrefixBytes {
		return prefix
	}
	return append([]byte{}, prefix...)
}
