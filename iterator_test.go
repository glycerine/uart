package art

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIterator(t *testing.T) {

	keys := []string{
		"1234",
		"1245",
		"1345",
		"1267",
	}
	sorted := make([]string, len(keys))
	copy(sorted, keys)
	sort.Strings(sorted)

	reversed := make([]string, len(keys))
	copy(reversed, keys)
	sort.Sort(sort.Reverse(sort.StringSlice(reversed)))

	for _, tc := range []struct {
		desc       string
		keys       []string
		start, end string
		reverse    bool
		rst        []string
	}{
		{
			desc: "full",
			keys: keys,
			rst:  sorted,
		},
		{
			desc: "empty",
			rst:  []string{},
		},
		{
			desc: "matching leaf",
			keys: keys[:1],
			rst:  keys[:1],
		},
		{
			desc:  "non matching leaf",
			keys:  keys[:1],
			rst:   []string{},
			start: "13",
		},
		{
			desc: "limited by end",
			keys: keys,
			end:  "125",
			rst:  sorted[:2],
		},
		{
			desc:  "limited by start",
			keys:  keys,
			start: "124",
			rst:   sorted[1:],
		},
		{
			desc:  "start is excluded",
			keys:  keys,
			start: "1234",
			rst:   sorted[1:],
		},
		{
			desc:  "start to end",
			keys:  keys,
			start: "125",
			end:   "1344",
			rst:   sorted[2:3],
		},
		{
			desc:    "reverse",
			keys:    keys,
			rst:     reversed,
			reverse: true,
		},
		{
			desc:    "reverse until",
			keys:    keys,
			start:   "1234",
			rst:     reversed[:4],
			reverse: true,
		},
		{
			desc:    "reverse from",
			keys:    keys,
			end:     "1268",
			rst:     reversed[1:],
			reverse: true,
		},
		{
			desc:    "reverse from until",
			keys:    keys,
			start:   "1235",
			end:     "1268",
			rst:     reversed[1:3],
			reverse: true,
		},
	} {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			tree := NewArtTree()
			for _, key := range tc.keys {
				tree.Insert([]byte(key), key)
			}
			iter := tree.Iterator([]byte(tc.start), []byte(tc.end))
			if tc.reverse {
				iter = iter.Reverse()
			}
			rst := []string{}
			for iter.Next() {
				rst = append(rst, iter.Value().(string))
			}
			require.Equal(t, tc.rst, rst)
		})
	}
}

func TestIterConcurrentExpansion(t *testing.T) {

	var (
		tree = NewArtTree()
		keys = [][]byte{
			[]byte("aaba"),
			[]byte("aabb"),
		}
	)

	for _, key := range keys {
		tree.Insert(key, key)
	}
	iter := tree.Iterator(nil, nil)
	require.True(t, iter.Next())
	require.Equal(t, Key(keys[0]), iter.Key())

	// adding a 3rd key, after iter started,
	// that is after the 2nd key we have not read yet.
	tree.Insert([]byte("aaca"), nil)
	require.True(t, iter.Next())
	require.Equal(t, Key(keys[1]), iter.Key())

	require.True(t, iter.Next())
	require.Equal(t, Key("aaca"), iter.Key())
}

func TestIterDeleteBehindFwd(t *testing.T) {

	tree := NewArtTree()
	N := 60000
	for i := range N {
		k := fmt.Sprintf("%09d", i)
		key := Key(k) // []byte
		tree.Insert(key, key)
	}
	//vv("full tree before any delete/iter: '%s'", tree)
	got := make(map[int]int)
	deleted := make(map[int]int)
	kept := make(map[int]int)

	iter := tree.Iterator(nil, nil)
	thresh := 5000
	for iter.Next() {
		sz := tree.Size()
		k := iter.Key()
		nk, err := strconv.Atoi(strings.TrimSpace(string(k)))
		panicOn(err)
		got[nk] = len(got)
		// e.g. for N=6 and thresh=4 => delete 0,1,2,3. keep 4,5
		if nk < thresh {
			gone, _ := tree.Remove(k)
			if !gone {
				panic("should have gone")
			}
			deleted[nk] = len(deleted)

			sz2 := tree.Size()
			if sz2 != sz-1 {
				//vv("tree now '%s'", tree)
				panic("should have shrunk tree")
			}
		} else {
			kept[nk] = len(kept)
		}
	}
	sz := tree.Size()
	//vv("after iter, sz = %v", sz)
	//vv("got (len %v) = '%#v'", len(got), got)
	//vv("deleted (len %v) = '%#v'", len(deleted), deleted)
	//vv("kept (len %v) = '%#v'", len(kept), kept)

	if thresh > N {
		thresh = N // simpler verification below, no change in above.
	}

	if sz != (N - thresh) {
		t.Fatalf("expected tree to be size %v, but see %v", N-thresh, sz)
	}
	if len(got) != N {
		t.Fatalf("expected got(len %v) to be len %v", len(got), N)
	}
	if len(deleted) != thresh {
		t.Fatalf("expected deleted(len %v) to be len %v",
			len(deleted), thresh)
	}
	//vv("tree at end '%s'", tree)
	for i := thresh; i < N; i++ {
		k := fmt.Sprintf("%09d", i)
		key := Key(k) // []b
		_, found := tree.FindExact(key)
		if !found {
			t.Fatalf("expected to find '%v' still in tree", k)
		}
	}

	for i := 0; i < N; i++ {
		if _, ok := got[i]; !ok {
			t.Fatalf("expected got[i=%v] to be present.", i)
		}
		if i < thresh {
			if _, ok := deleted[i]; !ok {
				t.Fatalf("expected deleted[i=%v] to be present.", i)
			}
		} else {
			if _, ok := kept[i]; !ok {
				t.Fatalf("expected kept[i=%v] to be present.", i)
			}
		}
	}
}

