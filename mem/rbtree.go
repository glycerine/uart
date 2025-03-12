package main

/* commented in uart to avoid the dependency on rbtree
import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"

	rb "github.com/glycerine/rbtree"
)

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

type kvs struct {
	key string
	val string
}

func newRBtree() *rb.Tree {
	rbtree := rb.NewTree(func(a, b rb.Item) int {
		av := a.(*kvs).key
		bv := b.(*kvs).key
		if av > bv {
			return -1
		}
		if av < bv {
			return 1
		}
		return 0
	})
	return rbtree
}

func main() {

	paths := loadTestFile("../assets/linux.txt")
	// note that paths are not sorted.

	rbtree := newRBtree()

	var prev uint64
	for j := range 3 {
		for i, w := range paths {
			_ = i
			ws := string(w)
			if j > 0 {
				ws += fmt.Sprintf("__%v", j)
			}

			pay := &kvs{
				key: ws,
			}
			rbtree.Insert(pay)
		}

		//runtime.GC()
		mstat := &runtime.MemStats{}
		runtime.ReadMemStats(mstat)
		ha := mstat.HeapAlloc
		fmt.Printf("mstat.HeapAlloc = '%v' (copies = %v; diff = %v bytes)\n", formatUnder(int(ha)), j+1, formatUnder(int(ha-prev)))
		prev = ha
	}

}

func formatUnder(n int) string {
	// Convert to string first
	str := strconv.FormatInt(int64(n), 10)

	// Handle numbers less than 1000
	if len(str) <= 3 {
		return str
	}

	// Work from right to left, adding underscores
	var result []byte
	for i := len(str) - 1; i >= 0; i-- {
		if (len(str)-1-i)%3 == 0 && i != len(str)-1 {
			result = append([]byte{'_'}, result...)
		}
		result = append([]byte{str[i]}, result...)
	}

	return string(result)
}
*/
