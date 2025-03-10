# the Adaptive radix tree (ART): sorted and speedy

Note: this is a minimal-dependency version of 
the Adaptive Radix Tree (ART) implementation at
https://github.com/glycerine/art-adaptive-radix-tree
and it comes without greenpack serialization support. 
Thus it is unserialized ART, or uart. See the mother project
above for full benchmark code and built-in serialization capabilities.

This project provides an implemention
of the Adaptive Radix Tree (ART) data structure[1]. 

Why? In read-heavy situations, ART
trees can have very good performance 
(e.g. 49ns/op vs 32ns/op for a standard Go
map wrapped with a RWMutex) while also providing
sorted-ordered-key lookups, range queries,
and remaining goroutine safe if writing
does become necessary (see the benchmarks 
below). Modern analytic-oriented databases 
like Tableau and DuckDB leverage ART trees
to implement their indexes because they
provide significant speedups[6].
They were designed in 2013 by the German academic
computer scientists whose HyPer database
was bought to provide the backend engine
for Tableau[7].

An ART tree is a sorted, key-value, in-memory
dictionary. It maps arbitrary []byte keys to
an any value. The ART tree provides both path compression
(vertical compression) and variable
sized inner nodes (horizontal compression)
for space-efficient fanout.

Path compression is particularly attractive in
situations where many keys have shared or redundant
prefixes. This is the common case for
many ordered-key-value-map use cases, such
as database indexes and file-system hierarchies.
The Google File System paper, for example,
emphasizes the efficiencies obtained
by exploiting prefix compression in their
distributed file system[2]. FoundationDB's
new Redwood backend provides it as a feature[3],
and users wish the API could be improved by
offering it[4] in query result APIs.

As an alternative to red-black trees,
AVL trees, and other kinds of balanced binary trees,
ART trees are attractive because of their speed and
space savings (especially though path compression). Like
those trees, ART offers an ordered index
of sorted keys allowing efficient O(log N) access
for each unique key. However, as the benchmarks
below indicate (1440 nsec per lookup for red-black tree,
versus 53 nsec for ART), lookups can be orders of
magnitude faster (27x in that benchmark, for the read-only case).
As a point of humility, we note that the skip-list
measured here was even faster. Skip
lists do not provide prefix compression
or reverse iteration (in their single-link typical form)
and have other trade-offs that are out of scope here.
Still they are an interesting data structure
that may also be worth investigating in your
application's context with more than 
this quick and cursory benchmark as a guide.

randomized/non-deterministic performance; and worst
case O(N) insertion for "tall towers".
Nonetheless, they may also be worth investigating in your
application context with more than 
this quick and cursory benchmark as a guide.

Ease of use: efficient greater-than/less-than key lookup
and range iteration, as well as the
ability to "treat the tree as a slice" using
integer indexes (based on the counted B-tree
idea[5] -- see the tree.At(i int) method), make this ART tree implementation
particularly easy to use in practice.

This ART tree supports only a single value for each
key -- it is not a "multi-map" in the C++ sense.
This makes it simple to use and implement. 
Note that the user can store any value, so 
being a unique-key-map is not really a limitation.
The user can simply point to a struct, slice or map
of same-key values in the Leaf.Value field.

Concurrency: this ART implementation is
goroutine safe, as it uses a sync.RWMutex
for synchronization. Thus it allows only a
single writer at a time, and any number
of readers when there is no writing. Readers will block until
the writer is done, and thus they see
a fully consistent view of the tree.
The RWMutex approach was the fastest
and easiest to reason about in our
applications without overly complicating
the code base. The SkipLocking flag can
be set to omit all locking if goroutine
coordination is provided by other means,
or unneeded (in the case of single goroutine
only access).

[1] "The Adaptive Radix Tree: ARTful
Indexing for Main-Memory Databases"
by Viktor Leis, Alfons Kemper, Thomas Neumann.
https://db.in.tum.de/~leis/papers/ART.pdf

