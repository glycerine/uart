package art

import (
	"fmt"
	"sync"
)

// Tree is a trie that implements
// the Adaptive Radix Tree (ART) algorithm
// to provide a sorted, key-value, in-memory
// dictionary[1]. The ART tree provides both path compression
// (vertical compression) and variable
// sized inner nodes (horizontal compression)
// for space-efficient fanout.
//
// Path compression is particularly attractive in
// situations where many keys have redudant
// prefixes. This is the common case for
// many ordered-key-value-map use cases, such
// as database indexes and file-system hierarchies.
// The Google File System paper, for example,
// emphasizes the efficiencies obtained
// by exploiting prefix compression in their
// distributed file system[2]. FoundationDB's
// new Redwood backend provides it as a feature[3],
// and users wish the API could be improved by
// offering it[4] in query result APIs.
//
// As an alternative to red-black trees,
// AVL trees, and other kinds of balanced binary trees,
// ART is particularly attractive. Like
// those trees, ART offers an ordered index
// of sorted keys allowing efficient O(log N) access
// for each unique key.
//
// Efficient key-range lookup and iteration, as well as the
// ability to treat the tree as array using
// integer indexes (based on the counted B-tree
// idea[5]), make this ART tree implementation
// particularly easy to use in practice.
//
// ART supports just a single value for each
// key -- it is not a "multi-map" in the C++ sense.
//
// Concurrency: this ART implementation is
// goroutine safe, as it uses a sync.RWMutex
// synchronization. Thus it allows only a
// single writer at a time, and any number
// of readers. Readers will block until
// the writer is done, and thus they see
// a fully consistent view of the tree.
// The RWMutex approach was the fastest
// and easiest to reason about in our
// applications without overly complicating
// the code base. The SkipLocking flag can
// be set to omit all locking if goroutine
// coordination is provided by other means,
// or unneeded (in the case of single goroutine
// only access).
//
// [1] "The Adaptive Radix Tree: ARTful
// Indexing for Main-Memory Databases"
// by Viktor Leis, Alfons Kemper, Thomas Neumann.
//
// [2] "The Google File System"
// SOSP’03, October 19–22, 2003, Bolton Landing, New York, USA.
// by Sanjay Ghemawat, Howard Gobioff, and Shun-Tak Leung.
// https://pdos.csail.mit.edu/6.824/papers/gfs.pdf
//
// [3] "How does FoundationDB store keys with duplicate prefixes?"
// https://forums.foundationdb.org/t/how-does-foundationdb-store-keys-with-duplicate-prefixes/1234
//
// [4] "Issue #2189: Prefix compress read range results"
// https://github.com/apple/foundationdb/issues/2189
//
// [5] "Counted B-Trees"
// https://www.chiark.greenend.org.uk/~sgtatham/algorithms/cbtree.html
type Tree struct {
	Rwmut sync.RWMutex `msg:"-"`

	root *bnode
	size int64

	// The treeVersion Update protocol:
	// Writers increment this treeVersion number
	// to allow iterators to continue
	// efficiently past tree modifications
	// (deletions and/or insertinos) that happen
	// behind them. If the iterator sees a
	// different treeVersion, it will use a
	// slightly more expensive way of getting
	// the next leaf, one that is resilient in
	// the face of tree structure changes.
	treeVersion int64

	// Leafz is for serialization. You must
	// set leafByLeaf=false if you want to
	// automatically serialize a Tree when it is
	// a field in other structs. In that case,
	// the pre-save and post-load hooks will
	// use Leafz as a serialization buffer.
	// Otherwise Leafz is unused.
	//
	// Using Leafz may require more memory, since
	// the tree is fully serialized (temporarily)
	// into Leafz before writing anything to disk.
	// When leafByLeaf is true, the tree is
	// streamed to disk incrementally. See
	// saver.go and the TreeSaver and TreeLoader
	// for standalone save/load facilities.
	//
	// Only leaf nodes are serialized to disk.
	// This saves 20x space.
	Leafz []*Leaf `zid:"0"`

	// SkipLocking means do no internal
	// synchronization, because a higher
	// component is doing so.
	//
	// Warning when using SkipLocking:
	// the user's code _must_ synchronize (prevent
	// overlap) of readers and writers who access the Tree.
	// Under this setting, the Tree will not do locking.
	// (it does by default, with SkipLocking false).
	// Without synchronization, there will be data races,
	// lost data, and panic segfaults from torn reads.
	//
	// The easiest way to do this is with a sync.RWMutex.
	// One such, the Rwmut on this Tree, will be
	// employed for you if SkipLocking is allowed to
	// default to false.
	SkipLocking bool `msg:"-"`
}

