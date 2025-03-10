# the Adaptive radix tree (ART): sorted and speedy

Note: this is a minimal-dependency version of 
the Adaptive Radix Tree (ART) implementation at
https://github.com/glycerine/art-adaptive-radix-tree
and it comes without greenpack serialization support. 
Thus it is unserialized ART, or uart. See the mother project
above for full benchmark/comparison capabilities.

This project provides an implemention
of the Adaptive Radix Tree (ART) data structure[1]. 

Why? In read-heavy situations, ART
trees can be competitive with the built in Go 
map and sync.Maps while _also_ providing
sorted-ordered-key lookups, range queries,
and remaining goroutine safe if writing
does become necessary (see the benchmarks 
below). Modern analytic-oriented databases 
like Tableau and DuckDB leverage ART trees
to implement their indexes because they
provide significant speedups[6].
They were designed in 2013 by the German academic
computer scientists whose HyPer project
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
space savings. Like
those trees, ART offers an ordered index
of sorted keys allowing efficient O(log N) access
for each unique key. However, as the benchmarks
below indicate (1440 nsec per lookup for red-black tree,
versus 53 nsec for ART), lookups can be orders of
magnitude faster (27x in that benchmark, for the read-only case).
As a point of humility, we note that the skip-list
measured here was even faster. Skip
lists do not provide prefix compression,
and have other trade-offs such as using
more memory for multiple forward pointers per node; having
randomized/non-deterministic performance; and worst
case O(N) insertion for "tall towers".
Nonetheless, they may also be worth investigating in your
application context with more than 
this quick and cursory benchmark as a guide.

Ease of use: efficient key-range lookup and iteration, as well as the
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

Docs: https://pkg.go.dev/github.com/glycerine/uart


-----
Author: Jason E. Aten, Ph.D.

Licence: MIT. See the LICENSE file.

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

Started Sun 2025 Mar 09

go test -v -run=blah -bench=. -benchmem
goos: darwin
goarch: amd64
pkg: github.com/glycerine/sync-adaptive-radix-tree
cpu: Intel(R) Core(TM) i7-1068NG7 CPU @ 2.30GHz


// our ART Tree implementation
BenchmarkArtReadWrite
BenchmarkArtReadWrite/frac_0
BenchmarkArtReadWrite/frac_0-8                 	 2265698	       633.2 ns/op	     144 B/op	       4 allocs/op
BenchmarkArtReadWrite/frac_1
BenchmarkArtReadWrite/frac_1-8                 	 2751024	       530.5 ns/op	     128 B/op	       3 allocs/op
BenchmarkArtReadWrite/frac_2
BenchmarkArtReadWrite/frac_2-8                 	 3181084	       451.5 ns/op	     114 B/op	       3 allocs/op
BenchmarkArtReadWrite/frac_3
BenchmarkArtReadWrite/frac_3-8                 	 3615133	       483.4 ns/op	     100 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_4
BenchmarkArtReadWrite/frac_4-8                 	 3769062	       411.0 ns/op	      85 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_5
BenchmarkArtReadWrite/frac_5-8                 	 4164261	       481.5 ns/op	      71 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_6
BenchmarkArtReadWrite/frac_6-8                 	 4822056	       360.1 ns/op	      57 B/op	       1 allocs/op
BenchmarkArtReadWrite/frac_7
BenchmarkArtReadWrite/frac_7-8                 	 5504163	       334.8 ns/op	      42 B/op	       1 allocs/op
BenchmarkArtReadWrite/frac_8
BenchmarkArtReadWrite/frac_8-8                 	 6221266	       345.1 ns/op	      28 B/op	       0 allocs/op
BenchmarkArtReadWrite/frac_9
BenchmarkArtReadWrite/frac_9-8                 	 6432708	       264.0 ns/op	      14 B/op	       0 allocs/op
BenchmarkArtReadWrite/frac_10
BenchmarkArtReadWrite/frac_10-8                	23893329	        52.41 ns/op	       0 B/op	       0 allocs/op


