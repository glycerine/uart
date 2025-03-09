package art

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	mathrand "math/rand"
	mathrand2 "math/rand/v2"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var _ = sort.Sort

type sliceByteSlice [][]byte

func (p sliceByteSlice) Len() int { return len(p) }
func (p sliceByteSlice) Less(i, j int) bool {
	return bytes.Compare(p[i], p[j]) <= 0
}
func (p sliceByteSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type sliceByteSliceRev [][]byte

func (p sliceByteSliceRev) Len() int { return len(p) }
func (p sliceByteSliceRev) Less(i, j int) bool {
	return bytes.Compare(p[i], p[j]) > 0
}
func (p sliceByteSliceRev) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func TestArtTree_InsertBasic(t *testing.T) {
	tree := NewArtTree()
	// insert one key
	tree.Insert(Key("I'm Key"), ByteSliceValue("I'm Value"))

	// search it
	value, found := tree.FindExact(Key("I'm Key"))
	assert.Equal(t, ByteSliceValue("I'm Value"), value)
	assert.True(t, found)
	//insert another key
	tree.Insert(Key("I'm Key2"), ByteSliceValue("I'm Value2"))

	// search it
	value, found = tree.FindExact(Key("I'm Key2"))
	assert.Equal(t, ByteSliceValue("I'm Value2"), value)

	// should be found
	value, found = tree.FindExact(Key("I'm Key"))
	assert.Equal(t, ByteSliceValue("I'm Value"), value)

	// lazy path expansion
	tree.Insert(Key("I'm"), ByteSliceValue("I'm"))

	// splitting, check depth on this one; should be 1.
	tree.Insert(Key("I"), ByteSliceValue("I"))

	//vv("tree = %v", tree.String())

	tree.Remove(Key("I"))

	//vv("tree = %v", tree.String())
}

type Set struct {
	key   Key
	value ByteSliceValue
}

func TestArtTree_InsertLongKey(t *testing.T) {
	tree := NewArtTree()
	tree.Insert(Key("sharedKey::1"), ByteSliceValue("value1"))
	tree.Insert(Key("sharedKey::1::created_at"), ByteSliceValue("created_at_value1"))

	value, found := tree.FindExact(Key("sharedKey::1"))
	assert.True(t, found)
	assert.Equal(t, ByteSliceValue("value1"), value)

	value, found = tree.FindExact(Key("sharedKey::1::created_at"))
	assert.True(t, found)
	assert.Equal(t, ByteSliceValue("created_at_value1"), value)
}

func TestArtTree_Insert2(t *testing.T) {
	tree := NewArtTree()
	sets := []Set{{
		Key("sharedKey::1"), ByteSliceValue("value1"),
	}, {
		Key("sharedKey::2"), ByteSliceValue("value2"),
	}, {
		Key("sharedKey::3"), ByteSliceValue("value3"),
	}, {
		Key("sharedKey::4"), ByteSliceValue("value4"),
	}, {
		Key("sharedKey::1::created_at"), ByteSliceValue("created_at_value1"),
	}, {
		Key("sharedKey::1::name"), ByteSliceValue("name_value1"),
	},
	}
	for _, set := range sets {
		tree.Insert(set.key, set.value)
	}
	for _, set := range sets {
		value, found := tree.FindExact(set.key)
		assert.True(t, found)
		assert.Equal(t, set.value, value)
	}
}

func TestArtTree_Insert3(t *testing.T) {
	tree := NewArtTree()
	tree.Insert(Key("sharedKey::1"), ByteSliceValue("value1"))
	tree.Insert(Key("sharedKey::2"), ByteSliceValue("value1"))
	tree.Insert(Key("sharedKey::3"), ByteSliceValue("value1"))
	tree.Insert(Key("sharedKey::4"), ByteSliceValue("value1"))

	tree.Insert(Key("sharedKey::1::created_at"), ByteSliceValue("created_at_value1"))

	tree.Insert(Key("sharedKey::1::name"), ByteSliceValue("name_value1"))

	value, found := tree.FindExact(Key("sharedKey::1::created_at"))
	assert.True(t, found)
	assert.Equal(t, ByteSliceValue("created_at_value1"), value)
}

func TestTree_Update(t *testing.T) {
	tree := NewArtTree()
	key := Key("I'm Key")

	// insert an entry
	tree.Insert(key, ByteSliceValue("I'm Value"))

	// should be found
	value, found := tree.FindExact(key)
	assert.Equal(t, ByteSliceValue("I'm Value"), value)
	assert.Truef(t, found, "The inserted key should be found")

	// try update inserted key
	updated := tree.Insert(key, ByteSliceValue("Value Updated"))
	assert.True(t, updated)

	value, found = tree.FindExact(key)
	assert.Truef(t, found, "The inserted key should be found")
	assert.Equal(t, ByteSliceValue("Value Updated"), value)
}

func TestArtTree_InsertSimilarPrefix(t *testing.T) {
	tree := NewArtTree()
	tree.Insert(Key{1}, []byte{1})
	tree.Insert(Key{1, 1}, []byte{1, 1})

	v, found := tree.FindExact(Key{1, 1})
	assert.True(t, found)
	assert.Equal(t, []byte{1, 1}, v)
}

func TestArtTree_InsertMoreKey(t *testing.T) {
	tree := NewArtTree()
	keys := []Key{{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 1}, {1, 1, 1}, {2, 1, 1}}
	for _, key := range keys {
		tree.Insert(key, ByteSliceValue(key))
	}
	for i, key := range keys {
		value, found := tree.FindExact(key)
		assert.Equalf(t, ByteSliceValue(key), value, "[run:%d],expected :%v but got: %v\n", i, key, value)
		assert.True(t, found)
	}
}

func TestArtTree_Remove(t *testing.T) {
	tree := NewArtTree()
	deleted, _ := tree.Remove(Key("wrong-key"))
	assert.False(t, deleted)

	//vv("tree = '%s'", tree) // empty tree

	tree.Insert(Key("sharedKey::1"), ByteSliceValue("value1"))
	tree.Insert(Key("sharedKey::2"), ByteSliceValue("value2"))

	//vv("tree = '%s'", tree) // node4 at root, 2 leaf, good.

	deleted, value := tree.Remove(Key("sharedKey::2"))
	assert.Equal(t, ByteSliceValue("value2"), value)
	assert.True(t, deleted)

	//vv("tree = '%s'", tree) // leaf at root, key "sharedKey::1"

	deleted, value = tree.Remove(Key("sharedKey::3"))
	//vv("value = '%v'", string(value.(ByteSliceValue))) // 'value1',bad!

	//vv("tree = '%s'", tree) // bad! empty tree, del killed wrong node.

	assert.Nil(t, value)     // red: expecting nil, getting 'value1'
	assert.False(t, deleted) // getting true?!?! expect false

	tree.Insert(Key("sharedKey::3"), ByteSliceValue("value3"))

	deleted, value = tree.Remove(Key("sharedKey"))
	assert.Nil(t, value)
	assert.False(t, deleted)

	tree.Insert(Key("sharedKey::4"), ByteSliceValue("value3"))

	deleted, value = tree.Remove(Key("sharedKey::5::xxx"))
	assert.Nil(t, value)
	assert.False(t, deleted)

	deleted, value = tree.Remove(Key("sharedKey::4xsfdasd"))
	assert.Nil(t, value)
	assert.False(t, deleted)

	tree.Insert(Key("sharedKey::4::created_at"), ByteSliceValue("value3"))
	deleted, value = tree.Remove(Key("sharedKey::4::created_at"))
	assert.True(t, deleted)
}

func TestArtTree_FindExact(t *testing.T) {
	tree := NewArtTree()
	value, found := tree.FindExact(Key("wrong-key"))
	assert.Nil(t, value)
	assert.False(t, found)

	tree.Insert(Key("sharedKey::1"), ByteSliceValue("value1"))

	value, found = tree.FindExact(Key("sharedKey"))
	assert.Nil(t, value)
	assert.False(t, found)
	value, found = tree.FindExact(Key("sharedKey::2"))
	assert.Nil(t, value)
	assert.False(t, found)

	tree.Insert(Key("sharedKey::2"), ByteSliceValue("value1"))

	value, found = tree.FindExact(Key("sharedKey::3"))
	assert.Nil(t, value)
	assert.False(t, found)

	value, found = tree.FindExact(Key("sharedKey"))
	assert.Nil(t, value)
	assert.False(t, found)
}

func TestArtTree_Remove2(t *testing.T) {
	tree := NewArtTree()
	sets := []Set{{
		Key("012345678:-1"), ByteSliceValue("value1"),
	}, {
		Key("012345678:-2"), ByteSliceValue("value2"),
	}, {
		Key("012345678:-3"), ByteSliceValue("value3"),
	}, {
		Key("012345678:-4"), ByteSliceValue("value4"),
	}, {
		Key("012345678:-1*&created_at"), ByteSliceValue("created_at_value1"),
	}, {
		Key("012345678:-1*&name"), ByteSliceValue("name_value1"),
	},
	}
	for _, set := range sets {
		tree.Insert(set.key, set.value)
	}
	for _, set := range sets {
		value, found := tree.FindExact(set.key)
		assert.True(t, found)
		assert.Equal(t, set.value, value)
	}
	for i, set := range sets {
		deleted, value := tree.Remove(set.key)
		assert.True(t, deleted)
		assert.Equalf(t, set.value, value, "[run:%d] should got deleted value:%v,bot got %v\n", i, set.value, value)
	}

}

type keyValueGenerator struct {
	cur       int
	generator func([]byte) []byte
}

func (g keyValueGenerator) getByteSliceValue(key Key) ByteSliceValue {
	return g.generator(key)
}

func (g *keyValueGenerator) prev() (Key, ByteSliceValue) {
	g.cur--
	k, v := g.get()
	return k, v
}

func (g *keyValueGenerator) get() (Key, ByteSliceValue) {
	var buf [8]byte
	binary.PutUvarint(buf[:], uint64(g.cur))
	return buf[:], g.generator(buf[:])
}

func (g *keyValueGenerator) next() (Key, ByteSliceValue) {
	k, v := g.get()
	g.cur++
	return k, v
}

func (g *keyValueGenerator) setCur(c int) {
	g.cur = c
}

func (g *keyValueGenerator) resetCur() {
	g.setCur(0)
}

func NewKeyValueGenerator() *keyValueGenerator {
	return &keyValueGenerator{cur: 0, generator: func(input []byte) []byte {
		return input
	}}
}

type CheckPoint struct {
	name       string
	totalNodes int
	expected   Kind
}

func TestArtTree_Grow(t *testing.T) {
	checkPoints := []CheckPoint{
		{totalNodes: 5, expected: Node16, name: "node4 growing test"},
		{totalNodes: 17, expected: Node48, name: "node16 growing test"},
		{totalNodes: 49, expected: Node256, name: "node256 growing test"},
	}
	for _, point := range checkPoints {
		tree := NewArtTree()
		g := NewKeyValueGenerator()
		for i := 0; i < point.totalNodes; i++ {
			tree.Insert(g.next())
		}
		assert.Equalf(t, int64(point.totalNodes), tree.size, "exected size %d but got %d", point.totalNodes, tree.size)
		assert.Equalf(t, point.expected, tree.root.Kind(), "exected kind %s got %s", point.expected, tree.root.Kind())
		g.resetCur()
		for i := 0; i < point.totalNodes; i++ {
			k, v := g.next()
			got, found := tree.FindExact(k)
			assert.True(t, found, "should found inserted (%v,%v) in test %s", k, v, point.name)
			assert.Equal(t, v, got, "should equal inserted (%v,%v) in test %s", k, v, point.name)
		}
	}
}

func TestArtTree_Shrink(t *testing.T) {
	tree := NewArtTree()
	g := NewKeyValueGenerator()
	// fill up an 256 node
	for i := 0; i < 256; i++ {
		tree.Insert(g.next())
	}
	// check inserted
	g.resetCur()
	for i := 0; i < 256; i++ {
		k, v := g.next()
		got, found := tree.FindExact(k)
		assert.True(t, found)
		assert.Equal(t, v, got)
	}
	// deleting nodes
	for i := 255; i >= 0; i-- {
		assert.Equal(t, int64(i+1), tree.size)
		k, v := g.prev()
		deleted, old := tree.Remove(k)
		assert.True(t, deleted)
		assert.Equalf(t, v, old, "idx:%d, remove k:%v,v:%v", i, k, v)
		switch tree.size {
		case 48:
			assert.Equal(t, Node48, tree.root.Kind())
		case 16:
			assert.Equal(t, Node16, tree.root.Kind())
		case 4:
			assert.Equal(t, Node4, tree.root.Kind())
		case 0:
			assert.Nil(t, tree.root)
		}
	}
}

func TestArtTree_ShrinkConcatenating(t *testing.T) {
	tree := NewArtTree()
	tree.Insert(Key("sharedKey::1"), ByteSliceValue("value1"))
	tree.Insert(Key("sharedKey::2"), ByteSliceValue("value1"))
	tree.Insert(Key("sharedKey::3"), ByteSliceValue("value1"))
	tree.Insert(Key("sharedKey::4"), ByteSliceValue("value1"))

	tree.Insert(Key("sharedKey::1::nested::name"), ByteSliceValue("created_at_value1"))
	tree.Insert(Key("sharedKey::1::nested::job"), ByteSliceValue("name_value1"))

	tree.Insert(Key("sharedKey::1::nested::name::firstname"), ByteSliceValue("created_at_value1"))
	tree.Insert(Key("sharedKey::1::nested::name::lastname"), ByteSliceValue("created_at_value1"))

	tree.Remove(Key("sharedKey::1::nested::name"))

	_, found := tree.FindExact(Key("sharedKey::1::nested::name"))
	assert.False(t, found)
}

func TestArtTree_LargeKeyShrink(t *testing.T) {
	tree := NewArtTree()
	g := NewLargeKeyValueGenerator([]byte("this a very long sharedKey::"))
	// fill up an 256 node
	for i := 0; i < 256; i++ {
		tree.Insert(g.next())
	}
	// check inserted
	g.resetCur()
	for i := 0; i < 256; i++ {
		k, v := g.next()
		got, found := tree.FindExact(k)
		assert.True(t, found)
		assert.Equal(t, v, got)
	}
	// deleting nodes
	for i := 255; i >= 0; i-- {
		assert.Equal(t, int64(i+1), tree.size)
		k, v := g.prev()
		deleted, old := tree.Remove(k)
		assert.True(t, deleted)
		assert.Equal(t, v, old)
		switch tree.size {
		case 48:
			assert.Equal(t, Node48, tree.root.Kind())
		case 16:
			assert.Equal(t, Node16, tree.root.Kind())
		case 4:
			assert.Equal(t, Node4, tree.root.Kind())
		case 0:
			assert.Nil(t, tree.root)
		}
	}
}

type largeKeyValueGenerator struct {
	cur       uint64
	generator func([]byte) []byte
	prefix    []byte
}

func NewLargeKeyValueGenerator(prefix []byte) *largeKeyValueGenerator {
	return &largeKeyValueGenerator{
		cur: 0,
		generator: func(input []byte) []byte {
			return input
		},
		prefix: prefix,
	}
}

func (g *largeKeyValueGenerator) get(cur uint64) (Key, ByteSliceValue) {
	prefixLen := len(g.prefix)
	var buf = make([]byte, prefixLen+8)
	copy(buf[:], g.prefix)
	binary.PutUvarint(buf[prefixLen:], cur)
	return buf, g.generator(buf)
}

func (g *largeKeyValueGenerator) prev() (Key, ByteSliceValue) {
	g.cur--
	k, v := g.get(g.cur)
	return k, v
}

func (g *largeKeyValueGenerator) next() (Key, ByteSliceValue) {
	k, v := g.get(g.cur)
	g.cur++
	return k, v
}

func (g *largeKeyValueGenerator) reset() {
	g.cur = 0
}

func (g *largeKeyValueGenerator) resetCur() {
	g.cur = 0
}

func TestArtTree_InsertOneAndDeleteOne(t *testing.T) {
	tree := NewArtTree()
	g := NewKeyValueGenerator()
	k, v := g.next()

	// insert one
	tree.Insert(k, v)

	//vv("tree = '%s'", tree)

	// delete inserted
	deleted, oldValue := tree.Remove(k)
	assert.Equal(t, v, oldValue)
	assert.True(t, deleted)

	//vv("tree = '%s', after +1, -1.", tree)

	// should be not found
	got, found := tree.FindExact(k)
	assert.Nil(t, got)
	assert.False(t, found)

	// insert another one
	k, v = g.next()
	tree.Insert(k, v)

	// try to delete a non-exist key
	deleted, oldValue = tree.Remove(Key("wrong-key"))
	assert.Nil(t, oldValue)
	assert.False(t, deleted)
}

func TestArtTest_InsertAndDelete(t *testing.T) {
	tree := NewArtTree()
	g := NewKeyValueGenerator()
	// insert 1000
	N := 1_000_000
	if true { // underRaceDetector {
		N = 100
	}
	for i := 0; i < N; i++ {
		_ = tree.Insert(g.next())
	}
	g.resetCur()
	// check inserted kv
	for i := 0; i < N; i++ {
		//if i%10_000 == 0 {
		//vv("search loop i = %v", i)
		//}
		k, v := g.next()
		got, found := tree.FindExact(k)
		assert.Equalf(t, v, got, "should insert key-value (%v:%v) but got %v", k, v, got)
		assert.True(t, found)
	}
	g.resetCur()
	for i := 0; i < N; i++ {
		//if i%10_000 == 0 {
		//vv("remove loop i = %v", i)
		//}
		k, v := g.next()
		deleted, got := tree.Remove(k)
		assert.Equal(t, v, got)
		assert.True(t, deleted)
	}
}

func TestArtTree_InsertLargeKeyAndDelete(t *testing.T) {
	tree := NewArtTree()
	g := NewLargeKeyValueGenerator([]byte("largeThanMax"))

	N := 1_000_000
	if true { // underRaceDetector {
		N = 100
	}
	for i := 0; i < N; i++ {
		_ = tree.Insert(g.next())
	}
	g.reset()
	// check inserted kv
	for i := 0; i < N; i++ {
		k, v := g.next()
		got, found := tree.FindExact(k)
		assert.Equalf(t, v, got, "should insert key-value (%v:%v)", k, v)
		assert.True(t, found)
	}
	g.resetCur()
	for i := 0; i < N; i++ {
		k, v := g.next()
		deleted, got := tree.Remove(k)
		assert.Equal(t, v, got)
		assert.True(t, deleted)
	}
}

// Benchmark
func loadTestFile(path string) [][]byte {
	file, err := os.Open(path)
	if err != nil {
		panic("Couldn't open " + path)
	}
	defer file.Close()

	var words [][]byte
	reader := bufio.NewReader(file)
	for {
		if line, err := reader.ReadBytes(byte('\n')); err != nil {
			break
		} else {
			if len(line) > 0 {
				words = append(words, line[:len(line)-1])
			}
		}
	}
	return words
}

type KV struct {
	Key            []byte
	ByteSliceValue []byte
}

func TestTree_InsertWordSets(t *testing.T) {
	words := loadTestFile("./assets/words.txt")
	tree := NewArtTree()

	if true { // underRaceDetector {
		words = words[:500]
	}
	for _, w := range words {
		tree.Insert(w, w)
	}
	for i, w := range words {
		v, found := tree.FindExact(w)
		assert.True(t, found)
		assert.Truef(t, bytes.Equal(v.([]byte), w), "[run:%d] should found %s,but got %s\n", i, w, v)
	}
	//TODO:
	for i, w := range words {
		deleted, v := tree.Remove(w)
		assert.True(t, deleted)
		assert.Truef(t, bytes.Equal(v.([]byte), w), "[run:%d] should got %s,but got %s\n", i, w, v)
	}
}

func Compare(a, b KV) bool {
	return bytes.Compare(a.Key, b.Key) < 0
}

func BenchmarkWordsArtInsert(b *testing.B) {
	words := loadTestFile("./assets/words.txt")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tree := NewArtTree()
		for _, w := range words {
			tree.Insert(w, w)
		}
	}
}

