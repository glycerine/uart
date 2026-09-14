package uart

import (
	"bytes"
	"fmt"

	"testing"
	"time"

	googbtree "github.com/google/btree"
)

// the original 620 on which btree/goog really shone, because
// the test plays to its strengths: sorted input plus huge degree.
func Test620_original_dream_version_for_googbtree_unlocked_read_comparison(t *testing.T) {

	// with data already in, how fast are we vs a map?

	// K = total number of keys (leaves) in the starting tree.
	// nothing fancy, just sequential integers -> strings.
	K := 10_000_000

	var keys []string
	var keyb [][]byte
	for k := range K {
		key := fmt.Sprintf("%09d", k)
		keys = append(keys, key)
		keyb = append(keyb, []byte(key))
	}

	tree := NewArtTree()
	tree.SkipLocking = true

	gomap := make(map[string]int)
	t0 := time.Now()
	for k, key := range keys {
		gomap[key] = k
	}
	e0 := time.Since(t0)
	rate0 := e0 / time.Duration(K)
	fmt.Printf("map time to store %v keys: %v (%v/op)\n", K, e0, rate0)

	t0 = time.Now()
	for k, v := range gomap {
		_, _ = k, v
		//if keys[v] != k {
		//	panic(fmt.Sprintf("gomap gave %v instead of %v", k, keys[v]))
		//}
	}
	e0 = time.Since(t0)
	rate0 = e0 / time.Duration(K)
	fmt.Printf("map reads %v keys: elapsed %v (%v/op)\n", K, e0, rate0)

	t1 := time.Now()
	for k, kb := range keyb {
		tree.Insert(kb, k)
	}
	e1 := time.Since(t1)
	rate1 := e1 / time.Duration(K)
	fmt.Printf("uart.Tree time to store %v keys: %v (%v/op)\n", K, e1, rate1)

	t1 = time.Now()
	for lf := range Ascend(tree, nil, nil) {
		_ = lf
	}
	e1 = time.Since(t1)
	rate1 = e1 / time.Duration(K)
	fmt.Printf("Ascend(tree) reads %v keys: elapsed %v (%v/op)\n", K, e1, rate1)

	// try the native iterator instead of iter.Seq

	it := tree.Iter(nil, nil)
	t1 = time.Now()
	for it.Next() {
		_ = it.Key
	}
	e1 = time.Since(t1)
	rate1 = e1 / time.Duration(K)
	fmt.Printf("uart Iter() reads %v keys: elapsed %v (%v/op)\n", K, e1, rate1)

	// and the integer indexing:

	t1 = time.Now()
	var lf *Leaf
	var ok bool
	var v int
	for i := range K {
		lf, ok = tree.At(i)
		v = lf.Value.(int)
		if !ok || v != i {
			panic(fmt.Sprintf("At(i=%v) gave %v instead of %v", i, v, i))
		}

	}
	e1 = time.Since(t1)
	rate1 = e1 / time.Duration(K)
	fmt.Printf("tree.At(i) reads %v keys: elapsed %v (%v/op)\n", K, e1, rate1)

	// we would like sequential iteration from
	// larger than 0 to work too. start from 10.
	t1 = time.Now()
	beg := 10
	for i := beg; i < K; i++ {
		lf, ok = tree.At(i)
		v = lf.Value.(int)
		if !ok || v != i {
			panic(fmt.Sprintf("At(i=%v) gave %v instead of %v", i, v, i))
		}

	}
	e1 = time.Since(t1)
	rate1 = e1 / time.Duration(K)
	fmt.Printf("tree.At(i) reads from %v: %v keys: elapsed %v (%v/op)\n", beg, K-beg, e1, rate1)

	// Atfar should work the same as un-cached At
	t1 = time.Now()
	for i := range K {
		lf, ok = tree.Atfar(i)
		v = lf.Value.(int)
		if !ok || v != i {
			panic(fmt.Sprintf("Atfar(i=%v) gave %v instead of %v", i, v, i))
		}

	}
	e1 = time.Since(t1)
	rate1 = e1 / time.Duration(K)
	fmt.Printf("tree.Atfar(i) reads %v keys: elapsed %v (%v/op)\n", K, e1, rate1)

	// with locking on
	tree.SkipLocking = false
	t1 = time.Now()
	for k := range K {
		tree.Atfar(k)
	}
	e1 = time.Since(t1)
	rate1 = e1 / time.Duration(K)
	fmt.Printf("Atfar() read-locked reads %v keys: elapsed %v (%v/op)\n", K, e1, rate1)

	// commented for no dependencies:

	// google/btree load and read

	degree := 3_000 // fastest; full table scan: 2 ns/key (put at 207 ns/key)
	//degree := 32 // full table scan: 6 ns/key (put at 241 ns/key)
	//degree := 10 // full table scan :  7 ns/key (put at 286 ns/key)
	//g := googbtree.NewG[string](degree, googbtree.Less[string]())
	g := googbtree.NewG[*Kint](degree, googbtree.LessFunc[*Kint](func(a, b *Kint) bool {
		return bytes.Compare(a.Key, b.Key) < 0
	}))

	t1 = time.Now()
	for k, kb := range keyb {
		kint := &Kint{
			Key: kb,
			Val: k,
		}
		//g.ReplaceOrInsert(ks)
		g.ReplaceOrInsert(kint)
	}
	e1 = time.Since(t1)
	rate1 = e1 / time.Duration(K)
	fmt.Printf("google/btree time to store %v keys: %v (%v/op)\n", K, e1, rate1)

	t1 = time.Now()
	g.Ascend(func(kint *Kint) bool { return true })

	e1 = time.Since(t1)
	rate1 = e1 / time.Duration(K)
	fmt.Printf("google/btree reads SEQUENTIALLY (in a FULL TABLE SCAN) %v keys: elapsed %v (%v/op)\n", K, e1, rate1)
	fmt.Printf("Note that random reads from the btree will be much slower(!)\n")

}