// our ART tree, with SkipLocking = true
BenchmarkArtReadWrite_NoLocking_NoParallel
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_0
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_0-8     	 1745264	       783.1 ns/op	     193 B/op	       4 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_1
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_1-8     	 1829658	       679.1 ns/op	     175 B/op	       3 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_2
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_2-8     	 2113701	       633.2 ns/op	     155 B/op	       3 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_3
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_3-8     	 2412417	       604.9 ns/op	     135 B/op	       3 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_4
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_4-8     	 2581794	       558.0 ns/op	     117 B/op	       2 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_5
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_5-8     	 2777422	       519.4 ns/op	      98 B/op	       2 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_6
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_6-8     	 3178274	       459.1 ns/op	      77 B/op	       1 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_7
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_7-8     	 3702620	       431.8 ns/op	      56 B/op	       1 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_8
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_8-8     	 4608860	       391.5 ns/op	      35 B/op	       0 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_9
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_9-8     	 8575534	       343.0 ns/op	      17 B/op	       0 allocs/op
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_10
BenchmarkArtReadWrite_NoLocking_NoParallel/frac_10-8    	88373481	        12.57 ns/op	       0 B/op	       0 allocs/op


// a skip list
BenchmarkSklReadWrite
BenchmarkSklReadWrite/frac_0
BenchmarkSklReadWrite/frac_0-8                 	 4144185	       369.8 ns/op	      46 B/op	       5 allocs/op
BenchmarkSklReadWrite/frac_1
BenchmarkSklReadWrite/frac_1-8                 	 4991278	       383.9 ns/op	      48 B/op	       6 allocs/op
BenchmarkSklReadWrite/frac_2
BenchmarkSklReadWrite/frac_2-8                 	 4811062	       362.7 ns/op	      42 B/op	       5 allocs/op
BenchmarkSklReadWrite/frac_3
BenchmarkSklReadWrite/frac_3-8                 	 5064134	       339.6 ns/op	      36 B/op	       4 allocs/op
BenchmarkSklReadWrite/frac_4
BenchmarkSklReadWrite/frac_4-8                 	 5994170	       328.8 ns/op	      30 B/op	       3 allocs/op
BenchmarkSklReadWrite/frac_5
BenchmarkSklReadWrite/frac_5-8                 	 6382962	       304.3 ns/op	      24 B/op	       3 allocs/op
BenchmarkSklReadWrite/frac_6
BenchmarkSklReadWrite/frac_6-8                 	 7161585	       281.7 ns/op	      17 B/op	       2 allocs/op
BenchmarkSklReadWrite/frac_7
BenchmarkSklReadWrite/frac_7-8                 	 8906937	       235.5 ns/op	      11 B/op	       1 allocs/op
BenchmarkSklReadWrite/frac_8
BenchmarkSklReadWrite/frac_8-8                 	10459196	       224.9 ns/op	       8 B/op	       1 allocs/op
BenchmarkSklReadWrite/frac_9
BenchmarkSklReadWrite/frac_9-8                 	11053722	       156.0 ns/op	       2 B/op	       0 allocs/op
BenchmarkSklReadWrite/frac_10
BenchmarkSklReadWrite/frac_10-8                	223748109	         5.216 ns/op	       0 B/op	       0 allocs/op


