package art

import "fmt"

type node16 struct {
	lth      int
	keys     [16]byte
	children [16]*bnode

	//dep        int
	//compressed []byte
}

func (n *node16) last() (byte, *bnode) {
	return n.keys[n.lth-1], n.children[n.lth-1]
}
func (n *node16) first() (byte, *bnode) {
	return n.keys[0], n.children[0]
}

/*
func (n *node16) getCompressed() []byte {
	return n.compressed
}
func (n *node16) setCompressed(pathpart []byte) {
	n.compressed = pathpart
}

func (n *node16) setDepth(d int) {
	n.dep = d
}
func (n *node16) depth() int {
	return n.dep
}
*/

func (n *node16) nchild() int {
	return int(n.lth)
}
func (n *node16) childkeysString() (s string) {
	s = "["
	for i := range n.lth {
		if i > 0 {
			s += ", "
		}
		k := n.keys[i]
		if k == 0 {
			s += "zero"
		} else {
			if k < 33 || k > '~' {
				s += fmt.Sprintf("0x%x", byte(k))
			} else {
				s += fmt.Sprintf("'%v'", string(k))
			}
		}
	}
	return s + "]"
}

func (n *node16) Kind() Kind {
	return Node16
}

func (n *node16) index(k byte) int {
	for i, b := range n.keys {
		if k <= b {
			return i
		}
	}
	return int(n.lth)
}

// we get more inlining by putting this in the
// same file. Faster than the assembly.
func index(key *byte, nkey *[16]byte) (int, bool) {
	for i := range nkey {
		if nkey[i] == *key {
			return i, true
		}
	}
	return 0, false
}

func (n *node16) child(k byte) (idx int, ch *bnode) {
	var key byte
	for idx, key = range n.keys {
		if key == k {
			ch = n.children[idx]
			return
		}
	}
	return
}

func (n *node16) next(k *byte) (byte, *bnode) {
	if k == nil {
		return n.keys[0], n.children[0]
	}
	for i, b := range n.keys {
		if b > *k {
			return b, n.children[i]
		}
	}
	return 0, nil
}

// A nil k will return the first key.
//
// Otherwise, we return the first
// (smallest) key that is >= *k.
//
// A nil bnode back means that all keys were < *k.
func (n *node16) gte(k *byte) (byte, *bnode) {
	if k == nil {
		return n.keys[0], n.children[0]
	}

	for i, b := range n.keys {
		if b >= *k {
			//vv("node16.gte() sees byte '%v' >= *k == %v, returning child i=%v", string(b), string(*k), i)
			return b, n.children[i]
		}
	}
	//vv("node16.gte() sees all keys < *k(%v): '%v'", string(*k), n.childkeysString())
	return 0, nil
}

// A nil k will return the first key.
//
// Otherwise, we return the first
// (smallest) key that is > *k.
//
// A nil bnode back means that all keys were <= *k.
func (n *node16) gt(k *byte) (byte, *bnode) {
	if k == nil {
		return n.keys[0], n.children[0]
	}
	for i, b := range n.keys {
		if b > *k {
			return b, n.children[i]
		}
	}
	return 0, nil
}

func (n *node16) prev(k *byte) (byte, *bnode) {
	if k == nil {
		idx := n.lth - 1
		return n.keys[idx], n.children[idx]
	}
	// we use an int for lnt now to avoid underflow.
	for i := n.lth - 1; i >= 0; i-- {
		if n.keys[i] < *k {
			return n.keys[i], n.children[i]
		}
	}
	return 0, nil
}

func (n *node16) replace(idx int, child *bnode) (old *bnode) {
	old = n.children[idx]
	if child == nil {
		copy(n.keys[idx:], n.keys[idx+1:])
		copy(n.children[idx:], n.children[idx+1:])
		n.keys[n.lth-1] = 0
		n.children[n.lth-1] = nil
		n.lth--
	} else {
		n.children[idx] = child
	}
	return
}

func (n *node16) full() bool {
	return n.lth == 16
}

func (n *node16) addChild(k byte, child *bnode) {
	idx := n.index(k)
	copy(n.children[idx+1:], n.children[idx:])
	copy(n.keys[idx+1:], n.keys[idx:])
	n.keys[idx] = k
	n.children[idx] = child
	n.lth++
}

func (n *node16) grow() Inode {
	nn := &node48{
		lth: n.lth,
		//compressed: append([]byte{}, n.compressed...),
		//dep:        n.dep,
	}
	copy(nn.children[:], n.children[:])
	for i, child := range n.children {
		if child == nil {
			continue
		}
		nn.keys[n.keys[i]] = uint16(i) + 1
	}
	return nn
}

func (n *node16) min() bool {
	return n.lth <= 5
}

func (n *node16) shrink() Inode {
	nn := node4{
		//compressed: append([]byte{}, n.compressed...),
		//dep:        n.dep,
	}
	copy(nn.keys[:], n.keys[:])
	copy(nn.children[:], n.children[:])
	nn.lth = n.lth
	return &nn
}

func (n *node16) String() string {
	return fmt.Sprintf("n16[%x]", n.keys[:n.lth])
}

// lt

// A nil k will return the last key.
//
// Otherwise, we return the largest
// (right-most) key that is < *k.
//
// A nil bnode back means that all keys were > *k.
func (n *node16) lt(k *byte) (keyb byte, ch *bnode) {
	if k == nil {
		return n.keys[n.lth-1], n.children[n.lth-1]
	}
	for idx := n.lth - 1; idx >= 0; idx-- {
		b := n.keys[idx]
		if b < *k {
			return b, n.children[idx]
		}
	}
	return 0, nil
}

// lte: A nil k will return the last key.
//
// Otherwise, we return the largest
// (right-most) key that is <= *k.
//
// A nil bnode back means that all keys were > *k.
func (n *node16) lte(k *byte) (byte, *bnode) {
	if k == nil {
		return n.keys[n.lth-1], n.children[n.lth-1]
	}
	for idx := n.lth - 1; idx >= 0; idx-- {
		b := n.keys[idx]
		if b <= *k {
			return b, n.children[idx]
		}
	}
	return 0, nil
}
