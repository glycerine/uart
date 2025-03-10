package art

import (
	"bytes"
	"iter"
)

// iterator will scan the tree in lexicographic order.
type iterator struct {
	tree *Tree

	closed bool

	start   []byte
	end     []byte
	reverse bool

	begIdx  int
	endxIdx int

	initialized bool
	started     bool

	// current:
	key   []byte
	value any
	leaf  *Leaf
}

// Next will iterate over all leaf nodes between specified prefixes
func (i *iterator) Next() (ok bool) {
	if i.closed {
		return false
	}
	if !i.initialized {
		// initialize iterator
		if exit, next := i.init(); exit {
			return next
		}
	}
	return i.iterate()
}

func (i *iterator) Leaf() *Leaf {
	return i.leaf
}

func (i *iterator) Value() any {
	return i.value
}

func (i *iterator) Key() Key {
	return i.key
}

func (i *iterator) inRange(key []byte) bool {
	if i.reverse {
		return (len(i.start) == 0 || (bytes.Compare(key, i.start) <= 0)) &&
			(len(i.end) == 0 || bytes.Compare(key, i.end) > 0)
	}
	return (len(i.start) == 0 || (bytes.Compare(key, i.start) >= 0)) &&
		(len(i.end) == 0 || bytes.Compare(key, i.end) < 0)
}

// exit returned true means only 0 or 1 nodes in tree,
// so Next() won't call iterate.
func (i *iterator) init() (exit bool, nextOK bool) {

	root := i.tree.root
	if root == nil {
		i.closed = true
		return true, false
	}

	if root.isLeaf {
		l := root.leaf
		i.closed = true
		if i.inRange(l.Key) {
			i.key = l.Key
			i.value = l.Value
			return true, true
		}
		return true, false
	}
	return false, false
}

func (i *iterator) iterate() bool {

	if i.reverse {
		if i.begIdx <= i.endxIdx {
			i.closed = true
			return false
		}
	} else {
		if i.begIdx >= i.endxIdx {
			i.closed = true
			return false
		}
	}
	lf, ok := i.tree.At(i.begIdx)
	if !ok {
		i.closed = true
		return false
	}
	if !i.inRange(lf.Key) {
		i.closed = true
		return false
	}
	if i.reverse {
		i.begIdx--
	} else {
		i.begIdx++
	}
	i.key = lf.Key
	i.value = lf.Value
	i.leaf = lf

	return true
}

func Ascend(t *Tree, beg, endx Key) iter.Seq2[Key, any] {
	return func(yield func(key Key, value any) bool) {
		//if t.Size() == 0 {
		//	return
		//}
		it := t.Iterator(beg, endx)
		for it.Next() {
			if !yield(it.Key(), it.Value()) {
				return
			}
		}
	}
}

// dfs does depth-first-search.
//
// Useful for debugging/visualizing
// the full tree. Used in some tests.
func dfs(root *bnode) iter.Seq2[*bnode, bool] {
	return func(yield func(*bnode, bool) bool) {

		// Helper function for recursive traversal
		var visit func(keybyte byte, root *bnode, depth int) bool
		visit = func(keybyte byte, root *bnode, d int) bool {

			if root.isLeaf {
				//case *Leaf:
				return yield(root, true)
			} else {
				//case *Inner:
				inode := root.inner.Node // interface
				switch n := inode.(type) {
				case *node4:
					for i := range n.children {
						if i < n.lth {
							if !visit(n.keys[i], n.children[i], d+1) {
								return false
							}
						}
					}
				case *node16:
					for i := range n.children {
						if i < n.lth {
							if !visit(n.keys[i], n.children[i], d+1) {
								return false
							}
						}
					}
				case *node48:
					for i, k := range n.keys {
						if k == 0 {
							continue
						}
						child := n.children[k-1]
						if !visit(byte(i), child, d+1) {
							return false
						}
					}
				case *node256:
					for i, child := range n.children {
						if child != nil {
							if !visit(byte(i), child, d+1) {
								return false
							}
						}
					}
				}
				// self after children
				return yield(root, true)
			}
			return true
		}
		// Start the recursion

		// the root keybyte is zero always.
		// This a pretense as there is no
		// keybyte that leads to the root really.
		var k byte

		if root.isLeaf {
			yield(root, true)
			return
		}
		visit(k, root, 0)
	}
}
