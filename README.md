# ART Trees in Go: an enhanced radix trie with path compression

* is also an Order-Statistics tree

This means you can access your dictionary 
like a Go slice, with integer indexes.

### See ["Memory Use"](#memory-use) and ["Benchmarks"](#benchmarks) below for a comparison to in-memory B-trees

(In memory B-trees like https://github.com/google/btree or 
https://github.com/tidwall/btree are more space and cache
efficient if you do alot of sequential-key queries).

## overview

Naming? This is a minimal-dependency version of 
my Adaptive Radix Tree (ART) implementation
and it comes without serialization support. 
Thus it is unserialized ART, or uart.

What exactly? This project provides an enhanced implemention
of the Adaptive Radix Tree (ART) data structure[1]. 
It is both a memory-based sorted key/value store and
an Order-Statistic tree. It offers ordered lookups,
range queries, and integer based indexing.

Why? In read-mostly situations, ART
trees can have very good performance 
(e.g. unlocked reads are 35 ns/op for ART vs 10 ns/op for a standard Go
map on a 10M key dictionary) while _also_ providing sorted order lookups
(e.g. find the next key greater-than this one) and range queries,
things that hash tables cannot do.

Who else? Modern analytics-oriented databases 
like Tableau and DuckDB leverage ART trees
to implement their indexes because of
their read heavy workloads[6].
ART trees are a radix tree with variable
sized inner nodes. They were designed 
in 2013 by the German computer scientists 
whose HyPer database became the backend engine
for Tableau[7].

Here is a nice slide deck summarizing the
paper.

https://bu-disc.github.io/CS561-Spring2023/slides/CAS-CS561-Class12.pdf


* more detail

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

Ease of use: efficient greater-than/less-than key lookup
and range iteration, as well as the
ability to "treat the tree as a slice" using
integer indexes (based on the counted B-tree
idea -- see the tree.At(i int) method), make this ART tree implementation
particularly easy to use in practice. 

A practical feature is that it is safe to 
delete from the tree, or insert into it, during
iteration. The iterator will simply resume
at the next available key beyond the previously
returned key. Reverse iteration and prefix-only
scanning are both supported. Each iterator
Next() call is an efficient O(log N).
A complete pass through the tree, even with 
inter-leaved deletes, is still only O(N log N).

The integer indexing makes this ART implementation
also an Order-Statistic tree, much like 
the Counted B-tree[5], so it is ideal for
quickly computing quantiles, medians, and
other statistics of interest. Jumping
forward by 500 keys, for example, is an 
efficient O(log N) time operation for
N keys in the tree. 

Trie operations are sometimes described as being O(k) time
where k is the string length of the
key. That may be technically more correct,
but I'ved opted for the more familiar
O(log N) description under the assumption that,
in practice, k will approximate log(N).
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
The SkipLocking flag can be set to omit 
all locking if goroutine coordination 
is provided by other means, or unneeded 
(in the case of single goroutine only access). 

Iterators are available. Be aware
they do no locking of their own, much
like the built-in Go map.

Full package docs: https://pkg.go.dev/github.com/glycerine/uart


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

## Memory use

I measured memory using the runtime.MemStats.HeapAlloc 
counter on four simple programs that did nothing 
else besides load the dictionary with one, two, or 
three copies of the 93790 linux kernel source tree 
paths in assets/linux.txt. The code for these simple 
programs is in the mem/ subdirectory.

So that we might get to see the benefits of path 
compression -- to see what difference that can make -- 
the second and third loads were exactly the same 
paths, but with __J appended to each,
where J is the additional copy number.

~~~
go run map.go     # the built-in Go map (go1.24 Swiss tables based)
mstat.HeapAlloc = '21_518_728' (copies = 1; diff = 21_518_728 bytes)
mstat.HeapAlloc = '37_891_552' (copies = 2; diff = 16_372_824 bytes)
mstat.HeapAlloc = '55_186_712' (copies = 3; diff = 17_295_160 bytes)

