package uart

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	googbtree "github.com/google/btree"
)

var memtableBenchSink int

type memtableBenchKV struct {
	Key []byte
	Val int
}

func memtableSequentialKeys(n int) [][]byte {
	keys := make([][]byte, n)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("%09d", i))
	}
	return keys
}

func memtablePermutedUint64Keys(n int) [][]byte {
	keys := make([][]byte, n)
	for i := range keys {
		key := make([]byte, 8)
		x := uint64(i) * 11400714819323198485
		binary.BigEndian.PutUint64(key, x)
		keys[i] = key
	}
	return keys
}

func benchmarkMemtableBuildScanART(b *testing.B, keys [][]byte, noCopy bool, scan bool) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	var sink int
	for range b.N {
		tree := NewArtTree()
		tree.SkipLocking = true
		for _, key := range keys {
			if noCopy {
				tree.InsertNoCopy(key, nil)
			} else {
				tree.Insert(key, nil)
			}
		}
		if scan {
			tree.ScanLeaves(func(lf *Leaf) bool {
				sink += len(lf.Key)
				return true
			})
		} else {
			it := tree.Iter(nil, nil)
			for it.Next() {
				lf := it.Leaf()
				sink += len(lf.Key)
			}
		}
	}
	memtableBenchSink = sink
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(keys)), "ns/key")
}

func benchmarkMemtableBuildScanARTBulkSorted(b *testing.B, keys [][]byte) {
	b.Helper()
	items := make([]BulkItem, len(keys))
	for i, key := range keys {
		items[i] = BulkItem{Key: key, Value: nil}
	}

	b.ReportAllocs()
	b.ResetTimer()

	var sink int
	for range b.N {
		tree := NewArtTreeFromSortedNoCopy(items)
		tree.SkipLocking = true
		tree.ScanLeaves(func(lf *Leaf) bool {
			sink += len(lf.Key)
			return true
		})
	}
	memtableBenchSink = sink
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(keys)), "ns/key")
}

func benchmarkMemtableBuildScanGoogleBTree(b *testing.B, keys [][]byte, degree int) {
	b.Helper()
	items := make([]*memtableBenchKV, len(keys))
	for i, key := range keys {
		items[i] = &memtableBenchKV{Key: key, Val: i}
	}
	less := googbtree.LessFunc[*memtableBenchKV](func(a, b *memtableBenchKV) bool {
		return bytes.Compare(a.Key, b.Key) < 0
	})

	b.ReportAllocs()
	b.ResetTimer()

	var sink int
	for range b.N {
		tree := googbtree.NewG[*memtableBenchKV](degree, less)
		for _, item := range items {
			tree.ReplaceOrInsert(item)
		}
		tree.Ascend(func(item *memtableBenchKV) bool {
			sink += len(item.Key)
			return true
		})
	}
	memtableBenchSink = sink
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(keys)), "ns/key")
}

func BenchmarkMemtableBuildScanSequential100K(b *testing.B) {
	keys := memtableSequentialKeys(100_000)
	benchmarkMemtableBuildScan(b, keys)
}

func BenchmarkMemtableBuildScanPermutedUint64_100K(b *testing.B) {
	keys := memtablePermutedUint64Keys(100_000)
	benchmarkMemtableBuildScan(b, keys)
}

func BenchmarkMemtableBuildScanLinuxPaths(b *testing.B) {
	keys := loadTestFile("assets/linux.txt")
	benchmarkMemtableBuildScan(b, keys)
}

func benchmarkMemtableBuildScan(b *testing.B, keys [][]byte) {
	b.Run("uart_insert_iter", func(b *testing.B) {
		benchmarkMemtableBuildScanART(b, keys, false, false)
	})
	b.Run("uart_insert_scan", func(b *testing.B) {
		benchmarkMemtableBuildScanART(b, keys, false, true)
	})
	b.Run("uart_insert_nocopy_scan", func(b *testing.B) {
		benchmarkMemtableBuildScanART(b, keys, true, true)
	})
	if sortedKeyBytes(keys) {
		b.Run("uart_bulk_sorted_nocopy_scan", func(b *testing.B) {
			benchmarkMemtableBuildScanARTBulkSorted(b, keys)
		})
	}
	b.Run("google_btree_degree_32", func(b *testing.B) {
		benchmarkMemtableBuildScanGoogleBTree(b, keys, 32)
	})
	b.Run("google_btree_degree_3000", func(b *testing.B) {
		benchmarkMemtableBuildScanGoogleBTree(b, keys, 3000)
	})
}