func BenchmarkWordsMapInsert(b *testing.B) {
	words := loadTestFile("./assets/words.txt")
	var strWords []string
	for _, word := range words {
		strWords = append(strWords, string(word))
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		m := make(map[string]string)
		for _, w := range strWords {
			m[w] = w
		}
	}
}

func TestDeleteConcurrentInsert(t *testing.T) {

	paths := loadTestFile("assets/linux.txt")
	n := len(paths)
	_ = n

	m := make(map[string]string)
	l := NewArtTree()
	for k := range paths {
		l.Insert(paths[k], paths[k])
		s := string(paths[k])
		m[s] = s
	}

	for i := 0; i <= 10; i++ {
		readFrac := float32(i) / 10.0
		_ = readFrac
		fmt.Println()
		fmt.Printf("on remove Frac %v\n", readFrac)
		rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
		for i := range 2 {
			//vv("on pass i = %v", i)
			fmt.Printf("%v ...", i)
			for k := range paths {
				s := string(paths[k])
				if rng.Float32() < readFrac {
					//l.FindExact(randomKey(rng))
					//l.FindExact(paths[k])
					l.Remove(paths[k])
					delete(m, s)
				} else {
					//l.Insert(randomKey(rng), value)
					l.Insert(paths[k], paths[k])
					m[s] = s
				}
			}
			if !mapTreeSame(m, l) {
				t.Fatal("map and tree different")
			}
		}
	}
}

