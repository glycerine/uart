package art

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	//"sync/atomic"

	cryrand "crypto/rand"
	mathrand2 "math/rand/v2"
)

func comparePrefix(k1, k2 []byte, depth int) int {
	idx, limit := depth, min(len(k1), len(k2))
	for ; idx < limit; idx++ {
		if k1[idx] != k2[idx] {
			break
		}
	}

	return idx - depth
}

func (n *Inner) Kind() Kind {
	return n.Node.Kind()
}

// max returned is len(n.Compressed)
func (n *Inner) compressedMismatch(key Key, depth int) (idx int) {

	maxCmp := min(len(n.compressed), len(key)-depth)
	for idx = 0; idx < maxCmp; idx++ {
		if n.compressed[idx] != key[depth+idx] {
			return idx // mismatch
		}
	}
	return maxCmp
}

// parentAnodeN should be the parent's child pointer
// (an *anode) that holds n inside it: such that
// INVAR holds: parentAnodeN.load().inner == n
//
// If restart == true on then retry the insert.
func (n *Inner) insert(lf *Leaf, depth int, selfb *bnode, tree *Tree, parent *Inner) (replacement *bnode, updated bool) {

	// biggest mis is len(n.Compressed) for
	// full matching with lf.Key
	mis := n.compressedMismatch(lf.Key, depth)

	if mis < len(n.compressed) {

		// lazy expand
		// we will overwrite ourself (n) with a new n4 split.
		// This newChild node will be a child of (us) n.Node
		newChildKey := n.compressed[mis]
		parentCompressed := append([]byte{}, n.compressed[:mis]...)

		newChild := &Inner{
			Node:       n.Node,
			compressed: n.compressed[mis+1:],
			// keep path stuff for debugging!
			//path:       append([]byte{}, lf.Key[:depth+mis]...),
			SubN: n.SubN,
		}
		//vv("assigned path '%v' to %p", string(newChild.path), newChild)
		newChild.Keybyte = newChildKey

		// n becomes the new parent of newChild and lf
		n4 := &node4{}
		leafKeybyte := lf.Key.At(depth + mis)
		lf.Keybyte = leafKeybyte
		n4.addChild(leafKeybyte, bnodeLeaf(lf))
		n4.addChild(newChildKey, bnodeInner(newChild))

		n.Node = n4

		// ======================================
		// keep this commented path stuff for debugging!
		// ======================================
		// compressDelta := len(n.compressed) - len(parentCompressed)
		// keep := len(n.path) - compressDelta
		// if keep < 0 {
		// 	keep = 0
		// }
		// n.path = n.path[:keep]
		// ======================================
		// end path stuff to be kept.
		// ======================================

		n.compressed = parentCompressed
		n.SubN++
		//n.Keybyte stays the same I think. likewise n.path.

		selfb.inner = n
		return selfb, false
	}
	// INVAR: mis == len(n.Compressed)
	// INVAR: prefixMismatchedIdx >= n.PrefixLen,
	// so we are extending previous leaf paths.

	nextDepth := depth + mis
	nextkey := lf.Key.At(nextDepth)
	idx, next := n.Node.child(nextkey)

	if next == nil {

		if n.Node.full() {
			n.Node = n.Node.grow()
		}
		addkey := lf.Key.At(nextDepth)
		lf.Keybyte = addkey
		n.Node.addChild(addkey, bnodeLeaf(lf))
		n.SubN++

		return selfb, false
	}

	if next.isLeaf {

		replacement, updated = next.insert(lf, nextDepth+1, next, tree, n)
		n.Node.replace(idx, replacement, false)
		n.SubN++
		if !replacement.isLeaf {
			replacement.inner.Keybyte = nextkey

			// keep commented out path stuff for debugging!
			//replacement.inner.path = next.inner.path
		}

		return selfb, updated
	}
	// INVAR: next is not a leaf.

	_, updated = next.insert(lf, nextDepth+1, next, tree, n)
	n.SubN++
	return selfb, updated
}

