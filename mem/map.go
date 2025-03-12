package main

import (
	"bufio"
	"os"
	"strconv"
	//"bytes"
	"fmt"
	"runtime"
	//mathrand2 "math/rand/v2"
	// ugh. not actually sorted. just preserves
	// the _insertion_ order of the keys.
	//"github.com/elliotchance/orderedmap/v3"
	// also not ordered. but very fast. And cool for having
	// O(1) snapshots. keep for speed comparison/inspiration.
	//art "github.com/glycerine/art-adaptive-radix-tree"
	//rb "github.com/glycerine/rbtree"
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

	var empty struct{}
	gomap := make(map[string]struct{})

	var prev uint64
	for j := range 3 {
		for i, w := range paths {
			_ = i
			ws := string(w)
			if j > 0 {
				ws += fmt.Sprintf("__%v", j)
			}
			gomap[ws] = empty
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