func sortedKeyBytes(keys [][]byte) bool {
	for i := 1; i < len(keys); i++ {
		if bytes.Compare(keys[i-1], keys[i]) >= 0 {
			return false
		}
	}
	return true
}

func BenchmarkMemtableInsertSequential100K(b *testing.B) {
	keys := memtableSequentialKeys(100_000)
	b.Run("uart_insert", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var sink int
		for range b.N {
			tree := NewArtTree()
			tree.SkipLocking = true
			for _, key := range keys {
				tree.Insert(key, nil)
			}
			sink += tree.Size()
		}
		memtableBenchSink = sink
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(keys)), "ns/key")
	})
	b.Run("uart_insert_nocopy", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var sink int
		for range b.N {
			tree := NewArtTree()
			tree.SkipLocking = true
			for _, key := range keys {
				tree.InsertNoCopy(key, nil)
			}
			sink += tree.Size()
		}
		memtableBenchSink = sink
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(keys)), "ns/key")
	})
	for _, degree := range []int{32, 3000} {
		degree := degree
		b.Run(fmt.Sprintf("google_btree_degree_%d", degree), func(b *testing.B) {
			items := make([]*memtableBenchKV, len(keys))
			for i, key := range keys {
				items[i] = &memtableBenchKV{Key: key, Val: i}
			}
			less := googbtree.LessFunc[*memtableBenchKV](func(a, b *memtableBenchKV) bool {
				return bytes.Compare(a.Key, b.Key) < 0
			})
			b.ReportAllocs()
			b.ResetTimer()
			var sink int
			for range b.N {
				tree := googbtree.NewG[*memtableBenchKV](degree, less)
				for _, item := range items {
					tree.ReplaceOrInsert(item)
				}
				sink += tree.Len()
			}
			memtableBenchSink = sink
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(keys)), "ns/key")
		})
	}
}

func BenchmarkMemtableScanSequential100K(b *testing.B) {
	keys := memtableSequentialKeys(100_000)

	tree := NewArtTree()
	tree.SkipLocking = true
	for _, key := range keys {
		tree.InsertNoCopy(key, nil)
	}

	b.Run("uart_iter", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var sink int
		for range b.N {
			it := tree.Iter(nil, nil)
			for it.Next() {
				sink += len(it.Leaf().Key)
			}
		}
		memtableBenchSink = sink
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(keys)), "ns/key")
	})
	b.Run("uart_scan", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var sink int
		for range b.N {
			tree.ScanLeaves(func(lf *Leaf) bool {
				sink += len(lf.Key)
				return true
			})
		}
		memtableBenchSink = sink
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(keys)), "ns/key")
	})
	for _, degree := range []int{32, 3000} {
		degree := degree
		b.Run(fmt.Sprintf("google_btree_degree_%d", degree), func(b *testing.B) {
			items := make([]*memtableBenchKV, len(keys))
			for i, key := range keys {
				items[i] = &memtableBenchKV{Key: key, Val: i}
			}
			less := googbtree.LessFunc[*memtableBenchKV](func(a, b *memtableBenchKV) bool {
				return bytes.Compare(a.Key, b.Key) < 0
			})
			g := googbtree.NewG[*memtableBenchKV](degree, less)
			for _, item := range items {
				g.ReplaceOrInsert(item)
			}
			b.ReportAllocs()
			b.ResetTimer()
			var sink int
			for range b.N {
				g.Ascend(func(item *memtableBenchKV) bool {
					sink += len(item.Key)
					return true
				})
			}
			memtableBenchSink = sink
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*len(keys)), "ns/key")
		})
	}
}
