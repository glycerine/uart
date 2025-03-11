# Adaptive Radix Tree (ART) in Go: sorted and speedy

Note: this is a minimal-dependency version of 
my Adaptive Radix Tree (ART) implementation
and it comes without serialization support. 
Thus it is unserialized ART, or uart.

This project provides an implemention
of the Adaptive Radix Tree (ART) data structure[1]. 
As a memory based sorted key/value store and a Order-Statistic tree,
it offers ordered lookups, range queries, and integer based indexing.

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
an `any` value. The ART tree is a trie, and 
provides both path compression
(vertical compression) and variable
sized inner nodes (horizontal compression)
for space-efficient fanout.

Path compression is particularly attractive in
situations where many keys have shared or redundant
prefixes. This is the common case for
many ordered key/value store use cases, such
as database indexes and file-system hierarchies.
The Google File System paper, for example,
mentions the efficiencies obtained
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
Note: red-black tree benchmarks are omittted here
to avoid any 3rd party dependencies.

Ease of use: efficient greater-than/less-than key lookup
and range iteration, as well as the
ability to "treat the tree as a slice" using
integer indexes (based on the counted B-tree
idea -- see the tree.At(i int) method), make this ART tree implementation
particularly easy to use in practice.

The integer indexing makes this ART implementation
also an Order-Statistic tree, much like 
the Counted B-tree[5], so it is ideal for
quickly computing quantiles, medians, and
other statistics of interest. Jumping
forward by 500 keys, for example, is an 
efficient O(log N) time operation for
N keys in the tree. Trie operations are
sometimes described as being O(k) time
where k is the string length of the
key, but I'ved opted for the more familiar
O(log N) description under the assumption, that,
in practice k will approximate log(N).
Path compression means there is an inner
node in the radix trie only where two keys differ, and this
closely resembles an (unbalanced) binary search tree.
ART trees never have to rebalance. See 
the journal paper for a full description[1].

This ART tree supports only a single value for each
key -- it is not a "multi-map" in the C++ sense.
This makes it simple to use and implement. 
Note that the user can store any value, so 
being a unique-key-map is not really a limitation.
The user can simply point to a struct, slice or map
holding the same-key values in the Leaf.Value field.

Concurrency: by default this ART implementation is
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

Licence: MIT

Originally based on, but much diverged from, the upstream repo
https://github.com/WenyXu/sync-adaptive-radix-tree . 

In particular, the racey and unfinished optimistic 
locking was removed, many bugs were fixed, and code was
added to support ordered-key queries (FindGE, FindGTE, FindLE,
FindLTE) and integer index access.
A comprehensive test suite is inclued to verify all operations.

## Benchmarks

For code, see [tree_bench_test.go](./tree_bench_test.go).

`frac_x` means `0.x` read fraction. frac_0 means write-only, frac_10 means read-only.

Note that our ART tree provide sorted element key
query, range queries, and integer indexing based access.

As hash tables, the Go map and sync.Map provide only unordered
exact-match lookups.