// NewArtTree creates and returns a new ART Tree,
// ready for use.
func NewArtTree() *Tree {
	return &Tree{}
}

// DeepSize enumerates all leaf nodes
// in order to compute the size. This is really only
// for testing. Prefer the cache based Size(),
// below, whenever possible.
func (t *Tree) DeepSize() (sz int) {

	for lf := range Ascend(t, nil, nil) {
		_ = lf
		sz++
	}
	return
}

// Size returns the number of keys
// (leaf nodes) stored in the tree.
func (t *Tree) Size() (sz int) {
	if t.SkipLocking {
		return int(t.size)
	}
	t.Rwmut.RLock()
	sz = int(t.size)
	t.Rwmut.RUnlock()
	return
}

func (t *Tree) String() string {
	sz := t.Size()
	if t.root == nil {
		return "empty tree"
	}
	return fmt.Sprintf("tree of size %v: ", sz) +
		t.root.FlatString(0, -1)
}

func (t *Tree) FlatString() string {
	sz := t.Size()
	if t.root == nil {
		return "empty tree"
	}

	return fmt.Sprintf("tree of size %v: \n", sz) +
		t.root.FlatString(0, -1)
}

// InsertX now copies the key to avoid bugs.
// The value is held by pointer in the interface.
// The x slice is not copied either.
func (t *Tree) InsertX(key Key, value any, x []byte) (updated bool) {

	key2 := Key(append([]byte{}, key...))
	lf := NewLeaf(key2, value, x)
	return t.InsertLeaf(lf)
}

// Insert makes a copy of key to avoid sharing bugs.
// The value is held by pointer in the interface.
func (t *Tree) Insert(key Key, value any) (updated bool) {

	// make a copy of key that we own, so
	// caller can alter/reuse without messing us up.
	// This was a frequent source of bugs, so
	// it is important. The benchmarks will crash
	// without it, for instance, since they
	// re-use key []byte memory alot.
	key2 := Key(append([]byte{}, key...))
	lf := NewLeaf(key2, value, nil)

	return t.InsertLeaf(lf)
}

// The *Leaf lf *must* own the lf.Key it holds.
// It cannot be shared. You must guarantee this,
// copying the slice if necessary.
func (t *Tree) InsertLeaf(lf *Leaf) (updated bool) {
	if t == nil {
		panic("t *Tree cannot be nil in InsertLeaf")
	}
	if !t.SkipLocking {
		t.Rwmut.Lock()
		defer t.Rwmut.Unlock()
	}

	var replacement *bnode

	if t.root == nil {
		// first node in tree
		t.size++
		t.root = bnodeLeaf(lf)
		t.treeVersion++
		return false
	}

	//vv("t.size = %v", t.size)
	replacement, updated = t.root.insert(lf, 0, t.root, t, nil)
	if replacement != nil {
		t.root = replacement
	}
	if !updated {
		t.size++
	}
	t.treeVersion++
	return
}

// FindGT returns the first element whose key
// is greater than the supplied key.
func (t *Tree) FindGT(key Key) (val any, idx int, found bool) {
	var lf *Leaf
	lf, idx, found = t.Find(GT, key)
	if found && lf != nil {
		val = lf.Value
	}
	return
}

// FindGTE returns the first element whose key
// is greater than, or equal to, the supplied key.
func (t *Tree) FindGTE(key Key) (val any, idx int, found bool) {
	var lf *Leaf
	lf, idx, found = t.Find(GTE, key)
	if found && lf != nil {
		val = lf.Value
	}
	return
}

// FindGT returns the first element whose key
// is less than the supplied key.
func (t *Tree) FindLT(key Key) (val any, idx int, found bool) {
	var lf *Leaf
	lf, idx, found = t.Find(LT, key)
	if found && lf != nil {
		val = lf.Value
	}
	return
}

