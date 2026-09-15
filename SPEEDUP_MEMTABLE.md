# Memtable Build And Scan Speedup Report

## Goal

The target workload is the memtable-shaped path:

1. Insert randomly distributed keys.
2. Read the entire table back once in sorted order.

The primary score is the sum of build time plus one full readback, reported as
`ns/key`. The important public APIs are the normal APIs: `Insert`, `Iter`,
`Ascend`, `Scan`, and `ScanLeaves`. Sorted-input-only paths are useful for
separate bulk-load cases, but they are not the benchmark target here.

The benchmark used for the main optimization loop was:

```sh
go test -run '^$' -bench '^BenchmarkMemtableBuildScanPermutedUint64_100K$' -benchtime=50x -count=3 -benchmem
```

This uses `memtablePermutedUint64Keys(100_000)`: fixed-width 8-byte keys in a
deterministic pseudo-random permutation. `ns/key` includes tree construction and
one full scan. Unless otherwise stated, numbers below are from an AMD Ryzen
Threadripper 3960X with the Go toolchain in this checkout.

Benchmarks were noisy because ART allocates pointer-rich arena chunks and Go GC
timing changes run to run. For repeated measurements, the report uses median
`ns/key`. Early single-run checkpoints are marked as such.

## Starting Point

Initial random-input baseline, before this optimization pass:

| Case                        | ns/key     | B/op       | allocs/op 
| ----------------------------|------------|------------|-----------
| `uart_insert_iter`          | 552.1      | 15,699,270 | 270,554 
| `uart_insert_scan`          | 339.0      | 15,528,470 | 100,137 
| `uart_insert_nocopy_scan`   | 282.7      | 14,728,460 | 137
| `google_btree_degree_32`    | 301.7      | 1,777,360  | 6,437
| `google_btree_degree_3000`  | 592.2      | 2,673,106  | 140

At this point default `Insert+ScanLeaves` was still slower than
`google/btree` degree 32, and default `Insert+Iter` was much slower.

## Final Current-Code Result

Final current-code run after reverting the non-winning `bytes.Equal` trial:

| Case                       | Runs, ns/key        | Median ns/key | B/op       | allocs/op
| ---------------------------|---------------------|---------------|------------|----
| `uart_insert_iter`         | 255.5, 255.9, 257.2 | 255.9         | 14,876,250 | 141
| `uart_insert_scan`         | 249.4, 249.0, 259.3 | 249.4         | 14,073,184 | 139
| `uart_insert_nocopy_scan`  | 243.0, 234.5, 238.0 | 238.0         | 13,024,584 | 137
| `google_btree_degree_32`   | 292.8, 297.8, 296.4 | 296.4         | 1,777,364  | 6,437
| `google_btree_degree_3000` | 569.4, 568.2, 560.1 | 568.2         | 2,673,108  | 140

Overall improvement versus the starting point:

| Case                     | Before       | After        | Improvement 
| -------------------------| -------------| -------------| -------------------
| `Insert+Iter(nil,nil)`   | 552.1 ns/key | 255.9 ns/key | 296.2 ns/key, 53.6%
| `Insert+ScanLeaves`      | 339.0 ns/key | 249.4 ns/key |  89.6 ns/key, 26.4%
| `InsertNoCopy+ScanLeaves`| 282.7 ns/key | 238.0 ns/key |  44.7 ns/key, 15.8%

Final default `Insert+Iter(nil,nil)` is 40.5 ns/key faster than
`google_btree_degree_32` on the same random build plus full-scan benchmark.

Final default `Insert+ScanLeaves` is 47.0 ns/key faster than
`google_btree_degree_32`.

## Retained Optimizations

### 1. Arena-Owned Key Copies For `Insert`

Files: `arena.go`, `tree.go`

Before this pass, `Insert` copied every key with `append([]byte{}, key...)`.
That made default `Insert` pay one heap allocation per key. The retained change
adds a `keyArena` and copies inserted key bytes into 1 MiB chunks.