func mapTreeSame(m map[string]string, tree *Tree) bool {
	nmap := len(m)
	ntree := tree.Size()
	if nmap != ntree {
		panic(fmt.Sprintf("nmap = %v; ntree = %v", nmap, ntree)) // 0,-1
	}
	//dup := make(map[string]string)
	//for k, v := range m {
	//	dup[k] = v
	//}
	for k := range m {
		_, found := tree.FindExact([]byte(k))
		if !found {
			panic(fmt.Sprintf("in map: '%v'; not in tree", k))
		}
	}
	return true
}

func TestArtTree_SearchMod_GTE(t *testing.T) {

	tree := NewArtTree()
	tree.Insert(Key("sharedKey::1"), ByteSliceValue("value1"))
	tree.Insert(Key("sharedKey::1::created_at"), ByteSliceValue("created_at_value1"))

	//vv("tree = '%s'", tree)

	v, found := tree.FindGTE(Key("sharedKey::1"))
	assert.True(t, found)
	assert.Equal(t, string(ByteSliceValue("value1")), string(v.(ByteSliceValue)))
	//vv("past GTE test!")

	v, found = tree.FindGT(Key("sharedKey::1"))
	assert.True(t, found)
	//vv("GT search got lf.Key = '%v'", string(lf.Key)) // 'sharedKey::1'
	// nil pointer deref!
	assert.Equal(t, string(ByteSliceValue("created_at_value1")), string(v.(ByteSliceValue)))

}

