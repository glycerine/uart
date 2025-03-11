package art

import (
	"bytes"
	"iter"
)

// Iter starts a traversal over the range [start, end)
// in ascending order.
//
// iter.Next() must be called to start the iteration before
// iter.Key(), iter.Value(), or iter.Leaf() will be meaningful.
//
// We begin with the first key that is >= start and < end.
//
// The end key must be > the start key, or no values
// will be returned. Either start or end
// can be nil to indicate the furthest possible range
// in that direction.
//
// For example, note that [x, x) will return the
// empty set, unless x is nil.
//
// For another example, suppose the keys {0, 1, 2} are
// in the tree, and tree.Iter(0, 2) is called.
// Forward iteration will return 0, then 1.
//
// The returned iterator is not concurrent/multiple goroutine safe.
// Iteration does no synchronization. This
// allows for single goroutine code that deletes from
// (or inserts into) the tree during the iteration,
// which is not an uncommon need.
func (t *Tree) Iter(start, end []byte) (iter *iterator) {

	if t.root == nil || t.size < 1 {
		return &iterator{
			initDone: true,
			closed:   true,
		}
	}

	// get the integer range [begIdx, endIdx]
	_, begIdx, ok := t.FindGTE(start)
	if !ok {
		// no such key to start from, iteration already over.
		return &iterator{
			initDone: true,
			closed:   true,
		}
	}

	_, endIdx, ok := t.FindLT(end)
	if !ok {
		return &iterator{
			initDone: true,
			closed:   true,
		}
	}

	return &iterator{
		tree:        t,
		treeVersion: t.treeVersion,
		cursor:      start,
		terminate:   end,
		begIdx:      begIdx - 1,
		curIdx:      begIdx - 1,
		endxIdx:     endIdx + 1,
	}
}

// RevIter starts a traversal over
// the range (end, start] in descending order.
//
// iter.Next() must be called to start the iteration before
// iter.Key(), iter.Value(), or iter.Leaf() will be meaningful.
//
// We begin with the first key that is <= start and > end.
//
// The end key must be < the start key, or no values
// will be returned. Either start or end
// can be nil to indicate the furthest possible range
// in that direction.
//
// For example, note that (x, x] will return the
// empty set, unless x is nil.
//
// For another example, suppose the keys {0, 1, 2} are
// in the tree, and tree.RevIter(0, 2) is called.
// Reverse iteration will return 2, then 1.
// The same holds true if start (2 here) is replaced by
// by any integer > 2.
//
// tree.RevIter(nil, 2) will yield 2, then 1, then 0;
// as will tree.RevIter(nil, nil).
//
// The returned iterator is not concurrent/multiple goroutine safe.
// Iteration does no synchronization. This
// allows for single goroutine code that deletes from
// (or inserts into) the tree during the iteration,
// which is not an uncommon need.
func (t *Tree) RevIter(end, start []byte) (iter *iterator) {

	if t.root == nil || t.size < 1 {
		return &iterator{
			initDone: true,
			closed:   true,
		}
	}

	// get the integer range [endIdx, begIdx]
	_, begIdx, ok := t.FindLTE(start)
	if !ok {
		vv("FindLTE start found nothing!")
		return &iterator{
			initDone: true,
			closed:   true,
		}
	}

	gtLeaf, endIdx, ok := t.FindGT(end)
	_ = gtLeaf
	if !ok {
		//vv("in revIt: FindGT(end='%v') got ok=false, endIdx = %v; gtLeaf='%v'", string(end), endIdx, gtLeaf)
		// this is okay if end is nil, of course

		//vv("FindGT end found nothing! end='%v'", string(end))
		return &iterator{
			initDone: true,
			closed:   true,
		}
	}

	//vv("revIt starting cursor=start='%v'", string(start))
	return &iterator{
		tree:        t,
		treeVersion: t.treeVersion,
		reverse:     true,
		cursor:      start,
		terminate:   end,
		begIdx:      begIdx + 1,
		curIdx:      begIdx + 1,
		endxIdx:     endIdx - 1,
	}
}

type checkpoint struct {
	node   *Inner
	curkey *byte

	prev *checkpoint
}

// iterator will scan the tree in lexicographic order.
type iterator struct {
	tree *Tree

	treeVersion int64

	stack *checkpoint

	initDone bool
	closed   bool

	cursor, terminate []byte
	reverse           bool

	begIdx  int // corresponding to initial cursor key - 1
	curIdx  int // corresponding to current key
	endxIdx int // corresponding to 1 past the last key

	//started bool

	// current:
	key   []byte
	value any
	leaf  *Leaf
}

// Next will iterate over all leaf nodes in
// the specified range in the chosen direction.
func (i *iterator) Next() (ok bool) {
	if i.closed {
		return false
	}
	if i.stack == nil {
		// initialize iterator
		if exit, next := i.init(); exit {
			return next
		}
	}
	return i.iterate()
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
	i.stack = &checkpoint{
		node: root.inner,
	}
	return false, false
}

func (i *iterator) iterate() bool {
	for i.stack != nil {
		more, restart := i.tryAdvance()
		if more {
			return more
		} else if restart {
			i.stack = i.stack.prev
			if i.stack == nil {
				// checkpoint is root
				i.stack = nil
				if exit, next := i.init(); exit {
					return next
				}
			}
		}
	}
	i.closed = true
	return false
}

func (i *iterator) tryAdvance() (bool, bool) {
	//vv("top of tryAdvance")
	//defer vv("end of tryAdvance")

	for adv := 0; ; adv++ {
		_ = adv

		tail := i.stack

		//vv("tryAdv calling i.next() with tail.curkey = '%#v'", tail.curkey) // nil on first call
		curkey, child := i.next(tail.node, tail.curkey)
		if child == nil {

			// Inner node is exhausted, move one level up the stack
			i.stack = tail.prev
			return false, false
		}
		// advance curkey
		//vv("setting tail.curkey = '%v'", string(curkey))
		tail.curkey = &curkey

		if child.isLeaf {
			l := child.leaf
			if i.inRange(l.Key) {
				i.key = l.Key
				i.value = l.Value
				i.cursor = l.Key
				i.leaf = l
				return true, false
			}
			return false, false
		}
		i.stack = &checkpoint{
			node: child.inner,
			prev: tail,
		}
		return false, false
	}
}

func (i *iterator) next(n *Inner, curkey *byte) (keyb byte, b *bnode) {
	//defer func() {
	//	vv("it.next returning keyb='%v', b='%v'", string(keyb), b.String())
	//}()
	if !i.reverse {
		return n.Node.next(curkey)
	}
	return n.Node.prev(curkey)
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

func (i *iterator) inRange(key []byte) (inside bool) {
	//defer func() {
	//vv("inRange returns inside=%v; reverse is %v; key='%v'; cursor='%v; terminate='%v'", inside, i.reverse, string(key), string(i.cursor), string(i.terminate))
	//}()
	if i.reverse {
		return (bytes.Compare(key, i.cursor) <= 0 || len(i.cursor) == 0) && (len(i.terminate) == 0 || bytes.Compare(key, i.terminate) > 0)
	}
	// forward iteration:
	return bytes.Compare(key, i.cursor) >= 0 && (len(i.terminate) == 0 || bytes.Compare(key, i.terminate) < 0)
}

func Ascend(t *Tree, beg, endx Key) iter.Seq2[Key, any] {
	return func(yield func(key Key, value any) bool) {
		//if t.Size() == 0 {
		//	return
		//}
		it := t.Iter(beg, endx)
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
