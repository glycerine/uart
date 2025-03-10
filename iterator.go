package art

import (
	"bytes"
	"iter"
)

// An iterator that wants to "delete behind"
// itself will also need a write lock (TODO
// implement a "writing iterator", or
// set SkipLocking below and manager
// synchronization at a higher level.

type checkpoint struct {
	node   *Inner
	curkey *byte

	prev *checkpoint
}

// iterator will scan the tree in lexicographic order.
type iterator struct {
	tree *Tree

	stack  *checkpoint
	closed bool

	cursor, terminate []byte
	reverse           bool

	allowDel bool
	started  bool

	// current:
	key   []byte
	value any
	leaf  *Leaf
}

// AllowDeletes during iteration uses a
// different traversal strategy that allows
// you to do deletes on the tree while
// you are traversing it. AllowDeletes
// must be called before calling Next on
// the iterator for the first time; it
// cannot be called after starting an
// iteration.
//
// AllowDeletes changes the
// state of the iterator (its receiver)
// and returns that same receiver as a convenience.
func (i *iterator) AllowDeletes() *iterator {
	i.allowDel = true
	return i
}

func (i *iterator) Reverse() *iterator {
	i.cursor, i.terminate = i.terminate, i.cursor
	i.reverse = true
	return i
}

// Next will iterate over all leaf nodes between specified prefixes
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

func (i *iterator) TheLeaf() *Leaf {
	return i.leaf
}

func (i *iterator) Value() any {
	return i.value
}

func (i *iterator) Key() Key {
	return i.key
}

func (i *iterator) inRange(key []byte) bool {
	if !i.reverse {
		return bytes.Compare(key, i.cursor) > 0 && (len(i.terminate) == 0 || bytes.Compare(key, i.terminate) <= 0)
	}
	return (bytes.Compare(key, i.cursor) < 0 || len(i.cursor) == 0) && (len(i.terminate) == 0 || bytes.Compare(key, i.terminate) >= 0)
}

// exit returned true means only 0 or 1 nodes in tree,
// so Next() won't call iterate.
func (i *iterator) init() (exit bool, nextOK bool) {

	root := i.tree.root
	if root == nil {
		i.closed = true
		return true, false
	}
	//l, isLeaf := root.(*Leaf)
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
		node: root.inner, // *Inner
	}
	return false, false
}

func (i *iterator) next(n *Inner, curkey *byte) (byte, *bnode) {
	if !i.reverse {
		return n.Node.next(curkey)
	}
	return n.Node.prev(curkey)
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

		//vv("adv = %v; about to tail.node.lock.RLock(); state is '%v'", adv, tail.node.lock.state) // adv is always 0
		//vv("past RLock; state = '%v'", tail.node.lock.state)

		curkey, child := i.next(tail.node, tail.curkey)
		if child == nil {

			// Inner node is exhausted, move one level up the stack
			i.stack = tail.prev
			return false, false
		}
		// advance curkey
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
			node: child.inner, // (*Inner),
			prev: tail,
		}
		return false, false
	}
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