This preserves the public safety contract of `Insert`: callers can still reuse
or mutate their original key slice after insertion. It also lets compressed ART
prefixes point directly into tree-owned key bytes.

Single-run effect after this step:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_scan` | 339.0 ns/key | 307.1 ns/key | 31.9 ns/key, 9.4% |
| `uart_insert_iter` | 552.1 ns/key | 536.0 ns/key | 16.1 ns/key, 2.9% |

Allocation effect:

| Case | Before allocs/op | After allocs/op |
| --- | ---: | ---: |
| `uart_insert_scan` | 100,137 | 139 |
| `uart_insert_iter` | 270,554 | 170,556 |

The iterator path was still slow because it was dominated by per-step iterator
machinery. That led to the next optimization.

### 2. Prefix Sharing Became Safe For Normal Inserts

Files: `arena.go`, `tree.go`

With arena-owned key bytes, compressed path fragments no longer need defensive
copies in normal `Insert`. `prefixBytes` now returns a slice of tree-owned key
storage, and `SharePrefixBytes` is kept only as a compatibility field.

The measured benefit was bundled with the key arena change above, because the
two changes are logically coupled: sharing prefixes is only safe for `Insert`
after the tree owns the copied key bytes. A later branch-removal run also showed
an iterator-path improvement of about 7 ns/key, though that isolated number was
within normal GC noise.

### 3. Full-Range Iterator Prefetch

Files: `iterator.go`, `tree.go`

`Iter(nil,nil)` is now recognized as the full-table scan case. It eagerly builds
or reuses an ordered `[]*Leaf` cache and then `Next` walks that slice. The cache
is invalidated on insert and delete. If a mutation happens during iteration, the
iterator drops the prefetched slice and resumes through the existing
tree-version lookup path.

This made the ordinary iterator API fast without asking the caller to pick a
separate "fast API" ahead of time.

Single-run effect of basic prefetch:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 536.0 ns/key | 317.1 ns/key | 218.9 ns/key, 40.8% |

Allocation effect:

| Metric | Before | After |
| --- | ---: | ---: |
| allocs/op | 170,556 | 141 |
| B/op | 15,947,787 | 16,580,186 |

The B/op increase is the ordered leaf-ref slice. That allocation is intentional:
it trades one dense pointer slice for much faster repeated `Next` calls. Since
the benchmark times iterator construction, this is not a hidden cost.

### 4. Fixed-Size Leaf-Ref Fill For Prefetch

Files: `iterator.go`

The first prefetch implementation used the generic recursive `scan` callback
and appended into the cache slice. That worked but still paid callback and
append checks during cache construction. It was replaced by `fillLeafRefs`,
which allocates the final-size slice and fills by index.

Single-run effect:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 317.1 ns/key | 309.8 ns/key | 7.3 ns/key, 2.3% |

The later `nextPrefetched` cleanup, which avoids copying `Key` and `Value` into
iterator fields on every prefetched step, moved the repeated median for this
phase to roughly 304 ns/key.

### 5. `Ascend(tree,nil,nil)` Uses The Same Full-Range Cache

Files: `iterator.go`

`Ascend` is also an ordinary scan API, so the full-range case now uses the same
ordered-leaf cache as `Iter(nil,nil)`. Ranged `Ascend` still uses the normal
versioned iterator.

The first attempt routed full-range `Ascend` through recursive `ScanLeaves`;
`Test620` showed that was worse in the `iter.Seq2` wrapper. That attempt was
rejected. The retained version uses the cache slice directly.

Diagnostic `Test620` output after the change, with `Iter` run first and warming
the cache:

| Case | Read time |
| --- | ---: |
| `Ascend(tree,nil,nil)` | 8 ns/op |
| `google/btree` full scan | 9 ns/op |

This is a scan-only diagnostic, not the primary build-plus-scan benchmark.

### 6. Shrink `bnode.pren` To `uint32`

Files: `node.go`, `gte.go`, `lte.go`, `inner.go`, `n4.go`, `n16.go`,
`n48.go`, `n256.go`, `tree_test.go`

`bnode.pren` stores the count of leaves before a child within its parent. It was
an `int`, which padded `bnode` to a larger amd64 layout. Changing it to `uint32`
reduced the pointer-scanned heap footprint and lowered GC work.

Median effect:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 304.0 ns/key | 289.7 ns/key | 14.3 ns/key, 4.7% |
| `uart_insert_scan` | 305.3 ns/key | 293.6 ns/key | 11.7 ns/key, 3.8% |

Memory effect for `uart_insert_iter`:

| Before B/op | After B/op | Saved |
| ---: | ---: | ---: |
| 16,580,182 | 15,466,068 | 1,114,114 B/op |

### 7. Pack `inner`

Files: `node.go`

The `inner` struct was reordered so the slice and interface fields sit before
small scalar fields. This removed padding without changing behavior.

Median effect:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 289.7 ns/key | 283.2 ns/key | 6.5 ns/key, 2.2% |
| `uart_insert_scan` | 293.6 ns/key | 288.1 ns/key | 5.5 ns/key, 1.9% |

Memory effect for `uart_insert_iter`:

| Before B/op | After B/op | Saved |
| ---: | ---: | ---: |
| 15,466,068 | 15,171,154 | 294,914 B/op |

### 8. Shrink `inner.SubN` To `uint32`

Files: `node.go`, `arena.go`, `inner.go`, `gte.go`, `lte.go`, `n4.go`,
`n16.go`, `n48.go`, `n256.go`, `tree.go`, `tree_test.go`

`SubN` is the subtree leaf count used by indexing and successor/predecessor
queries. It was also an `int`. Shrinking it to `uint32` removed more heap
footprint. Public indexes still use `int`; conversions happen at the API/query
edges.

Because this makes subtree counts 32-bit, `tree.go` now has a cold guard at
`2^32-1` leaves. Existing practical memtable sizes are far below that.

Median effect:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 283.2 ns/key | 276.2 ns/key | 7.0 ns/key, 2.5% |
| `uart_insert_scan` | 288.1 ns/key | 271.2 ns/key | 16.9 ns/key, 5.9% |

Memory effect for `uart_insert_iter`:

| Before B/op | After B/op | Saved |
| ---: | ---: | ---: |
| 15,171,154 | 14,876,255 | 294,899 B/op |

### 9. Remove No-Op `redoPren` Calls

Files: `arena.go`, `inner.go`, `n4.go`, `n16.go`, `n48.go`, `n256.go`

The node `redoPren` methods now immediately return because prefix counts are
lazy and recomputed by `subTreeRedoPren` when needed. Several hot insert/grow
paths still called those methods. Those calls were removed.

The standalone benchmark result was within noise:

| Case | Before | After |
| --- | ---: | ---: |
| `uart_insert_iter` | 276.2 ns/key | 278.9 ns/key |
| `uart_insert_scan` | 271.2 ns/key | 271.6 ns/key |

This was retained as a hot-path cleanup because it removes interface calls to
methods that do no work, and the final combined benchmark remained strongly
positive. It should not be counted as an independently proven speedup.

### 10. Reduce Leaf And Arena Write Traffic

Files: `arena.go`

`newLeaf` now writes only the fields that are needed. In the nil-value memtable
benchmark, it avoids writing the `Value` interface field. Arena-backed
`newBnodeLeaf`, `newBnodeInner`, `newInner`, and `newNode*` constructors also
avoid assigning whole zero-value structs into freshly zeroed arena slots.

Effect of the `newLeaf` field-write change:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 278.9 ns/key | 276.2 ns/key | 2.7 ns/key, 1.0% |
| `uart_insert_scan` | 271.6 ns/key | 263.3 ns/key | 8.3 ns/key, 3.1% |

Effect of avoiding whole-struct zero assignments in arena constructors:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_scan` | 253.6 ns/key | 247.4 ns/key | 6.2 ns/key, 2.4% |