// FindLTE returns the first element whose key
// is less-than-or-equal to the supplied key.
func (t *Tree) FindLTE(key Key) (val any, idx int, found bool) {
	var lf *Leaf
	lf, idx, found = t.Find(LTE, key)
	if found && lf != nil {
		val = lf.Value
	}
	return
}

// FindExact returns the element whose key
// matches the supplied key.
func (t *Tree) FindExact(key Key) (val any, idx int, found bool) {
	var lf *Leaf
	lf, idx, found = t.Find(Exact, key)
	if found && lf != nil {
		val = lf.Value
	}
	return
}

// FirstLeaf returns the first leaf in the Tree.
func (t *Tree) FirstLeaf() (lf *Leaf, idx int, found bool) {
	return t.Find(GTE, nil)
}

// FirstLeaf returns the last leaf in the Tree.
func (t *Tree) LastLeaf() (lf *Leaf, idx int, found bool) {
	return t.Find(LTE, nil)
}

// Find allows GTE, GT, LTE, LT, and Exact searches.
//
// GTE: find a leaf greater-than-or-equal to key;
// the smallest such key.
//
// GT: find a leaf strictly greater-than key;
// the smallest such key.
//
// LTE: find a leaf less-than-or-equal to key;
// the largest such key.
//
// LT: find a leaf less-than key; the
// largest such key.
//
// Exact: find leaf whose key matches the supplied
// key exactly. This is the default. It acts
// like a hash table. A key can only be stored
// once in the tree. (It is not a multi-map
// in the C++ STL sense).
//
// If key is nil, then GTE and GT return
// the first leaf in the tree, while LTE
// and LT return the last leaf in the tree.
func (t *Tree) Find(smod SearchModifier, key Key) (lf *Leaf, idx int, found bool) {
	if !t.SkipLocking {
		t.Rwmut.RLock()
		defer t.Rwmut.RUnlock()
	}
	if t.root == nil {
		return
	}
	if len(key) == 0 && t.size == 1 {
		// nil query asks for first leaf, or last, depending.
		// here it is the same.
		return t.root.leaf, 0, true
	}
	var b *bnode
	switch smod {
	case GTE, GT:
		b, found, _, idx = t.root.getGTE(key, 0, smod, t.root, t, 0, false, 0)
	case LTE, LT:
		b, found, _, idx = t.root.getLTE(key, 0, smod, t.root, t, 0, false, 0)
	default:
		b, found, _, idx = t.root.get(key, 0, t.root)
	}
	if b != nil {
		lf = b.leaf
	}
	return
}

type SearchModifier int

const (
	// Exact is the default.
	Exact SearchModifier = 0 // exact matches only; like a hash table
	GTE   SearchModifier = 1 // greater than or equal to this key.
	LTE   SearchModifier = 2 // less than or equal to this key.
	GT    SearchModifier = 3 // strictly greater than this key.
	LT    SearchModifier = 4 // strictly less than this key.
)

func (smod SearchModifier) String() string {
	switch smod {
	case Exact:
		return "Exact"
	case GTE:
		return "GTE"
	case LTE:
		return "LTE"
	case GT:
		return "GT"
	case LT:
		return "LT"
	}
	panic(fmt.Sprintf("unknown smod '%v'", int(smod)))
}

// Remove deletes the key from the Tree.
func (t *Tree) Remove(key Key) (deleted bool, value any) {

	if !t.SkipLocking {
		t.Rwmut.Lock()
		defer t.Rwmut.Unlock()
	}

	var deletedNode *bnode
	if t.root == nil {
		return
	}

	deleted, deletedNode = t.root.del(key, 0, t.root, func(rn *bnode) {
		t.root = rn
	})
	if deleted {
		value = deletedNode.leaf.Value
		t.size--
		t.treeVersion++
	}
	return deleted, value
}

// IsEmpty returns true iff the Tree is empty.
func (t *Tree) IsEmpty() (empty bool) {
	if t.SkipLocking {
		return t.root == nil
	}
	t.Rwmut.RLock()
	empty = t.root == nil
	t.Rwmut.RUnlock()
	return
}