go run art.go     # our ART tree
mstat.HeapAlloc = '34_789_632' (copies = 1; diff = 34_789_632 bytes)
mstat.HeapAlloc = '70_789_824' (copies = 2; diff = 36_000_192 bytes)
mstat.HeapAlloc = '101_813_560' (copies = 3; diff = 31_023_736 bytes)

go run rbtree.go  # a red-black tree
mstat.HeapAlloc = '22_911_096' (copies = 1; diff = 22_911_096 bytes)
mstat.HeapAlloc = '42_296_912' (copies = 2; diff = 19_385_816 bytes)
mstat.HeapAlloc = '55_081_880' (copies = 3; diff = 12_784_968 bytes)

go run googbtree.go  # github.com/google/btree degree=30 b-tree
mstat.HeapAlloc = '18_536_872' (copies = 1; diff = 18_536_872 bytes)
mstat.HeapAlloc = '30_895_664' (copies = 2; diff = 12_358_792 bytes)
mstat.HeapAlloc = '37_989_176' (copies = 3; diff = 7_093_512 bytes)
~~~

Conclusions: the Go map and the red-black tree use about the
same amount of memory. The ART tree uses about 2x the memory of those.

The btree consumes much less memory than the map or red-black tree.
ART seems memory hungry in comparison, using 2x to 3x or more 
memory compared to the btree. This is due to the extra internal nodes
and leaf nodes. The incrementally more leaf and internal nodes can
consume about the same memory as the more compact btree itself,
resulting in 3 fold the memory consumption of the much more
compact btree. This is a pretty serious drawback to ART trees.
For random access (instead of sequential) reads, ART can 
be slightly faster than even a well tuned btree, but only 
about 15% faster. That speed hardly makes up for 3x more memory
to my thinking.

Path/prefix compression too really seems like a wash on this data set.
The second copy actually consumed more memory (3.5% more) than the
inital set plus the baseline of memory for the runtime. 
The third copy consumed about 14% less than 
either of the first two. Prefix compression is heavily 
data dependent, of course. The longest available 
shared prefix in that data set was only
48 bytes, and occurred only twice. The most frequent
compressed out prefix was only 1 byte long, and
had 103094 instances. I added the `tree.CompressedStats()`
method to allow you to analyze your own data sets.
Here is the output for the linux.txt paths:

~~~
compressed stats: 'map[int]int{0:42130, 1:103094, 2:7770, 3:9886, 4:11357, 5:10560, 6:10469, 7:10634, 8:6340, 9:4769, 10:3635, 11:3080, 12:2293, 13:1839, 14:1444, 15:1277, 16:990, 17:782, 18:628, 19:474, 20:346, 21:283, 22:255, 23:168, 24:143, 25:89, 26:92, 27:58, 28:38, 29:27, 30:26, 31:18, 32:8, 33:19, 34:9, 35:4, 36:6, 37:3, 38:3, 39:1, 40:2, 41:2, 44:1, 46:2, 48:2}'
~~~

The total bytes saved through prefix compression 
here was 12_628_922 bytes (12MB). Which is to say,
the ART total of 101_813_560 bytes (97MB) would have
been at least 12MB larger (about 12% larger) and
probably much, much larger with all the extra
inner nodes, without the prefix compression.

While your mileage may very because you have
a alot of shared sub-strings in your key space
(remember that ART will compress "prefixes"
in the middle of a key as well as at the
beginning), my take-home summary here 
is that I don't think the prefix 
compression feature of ART should be a big
deciding factor. 

ART trees are about 2-5x as fast as the red-black tree
used in my measurements (depending on the read/write mix),
so in a sense this is a straight time-for-space 
trade-off: 2/3x as fast, for 2/3x the memory use versus
the red-black tree. However the in-memory btree does so 
much better than the red-black tree; it is the real competition to ART.

A note about reading keys sequentially, the
"full-table" scan case:

Without synchronization, a degree 32 b-tree 
github.com/google/btree, when reading sequential 
values in-order (its sweet spot), kicks ART's bootie 
to the curb, in both time and space. 

The google/btree reads are 2x faster than the Go map Swiss 
table, and 9x faster than my ART. Writes are 15% faster
than the Go map, and 75% faster than my ART. Measurements below.
Code in mem/googbtree.go and commented in tree_test.go Test620 (on branch).
If no deletions are needed, the btree
with bigger degree (say degree 3000) performs even better
than the degree 32 btree, with the understanding that
deletes are then 6x slower due to the large copies involved.

Still without synchronization: for random access, 
this ART is slightly faster than the btree on reads,
and ART is slightly faster than the btree on writes (inserts). 
So the access pattern matters a great deal. 
For random writes and reads (single goroutine, 
unsynchronized access), this ART is only a
little slower than the built-in Go map (which
does not provide next-key-greater-than queries).
You can use the tests in tree_bench_test.go
to measure performance on your own data
and access patterns.

In short, this ART is faster than the sync.Map in many
cases, and competitive with the built-in Go map,
and offers a sorted dictionary and fast
order statstics. Nonetheless, if sequential
full read of all keys (a full table scan) is common,
an in-memory btree will save you
a ton of time and space, and should be preferred
to the ART tree. If your key access is random,
you have memory to spare, and you absolutely 
need the last ounce of speed, the ART tree may
be the marginally faster choice.

To go deeper into the rationale as to why
in-memory B-trees do so well:  The article here 
http://google-opensource.blogspot.com/2013/01/c-containers-that-save-memory-and-time.html
says

> For small data types, B-tree containers 
> typically reduce memory use by 50 to 80% 
> compared with Red-Black tree containers.

and

> Storing multiple elements per node can also 
> improve performance, especially in large containers 
> with inexpensive key-compare functions (e.g., integer 
> keys using less-than), relative to Red-Black tree 
> containers. This is because performance in these 
> cases is effectively governed by the number of 
> cache misses, not the number of key-compare 
> operations. For large data sets, using these 
> B-tree containers will save memory and improve performance.

In the sequential read/full table scan, the btree has
most reads cached from the last read, and so suffers
very few L1 cache misses. 

## Benchmarks

For code, see [tree_bench_test.go](./tree_bench_test.go).

`frac_x` means `0.x` read fraction. frac_0 means write-only, frac_10 means read-only.

Note that our ART tree provide sorted element key
query, range queries, and integer indexing based access.

As hash tables, the Go map and sync.Map provide only unordered
exact-match lookups.