// standard Go map, wrapped with a sync.RWMutex
BenchmarkReadWrite_map_RWMutex_wrapped
BenchmarkReadWrite_map_RWMutex_wrapped/frac_0
BenchmarkReadWrite_map_RWMutex_wrapped/frac_0-8         	 4166245	       317.0 ns/op	      32 B/op	       1 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_1
BenchmarkReadWrite_map_RWMutex_wrapped/frac_1-8         	 5437664	       258.7 ns/op	      25 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_2
BenchmarkReadWrite_map_RWMutex_wrapped/frac_2-8         	 6511797	       230.7 ns/op	      21 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_3
BenchmarkReadWrite_map_RWMutex_wrapped/frac_3-8         	 7210330	       198.4 ns/op	      19 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_4
BenchmarkReadWrite_map_RWMutex_wrapped/frac_4-8         	 7947590	       185.6 ns/op	      17 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_5
BenchmarkReadWrite_map_RWMutex_wrapped/frac_5-8         	 8584722	       181.2 ns/op	      15 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_6
BenchmarkReadWrite_map_RWMutex_wrapped/frac_6-8         	 9268635	       171.5 ns/op	      14 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_7
BenchmarkReadWrite_map_RWMutex_wrapped/frac_7-8         	 9587798	       153.8 ns/op	       7 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_8
BenchmarkReadWrite_map_RWMutex_wrapped/frac_8-8         	 9703442	       160.0 ns/op	       6 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_9
BenchmarkReadWrite_map_RWMutex_wrapped/frac_9-8         	 8902550	       152.1 ns/op	       2 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_10
BenchmarkReadWrite_map_RWMutex_wrapped/frac_10-8        	35992018	        32.58 ns/op	       0 B/op	       0 allocs/op


// standard Go map
BenchmarkReadWrite_Map_NoMutex_NoParallel
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_0
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_0-8      	 3553071	       331.6 ns/op	     139 B/op	       1 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_1
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_1-8      	 4549849	       395.9 ns/op	     184 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_2
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_2-8      	 5306785	       349.1 ns/op	     158 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_3
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_3-8      	 5213368	       303.6 ns/op	     115 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_4
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_4-8      	 5108922	       282.0 ns/op	      83 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_5
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_5-8      	 5929446	       246.4 ns/op	      71 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_6
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_6-8      	 8627586	       239.8 ns/op	      51 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_7
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_7-8      	 9746292	       214.2 ns/op	      43 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_8
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_8-8      	14679614	       199.6 ns/op	      29 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_9
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_9-8      	21443563	       181.7 ns/op	      19 B/op	       0 allocs/op
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_10
BenchmarkReadWrite_Map_NoMutex_NoParallel/frac_10-8     	86117709	        12.53 ns/op	       0 B/op	       0 allocs/op


// a red-black tree
BenchmarkReadWrite_RedBlackTree
BenchmarkReadWrite_RedBlackTree/frac_0
BenchmarkReadWrite_RedBlackTree/frac_0-8                	 1000000	      1589 ns/op	      87 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_1
BenchmarkReadWrite_RedBlackTree/frac_1-8                	 1000000	      1481 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_2
BenchmarkReadWrite_RedBlackTree/frac_2-8                	 1000000	      1499 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_3
BenchmarkReadWrite_RedBlackTree/frac_3-8                	 1000000	      1503 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_4
BenchmarkReadWrite_RedBlackTree/frac_4-8                	 1000000	      1453 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_5
BenchmarkReadWrite_RedBlackTree/frac_5-8                	 1000000	      1455 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_6
BenchmarkReadWrite_RedBlackTree/frac_6-8                	 1000000	      1471 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_7
BenchmarkReadWrite_RedBlackTree/frac_7-8                	 1000000	      1461 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_8
BenchmarkReadWrite_RedBlackTree/frac_8-8                	 1000000	      1440 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_9
BenchmarkReadWrite_RedBlackTree/frac_9-8                	 1000000	      1428 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_10
BenchmarkReadWrite_RedBlackTree/frac_10-8               	 1000000	      1440 ns/op	      40 B/op	       2 allocs/op