func TestIterDeleteBehindReverse(t *testing.T) {

	tree := NewArtTree()
	N := 60_000
	if N >= 1_000_000_000 {
		panic(`must bump up the Sprintf("%09d", i) ` +
			`have sufficient lead 0 padding`)
	}
	for i := range N {
		// if we don't zero pad, then lexicographic
		// delete order is very different from
		// numerical order, and we might get
		// confused below--like we did at first
		// when wondering why 8 and 9 are the
		// first two deletions with 60 keys
		// in the tree. Lexicographically, they
		// are the largest.
		k := fmt.Sprintf("%09d", i)
		key := Key(k) // []byte
		tree.Insert(key, key)
	}
	//vv("full tree before any delete/iter: '%s'", tree)
	got := make(map[int]int)
	deleted := make(map[int]int)
	kept := make(map[int]int)

	iter := tree.Iterator(nil, nil)
	iter = iter.Reverse()

	thresh := 20_000
	callcount := 0
	for iter.Next() {
		callcount++
		sz := tree.Size()
		k := iter.Key()
		nk, err := strconv.Atoi(strings.TrimSpace(string(k)))
		panicOn(err)
		got[nk] = len(got)
		// reversed testing uses callcount here,
		// so that reversed (order issued) actually matters.
		// e.g. for N=6, iter should return     5,4,3,2,1,0
		// and so for thresh = 2, we should del 5,4         (len thresh)
		//                         and keep         3,2,1,0 (len N-thresh)
		// kept is < N-thresh;
		// deleted is >= N-thresh
		if callcount <= thresh {
			//vv("calling Remove(%v)", nk)
			gone, _ := tree.Remove(k)
			if !gone {
				panic("should have gone")
			}
			deleted[nk] = len(deleted)

			sz2 := tree.Size()
			if sz2 != sz-1 {
				//vv("tree now '%s'", tree)
				panic("should have shrunk tree")
			}
		} else {
			kept[nk] = len(kept)
		}
	}
	sz := tree.Size()
	//vv("after iter, sz = %v", sz)
	//vv("got (len %v) = '%#v'", len(got), got)
	//vv("deleted (len %v) = '%#v'", len(deleted), deleted)
	//vv("kept (len %v) = '%#v'", len(kept), kept)

	if thresh > N {
		thresh = N // simpler verification below
	}

	if sz != (N - thresh) {
		t.Fatalf("expected tree to be size %v, but see %v", N-thresh, sz)
	}
	if len(got) != N {
		t.Fatalf("expected got(len %v) to be len %v", len(got), N)
	}
	if len(deleted) != thresh {
		t.Fatalf("expected deleted(len %v) to be len %v",
			len(deleted), thresh)
	}
	//vv("tree at end '%s'", tree)
	// kept: i < N-thresh
	for i := 0; i < N-thresh; i++ {
		k := fmt.Sprintf("%09d", i)
		key := Key(k) // []b
		_, found := tree.FindExact(key)
		if !found {
			t.Fatalf("expected to find '%v' still in tree", k)
		}
	}

	for i := 0; i < N; i++ {
		if _, ok := got[i]; !ok {
			t.Fatalf("expected got[i=%v] to be present.", i)
		}
		if i >= N-thresh {
			if _, ok := deleted[i]; !ok {
				t.Fatalf("expected deleted[i=%v] to be present.", i)
			}
		} else {
			if _, ok := kept[i]; !ok {
				t.Fatalf("expected kept[i=%v] to be present.", i)
			}
		}
	}
}