The iterator measurement for the arena-constructor change alone was noisy and
not a reliable standalone win, but the scan path improved and the change reduces
write barriers and dead stores.

### 11. Faster `Key.At` Bounds Check

Files: `node.go`

`Key.At` is called frequently during insertion. The bounds check changed from:

```go
if pos < 0 || pos >= len(key)
```

to the common compiler-friendly unsigned form:

```go
if uint(pos) >= uint(len(key))
```

Median effect:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 276.2 ns/key | 270.0 ns/key | 6.2 ns/key, 2.2% |

The scan-path number moved the other way in that run, so the retained evidence
is mainly for the default iterator API.

### 12. Hand-Inline Tiny `node4.addChild` Shifts

Files: `n4.go`

`node4.addChild` used `copy` to shift at most three keys and three child
pointers. The hot path now performs those tiny shifts directly with a switch.

Median effect:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 275.7 ns/key | 264.5 ns/key | 11.2 ns/key, 4.1% |
| `uart_insert_scan` | 275.4 ns/key | 263.0 ns/key | 12.4 ns/key, 4.5% |

This is useful because random ART insertion creates and fills many small
`node4` instances.

### 13. Directly Fill Fresh Two-Child `node4` Splits

Files: `leaf.go`, `inner.go`

When a leaf splits, or a compressed prefix splits, the code creates a fresh
`node4` that will contain exactly two children. The old code called `addChild`
twice. The retained code computes the sorted order and writes both slots
directly.