func (n *Inner) del(key Key, depth int, selfb *bnode, parentUpdate func(*bnode)) (deleted bool, deletedNode *bnode) {

	if _, fullmatch, _ := n.checkCompressed(key, depth); !fullmatch {
		// key is not found, check for concurrent writes and exit
		return false, nil
	}

	nextDepth := depth + len(n.compressed)
	delkey := key.At(nextDepth)
	idx, next := n.Node.child(delkey)

	if next == nil {
		// key is not found
		return false, nil
	}
	n.SubN--

	if next.isLeaf && next.leaf.cmp(key) {

		// deleting a leaf in next
		_, isNode4 := n.Node.(*node4)
		atmin := n.Node.min()
		if isNode4 && atmin {
			// update parent pointer. current node will
			// be collapsed from n4 -> leaf.

			deletedNode = n.Node.replace(idx, nil, true)

			// get the left node
			leftKey, left := n.Node.next(nil)

			// during delete of n, have to give leftB n's prefix
			if left.isLeaf {
				left.leaf.addPrefixBefore(n, leftKey)
			} else {
				left.inner.addPrefixBefore(n, leftKey)
			}
			// left.addPrefixBefore(n, leftB)

			// left is replacing n, because n shrank.
			parentUpdate(left)

			// NB: replace() is used to delete as well as update,
			// and happens via the above parentUpdate callback.
			// In particular, the keys are updated alongside
			// children pointers.

			// deleted, deletedNode
			return true, deletedNode
		}
		// deleting a leaf in next.
		// n is > node4

		// local change. parent not affected.

		deletedNode = n.Node.replace(idx, nil, true)
		if atmin && !isNode4 {
			n.Node = n.Node.shrink()
		}
		return true, deletedNode

	} else if next.isLeaf {
		// key is not found.
		return false, deletedNode
	}
	// INVAR: next is not a leaf

	deleted, deletedNode = next.del(key, nextDepth+1, next, func(bn *bnode) {
		n.Node.replace(idx, bn, true)
	})
	n.Node.redoPren() // essential! for LeafIndex/id to be correct.
	return deleted, deletedNode
}

// checkCompressed returns the number
// of prefix characters shared between
// the key and node. fullmatch is
// returned true iff the key matches completely.
// greaterThan returns true iff the mismatched
// key byte is > the compressed path byte.
// On fullmatch true, greaterThan should be
// ignored, as it is not meaningful.
func (n *Inner) checkCompressed(key Key, depth int) (idx int, fullmatch bool, greaterThan bool) {

	maxCmp := min(len(n.compressed), len(key)-depth)
	for idx = 0; idx < maxCmp; idx++ {
		ci := n.compressed[idx]
		kdi := key[depth+idx]
		if ci != kdi {
			return idx, false, kdi > ci
		}
	}
	return idx, true, false
}

// direc is returned from get() to tell
// recursive get() calls if/where to retry on backtrack.
// Similar to the output of bytes.Compare: 0, 1, or -1
//
// Update: added two more states:
// 2 means go forward (largest), but smallest-will-do.
// -2 will mean previous (smaller), but largest-will-do,
// (once we implement LTE).
type direc int

const needNextLeaf direc = 1
const needPrevLeaf direc = -1
const nextButSmallestWillDo = 2
const prevButLargestWillDo = -2

func (n *Inner) get(key Key, depth int, selfb *bnode) (value *bnode, found bool, dir direc, id int) {

	//pp("top of get(), we are '%v'", n.FlatString(depth, 0))

	//_, fullmatch, gt := n.checkCompressed(key, depth)

	// Let's inline checkCompressed, as it profiles hot.
	maxCmp := len(n.compressed)
	kd := len(key) - depth
	if kd < maxCmp {
		maxCmp = kd
	}
	for idx := 0; idx < maxCmp; idx++ {
		ci := n.compressed[idx]
		kdi := key[depth+idx]
		if ci != kdi {
			return
		}
	}

	// have full match of compressed prefix, or a nil key.
	//pp("full match of compressed = '%v' from key '%v'", string(n.compressed), string(key))

	nextDepth := depth + len(n.compressed)

	var querykey byte
	if nextDepth < len(key) {
		querykey = key[nextDepth]
	}

	_, next := n.Node.child(querykey)
	if next == nil {
		return nil, false, 0, 0
	}

	//pp("about to call next.get on next = '%v' with inquiry '%v'", next.FlatString(nextDepth+1, 0), string(key[:nextDepth]))

	value, found, dir, id = next.get(key, nextDepth+1, next)
	id += next.pren
	return
}