```bash

Unlocked apples-to-apples versus the Go map and google/btree:

(To take synchronization overhead out of the picture.)

=== RUN   Test620_unlocked_read_comparison
map time to store 10_000_000 keys: 2.466118634s (246ns/op)
map reads 10_000_000 keys: elapsed 101.076718ms (10ns/op)
map deletes 10_000_000 keys: elapsed 1.36421433s (136ns/op)

uart.Tree time to store 10_000_000 keys: 3.665080254s (366ns/op)
Ascend(tree) reads 10_000_000 keys: elapsed 368.400458ms (36ns/op)
uart Iter() reads 10_000_000 keys: elapsed 354.294422ms (35ns/op)
tree.At(i) reads 10_000_000 keys: elapsed 381.973196ms (38ns/op)
tree.At(i) reads from 10: 9999990 keys: elapsed 381.521877ms (38ns/op)
my ART: delete 10000000 keys: elapsed 1.059195931s (105ns/op)

Notice: as Atfar does not use an iterator to cache 
sequential At calls, it is 6x slower than the iterator
or sequential At calls. This was the motivation for
adding the Tree.atCache mechanism to speed up 
sequential At(i), At(i+1), At(i+2), ... calls.

tree.Atfar(i) reads 10_000_000 keys: elapsed 2.431009745s (243ns/op)

// degree 32 b-tree github.com/google/btree
//
// (Code kept on a branch to keep zero dependencies).
// See https://github.com/glycerine/uart/tree/bench_goog_btree
//
google/btree time to store 10_000_000 keys: 2.097327599s (209ns/op)
google/btree read all keys sequentially: elapsed 49.415972ms (4ns/op)
google/btree delete all keys: elapsed 2.024685985s (202ns/op)

--- PASS: Test620_unlocked_read_comparison (12.14s)


started at Tue 2025 Mar 12 18:46:28

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
BenchmarkArtReadWrite/frac_0-8                 	 2639517	       530.6 ns/op	     118 B/op	       4 allocs/op
BenchmarkArtReadWrite/frac_1
BenchmarkArtReadWrite/frac_1-8                 	 1934983	       748.4 ns/op	     106 B/op	       3 allocs/op
BenchmarkArtReadWrite/frac_2
BenchmarkArtReadWrite/frac_2-8                 	 2144061	       687.2 ns/op	      94 B/op	       3 allocs/op
BenchmarkArtReadWrite/frac_3
BenchmarkArtReadWrite/frac_3-8                 	 2418021	       627.3 ns/op	      83 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_4
BenchmarkArtReadWrite/frac_4-8                 	 2569453	       601.9 ns/op	      71 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_5
BenchmarkArtReadWrite/frac_5-8                 	 2924372	       565.7 ns/op	      59 B/op	       2 allocs/op
BenchmarkArtReadWrite/frac_6
BenchmarkArtReadWrite/frac_6-8                 	 3415107	       493.2 ns/op	      47 B/op	       1 allocs/op
BenchmarkArtReadWrite/frac_7
BenchmarkArtReadWrite/frac_7-8                 	 3958209	       432.9 ns/op	      35 B/op	       1 allocs/op
BenchmarkArtReadWrite/frac_8
BenchmarkArtReadWrite/frac_8-8                 	 4604090	       372.3 ns/op	      23 B/op	       0 allocs/op
BenchmarkArtReadWrite/frac_9
BenchmarkArtReadWrite/frac_9-8                 	 4809241	       295.1 ns/op	      11 B/op	       0 allocs/op
BenchmarkArtReadWrite/frac_10
BenchmarkArtReadWrite/frac_10-8                	23420194	        49.64 ns/op	       0 B/op	       0 allocs/op


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


// a red-black tree; "github.com/glycerine/rbtree"

BenchmarkReadWrite_RedBlackTree
BenchmarkReadWrite_RedBlackTree/frac_0
BenchmarkReadWrite_RedBlackTree/frac_0-8                	 1000000	      1557 ns/op	      87 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_1
BenchmarkReadWrite_RedBlackTree/frac_1-8                	 1000000	      1521 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_2
BenchmarkReadWrite_RedBlackTree/frac_2-8                	 1000000	      1554 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_3
BenchmarkReadWrite_RedBlackTree/frac_3-8                	 1000000	      1485 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_4
BenchmarkReadWrite_RedBlackTree/frac_4-8                	 1000000	      1476 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_5
BenchmarkReadWrite_RedBlackTree/frac_5-8                	 1000000	      1548 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_6
BenchmarkReadWrite_RedBlackTree/frac_6-8                	 1000000	      1524 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_7
BenchmarkReadWrite_RedBlackTree/frac_7-8                	 1000000	      1638 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_8
BenchmarkReadWrite_RedBlackTree/frac_8-8                	 1000000	      1543 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_9
BenchmarkReadWrite_RedBlackTree/frac_9-8                	 1000000	      1548 ns/op	      40 B/op	       2 allocs/op
BenchmarkReadWrite_RedBlackTree/frac_10
BenchmarkReadWrite_RedBlackTree/frac_10-8               	 1000000	      1467 ns/op	      40 B/op	       2 allocs/op


PASS
ok  	github.com/glycerine/uart	137.808s

finished at Wed 2025 Mar 12 18:47:16
```