Median effect:

| Case | Before | After | Improvement |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 264.5 ns/key | 251.8 ns/key | 12.7 ns/key, 4.8% |
| `uart_insert_scan` | 263.0 ns/key | 253.6 ns/key | 9.4 ns/key, 3.6% |

This was one of the highest-signal insertion micro-optimizations.

## `Test620_unlocked_read_comparison`

The diagnostic test was changed because it had become misleading for this goal:

- It generated already-sorted decimal string keys.
- It used `google/btree` degree 3000, which is excellent for sorted insertion
  and poor for the random insertion target.
- It constructed `Iter(nil,nil)` before starting the scan timer, which would
  hide prefetch cost.
- It spent minutes in unrelated full 10M-key `Atfar` passes once the input was
  randomized.

The retained test now:

- Uses permuted fixed-width keys.
- Times iterator construction inside the iterator scan measurement.
- Uses btree degree 32 for the random-load diagnostic.
- Samples the expensive indexed-read sections.

One run after the update:

| Case | Result |
| --- | ---: |
| `uart.Tree` store | 247 ns/op |
| `uart Iter`, including construction | 66 ns/op |
| `Ascend(tree,nil,nil)` after cache warmup | 8 ns/op |
| `uart ScanLeaves` | 1 ns/op |
| `google/btree` store | 1.256 us/op |
| `google/btree` full scan | 9 ns/op |

`Test620` is still a diagnostic. The benchmark above is the primary
build-plus-readback score because it times the whole memtable pattern in a
repeatable `testing.B` harness.

## Rejected Or Diagnostic Experiments

These were tried but not kept as speedup claims:

| Experiment | Result |
| --- | --- |
| Sorted bulk constructor as primary comparison | Rejected for this goal. It remains useful for sorted bulk-load cases, but the memtable target assumes random inputs. |
| `node4.lth uint8` | Did not reduce B/op and added casts; timing did not improve. Reverted. |
| Direct `inodeChild` type-switch helper | Regressed the random benchmark versus interface dispatch. Reverted. |
| `bytes.Equal` for leaf equality | No reliable improvement; reverted before the final benchmark. |
| Full-range `Ascend` implemented via recursive `ScanLeaves` | `Test620` showed it was slower inside `iter.Seq2`; replaced with direct cache-slice traversal. |
| `GOGC=off` diagnostic | Not a retained setting. It showed ART algorithmic cost was already much lower than btree, and that GC scanning of pointer-rich structures was the real limiter. |
| Passing `int` values directly to ART in the benchmark | Caused one interface-box allocation per key for ART while btree values were prebuilt outside the timer. Useful fairness warning, not kept as the main benchmark. |
| `uint32` child refs instead of `*bnode` child pointers | Correct, but slower on the target build-plus-scan benchmark. Reverted. |