func TestArtTree_SearchMod_GT_requires_backtracking(t *testing.T) {

	// when deep in a GT search and encountering
	// GT a key that is the right-most (greatest) child
	// in a left (lesser) subtree, the search may
	// have to tell the parent (recursively) to give
	// the next child's first subtree (leftmost) leaf.

	tree := NewArtTree()
	// a side, 3 levels deep
	tree.Insert(Key("a01"), ByteSliceValue("a01"))
	tree.Insert(Key("a02"), ByteSliceValue("a02"))
	tree.Insert(Key("a13"), ByteSliceValue("a13"))
	tree.Insert(Key("a14"), ByteSliceValue("a14"))

	// b side, 3 levels deep
	tree.Insert(Key("b01"), ByteSliceValue("b01"))
	tree.Insert(Key("b02"), ByteSliceValue("b02"))
	tree.Insert(Key("b13"), ByteSliceValue("b13"))
	tree.Insert(Key("b14"), ByteSliceValue("b14"))

	// so the GT from a14 -> b01 is the mechanism
	// to test/implement.

	//vv("tree = '%s'", tree)

	v, found := tree.FindGT(Key("a14"))
	//vv("search for > 'a14' => found = %v; lf = '%s'", found, lf)
	assert.True(t, found)
	assert.Equal(t, string(ByteSliceValue("b01")), string(v.(ByteSliceValue)))
	//vv("past GT with recursion test!")
}

func TestArtTree_SearchMod_LT_requires_backtracking(t *testing.T) {

	// when deep in a LT search and encountering
	// LT a key that is the left-most (smallest) child
	// in a right (greater) subtree, the search may
	// have to tell the parent (recursively) to give
	// the prev child's last subtree (rightmost) leaf.

	tree := NewArtTree()
	// a side, 3 levels deep
	tree.Insert(Key("a01"), ByteSliceValue("a01"))
	tree.Insert(Key("a02"), ByteSliceValue("a02"))
	tree.Insert(Key("a13"), ByteSliceValue("a13"))
	tree.Insert(Key("a14"), ByteSliceValue("a14"))

	// b side, 3 levels deep
	tree.Insert(Key("b01"), ByteSliceValue("b01"))
	tree.Insert(Key("b02"), ByteSliceValue("b02"))
	tree.Insert(Key("b13"), ByteSliceValue("b13"))
	tree.Insert(Key("b14"), ByteSliceValue("b14"))

	// so the LT from b01 -> a14 is the mechanism
	// to test/implement.

	//vv("tree = '%s'", tree)

	lf, found := tree.FindLT(Key("b01"))
	assert.True(t, found)
	//vv("search for 'b01' => found = %v; lf = '%s'", found, lf)
	assert.Equal(t, string(ByteSliceValue("a14")), string(lf.(ByteSliceValue)))
	//vv("past LT with back-track-recursion test!")
}

func TestArtTree_SearchMod_big_GT_only(t *testing.T) {
	// GT should work on big trees

	return // GT not done yet! only have GTE at the moment.
	tree := NewArtTree()
	paths := loadTestFile("assets/linux.txt")
	// paths are not sorted.

	// check the tree against sorted paths
	sorted := loadTestFile("assets/linux.txt")
	sort.Sort(sliceByteSlice(sorted))

	for i, w := range paths {
		_ = i
		if tree.Insert(w, w) {
			t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
		}
	}

	sz := tree.Size()
	//vv("sz = %v", sz)
	var key []byte
	for i := 0; i < sz; i++ {
		lf, found := tree.Find(GT, key)
		if !found {
			panic(fmt.Sprintf("could not find key GT '%v' at i=%v", string(key), i))
		}
		_ = lf
		lfkey := string(lf.Key)
		wanted := string(sorted[i])
		if lfkey != wanted {
			panic(fmt.Sprintf("on i = %v, wanted = '%v' but lfkey ='%v'", i, wanted, lfkey))
		}
		//vv("good: i=%v, lfkey == wanted(%v)", i, wanted)
		// search past lf.Key next time.
		key = lf.Key
	}
}