```bash

started at Tue 2025 Mar 11 11:14:51

go test -v -run=blah -bench=. -benchmem
goos: darwin
goarch: amd64
pkg: github.com/glycerine/uart
cpu: Intel(R) Core(TM) i7-1068NG7 CPU @ 2.30GHz

// our ART tree, using the default sync.RWMutex.

// The first line represents 100% write/write conflict.
// The last line shows 100% reading.

BenchmarkArtReadWrite
BenchmarkArtReadWrite/frac_0
BenchmarkArtReadWrite/frac_0-8                 	 1000000	      1418 ns/op	     120 B/op	       4 allocs/op
BenchmarkArtReadWrite/frac_1
BenchmarkArtReadWrite/frac_1-8                 	 1000000	      1286 ns/op	     107 B/op	       3 allocs/op
BenchmarkArtReadWrite/frac_2
BenchmarkArtReadWrite/frac_2-8                 	 1000000	      1201 ns/op	      95 B/op	       3 allocs/op
BenchmarkArtReadWrite/frac_3
BenchmarkArtReadWrite/frac_3-8                 	 1000000	      1065 ns/op	      83 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_4
BenchmarkArtReadWrite/frac_4-8                 	 1267842	       940.6 ns/op	      71 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_5
BenchmarkArtReadWrite/frac_5-8                 	 1542188	       865.0 ns/op	      59 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_6
BenchmarkArtReadWrite/frac_6-8                 	 1891710	       705.0 ns/op	      47 B/op	       1 allocs/op
BenchmarkArtReadWrite/frac_7
BenchmarkArtReadWrite/frac_7-8                 	 2465361	       639.1 ns/op	      35 B/op	       1 allocs/op
BenchmarkArtReadWrite/frac_8
BenchmarkArtReadWrite/frac_8-8                 	 3296506	       538.8 ns/op	      23 B/op	       0 allocs/op
BenchmarkArtReadWrite/frac_9
BenchmarkArtReadWrite/frac_9-8                 	 5126754	       363.6 ns/op	      12 B/op	       0 allocs/op
BenchmarkArtReadWrite/frac_10
BenchmarkArtReadWrite/frac_10-8                	23695676	        46.43 ns/op	       0 B/op	       0 allocs/op


// standard Go map wrapped with a sync.RWMutex (no range queries)

BenchmarkReadWrite_map_RWMutex_wrapped
BenchmarkReadWrite_map_RWMutex_wrapped/frac_0
BenchmarkReadWrite_map_RWMutex_wrapped/frac_0-8         	 5315512	       269.8 ns/op	      26 B/op	       1 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_1
BenchmarkReadWrite_map_RWMutex_wrapped/frac_1-8         	 5972695	       255.8 ns/op	      24 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_2
BenchmarkReadWrite_map_RWMutex_wrapped/frac_2-8         	 6723937	       206.8 ns/op	      21 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_3
BenchmarkReadWrite_map_RWMutex_wrapped/frac_3-8         	 7515493	       194.9 ns/op	      19 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_4
BenchmarkReadWrite_map_RWMutex_wrapped/frac_4-8         	 8147653	       179.1 ns/op	      17 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_5
BenchmarkReadWrite_map_RWMutex_wrapped/frac_5-8         	 8596716	       174.9 ns/op	      15 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_6
BenchmarkReadWrite_map_RWMutex_wrapped/frac_6-8         	 9224728	       165.4 ns/op	      13 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_7
BenchmarkReadWrite_map_RWMutex_wrapped/frac_7-8         	 9719365	       151.7 ns/op	       7 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_8
BenchmarkReadWrite_map_RWMutex_wrapped/frac_8-8         	10205706	       148.0 ns/op	       6 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_9
BenchmarkReadWrite_map_RWMutex_wrapped/frac_9-8         	 9078074	       149.2 ns/op	       3 B/op	       0 allocs/op
BenchmarkReadWrite_map_RWMutex_wrapped/frac_10
BenchmarkReadWrite_map_RWMutex_wrapped/frac_10-8        	35471313	        33.72 ns/op	       0 B/op	       0 allocs/op


// standard library sync.Map (no range queries)

BenchmarkReadWriteSyncMap
BenchmarkReadWriteSyncMap/frac_0
BenchmarkReadWriteSyncMap/frac_0-8                      	11585511	       128.9 ns/op	     111 B/op	       5 allocs/op
BenchmarkReadWriteSyncMap/frac_1
BenchmarkReadWriteSyncMap/frac_1-8                      	12977352	       131.1 ns/op	     101 B/op	       4 allocs/op
BenchmarkReadWriteSyncMap/frac_2
BenchmarkReadWriteSyncMap/frac_2-8                      	14838945	       122.2 ns/op	      90 B/op	       4 allocs/op
BenchmarkReadWriteSyncMap/frac_3
BenchmarkReadWriteSyncMap/frac_3-8                      	14907528	       114.5 ns/op	      80 B/op	       3 allocs/op
BenchmarkReadWriteSyncMap/frac_4
BenchmarkReadWriteSyncMap/frac_4-8                      	18078240	       100.8 ns/op	      70 B/op	       3 allocs/op
BenchmarkReadWriteSyncMap/frac_5
BenchmarkReadWriteSyncMap/frac_5-8                      	18791480	        89.52 ns/op	      59 B/op	       3 allocs/op
BenchmarkReadWriteSyncMap/frac_6
BenchmarkReadWriteSyncMap/frac_6-8                      	21172922	        79.55 ns/op	      49 B/op	       2 allocs/op
BenchmarkReadWriteSyncMap/frac_7
BenchmarkReadWriteSyncMap/frac_7-8                      	25711459	        73.04 ns/op	      38 B/op	       2 allocs/op
BenchmarkReadWriteSyncMap/frac_8
BenchmarkReadWriteSyncMap/frac_8-8                      	29925382	        60.51 ns/op	      28 B/op	       1 allocs/op
BenchmarkReadWriteSyncMap/frac_9
BenchmarkReadWriteSyncMap/frac_9-8                      	41599728	        46.34 ns/op	      18 B/op	       1 allocs/op
BenchmarkReadWriteSyncMap/frac_10
BenchmarkReadWriteSyncMap/frac_10-8                     	100000000	        11.96 ns/op	       8 B/op	       1 allocs/op
PASS
ok  	github.com/glycerine/uart	137.808s

finished at Tue 2025 Mar 11 11:17:09
```
