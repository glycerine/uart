package main

/* commented to avoid any 3rd party packages
import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"

	googbtree "github.com/google/btree"
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

func main() {

	paths := loadTestFile("../assets/linux.txt")
	// note that paths are not sorted.

	degree := 30 // fastest with all writes that I saw: 876 ns/op
	tree := googbtree.NewG[string](degree, googbtree.Less[string]())

	var prev uint64
	for j := range 3 {
		for i, w := range paths {
			_ = i
			ws := string(w)
			if j > 0 {
				ws += fmt.Sprintf("__%v", j)
			}
			tree.ReplaceOrInsert(ws)
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