func Test_707_ArtTree_SearchMod_big_GTE(t *testing.T) {
	return
	// GTE should work on big trees

	tree := NewArtTree()
	//paths := loadTestFile("assets/linux.txt")
	// paths are not sorted.

	// check the tree against sorted paths
	sorted := loadTestFile("assets/linux.txt")
	sort.Sort(sliceByteSlice(sorted))

	// while debugging
	smalltest := true
	if smalltest {
		sorted = sorted[:1200] // 306,600 green, 1200 red on i=302
		//sorted = sorted[93774:]
	}

	for i, w := range sorted {
		// only insert the evens, so we can search GTE on odds.
		if i%2 == 0 {
			if tree.Insert(w, w) {
				t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
			}
		}
	}

	sz := tree.Size()
	//vv("sz = %v", sz)

	//vv("tree = '%s'", tree)
	if true {
		for i := 250; i < 320; i++ {
			//for i := range sorted {
			if i%2 == 0 {
				fmt.Printf("sorted[%02d] %v      *in tree*\n", i, string(sorted[i]))
			} else {
				fmt.Printf("sorted[%02d] %v\n", i, string(sorted[i]))
			}
		}
	}
	var key []byte

	// do GTE the odd keys, expecting to get the next even.
	//vv("begin GTE on odds expecting even hits.")

	var wrong []int

	//for i := 2; i < sz; i += 2 {

	// panic at i = 28, with :306 in sorted.
	for i := 304; i < sz; i += 2 {

		key = sorted[i-1]
		//vv("i=%v, searching GTE key '%v', expecting to hit '%v'", i, string(key), string(sorted[i]))
		lf, found := tree.Find(GTE, key)
		if !found {
			panic(fmt.Sprintf("could not find key GTE '%v' at i=%v", string(key), i))
		}

		lfkey := string(lf.Key)
		wanted := string(sorted[i])
		if lfkey != wanted {
			wrong = append(wrong, i)
			panic(fmt.Sprintf("on i = %v, wanted = '%v' but lfkey ='%v'", i, wanted, lfkey))
		}
		//vv("good: i=%v, lfkey == wanted(%v)", i, wanted)
	}

	//vv("wrong = '%#v'", wrong)
	// tree_test.go:924 2025-03-06 22:18:23.213 -0600 CST wrong = '[]int{28, 60, 62, 92, 108}' with  :306

	// verify nil gives the first key in the tree in GTE
	lf, found := tree.Find(GTE, nil)
	if !found {
		panic("not found GTE nil key")
	}
	if bytes.Compare(lf.Key, sorted[0]) != 0 {
		panic("nil GTE search did not give first leaf")
	}
}

func Test_808_ArtTree_SearchMod_big_LT_only(t *testing.T) {
	// LT should work on big trees

	return // LT not done yet!

	tree := NewArtTree()
	paths := loadTestFile("assets/linux.txt")
	// paths are not sorted.

	// check the tree against sorted paths
	sorted := loadTestFile("assets/linux.txt")
	sort.Sort(sliceByteSliceRev(sorted))

	for i, w := range paths {
		_ = i
		if tree.Insert(w, w) {
			t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
		}
	}

	sz := tree.Size()
	////vv("sz = %v", sz)

	// debug a specific problem we found going LT from here.
	// This found a bug in n48 prev().
	prob := "../../torvalds/linux/virt"
	expect := "../../torvalds/linux/usr/initramfs_data.S"
	lf, found := tree.Find(LT, []byte(prob))
	if !found {
		panic("LT prob not found")
	}
	lfkey := string(lf.Key)
	if lfkey != expect {
		panic(fmt.Sprintf("want '%v'; got '%v'", expect, lfkey))
	}

	var key []byte
	for i := 0; i < sz; i++ {
		lf, found := tree.Find(LT, key)
		if !found {
			panic(fmt.Sprintf("could not find key LT '%v' at i=%v", string(key), i))
		}
		_ = lf
		lfkey := string(lf.Key)
		wanted := string(sorted[i])
		if lfkey != wanted {
			panic(fmt.Sprintf("on i = %v, wanted = '%v' but lfkey ='%v'", i, wanted, lfkey))
		}
		////vv("good: i=%v, lfkey == wanted(%v)", i, wanted)
		// search past lf.Key next time.
		key = lf.Key
	}
}

func Test909_ArtTree_SearchMod_numbered_GTE(t *testing.T) {
	//return
	// GTE should work on big trees, but make them
	// numbered for ease of inspection.

	tree := NewArtTree()

	var sorted [][]byte
	for i := range 10_000 {
		sorted = append(sorted, []byte(fmt.Sprintf("%03d", i)))
	}
	sort.Sort(sliceByteSlice(sorted))

	for i, w := range sorted {
		// only insert the evens, so we can search GTE on odds.
		if i%2 == 0 {
			if tree.Insert(w, w) {
				t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
			}
		}
	}

	sz := tree.Size()
	//vv("sz = %v", sz)

	////vv("tree = '%s'", tree)
	if false {
		for i := range sorted {
			if i%2 == 0 {
				fmt.Printf("sorted[%02d] %v      *in tree*\n", i, string(sorted[i]))
			} else {
				fmt.Printf("sorted[%02d] %v\n", i, string(sorted[i]))
			}
		}
	}
	var key []byte

	// do GTE the odd keys, expecting to get the next even.
	//vv("begin GTE on odds expecting even hits.")

	var wrong []int

	for i := 2; i < sz; i += 2 {

		key = sorted[i-1]
		////vv("i=%v, searching GTE key '%v', expecting to hit '%v'", i, string(key), string(sorted[i]))
		lf, found := tree.Find(GTE, key)
		if !found {
			panic(fmt.Sprintf("could not find key GTE '%v' at i=%v", string(key), i))
		}

		lfkey := string(lf.Key)
		wanted := string(sorted[i])
		if lfkey != wanted {
			wrong = append(wrong, i)
			panic(fmt.Sprintf("on i = %v, wanted = '%v' but lfkey ='%v'", i, wanted, lfkey))
		}
		////vv("good: i=%v, lfkey == wanted(%v)", i, wanted)
	}

	//vv("wrong = '%#v'", wrong)
	// tree_test.go:924 2025-03-06 22:18:23.213 -0600 CST wrong = '[]int{28, 60, 62, 92, 108}' with  :306

	// verify nil gives the first key in the tree in GTE
	lf, found := tree.Find(GTE, nil)
	if !found {
		panic("not found GTE nil key")
	}
	if bytes.Compare(lf.Key, sorted[0]) != 0 {
		panic("nil GTE search did not give first leaf")
	}
}

