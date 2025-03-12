package uart

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
)

var _ = sync.RWMutex{}

type TestBytes struct {
	Slc []byte `zid:"0"`
}

// ByteSlice is an alias for []byte. It
// can be ignored in the uart Unserserialized
// ART project, as it is only used for
// serialization purposes elsewhere.
//
// ByteSlice is a simple wrapper header on all msgpack
// messages; has the length and the bytes.
// Allows us length delimited messages;
// with length knowledge up front.
type ByteSlice []byte

// Key is the []byte which the tree
// sorts in lexicographic order. It
// is an arbitrary string of bytes, and
// in particular can contain the 0 byte
// anywhere in the slice.
type Key []byte

type Leaf struct {
	// Keybyte holds parent's name for us;
	// it is a kind of "back-pointer" value
	// that is primary useful for debug logging
	// and diagnostics.
	// Keybyte will be somewhere in our Key,
	// but we don't know where because it
	// depends on the other keys the parent
	// has/path compression.
	Keybyte byte `zid:"1"`

	Key   Key         `zid:"0"`
	Value interface{} `msg:"-"`
}

func (n *Leaf) depth() int {
	return len(n.Key)
}
func (n *Leaf) clone() (c *Leaf) {
	c = &Leaf{
		Key:     append([]byte{}, n.Key...),
		Value:   n.Value, // shared interface (pointer to Value)
		Keybyte: n.Keybyte,
	}
	return c
}

func NewLeaf(key Key, v any, x []byte) *Leaf {
	return &Leaf{
		Key:   key,
		Value: v,
	}
}

func (lf *Leaf) kind() kind {
	return _Leafy
}

func (lf *Leaf) insert(other *Leaf, depth int, selfb *bnode, tree *Tree, par *inner) (value *bnode, updated bool) {

	if lf == other {
		// due to restarts (now elided though),
		// we might be trying to put ourselves in
		// the tree twice.
		return selfb, false
	}

	if other.equal(lf.Key) {
		value = bnodeLeaf(other)
		updated = true
		// avoid forcing a full re-compute of pren.
		value.pren = selfb.pren
		return
	}

	longestPrefix := comparePrefix(lf.Key, other.Key, depth)
	//vv("longestPrefix = %v; lf.Key='%v', other.key='%v', depth=%v", longestPrefix, string(lf.Key), string(other.Key), depth)
	n4 := &node4{}
	nn := &inner{
		Node: n4,

		// keep commented out path stuff for debugging!
		//path: append([]byte{}, lf.Key[:depth+longestPrefix]...),
		SubN: 2,
	}
	//vv("assigned path '%v' to %p", string(nn.path), nn)
	if longestPrefix > 0 {
		nn.compressed = append([]byte{}, lf.Key[depth:depth+longestPrefix]...)
	}
	//vv("leaf insert: lef nn.PrefixLen = %v (longestPrefix)", nn.PrefixLen)

	child0key := lf.Key.At(depth + longestPrefix)
	child1key := other.Key.At(depth + longestPrefix)

	//vv("child0key = 0x%x; lf.Key = '%v' (len %v); depth=%v; longestPrefix=%v; depth+longestPrefix=%v", child0key, string(lf.Key), len(lf.Key), depth, longestPrefix, depth+longestPrefix)

	nn.Node.addChild(child0key, bnodeLeaf(lf))
	nn.Node.addChild(child1key, bnodeLeaf(other))

	selfb.isLeaf = false
	selfb.inner = nn
	return selfb, false
}

func (lf *Leaf) del(key Key, depth int, selfb *bnode, parentUpdate func(*bnode)) (deleted bool, deletedNode *bnode) {

	if !lf.equalUnlocked(key) {
		return false, nil
	}

	parentUpdate(nil)

	return true, selfb
}

func (lf *Leaf) get(key Key, i int, selfb *bnode) (value *bnode, found bool, dir direc, id int) {
	cmp := bytes.Compare(key, lf.Key)
	//pp("top of Leaf get, cmp = %v from lf.Key='%v'; key='%v'", cmp, string(lf.Key), string(key))
	//defer func() {
	//pp("Leaf '%v' returns found=%v, dir=%v", string(lf.Key), found, dir)
	//}()

	// return ourselves even if not exact match, to avoid
	// a second recursive descent on GTE, for example.
	return selfb, cmp == 0, direc(cmp), 0
}

func (lf *Leaf) addPrefixBefore(node *inner, key byte) {
	// Leaf does not store prefixes, only inner.
}

func (lf *Leaf) isLeaf() bool {
	return true
}

func (lf *Leaf) String() string {
	//return fmt.Sprintf("leaf[%q]", string(lf.Key))
	return lf.FlatString(0, 0)
}

// used by get
func (lf *Leaf) equal(other []byte) (equal bool) {
	return bytes.Compare(lf.Key, other) == 0
}

// use by del, already holding Lock
func (lf *Leaf) equalUnlocked(other []byte) (equal bool) {
	equal = bytes.Compare(lf.Key, other) == 0
	return
}

func (n *Leaf) FlatString(depth int, recurse int) (s string) {
	rep := strings.Repeat("    ", depth)
	return fmt.Sprintf(`%[1]v %p leaf: key '%v' (len %v)%v`,
		rep,
		n,
		viznlString(n.Key),
		len(n.Key),
		"\n",
	)
}

func (n *Leaf) stringNoKeys(depth int) (s string) {
	rep := strings.Repeat("    ", depth)
	return fmt.Sprintf(`%[1]v %p leaf:%v`,
		rep,
		n,
		"\n",
	)
}

func (n *Leaf) str() string {
	return string(n.Key)
}

// essential utility.
func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}