### `uint32` Child Refs For Node Fanout Arrays

This experiment tested the next arena idea: replace the persistent `*bnode`
child slots in `node4`, `node16`, `node48`, and `node256` with integer refs.
The prototype assigned each bnode a one-based `uint32` index in `bnodeArena`
and changed node navigation to resolve refs through the owning `Tree`. The
resolver used shift/mask arithmetic for the 4096-element bnode chunks after an
initial version using ordinary division/modulo was also slow.

This did remove GC-visible child pointers from the four node-size payloads, but
it added an extra arena lookup on every descent, split update, scan, and pren
refresh. It also required a bnode ref field. Net allocation bytes went up by
about 57 KiB per 100K-key benchmark run because the bnode size increase mostly
offset the smaller fanout arrays.

Correctness passed:

```sh
go test -timeout=3m .
```

Measured target result after the shift/mask resolver:

| Case | Current Baseline | Child-Ref Prototype | Delta |
| --- | ---: | ---: | ---: |
| `uart_insert_iter` | 255.9 ns/key | 258.6 ns/key | 2.7 ns/key slower |
| `uart_insert_scan` | 249.4 ns/key | 272.2 ns/key | 22.8 ns/key slower |
| `uart_insert_nocopy_scan` | 238.0 ns/key | 249.9 ns/key | 11.9 ns/key slower |

The 10M diagnostic was also not a default-API win:

| Case | Child-Ref Prototype |
| --- | ---: |
| `uart.Tree` store | 238 ns/op |
| `uart Iter`, including construction | 94 ns/op |
| `Ascend(tree,nil,nil)` after cache warmup | 8 ns/op |
| `uart ScanLeaves` | 1 ns/op |

That made `Insert+Iter(nil,nil)` about 332 ns/key in the diagnostic, versus the
previous reported 313 ns/key. `ScanLeaves` remained fast after the ordered-leaf
cache existed, but the benchmark target includes the construction/readback
path, not only a warmed scan.

A fully movable "double and copy" arena for the four node sizes would require
one more structural step: `inner.Node inode` currently stores an interface that
points directly at a `node4`/`node16`/`node48`/`node256`. Copying those arenas
would invalidate that pointer. The next refactor would therefore need to turn
`inner.Node` itself into a typed node ref as well. Since the child-ref half
already regressed the build-plus-scan score, this deeper version was not kept.

The `GOGC=off` diagnostic numbers were especially useful:

| Case | ns/key with `GOGC=off` |
| --- | ---: |
| `uart_insert_iter` | 199.3 |
| `uart_insert_scan` | 126.4 |
| `google_btree_degree_32` | 273.2 |

That motivated the retained layout and allocation-pressure changes.

## Final Takeaway

For randomly distributed 8-byte keys and the normal public APIs, ART now beats
`google/btree` degree 32 on the common memtable pattern of "write the table,
then read it all back once":

- `Insert+Iter(nil,nil)`: 255.9 ns/key versus btree degree 32 at 296.4 ns/key.
- `Insert+ScanLeaves`: 249.4 ns/key versus btree degree 32 at 296.4 ns/key.
- `InsertNoCopy+ScanLeaves`: 238.0 ns/key when key ownership is already known.

The largest wins came from removing default-API key-copy allocations, making the
full-range iterator prefetch ordered leaves, reducing pointer-scanned heap
footprint, and speeding up fresh `node4` split construction.