func Test505_ArtTree_SearchMod_random_numbered_GTE(t *testing.T) {

	// GTE should work on big trees, but make them
	// numbered for ease of inspection.

	// j=total number of leaves in the tree.
	// titrate up to find the smallest tree with
	// a problem. -- to make for easier diagnostics.
	//for j := 7818; j < 7819; j++ {

	for j := 1; j < 5000; j++ {

		// green, but takes 2 minutes.
		//for j := 1; j < 20025; j++ {

		// red j=93; i = 34 without path comparison!
		//for j := 93; j < 94; j++ {
		//for j := 1; j < 94; j++ {
		//for j := 68; j < 69; j++ {

		//for j := 1; j < 5000; j++ {
		//if j%500 == 0 {
		//	//vv("on j = %v", j)
		//}

		tree := NewArtTree()

		var seed32 [32]byte
		chacha8 := mathrand2.NewChaCha8(seed32)

		var sorted [][]byte
		var N uint64 = 100000 // domain for leaf keys.

		// j = number of leaves in the tree.

		used := make(map[int]bool) // tree may dedup, but sorted needs too.
		for range j {
			r := int(chacha8.Uint64() % N)
			if used[r] {
				continue
			}
			used[r] = true
			sorted = append(sorted, []byte(fmt.Sprintf("%06d", r)))
		}
		sort.Sort(sliceByteSlice(sorted))

		for i, w := range sorted {
			// only insert the evens, so we can search GTE on odds.
			if i%2 == 0 {
				if tree.Insert(w, w) {
					// make sure leaves are unique.
					t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
				}
			}
		}

		sz := tree.Size()
		_ = sz
		////vv("sz = %v", sz)
		////vv("tree = '%s'", tree)
		showlist := func(want int, got string) {

			for i, nd := range sorted {
				extra := ""
				ssi := string(sorted[i])
				if i == want {
					extra += " <<< want!"
				}
				if ssi == got {
					extra += " <<< got!"
				}
				if i%2 == 0 {
					fmt.Printf("%p sorted[%02d] %v      *in tree*  %v\n", nd, i, ssi, extra)
				} else {
					fmt.Printf("%p sorted[%02d] %v  %v\n", nd, i, ssi, extra)
				}
			}
		}
		_ = showlist
		//showlist(-1, "")

		var key []byte

		// do GTE the odd keys, expecting to get the next even.
		////vv("begin GTE on odds expecting even hits.")

		var wrong []int

		//for i := 18; i < 19; i += 2 {
		for i := 2; i < sz; i += 2 {
			//for i := 3646; i < 3647; i += 2 {
			//for i := 34; i < 35; i += 2 {

			key = sorted[i-1]
			//pp("i=%v, searching GTE key '%v', expecting to hit '%v'", i, string(key), string(sorted[i]))
			lf, found := tree.Find(GTE, key)
			if !found {
				panic(fmt.Sprintf("could not find key GTE '%v' at i=%v", string(key), i))
			}
			wanted := string(sorted[i])
			if lf == nil {
				//vv("nil leaf back, not good!")
				wrong = append(wrong, i)

				panic(fmt.Sprintf("on i = %v, (GTE key '%v') wanted = '%v' but lf was nil", i, string(key), wanted))
			}
			lfkey := string(lf.Key)
			if lfkey != wanted {
				wrong = append(wrong, i)

				if false {
					// yes, we confirmed that '048261' is in tree with this search; should have seen it on search GTE 048255, but got 048330 instead.
					fmt.Printf("\n\n======= problem! now debugging with a Search2 call! ======\n\n")

					regularSearchLeaf, regularFound := tree.FindExact(sorted[i])
					_ = regularSearchLeaf
					_ = regularFound
					//vv("confirm key '%v' remains in tree: regularFound='%v'; regularSearch='%v'", string(sorted[i]), regularFound, regularSearchLeaf)
				}

				//showlist(i, lfkey)
				panic(fmt.Sprintf("on j=%v; i = %v, (GTE key '%v') wanted = '%v' but lfkey ='%v'", j, i, string(key), wanted, lfkey))
			}
			////vv("good: i=%v, lfkey == wanted(%v)", i, wanted)
		}

		////vv("wrong = '%#v'", wrong)
		// tree_test.go:924 2025-03-06 22:18:23.213 -0600 CST wrong = '[]int{28, 60, 62, 92, 108}' with  :306

		// verify nil gives the first key in the tree in GTE
		lf, found := tree.Find(GTE, nil)
		if !found {
			panic("not found GTE nil key")
		}
		if bytes.Compare(lf.Key, sorted[0]) != 0 {
			panic("nil GTE search did not give first leaf")
		}
	}
}

func Test506_ArtTree_SearchMod_random_numbered_GT_(t *testing.T) {

	// same as 505 but for GT now (rather than GTE).

	// j=total number of leaves in the tree.
	for j := 1; j < 5000; j++ {
		//for j := 1; j < 20000; j++ {
		//for j := 7; j < 8; j++ {

		// green, but takes 2 minutes.
		//for j := 1; j < 20025; j++ {

		// red j=93; i = 34 without path comparison!
		//for j := 93; j < 94; j++ {
		//for j := 1; j < 94; j++ {
		//for j := 68; j < 69; j++ {

		//for j := 1; j < 5000; j++ {
		//if j%500 == 0 {
		//	//vv("on j = %v", j)
		//}

		tree := NewArtTree()

		var seed32 [32]byte
		chacha8 := mathrand2.NewChaCha8(seed32)

		var sorted [][]byte
		var N uint64 = 100000 // domain for leaf keys.

		// j = number of leaves in the tree.

		used := make(map[int]bool) // tree may dedup, but sorted needs too.
		for range j {
			r := int(chacha8.Uint64() % N)
			if used[r] {
				continue
			}
			used[r] = true
			sorted = append(sorted, []byte(fmt.Sprintf("%06d", r)))
		}
		sort.Sort(sliceByteSlice(sorted))

		for i, w := range sorted {
			// only insert the evens
			if i%2 == 0 {
				if tree.Insert(w, w) {
					// make sure leaves are unique.
					t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
				}
			}
		}

		sz := tree.Size()
		_ = sz
		////vv("sz = %v", sz)
		////vv("tree = '%s'", tree)
		showlist := func(want int, got string) {

			for i, nd := range sorted {
				extra := ""
				ssi := string(sorted[i])
				if i == want {
					extra += " <<< want!"
				}
				if ssi == got {
					extra += " <<< got!"
				}
				if i%2 == 0 {
					fmt.Printf("%p sorted[%02d] %v      *in tree*  %v\n", nd, i, ssi, extra)
				} else {
					fmt.Printf("%p sorted[%02d] %v  %v\n", nd, i, ssi, extra)
				}
			}
		}
		_ = showlist
		//showlist(-1, "")

		var key []byte

		// do GT the odd keys, expecting to get the next even.
		////vv("begin GT on odds expecting even hits.")

		var wrong []int

		lim := sz - 2
		for i := 0; i < lim; i++ {

			// we query every key, on each i (GTE went by 2s).
			var wanted string
			var wanti int
			if i%2 == 0 {
				// evens are in the tree, odds are not

				// querying for > an even should give
				// the next even, since odds are not in the tree.
				wanti = i + 2
			} else {
				wanti = i + 1
				// query for GT an odd should give the next (even) leaf.
			}
			wanted = string(sorted[wanti])

			key = sorted[i]

			//pp("i=%v, searching GT key '%v', expecting to hit '%v'", i, string(key), string(sorted[wanti]))
			lf, found := tree.Find(GT, key)
			if !found {
				showlist(wanti, "")
				panic(fmt.Sprintf("could not find key GT '%v' at i=%v", string(key), i))
			}

			////vv("i=%v; wanti=%v; wanted='%v'", i, wanti, wanted)

			if lf == nil {
				//vv("nil leaf back, not good!")
				wrong = append(wrong, i)

				panic(fmt.Sprintf("on i = %v, (GT key '%v') wanted = '%v' but lf was nil", i, string(key), wanted))
			}
			lfkey := string(lf.Key)
			if lfkey != wanted {
				wrong = append(wrong, wanti)
				showlist(wanti, lfkey)
				panic(fmt.Sprintf("on j=%v; i = %v, (GT key '%v') wanted = '%v' but lfkey ='%v'", j, i, string(key), wanted, lfkey))
			}
			////vv("good: i=%v, lfkey == wanted(%v)", i, wanted)
		}

		////vv("wrong = '%#v'", wrong)
		// tree_test.go:924 2025-03-06 22:18:23.213 -0600 CST wrong = '[]int{28, 60, 62, 92, 108}' with  :306

		// verify nil gives the first key in the tree in GT
		lf, found := tree.Find(GT, nil)
		if !found {
			panic("not found GT nil key")
		}
		if bytes.Compare(lf.Key, sorted[0]) != 0 {
			panic("nil GT search did not give first leaf")
		}
	}
}