func memcpy[T any](dst []T, src []T, len int) {
	copy(dst[:], src[:len])
}

// durring delete of node, n needs to have nodes' prefix pre-pended.
func (n *Inner) addPrefixBefore(node *Inner, key byte) {

	// new prefix: { node prefix } { key } { n(this) prefix }
	nCompressed := n.compressed
	nodeCompressed := node.compressed

	newpre := make([]byte, len(nodeCompressed)+1+len(nCompressed))

	i := copy(newpre, nodeCompressed)
	newpre[i] = key
	copy(newpre[i+1:], nCompressed)

	n.compressed = newpre
}

func (n *Inner) String() string {
	return n.FlatString(0, 0) // -1 to recurse.
}

func (n *Inner) isLeaf() bool {
	return false
}

// debug facility
// not cryptographically random.
func randomID(n int) string {
	pseudo := make([]byte, n)
	chacha8randMut.Lock()
	chacha8rand.Read(pseudo)
	chacha8randMut.Unlock()
	return fmt.Sprintf("%x", pseudo)
}

var chacha8randMut sync.Mutex
var chacha8rand *mathrand2.ChaCha8 = newCryrandSeededChaCha8()

func newCryrandSeededChaCha8() *mathrand2.ChaCha8 {
	var seed [32]byte
	_, err := cryrand.Read(seed[:])
	panicOn(err)
	return mathrand2.NewChaCha8(seed)
}

func (n *Inner) FlatString(depth int, recurse int) (s string) {

	keystr := string(n.Keybyte)
	if n.Keybyte == 0 {
		keystr = "(zero)"
	}

	rep := strings.Repeat("    ", depth)

	s += fmt.Sprintf(`%v %p %v, key '%v' childkeys: %v (treedepth %v) compressed='%v' path='%v' (subN: %v)%v`,
		rep,
		n,
		n.Kind().String(),
		keystr,
		n.Node.childkeysString(),
		depth,
		string(n.compressed),
		// keep commented out path stuff for debugging!
		//string(n.path),
		"(paths commented out atm)",
		n.SubN,
		"\n",
	)

	if recurse == 0 {
		return s // just this node.
	}
	key, node := n.Node.next(nil)
	k := 0
	_ = k
	for node != nil {
		s += node.FlatString(depth+1, recurse-1)
		key, node = n.Node.next(&key)
		k++
	}
	return s
}

func viznl(s string) string {
	if s == "\n" {
		return "\\n" // 2 runes, to keep newline keys on the same line.
	}
	return s
}

func viznlString(by []byte) string {
	numnl := bytes.Count(by, []byte{10})
	out := make([]byte, 0, len(by)+numnl)
	for _, c := range by {
		if c == 10 {
			out = append(out, []byte("\\n")...)
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}

func (n *Inner) rangestr() string {
	return fmt.Sprintf(" with range [%v :to: %v]",
		n.rfirst().str(), n.rlast().str())
}
func (b *bnode) rangestr() string {
	if b.isLeaf {
		return ""
	}
	return b.inner.rangestr()
}

func (n *Inner) rfirst() *Leaf {
	_, b := n.first()
	for {
		if b.isLeaf {
			return b.leaf
		}
		_, b = b.first()
	}
}
func (n *Inner) rlast() *Leaf {
	_, b := n.last()
	for {
		if b.isLeaf {
			return b.leaf
		}
		_, b = b.last()
	}
}