[2] "The Google File System"
SOSP’03, October 19–22, 2003, Bolton Landing, New York, USA.
by Sanjay Ghemawat, Howard Gobioff, and Shun-Tak Leung.
https://pdos.csail.mit.edu/6.824/papers/gfs.pdf

[3] "How does FoundationDB store keys with duplicate prefixes?"
https://forums.foundationdb.org/t/how-does-foundationdb-store-keys-with-duplicate-prefixes/1234

[4] "Issue #2189: Prefix compress read range results"
https://github.com/apple/foundationdb/issues/2189

[5] "Counted B-Trees"
https://www.chiark.greenend.org.uk/~sgtatham/algorithms/cbtree.html

[6] "Persistent Storage of Adaptive Radix Trees in DuckDB"
https://duckdb.org/2022/07/27/art-storage.html

[7] https://hyper-db.de/  https://tableau.github.io/hyper-db/journey

Docs: https://pkg.go.dev/github.com/glycerine/art-adaptive-radix-tree


Serialization to disk facilities are provided via greenpack (https://github.com/glycerine/greenpack).

The implementation is based on following paper:

- [The Adaptive Radix Tree: ARTful Indexing for Main-Memory Databases](https://db.in.tum.de/~leis/papers/ART.pdf)

-----
Author: Jason E. Aten, Ph.D.

Licence: MIT

Originally based on, but much diverged from, the upstream repo
https://github.com/WenyXu/sync-adaptive-radix-tree . 

In particular, the racey and unfinished optimistic 
locking was removed, many bugs were fixed, and code was
added to support queries (FindGE, FindGTE, FindLE,
FindLTE) based on the ordering of keys in the tree. 
A comprehensive test suite is inclued to verify all operations.

## Benchmarks

The benchmarks are located in [tree_concurrent_test.go](./tree_concurrent_test.go) and [tree_bench_test.go](./tree_bench_test.go).

`frac_x` means `0.x` read fraction. frac_0 means write-only, frac_10 means read-only.

Note that our ART tree, the skip-list, and the red-black tree
provide sorted element access. They provide for range queries
and finding elements greater-than, greater-than-or-equal,
less-than, and less-than-or-equal to a key.

The Go map, sync.Map, and Ctrie provide only unordered
lookups.

```bash

started at Mon 2025 Mar 10 08:49:11

go test -v -run=blah -bench=ReadWrite -benchmem
goos: darwin
goarch: amd64
pkg: github.com/glycerine/art-adaptive-radix-tree
cpu: Intel(R) Core(TM) i7-1068NG7 CPU @ 2.30GHz


// our ART Tree implementation
BenchmarkArtReadWrite
BenchmarkArtReadWrite/frac_0
BenchmarkArtReadWrite/frac_0-8                 	 1717491	       640.9 ns/op	     154 B/op	       4 allocs/op
BenchmarkArtReadWrite/frac_1
BenchmarkArtReadWrite/frac_1-8                 	 1982194	       565.8 ns/op	     137 B/op	       3 allocs/op
BenchmarkArtReadWrite/frac_2
BenchmarkArtReadWrite/frac_2-8                 	 2363079	       516.4 ns/op	     121 B/op	       3 allocs/op
BenchmarkArtReadWrite/frac_3
BenchmarkArtReadWrite/frac_3-8                 	 2552164	       540.9 ns/op	     105 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_4
BenchmarkArtReadWrite/frac_4-8                 	 2971687	       447.1 ns/op	      90 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_5
BenchmarkArtReadWrite/frac_5-8                 	 3663865	       433.1 ns/op	      75 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_6
BenchmarkArtReadWrite/frac_6-8                 	 3957212	       377.5 ns/op	      60 B/op	       1 allocs/op
BenchmarkArtReadWrite/frac_7
BenchmarkArtReadWrite/frac_7-8                 	 4796148	       352.1 ns/op	      45 B/op	       1 allocs/op
BenchmarkArtReadWrite/frac_8
BenchmarkArtReadWrite/frac_8-8                 	 5674362	       374.0 ns/op	      30 B/op	       0 allocs/op
BenchmarkArtReadWrite/frac_9
BenchmarkArtReadWrite/frac_9-8                 	 6275383	       319.9 ns/op	      15 B/op	       0 allocs/op
BenchmarkArtReadWrite/frac_10
BenchmarkArtReadWrite/frac_10-8                	24074565	        49.25 ns/op	       0 B/op	       0 allocs/op


// a skip list
BenchmarkSklReadWrite
BenchmarkSklReadWrite/frac_0
BenchmarkSklReadWrite/frac_0-8                 	 4054360	       402.5 ns/op	      56 B/op	       7 allocs/op
BenchmarkSklReadWrite/frac_1
BenchmarkSklReadWrite/frac_1-8                 	 4879401	       394.9 ns/op	      49 B/op	       6 allocs/op
BenchmarkSklReadWrite/frac_2
BenchmarkSklReadWrite/frac_2-8                 	 4886041	       365.7 ns/op	      43 B/op	       5 allocs/op
BenchmarkSklReadWrite/frac_3
BenchmarkSklReadWrite/frac_3-8                 	 5406398	       342.4 ns/op	      35 B/op	       4 allocs/op
BenchmarkSklReadWrite/frac_4
BenchmarkSklReadWrite/frac_4-8                 	 6880112	       325.2 ns/op	      32 B/op	       4 allocs/op
BenchmarkSklReadWrite/frac_5
BenchmarkSklReadWrite/frac_5-8                 	 7611789	       311.7 ns/op	      26 B/op	       3 allocs/op
BenchmarkSklReadWrite/frac_6
BenchmarkSklReadWrite/frac_6-8                 	 9188644	       289.9 ns/op	      17 B/op	       2 allocs/op
BenchmarkSklReadWrite/frac_7
BenchmarkSklReadWrite/frac_7-8                 	 9963705	       244.9 ns/op	      11 B/op	       1 allocs/op
BenchmarkSklReadWrite/frac_8
BenchmarkSklReadWrite/frac_8-8                 	10654891	       190.5 ns/op	       6 B/op	       0 allocs/op
BenchmarkSklReadWrite/frac_9
BenchmarkSklReadWrite/frac_9-8                 	14126460	       172.0 ns/op	       4 B/op	       0 allocs/op
BenchmarkSklReadWrite/frac_10
BenchmarkSklReadWrite/frac_10-8                	252937036	         4.586 ns/op	       0 B/op	       0 allocs/op


// standard Go map, RWMutex protected.
BenchmarkReadWrite_map_RWMutex_wrapped
BenchmarkReadWrite_map_RWMutex_wrapped/frac_0
BenchmarkReadWrite_map_RWMutex_wrapped/frac_0-8         	 5356084	       265.2 ns/op	      26 B/op	       1 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_1
BenchmarkReadWrite_map_RWMutex_wrapped/frac_1-8         	 6070125	       232.7 ns/op	      23 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_2
BenchmarkReadWrite_map_RWMutex_wrapped/frac_2-8         	 7224242	       218.1 ns/op	      20 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_3
BenchmarkReadWrite_map_RWMutex_wrapped/frac_3-8         	 7557686	       178.9 ns/op	      18 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_4
BenchmarkReadWrite_map_RWMutex_wrapped/frac_4-8         	 8384904	       194.5 ns/op	      16 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_5
BenchmarkReadWrite_map_RWMutex_wrapped/frac_5-8         	 8439982	       184.3 ns/op	      15 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_6
BenchmarkReadWrite_map_RWMutex_wrapped/frac_6-8         	 9379390	       168.8 ns/op	      13 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_7
BenchmarkReadWrite_map_RWMutex_wrapped/frac_7-8         	 9558462	       154.3 ns/op	       7 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_8
BenchmarkReadWrite_map_RWMutex_wrapped/frac_8-8         	 9948544	       161.8 ns/op	       6 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_9
BenchmarkReadWrite_map_RWMutex_wrapped/frac_9-8         	 8952806	       154.4 ns/op	       2 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_10
BenchmarkReadWrite_map_RWMutex_wrapped/frac_10-8        	35561499	        32.25 ns/op	       0 B/op	       0 allocs/op


// standard Go map, no mutex, no parallel benchmark
BenchmarkReadWrite_Map_NoMutex_NoParallel
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_0
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_0-8      	 3550207	       333.7 ns/op	     139 B/op	       1 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_1
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_1-8      	 4386049	       394.0 ns/op	     189 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_2
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_2-8      	 5356921	       346.3 ns/op	     156 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_3
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_3-8      	 5310548	       339.0 ns/op	     129 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_4
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_4-8      	 5802025	       261.4 ns/op	      78 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_5
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_5-8      	 6109118	       239.0 ns/op	      69 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_6
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_6-8      	 8569330	       228.6 ns/op	      51 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_7
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_7-8      	 9681540	       211.4 ns/op	      44 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_8
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_8-8      	11827711	       194.3 ns/op	      35 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_9
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_9-8      	19999615	       174.2 ns/op	      20 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_10
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_10-8     	88869356	        12.41 ns/op	       0 B/op	       0 allocs/op


// our ART tree in a non parallel benchmark
BenchmarkArtReadWrite_NoLocking_NoParallel
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_0
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_0-8     	 1252537	      1029 ns/op	     201 B/op	       4 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_1
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_1-8     	 1435491	       966.5 ns/op	     182 B/op	       3 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_2
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_2-8     	 1620576	       894.5 ns/op	     162 B/op	       3 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_3
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_3-8     	 1722978	       799.4 ns/op	     139 B/op	       3 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_4
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_4-8     	 1909138	       721.1 ns/op	     118 B/op	       2 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_5
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_5-8     	 2270104	       648.4 ns/op	      98 B/op	       2 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_6
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_6-8     	 2586502	       561.3 ns/op	      76 B/op	       1 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_7
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_7-8     	 2848957	       458.0 ns/op	      54 B/op	       1 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_8
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_8-8     	 3826602	       380.5 ns/op	      36 B/op	       0 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_9
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_9-8     	 6270745	       295.5 ns/op	      18 B/op	       0 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_10
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_10-8    	87189405	        12.51 ns/op	       0 B/op	       0 allocs/op


// a red-black tree
BenchmarkReadWrite_RedBlackTree
BenchmarkReadWrite_RedBlackTree/frac_0
BenchmarkReadWrite_RedBlackTree/frac_0-8                	 1000000	      1580 ns/op	      87 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_1
BenchmarkReadWrite_RedBlackTree/frac_1-8                	 1000000	      1439 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_2
BenchmarkReadWrite_RedBlackTree/frac_2-8                	 1000000	      1460 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_3
BenchmarkReadWrite_RedBlackTree/frac_3-8                	 1000000	      1445 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_4
BenchmarkReadWrite_RedBlackTree/frac_4-8                	 1000000	      1481 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_5
BenchmarkReadWrite_RedBlackTree/frac_5-8                	 1000000	      1507 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_6
BenchmarkReadWrite_RedBlackTree/frac_6-8                	 1000000	      1447 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_7
BenchmarkReadWrite_RedBlackTree/frac_7-8                	 1000000	      1452 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_8
BenchmarkReadWrite_RedBlackTree/frac_8-8                	 1000000	      1436 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_9
BenchmarkReadWrite_RedBlackTree/frac_9-8                	 1000000	      1462 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_10
BenchmarkReadWrite_RedBlackTree/frac_10-8               	 1000000	      1452 ns/op	      40 B/op	       2 allocs/op


// standard Go lib sync.Map
BenchmarkReadWriteSyncMap
BenchmarkReadWriteSyncMap/frac_0
BenchmarkReadWriteSyncMap/frac_0-8                      	11691332	       130.6 ns/op	     111 B/op	       5 allocs/op
BenchmarkReadWriteSyncMap/frac_1
BenchmarkReadWriteSyncMap/frac_1-8                      	13385636	       124.7 ns/op	     101 B/op	       4 allocs/op
BenchmarkReadWriteSyncMap/frac_2
BenchmarkReadWriteSyncMap/frac_2-8                      	14851813	       121.7 ns/op	      90 B/op	       4 allocs/op
BenchmarkReadWriteSyncMap/frac_3
BenchmarkReadWriteSyncMap/frac_3-8                      	15685892	       111.9 ns/op	      80 B/op	       3 allocs/op
BenchmarkReadWriteSyncMap/frac_4
BenchmarkReadWriteSyncMap/frac_4-8                      	18599248	       104.1 ns/op	      70 B/op	       3 allocs/op
BenchmarkReadWriteSyncMap/frac_5
BenchmarkReadWriteSyncMap/frac_5-8                      	19203139	        90.68 ns/op	      59 B/op	       3 allocs/op
BenchmarkReadWriteSyncMap/frac_6
BenchmarkReadWriteSyncMap/frac_6-8                      	22553433	        81.18 ns/op	      49 B/op	       2 allocs/op
BenchmarkReadWriteSyncMap/frac_7
BenchmarkReadWriteSyncMap/frac_7-8                      	26940105	        72.88 ns/op	      39 B/op	       2 allocs/op
BenchmarkReadWriteSyncMap/frac_8
BenchmarkReadWriteSyncMap/frac_8-8                      	34950993	        59.05 ns/op	      28 B/op	       1 allocs/op
BenchmarkReadWriteSyncMap/frac_9
BenchmarkReadWriteSyncMap/frac_9-8                      	40551128	        46.06 ns/op	      18 B/op	       1 allocs/op
BenchmarkReadWriteSyncMap/frac_10
BenchmarkReadWriteSyncMap/frac_10-8                     	79313336	        12.80 ns/op	       8 B/op	       1 allocs/op


// Ctrie
BenchmarkReadWriteCtrie
BenchmarkReadWriteCtrie/frac_0
BenchmarkReadWriteCtrie/frac_0-8                        	 4175851	       322.0 ns/op	     340 B/op	       8 allocs/op
BenchmarkReadWriteCtrie/frac_1
BenchmarkReadWriteCtrie/frac_1-8                        	 4776138	       310.6 ns/op	     304 B/op	       7 allocs/op
BenchmarkReadWriteCtrie/frac_2
BenchmarkReadWriteCtrie/frac_2-8                        	 4469605	       278.1 ns/op	     272 B/op	       7 allocs/op
BenchmarkReadWriteCtrie/frac_3
BenchmarkReadWriteCtrie/frac_3-8                        	 5722580	       267.6 ns/op	     257 B/op	       6 allocs/op
BenchmarkReadWriteCtrie/frac_4
BenchmarkReadWriteCtrie/frac_4-8                        	 5225223	       236.2 ns/op	     219 B/op	       6 allocs/op
BenchmarkReadWriteCtrie/frac_5
BenchmarkReadWriteCtrie/frac_5-8                        	 6660027	       233.4 ns/op	     192 B/op	       5 allocs/op
BenchmarkReadWriteCtrie/frac_6
BenchmarkReadWriteCtrie/frac_6-8                        	 6915661	       200.6 ns/op	     166 B/op	       5 allocs/op
BenchmarkReadWriteCtrie/frac_7
BenchmarkReadWriteCtrie/frac_7-8                        	 9343092	       163.1 ns/op	     141 B/op	       4 allocs/op
BenchmarkReadWriteCtrie/frac_8
BenchmarkReadWriteCtrie/frac_8-8                        	11817272	       141.2 ns/op	     113 B/op	       4 allocs/op
BenchmarkReadWriteCtrie/frac_9
BenchmarkReadWriteCtrie/frac_9-8                        	13555288	       108.1 ns/op	      89 B/op	       3 allocs/op
BenchmarkReadWriteCtrie/frac_10
BenchmarkReadWriteCtrie/frac_10-8                       	22852215	        44.80 ns/op	      64 B/op	       3 allocs/op
PASS
ok  	github.com/glycerine/art-adaptive-radix-tree	177.065s

finished at Mon Mar 10 08:52:09
```
