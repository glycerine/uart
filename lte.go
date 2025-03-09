package art

import (
	"bytes"
	"fmt"
)

var _ = fmt.Sprintf

// getLTE returns a leaf with a leaf.Key <= key.
// And, importantly, it returns the leaf with the
// largest such key (the largest leaf.Key that is <= key).
//
// Since we don't want just any leaf < key, but
// the largest available, we have to find where the key
// would be inserted (or its exact matching location,
// if the key is already in the tree).
//
// When we are back-tracking
// up the tree, we need some way to convey the
// idea that "largest will do" to the caller. We added
// another dir state; -2 to mean less (search
// backward) but the next leaf encountered will do. A caller
// can, in turn, convey this to other
// recursive invocations of getLTE() in
// its largestWillDo parameter.
//
// So: set largestWillDo when we are in the phase
// of the search where we know anything in
// this subtree will be LTE our query. This
// is not true at the start on the full tree,
// of course, because that would always
// return the minimum leaf. But once we
// have found the natural insert place for key,
// we can leverage the fact that any leaf
// after that will be LTE, even if we
// have to backtrack up and back-down-to-the-right
// in the tree to find it. Simply put, the +2
// direction means go forward (greater keys) and
// return the very next (smallest) key/leaf on
// the forward path.
//
// The keyCmpPath maintains the comparison of
// key to path as we descend the tree. It
// saves us from having to store the full
// paths in the inner nodes, which would
// consume lots of redundant memory.
//
// In debugging, the paths are super helpful,
// so we'll try to retain some efficient
// means of activating them for diagnostics.
//
// On LTE verus LT search:
//
// LT is identical to LTE until we find a leaf.
// Then LT moves left by one iff the leaf is
// an exact match to the key. The differences
// are compact and found right after the
// first getLTE() recursion.
func (n *Inner) getLTE(
	key Key,
	depth int, // how far along the key we are
	smod SearchModifier,
	selfb *bnode, // selfb.inner == n
	tree *Tree,
	calldepth int, // easier diagnostics
	largestWillDo bool, // want the next largest key.
	keyCmpPath int, // equal to binary.Compare(key, n.path)

) (value *bnode, found bool, dir direc, id int) {

	//pp("%p top of getLTE(), path='%v'; we are '%v' %v;  largestWillDo=%v", n, string(n.path), n.FlatString(depth, 0), n.rangestr(), largestWillDo)

	//defer func() {
	//pp("%p returning from calldepth=%v getLTE value='%v', found='%v'; dir='%v'; id='%v'; my Inner %v;  largestWillDo=%v; pathMatch=%v", n, calldepth, value, found, dir, id, n.rangestr(), largestWillDo, pathMatch)
	//}()

	// PHASE ZERO: largestWillDo during backtracking.
	if largestWillDo {
		dir = 0
		found = true
		value, _ = n.recursiveLast()
		return
	}

	// So largestWillDo is false
	// for a while after here, but it does
	// get set below for some of the
	// recursive calls, and informs
	// the dir we return.

	// PHASE ONE: handle nil keys.
	// They are used to ask for the first (last in LTE)
	// leaf in the tree.
	if len(key) == 0 {
		value, found = n.recursiveLast()
		return
	}

	// During LTE, we could have a mismatch in key
	// at a place in the path higher up than depth,
	// because we are searching for the place that
	// a key which is not (necessarily) in the tree
	// would go. So compare paths using keyCmpPath first.

	switch keyCmpPath {
	case 1:
		dir = needNextLeaf
		value, _ = n.recursiveLast()
		return
	case -1:
		dir = needPrevLeaf
		value, _ = n.recursiveFirst()
		return
	}
	// end of keyCmpPath path comparison.

	// inlined below
	//_, fullmatch, gt := n.checkCompressed(key, depth)

	// Let's inline checkCompressed, as it profiles hot.
	var gt bool
	fullmatch := true
	maxCmp := len(n.compressed)
	kd := len(key) - depth
	if kd < maxCmp {
		maxCmp = kd
	}
	for idx := 0; idx < maxCmp; idx++ {
		ci := n.compressed[idx]
		kdi := key[depth+idx]
		if ci != kdi {
			if kdi > ci {
				gt = true
			}
			fullmatch = false
			break
		}
	}

	// PHASE TWO: handle incomplete match with the compressed prefix.

	if !fullmatch {

		if gt {
			dir = needNextLeaf
			value, _ = n.recursiveLast()
			return
		} else {
			dir = needPrevLeaf
			value, _ = n.recursiveFirst()
			return
		}
	} // end if !fullmatch

	// We have a full match of compressed prefix.

	// The adjacency condition to terminate the
	// LTE search: we can stop searching when
	// we have found a smaller (key) node that
	// says go in a positive direction, and
	// a larger (key) node that says go
	// in a negative direction. This still holds
	// if we substitute subtree for node.
	// The leftmost-leaf of the greater subtree
	// gives the LTE search result leaf.

	// PHASE THREE: we have a full compressed prefix match,
	// so our subtree is relavant and must be searched.
	// Do a bisecting LTE search.

	// This is imporant, because only we (a this
	// stage in the tree descent) can make the
	// determination that two adjacent leaves meet
	// the LTE termination condition. Otherwise the
	// desired leaf could well be outside our purview;
	// either greater than our largest leaf, or less
	// than our smallest leaf. We want to efficiently
	// discover this, without having to go deeper
	// than necessary.

	nextDepth := depth + len(n.compressed)

	var querykey byte
	if nextDepth < len(key) {
		querykey = key[nextDepth]
	}

	// the LTE bisecting call:
	prevKeyb, prev := n.Node.lte(&querykey)

	// nil means querykey < all keys in n.Node
	if prev == nil {
		// dir = 2 will tell our caller to
		// look for the very prev
		// biggest key: the maximal key in the previous/
		// lesser subtree (or their last child).
		dir = -2
		// value never going to be used, so don't bother.
		return nil, false, dir, 0
	}

	// It could still be that key < all of our
	// sub-tree keys, even thought prev != nil,
	// because we've only looked at one additional byte,
	// the querykey. We must recurse with next.getLTE()
	// check the next bytes.

	// PHASE FOUR: recursion. A play in three acts.
	//
	// There are three recursive getLTE() calls below,
	// but only at most two of them will happen.
	//
	// We will always do the prev.getLTE().
	//
	// Afterwards, depending on the results,
	// may also do a prevprev.getLTE() or
	// a next.getLTE() call.

	// This is the first recursive getLTE call.
	value, found, dir, _ = prev.getLTE(
		key,
		nextDepth+1,
		smod,
		prev,
		tree,
		calldepth+1,
		false, // false for largestWillDo
		byteCmp(querykey, prevKeyb, keyCmpPath),
	)

	if found {
		// exact LTE match
		switch smod {
		case LTE:
			return value, true, 0, 0
		case LT:
			cmp := bytes.Compare(value.leaf.Key, key)
			if cmp < 0 {
				// strictly less, done!
				return value, true, 0, 0
			}
			// check do we have other sibs before returning
			_, prevLocal := n.Node.prev(&prevKeyb)
			if prevLocal == nil {
				// gotta pop up and then back down.
				dir = -2 // first greater encounted, take it!
				found = false
				value = nil
				return
			}
			// the local prev is available, use it.
			value, _ = prevLocal.recursiveLast()
			found = true
			dir = 0
			return
			// end LT
		}
	}
	// INVAR: found is false.

	// a) We know that prev has a keybyte <= key[some depth],
	// so it could qualify as a LTE subtree.
	//
	// b) We know prev.getLTE returned without finding
	// an exact answer.
	//
	// Could this mean that key was < all leaves in next?
	// only if dir was negative.

	if dir < 0 {
		// key is < all leaves in next.
		// The key must fall between
		// prev and prev.prev. Since it
		// was not found in prev, it must
		// be in prev.prev, or in the prev adjacent
		// subtree if prev.prev is nil.

		prevprevKeyb, prevprev := n.Node.prev(&querykey)
		if prevprev == nil {
			_, value = n.Node.first()
			return value, false, needPrevLeaf, 0
		}
		if dir < -1 {
			largestWillDo = true
		}

		// the second recursive getLTE() call.
		value2, found2, dir2, _ := prevprev.getLTE(
			key,
			nextDepth,
			smod,
			prevprev,
			tree,
			calldepth+1,
			largestWillDo,
			byteCmp(querykey, prevprevKeyb, keyCmpPath),
		)

		if found2 {
			return value2, true, 0, 0
		}

		// dir < 0 here.
		if dir2 > 0 {
			// The adjacency condition holds,
			// we have found our LTE value.
			found = true
			value, _ = value2.recursiveLast()
			dir = 0
			return
		}

		dir = needNextLeaf
		if largestWillDo {
			dir = -2
		}
		return value2, false, dir, 0

	} // end if dir < 0

	// still handling the results of the
	// 	value, found, dir = prev.getLTE(key, nextDepth+1, smod, next)
	// call.
	//
	// We know: !found
	// We know: dir > 0
	// We know 'prev' was the largest such that: prevKeyb <= key.
	//
	// Do we have an adjacency conclusion here? Maybe!

	nextKeyb, next := n.Node.next(&querykey)
	if next == nil {
		// goal could be in next subtree.
		_, value = n.Node.last()
		return value, false, needNextLeaf, 0
	}

	if dir < -1 {
		largestWillDo = true
	}

	// the third recursive getLTE() call.
	value2, found2, dir2, _ := next.getLTE(
		key,
		nextDepth+1,
		smod,
		prev,
		tree,
		calldepth+1,
		largestWillDo,
		byteCmp(querykey, nextKeyb, keyCmpPath),
	)

	if found2 {
		return value2, true, 0, 0
	}

	if dir2 < 0 {
		// adjacency conclusion holds: the
		// prev.recursiveLast() is our goal node.
		value, _ = value.recursiveLast()
		return value, true, 0, 0
	}
	if dir2 < 0 && largestWillDo {
		dir2 = -2
	}
	return value2, false, dir2, 0
}