//

func Test507_ArtTree_SearchMod_random_numbered_LTE(t *testing.T) {

	// LTE should work on big trees, but make them
	// numbered for ease of inspection.

	// j=total number of leaves in the tree.
	// titrate up to find the smallest tree with
	// a problem. -- to make for easier diagnostics.

	for j := 1; j < 5000; j++ {

		// green, but takes 2 minutes.
		//for j := 1; j < 20025; j++ {

		//for j := 1; j < 5000; j++ {
		//if j%100 == 0 {
		//	//vv("on j = %v", j)
		//}

		tree := NewArtTree()

		var seed32 [32]byte
		chacha8 := mathrand2.NewChaCha8(seed32)

		var sorted [][]byte
		var N uint64 = 100000 // domain for leaf keys.

		// j = number of leaves in the tree.

		used := make(map[int]bool) // tree may dedup, but sorted needs too.
		for range j {
			r := int(chacha8.Uint64() % N)
			if used[r] {
				continue
			}
			used[r] = true
			sorted = append(sorted, []byte(fmt.Sprintf("%06d", r)))
		}
		sort.Sort(sliceByteSlice(sorted))

		var lastLeaf *Leaf
		for i, w := range sorted {
			// only insert the evens, so we can search LTE on odds.
			if i%2 == 0 {
				key2 := Key(append([]byte{}, w...))
				lf := NewLeaf(key2, key2, nil)
				if tree.InsertLeaf(lf) {
					// make sure leaves are unique.
					t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
				}
				lastLeaf = lf
			}
		}

		sz := tree.Size()
		_ = sz
		////vv("sz = %v", sz)
		////vv("tree = '%s'", tree)
		showlist := func(want int, got string) {

			for i, nd := range sorted {
				extra := ""
				ssi := string(sorted[i])
				if i == want {
					extra += " <<< want!"
				}
				if ssi == got {
					extra += " <<< got!"
				}
				if i%2 == 0 {
					fmt.Printf("%p sorted[%02d] %v      *in tree*  %v\n", nd, i, ssi, extra)
				} else {
					fmt.Printf("%p sorted[%02d] %v  %v\n", nd, i, ssi, extra)
				}
			}
		}
		_ = showlist
		//showlist(-1, "")

		var key []byte

		// do LTE the odd keys, expecting to get the previous even.
		////vv("begin LTE on odds expecting lesser even hits.")

		var wrong []int

		for i := 0; i < sz-1; i += 2 {

			key = sorted[i+1]
			//pp("i=%v, searching LTE key '%v', expecting to hit '%v'", i, string(key), string(sorted[i]))
			lf, found := tree.Find(LTE, key)
			if !found {
				panic(fmt.Sprintf("could not find key LTE '%v' at i=%v", string(key), i))
			}
			wanted := string(sorted[i])
			if lf == nil {
				//vv("nil leaf back, not good!")
				wrong = append(wrong, i)

				panic(fmt.Sprintf("on i = %v, (LTE key '%v') wanted = '%v' but lf was nil", i, string(key), wanted))
			}
			lfkey := string(lf.Key)
			if lfkey != wanted {
				wrong = append(wrong, i)

				//showlist(i, lfkey)
				panic(fmt.Sprintf("on j=%v; i = %v, (LTE key '%v') wanted = '%v' but lfkey ='%v'", j, i, string(key), wanted, lfkey))
			}
			////vv("good: i=%v, lfkey == wanted(%v)", i, wanted)
		}

		////vv("wrong = '%#v'", wrong)

		// verify nil gives the last key in the tree in LTE
		lf, found := tree.Find(LTE, nil)
		if !found {
			panic("not found LTE nil key")
		}
		if lf != lastLeaf {
			panic(fmt.Sprintf("nil LTE search did not give last leaf. lf='%v'; lastLeaf='%v'", lf, lastLeaf))
		}
	}

}

func Test508_ArtTree_SearchMod_random_numbered_LT_(t *testing.T) {

	// same as 506 but for LT now

	// j=total number of leaves in the tree.
	for j := 1; j < 5000; j++ {
		//for j := 1; j < 20000; j++ {
		//for j := 7; j < 8; j++ {

		// green, but takes 2 minutes.
		//for j := 1; j < 20025; j++ {

		// red j=93; i = 34 without path comparison!
		//for j := 93; j < 94; j++ {
		//for j := 1; j < 94; j++ {
		//for j := 68; j < 69; j++ {

		//for j := 1; j < 5000; j++ {
		//if j%100 == 0 {
		//	//vv("on j = %v", j)
		//}

		tree := NewArtTree()

		var seed32 [32]byte
		chacha8 := mathrand2.NewChaCha8(seed32)

		var sorted [][]byte
		var N uint64 = 100000 // domain for leaf keys.

		// j = number of leaves in the tree.

		used := make(map[int]bool) // tree may dedup, but sorted needs too.
		for range j {
			r := int(chacha8.Uint64() % N)
			if used[r] {
				continue
			}
			used[r] = true
			sorted = append(sorted, []byte(fmt.Sprintf("%06d", r)))
		}
		sort.Sort(sliceByteSlice(sorted))

		var lastLeaf *Leaf
		for i, w := range sorted {
			// only insert the evens, so we can search LTE on odds.
			if i%2 == 0 {
				key2 := Key(append([]byte{}, w...))
				lf := NewLeaf(key2, key2, nil)
				if tree.InsertLeaf(lf) {
					// make sure leaves are unique.
					t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
				}
				lastLeaf = lf
			}
		}

		sz := tree.Size()
		_ = sz
		////vv("sz = %v", sz)
		////vv("tree = '%s'", tree)
		showlist := func(want int, got string) {

			for i, nd := range sorted {
				extra := ""
				ssi := string(sorted[i])
				if i == want {
					extra += " <<< want!"
				}
				if ssi == got {
					extra += " <<< got!"
				}
				if i%2 == 0 {
					fmt.Printf("%p sorted[%02d] %v      *in tree*  %v\n", nd, i, ssi, extra)
				} else {
					fmt.Printf("%p sorted[%02d] %v  %v\n", nd, i, ssi, extra)
				}
			}
		}
		_ = showlist
		//showlist(-1, "")

		var key []byte

		// do LT the odd keys, expecting to get the next even.
		////vv("begin LT on odds expecting even hits.")

		var wrong []int

		lim := sz - 2
		for i := 2; i < lim; i++ {

			// we query every key, on each i
			var wanted string
			var wanti int
			if i%2 == 0 {
				// evens are in the tree, odds are not

				// querying for < an even should give
				// the next even, since odds are not in the tree.
				wanti = i - 2
			} else {
				wanti = i - 1
				// query for LT an odd should give the next (even) leaf.
			}
			wanted = string(sorted[wanti])

			key = sorted[i]

			//pp("i=%v, searching LT key '%v', expecting to hit '%v'", i, string(key), string(sorted[wanti]))
			lf, found := tree.Find(LT, key)
			if !found {
				showlist(wanti, "")
				panic(fmt.Sprintf("could not find key LT '%v' at i=%v", string(key), i))
			}

			////vv("i=%v; wanti=%v; wanted='%v'", i, wanti, wanted)

			if lf == nil {
				//vv("nil leaf back, not good!")
				wrong = append(wrong, i)

				panic(fmt.Sprintf("on i = %v, (LT key '%v') wanted = '%v' but lf was nil", i, string(key), wanted))
			}
			lfkey := string(lf.Key)
			if lfkey != wanted {
				wrong = append(wrong, wanti)
				showlist(wanti, lfkey)
				panic(fmt.Sprintf("on j=%v; i = %v, (LT key '%v') wanted = '%v' but lfkey ='%v'", j, i, string(key), wanted, lfkey))
			}
			////vv("good: i=%v, lfkey == wanted(%v)", i, wanted)
		}

		////vv("wrong = '%#v'", wrong)

		// verify nil gives the first key in the tree in LT
		lf, found := tree.Find(LT, nil)
		if !found {
			panic("not found LT nil key")
		}
		if lf != lastLeaf {
			panic("nil LT search did not give last leaf")
		}
	}
}