// standard Go sync.Map
BenchmarkReadWriteSyncMap
BenchmarkReadWriteSyncMap/frac_0
BenchmarkReadWriteSyncMap/frac_0-8                      	10100846	       160.1 ns/op	     111 B/op	       5 allocs/op
BenchmarkReadWriteSyncMap/frac_1
BenchmarkReadWriteSyncMap/frac_1-8                      	12105478	       148.8 ns/op	     101 B/op	       4 allocs/op
BenchmarkReadWriteSyncMap/frac_2
BenchmarkReadWriteSyncMap/frac_2-8                      	13022816	       136.3 ns/op	      90 B/op	       4 allocs/op
BenchmarkReadWriteSyncMap/frac_3
BenchmarkReadWriteSyncMap/frac_3-8                      	15898324	       124.7 ns/op	      80 B/op	       3 allocs/op
BenchmarkReadWriteSyncMap/frac_4
BenchmarkReadWriteSyncMap/frac_4-8                      	17262078	       117.8 ns/op	      70 B/op	       3 allocs/op
BenchmarkReadWriteSyncMap/frac_5
BenchmarkReadWriteSyncMap/frac_5-8                      	16596974	       105.2 ns/op	      59 B/op	       3 allocs/op
BenchmarkReadWriteSyncMap/frac_6
BenchmarkReadWriteSyncMap/frac_6-8                      	20768742	        90.00 ns/op	      49 B/op	       2 allocs/op
BenchmarkReadWriteSyncMap/frac_7
BenchmarkReadWriteSyncMap/frac_7-8                      	26203155	        75.80 ns/op	      38 B/op	       2 allocs/op
BenchmarkReadWriteSyncMap/frac_8
BenchmarkReadWriteSyncMap/frac_8-8                      	27639141	        69.48 ns/op	      28 B/op	       1 allocs/op
BenchmarkReadWriteSyncMap/frac_9
BenchmarkReadWriteSyncMap/frac_9-8                      	36688646	        52.62 ns/op	      18 B/op	       1 allocs/op
BenchmarkReadWriteSyncMap/frac_10
BenchmarkReadWriteSyncMap/frac_10-8                     	100000000	        12.32 ns/op	       8 B/op	       1 allocs/op


// Ctrie
BenchmarkReadWriteCtrie
BenchmarkReadWriteCtrie/frac_0
BenchmarkReadWriteCtrie/frac_0-8                        	 3483632	       369.5 ns/op	     321 B/op	       8 allocs/op
BenchmarkReadWriteCtrie/frac_1
BenchmarkReadWriteCtrie/frac_1-8                        	 4331516	       338.2 ns/op	     299 B/op	       7 allocs/op
BenchmarkReadWriteCtrie/frac_2
BenchmarkReadWriteCtrie/frac_2-8                        	 4819260	       287.0 ns/op	     277 B/op	       7 allocs/op
BenchmarkReadWriteCtrie/frac_3
BenchmarkReadWriteCtrie/frac_3-8                        	 5175825	       278.8 ns/op	     243 B/op	       6 allocs/op
BenchmarkReadWriteCtrie/frac_4
BenchmarkReadWriteCtrie/frac_4-8                        	 5936182	       238.2 ns/op	     221 B/op	       6 allocs/op
BenchmarkReadWriteCtrie/frac_5
BenchmarkReadWriteCtrie/frac_5-8                        	 6718149	       216.7 ns/op	     194 B/op	       5 allocs/op
BenchmarkReadWriteCtrie/frac_6
BenchmarkReadWriteCtrie/frac_6-8                        	 7688214	       217.2 ns/op	     165 B/op	       5 allocs/op
BenchmarkReadWriteCtrie/frac_7
BenchmarkReadWriteCtrie/frac_7-8                        	 7845292	       192.2 ns/op	     137 B/op	       4 allocs/op
BenchmarkReadWriteCtrie/frac_8
BenchmarkReadWriteCtrie/frac_8-8                        	10784320	       146.7 ns/op	     113 B/op	       4 allocs/op
BenchmarkReadWriteCtrie/frac_9
BenchmarkReadWriteCtrie/frac_9-8                        	11528384	       120.9 ns/op	      89 B/op	       3 allocs/op
BenchmarkReadWriteCtrie/frac_10
BenchmarkReadWriteCtrie/frac_10-8                       	21159684	        51.13 ns/op	      64 B/op	       3 allocs/op

finished at Sun 2025 Mar 09
```