// Iterator starts a traversal over the range [start, end).
// Use a nil start to begin with the first key.
// Use a nil end to proceed through the last key.
//
// For example, suppose the keys {0, 1, 2} are
// in the tree, and tree.Iterator(0, 2) is called.
// Forward iteration will return 0, then 1.
//
// The returned iterator is not concurrent/multiple goroutine safe.
// Iteration does no synchronization. If concurrent
// writes are possible, the user must
// ensure the equivalent of a read-lock is in place
// during iteration, using the t.Rwmut if necessary.
// Calling `tree.Rwmut.Rlock()` followed by
// `defer tree.Rwmut.RUnlock()` is typical.
func (t *Tree) Iterator(start, end []byte) *iterator {

	if t.root == nil || t.size < 1 {
		return &iterator{
			initialized: true,
			closed:      true,
		}
	}

	// get the integer range [begIdx, endIdx]
	_, begIdx, ok := t.FindGTE(start)
	if !ok {
		panic("what? internal logic error, t.size was >= 1")
	}

	_, endIdx, ok := t.FindLT(end)
	if !ok {
		panic("what? internal logic error, t.size was >= 1")
	}

	return &iterator{
		tree:    t,
		start:   start,
		end:     end,
		begIdx:  begIdx,
		endxIdx: endIdx + 1,
	}
}

// ReverseIterator starts a traversal over
// the range (end, start] and returns keys in descending order
// beginning with the first key that is <= start.
// The start key must be >= the end key. Either
// can be nil to indicate the furthest possible range
// in that direction.
//
// For example, suppose the keys {0, 1, 2} are
// in the tree, and tree.ReverseIterator(0, 2) is called.
// Reverse iteration will return 2, then 1.
//
// tree.ReverseIterator(nil, 2) will yield 2, then 1, then 0;
// as will tree.ReverseIterator(nil, nil).
//
// The returned iterator is not concurrent/multiple goroutine safe.
// Iteration does no synchronization. If concurrent
// writes are possible, the user must
// ensure the equivalent of a read-lock is in place
// during iteration, using the Tree.Rwmut if necessary.
// Calling `tree.Rwmut.Rlock()` followed by
// `defer tree.Rwmut.RUnlock()` is typical.
func (t *Tree) ReverseIterator(end, start []byte) *iterator {
	if t.root == nil || t.size < 1 {
		return &iterator{
			initialized: true,
			closed:      true,
		}
	}

	// get the integer range [endIdx, begIdx]
	_, begIdx, ok := t.FindLTE(start)
	if !ok {
		panic("what? internal logic error, t.size was >= 1")
	}

	_, endIdx, ok := t.FindGT(end)
	if !ok {
		panic("what? internal logic error, t.size was >= 1")
	}

	return &iterator{
		tree:    t,
		start:   start,
		end:     end,
		begIdx:  begIdx,
		endxIdx: endIdx - 1,
	}
}

// At(i) lets us think of the tree as a
// array, returning the i-th leaf
// from the sorted leaf nodes, using
// an efficient O(log N) time algorithm.
// Here N is the size or count of elements
// stored in the tree.
//
// At() uses the counted B-tree approach
// described by Simon Tatham of PuTTY fame[1].
// [1] Reference:
// https://www.chiark.greenend.org.uk/~sgtatham/algorithms/cbtree.html
func (t *Tree) At(i int) (lf *Leaf, ok bool) {
	if t.SkipLocking {
		lf, ok = t.root.at(i)
		return
	}
	t.Rwmut.RLock()
	lf, ok = t.root.at(i)
	t.Rwmut.RUnlock()
	return
}

// Atv(i) is like At(i) but returns the value
// from the Leaf instead of the actual *Leaf itself,
// simply for convenience.
func (t *Tree) Atv(i int) (val any, ok bool) {
	var lf *Leaf
	if t.SkipLocking {
		lf, ok = t.root.at(i)
		if ok {
			val = lf.Value
			return
		}
		return
	}
	t.Rwmut.RLock()
	lf, ok = t.root.at(i)
	if ok {
		val = lf.Value
	}
	t.Rwmut.RUnlock()
	return
}

func (t *Tree) LeafIndex(leaf *Leaf) (idx int, ok bool) {
	t.Rwmut.RLock()
	_, idx, ok = t.FindExact(leaf.Key)
	t.Rwmut.RUnlock()
	return
}
