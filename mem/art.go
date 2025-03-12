package main

import (
	"strconv"
	//"bytes"
	"bufio"
	"fmt"
	art "github.com/glycerine/art-adaptive-radix-tree"
	"os"
	"runtime"
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
	tree := art.NewArtTree()
	paths := loadTestFile("../assets/linux.txt")
	// note that paths are not sorted.

	var prev uint64
	for j := range 3 {
		for i, w := range paths {
			_ = i

			if j > 0 {
				w2 := append([]byte{}, w...)
				w2 = append(w2, []byte(fmt.Sprintf("__%v", j))...)
				if tree.Insert(w2, nil) {
					panic(fmt.Sprintf("i=%v, could not add '%v', already in tree", i, string(w2)))
				}

			} else {
				if tree.Insert(w, nil) {
					panic(fmt.Sprintf("i=%v, could not add '%v', already in tree", i, string(w)))
				}
			}
		}
		//sz := tree.Size()
		//fmt.Printf("sz of assets/linux.txt path list = %v; tree sz = %v\n", len(paths), sz)

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