// SubN updates
func Test510_SubN_maintained_for_At_indexing_(t *testing.T) {

	// j=total number of leaves in the tree.
	//for j := 1; j < 5000; j++ {
	for j := 1; j < 1000; j++ {

		//if j%100 == 0 {
		//	//vv("on j = %v", j)
		//}

		tree := NewArtTree()

		var seed32 [32]byte
		chacha8 := mathrand2.NewChaCha8(seed32)

		var sorted [][]byte
		var N uint64 = 100000 // domain for leaf keys.

		// j = number of leaves in the tree.

		used := make(map[int]bool) // tree may dedup, but sorted needs too.
		for range j {
			r := int(chacha8.Uint64() % N)
			if used[r] {
				continue
			}
			used[r] = true
			sorted = append(sorted, []byte(fmt.Sprintf("%06d", r)))
		}
		sort.Sort(sliceByteSlice(sorted))

		//vv("verifying SubN after each insert")

		var lastLeaf *Leaf
		_ = lastLeaf
		for i, w := range sorted {

			key2 := Key(append([]byte{}, w...))
			lf := NewLeaf(key2, key2, nil)
			if tree.InsertLeaf(lf) {
				// make sure leaves are unique.
				t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
			}
			lastLeaf = lf

			// after each insert, verify correct SubN counts.
			verifySubN(tree.root)
		}

		//vv("verifying SubN after removal")
		sz := tree.Size()

		var key []byte

		//vv("starting tree = '%v'", tree)

		for i := range sz {
			key = sorted[i]
			tree.Remove(key)

			//vv("after %v removal, tree = '%v'", i+1, tree)
			// after each delete, verify correct SubN counts.
			verifySubN(tree.root)
		}
	}
}

// verifySubN:
// walk through the subtree at root, counting children.
// At each inner node, verify that SubN
// has an accurate count of children, or panic.
func verifySubN(root *bnode) (leafcount int) {

	if root == nil {
		return 0
	}
	if root.isLeaf {
		return 1
	} else {

		inode := root.inner.Node
		switch n := inode.(type) {
		case *node4:
			for i := range n.children {
				if i < n.lth {
					leafcount += verifySubN(n.children[i])
				}
			}
		case *node16:
			for i := range n.children {
				if i < n.lth {
					leafcount += verifySubN(n.children[i])
				}
			}
		case *node48:
			for _, k := range n.keys {

				if k == 0 {
					continue
				}
				child := n.children[k-1]
				leafcount += verifySubN(child)
			}
		case *node256:
			for _, child := range n.children {
				if child != nil {
					leafcount += verifySubN(child)
				}
			}
		}

		if root.inner.SubN != leafcount {
			panic(fmt.Sprintf("leafcount=%v, but n.SubN = %v", leafcount, root.inner.SubN))
		}
	}
	return leafcount
}

// At(i) returns the ith-element in the sorted tree.
func Test511_At_index_the_tree_like_an_array(t *testing.T) {

	// j=total number of leaves in the tree.
	//for j := 1; j < 10_000; j++ { // 42 sec
	for j := 1; j < 500; j++ { // 0.10 sec

		//if j%100 == 0 {
		//	//vv("on j = %v", j)
		//}

		tree := NewArtTree()

		var seed32 [32]byte
		chacha8 := mathrand2.NewChaCha8(seed32)

		var sorted [][]byte
		var N uint64 = 100000 // domain for leaf keys.

		// j = number of leaves in the tree.

		used := make(map[int]bool) // tree may dedup, but sorted needs too.
		for range j {
			r := int(chacha8.Uint64() % N)
			if used[r] {
				continue
			}
			used[r] = true
			sorted = append(sorted, []byte(fmt.Sprintf("%06d", r)))
		}
		sort.Sort(sliceByteSlice(sorted))

		var lastLeaf *Leaf
		_ = lastLeaf
		for i, w := range sorted {

			key2 := Key(append([]byte{}, w...))
			lf := NewLeaf(key2, key2, nil)
			if tree.InsertLeaf(lf) {
				// make sure leaves are unique.
				t.Fatalf("i=%v, could not add '%v', already in tree", i, string(w))
			}
			lastLeaf = lf
		}

		//vv("verifying SubN after removal")
		sz := tree.Size()

		//vv("starting tree = '%v'", tree)

		for i := range sz {
			lf, ok := tree.At(i)
			if !ok {
				panic(fmt.Sprintf("missing leaf!?! j=%v; i=%v not ok", j, i))
			}
			// test Atv too.
			val, ok2 := tree.Atv(i)
			if !ok2 {
				panic(fmt.Sprintf("missing leaf!?! j=%v; i=%v not ok2", j, i))
			}
			got := string(lf.Key)
			want := string(sorted[i])
			if got != want {
				panic(fmt.Sprintf("at j=%v; i=%v, want '%v'; got '%v'", j, i, want, got))
			}
			got2 := string(val.(Key))
			if got2 != want {
				panic(fmt.Sprintf("at j=%v; i=%v, want '%v'; got2 '%v'", j, i, want, got2))
			}
		}
	}
}
