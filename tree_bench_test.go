package uart

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
	//rb "github.com/glycerine/rbtree"

	googbtree "github.com/google/btree"

	"github.com/glycerine/uart/drwmutex"
)

const seed = 1

func newValue(v int) []byte {
	return []byte(fmt.Sprintf("%05d", v))
}

func randomKey(rng *rand.Rand, b []byte) []byte {
	key := rng.Uint32()
	key2 := rng.Uint32()
	binary.LittleEndian.PutUint32(b, key)
	binary.LittleEndian.PutUint32(b[4:], key2)
	return b
}

func randomKey2(rng *rand.Rand) []byte {
	b := make([]byte, 8)
	key := rng.Uint32()
	key2 := rng.Uint32()
	binary.LittleEndian.PutUint32(b, key)
	binary.LittleEndian.PutUint32(b[4:], key2)
	return b
}

/* this is the wrong way to benchmark readers vs writers. We
must have them on their own goroutines for the DRW lock sharding to work.

// Insert and Read benchmark. A varied fraction is
// read vs inserted. sync.RWMutex ocking is used.
func BenchmarkArtReadWrite(b *testing.B) {
	value := newValue(123)
	for i := 0; i <= 10; i++ {
		readFrac := float32(i) / 10.0
		b.Run(fmt.Sprintf("frac_%d", i), func(b *testing.B) {
			tree := NewArtTree()
			tree.SkipLocking = true
			b.ResetTimer()
			//var count int
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				rlock := tree.DRWmut.RLocker()
				rng := rand.New(rand.NewSource(seed))
				var rkey [8]byte
				for pb.Next() {
					i++
					rk := randomKey(rng, rkey[:])

					if rng.Float32() < readFrac {
						if i%2000 == 0 {
							// refresh the lock, in case
							// we are different core.
							rlock = tree.DRWmut.RLocker()
						}
						rlock.RLock()
						tree.FindExact(rk)
						rlock.RUnlock()
					} else {
						tree.DRWmut.Lock()
						tree.Insert(rk, value)
						tree.DRWmut.Unlock()
					}
				}
			})
		})
	}
}
*/

func TestArtReadWrite_readers_writers_on_own_goro_DRWMutex(t *testing.T) {
	value := newValue(123)
	for i := 0; i <= 10; i++ {
		//readFrac := float32(i) / 10.0
		//fmt.Printf("frac_%d", i)

		//vv("top of Run func: i = %v", i)

		tree := NewArtTree()
		tree.SkipLocking = true // we do locking manually below
		t0 := time.Now()

		const ops = 10_0000
		//const ops = 1000
		var wg sync.WaitGroup
		Ngoro := 100
		elaps := make([]time.Duration, Ngoro)
		wg.Add(Ngoro)
		for j := range Ngoro {
			isReader := j < i*10
			//vv("on i=%v; j=%v; am reader? %v", i, j, isReader)
			go func(isReader bool, j int) {
				defer wg.Done()

				rng := rand.New(rand.NewSource(seed))
				var rkey [8]byte
				t1 := time.Now()
				if isReader {
					rlock := tree.DRWmut.RLocker()
					rlock.RLock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						tree.FindExact(rk)
					}
					rlock.RUnlock()
				} else {
					// is writer
					tree.DRWmut.Lock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						tree.Insert(rk, value)
					}
					tree.DRWmut.Unlock()
				}
				elaps[j] = time.Since(t1)
			}(isReader, j)
		} // end j over all goro
		wg.Wait()
		e0 := time.Since(t0).Truncate(time.Microsecond)
		if i == 0 || i == 10 {
			fmt.Printf("%v %% read: elapsed %v; %v reads; %v writes (%0.3f ns/op)\n", i*10, e0, formatUnder(i*Ngoro*ops), formatUnder((10-i)*Ngoro*ops), float64(e0)/float64(Ngoro*ops))
			//fmt.Printf("%v %% read: elapsed %v; %v reads; %v writes (%0.3f ns/op); elaps='%#v'\n", i*10, e0, formatUnder(i*Ngoro*ops), formatUnder((10-i)*Ngoro*ops), float64(e0)/float64(Ngoro*ops), elaps)
		}

	}
}

/* Linux 48 core:
go test -v -run TestArtReadWrite_readers_writers_on_own_goro
48/48 cpus found in 23.838389ms: map[0:0 1:24 2:1 3:25 4:2 5:26 8:3 9:27 10:4 11:28 12:5 13:29 16:6 17:30 18:7 19:31 20:8 21:32 24:9 25:33 26:10 27:34 28:11 29:35 32:12 33:36 34:13 35:37 36:14 37:38 40:15 41:39 42:16 43:40 44:17 45:41 48:18 49:42 50:19 51:43 52:20 53:44 56:21 57:45 58:22 59:46 60:23 61:47]
=== RUN   TestArtReadWrite_readers_writers_on_own_goro
 0 % read: elapsed 4.748993s; 0 reads; 100_000_000 writes (474.899 ns/op)
10 % read: elapsed 4.293294s; 10_000_000 reads; 90_000_000 writes (429.329 ns/op)
20 % read: elapsed 3.824588s; 20_000_000 reads; 80_000_000 writes (382.459 ns/op)
30 % read: elapsed 3.475722s; 30_000_000 reads; 70_000_000 writes (347.572 ns/op)
40 % read: elapsed 2.85854s; 40_000_000 reads; 60_000_000 writes (285.854 ns/op)
50 % read: elapsed 2.463963s; 50_000_000 reads; 50_000_000 writes (246.396 ns/op)
60 % read: elapsed 1.919834s; 60_000_000 reads; 40_000_000 writes (191.983 ns/op)
70 % read: elapsed 1.465703s; 70_000_000 reads; 30_000_000 writes (146.570 ns/op)
80 % read: elapsed 990.867ms; 80_000_000 reads; 20_000_000 writes (99.087 ns/op)
90 % read: elapsed 512.86ms; 90_000_000 reads; 10_000_000 writes (51.286 ns/op)
100 % read: elapsed 8.946ms; 100_000_000 reads; 0 writes (0.895 ns/op)
--- PASS: TestArtReadWrite_readers_writers_on_own_goro (26.56s)

// compare to sync.Map at 100% reads. On Linux:

> a=c(0.9,1.0, 0.92, 0.895)
> summary(a)
   Min. 1st Qu.  Median    Mean 3rd Qu.    Max.
 0.8950  0.8988  0.9100  0.9287  0.9400  1.0000
> b=c(1.1,1.2,1.1,0.964)
> summary(b)
   Min. 1st Qu.  Median    Mean 3rd Qu.    Max.
  0.964   1.066   1.100   1.091   1.125   1.200
> d=1.091 -0.9287
> d
[1] 0.1623
> d/1.091
[1] 0.1487626

//  So ART reads are 15% faster than sync.Map in 100% read with DRWMutex on Linux;
// and ART reads are  2x faster than sync.Map in 100% read with DRWMutex on Darwin.

Darwin, 8 core:

go test -v -run=ArtReadWrite_readers_writers_on_own_goro
8/8 cpus found in 8.209µs: map[0:0 1:1 2:2 3:3 4:4 5:5 6:6 7:7]
=== RUN   TestArtReadWrite_readers_writers_on_own_goro
 0 % read: elapsed 3.093764s; 0 reads; 100_000_000 writes (309.376 ns/op)
10 % read: elapsed 2.798005s; 10_000_000 reads; 90_000_000 writes (279.800 ns/op)
20 % read: elapsed 2.527697s; 20_000_000 reads; 80_000_000 writes (252.770 ns/op)
30 % read: elapsed 2.255554s; 30_000_000 reads; 70_000_000 writes (225.555 ns/op)
40 % read: elapsed 1.939679s; 40_000_000 reads; 60_000_000 writes (193.968 ns/op)
50 % read: elapsed 1.733701s; 50_000_000 reads; 50_000_000 writes (173.370 ns/op)
60 % read: elapsed 1.401226s; 60_000_000 reads; 40_000_000 writes (140.123 ns/op)
70 % read: elapsed 1.12676s; 70_000_000 reads; 30_000_000 writes (112.676 ns/op)
80 % read: elapsed 811.226ms; 80_000_000 reads; 20_000_000 writes (81.123 ns/op)
90 % read: elapsed 540.362ms; 90_000_000 reads; 10_000_000 writes (54.036 ns/op)
100 % read: elapsed 26.126ms; 100_000_000 reads; 0 writes (2.613 ns/op)
--- PASS: TestArtReadWrite_readers_writers_on_own_goro (18.25s)

*/

func TestArtReadWrite_sync_RWMutex_readers_writers_on_own_goro(t *testing.T) {
	value := newValue(123)
	for i := 0; i <= 10; i++ {
		//readFrac := float32(i) / 10.0
		//fmt.Printf("frac_%d", i)

		//vv("top of Run func: i = %v", i)

		var rwmut sync.RWMutex // just one, shared not sharded.

		tree := NewArtTree()
		tree.SkipLocking = true // we do locking manually below
		t0 := time.Now()

		const ops = 10_0000
		//const ops = 1000
		var wg sync.WaitGroup
		Ngoro := 100
		elaps := make([]time.Duration, Ngoro)
		wg.Add(Ngoro)
		for j := range Ngoro {
			isReader := j < i*10
			//vv("on i=%v; j=%v; am reader? %v", i, j, isReader)
			go func(isReader bool, j int) {
				defer wg.Done()

				rng := rand.New(rand.NewSource(seed))
				var rkey [8]byte
				t1 := time.Now()
				if isReader {
					rwmut.RLock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						tree.FindExact(rk)
					}
					rwmut.RUnlock()
				} else {
					// is writer
					rwmut.Lock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						tree.Insert(rk, value)
					}
					rwmut.Unlock()
				}
				elaps[j] = time.Since(t1)
			}(isReader, j)
		} // end j over all goro
		wg.Wait()
		e0 := time.Since(t0).Truncate(time.Microsecond)
		if i == 0 || i == 10 {
			fmt.Printf("%v %% read: elapsed %v; %v reads; %v writes (%0.3f ns/op)\n", i*10, e0, formatUnder(i*Ngoro*ops), formatUnder((10-i)*Ngoro*ops), float64(e0)/float64(Ngoro*ops))
			//fmt.Printf("%v %% read: elapsed %v; %v reads; %v writes (%0.3f ns/op); elaps='%#v'\n", i*10, e0, formatUnder(i*Ngoro*ops), formatUnder((10-i)*Ngoro*ops), float64(e0)/float64(Ngoro*ops), elaps)
		}
	}
}

/*

Linux 48 core:

go test -v -run TestArtReadWrite_sync_RWMutex_readers_writers_on_own_goro
48/48 cpus found in 26.191842ms: map[0:0 1:24 2:1 3:25 4:2 5:26 8:3 9:27 10:4 11:28 12:5 13:29 16:6 17:30 18:7 19:31 20:8 21:32 24:9 25:33 26:10 27:34 28:11 29:35 32:12 33:36 34:13 35:37 36:14 37:38 40:15 41:39 42:16 43:40 44:17 45:41 48:18 49:42 50:19 51:43 52:20 53:44 56:21 57:45 58:22 59:46 60:23 61:47]
=== RUN   TestArtReadWrite_sync_RWMutex_readers_writers_on_own_goro
0 % read: elapsed 4.83643s; 0 reads; 100_000_000 writes (483.643 ns/op)
10 % read: elapsed 4.393822s; 10_000_000 reads; 90_000_000 writes (439.382 ns/op)
20 % read: elapsed 3.885179s; 20_000_000 reads; 80_000_000 writes (388.518 ns/op)
30 % read: elapsed 3.400687s; 30_000_000 reads; 70_000_000 writes (340.069 ns/op)
40 % read: elapsed 2.93056s; 40_000_000 reads; 60_000_000 writes (293.056 ns/op)
50 % read: elapsed 2.479043s; 50_000_000 reads; 50_000_000 writes (247.904 ns/op)
60 % read: elapsed 1.999989s; 60_000_000 reads; 40_000_000 writes (199.999 ns/op)
70 % read: elapsed 1.519092s; 70_000_000 reads; 30_000_000 writes (151.909 ns/op)
80 % read: elapsed 1.009523s; 80_000_000 reads; 20_000_000 writes (100.952 ns/op)
90 % read: elapsed 527.528ms; 90_000_000 reads; 10_000_000 writes (52.753 ns/op)
100 % read: elapsed 7.984ms; 100_000_000 reads; 0 writes (0.798 ns/op)
--- PASS: TestArtReadWrite_sync_RWMutex_readers_writers_on_own_goro (26.99s)

Darwin 8 core:

go test -v -run TestArtReadWrite_sync_RWMutex_readers_writers_on_own_goro
8/8 cpus found in 9.307µs: map[0:0 1:1 2:2 3:3 4:4 5:5 6:6 7:7]
=== RUN   TestArtReadWrite_sync_RWMutex_readers_writers_on_own_goro
0 % read: elapsed 3.060976s; 0 reads; 100_000_000 writes (306.098 ns/op)
10 % read: elapsed 2.818532s; 10_000_000 reads; 90_000_000 writes (281.853 ns/op)
20 % read: elapsed 2.596279s; 20_000_000 reads; 80_000_000 writes (259.628 ns/op)
30 % read: elapsed 2.256704s; 30_000_000 reads; 70_000_000 writes (225.670 ns/op)
40 % read: elapsed 1.977007s; 40_000_000 reads; 60_000_000 writes (197.701 ns/op)
50 % read: elapsed 1.665417s; 50_000_000 reads; 50_000_000 writes (166.542 ns/op)
60 % read: elapsed 1.388382s; 60_000_000 reads; 40_000_000 writes (138.838 ns/op)
70 % read: elapsed 1.117619s; 70_000_000 reads; 30_000_000 writes (111.762 ns/op)
80 % read: elapsed 820.719ms; 80_000_000 reads; 20_000_000 writes (82.072 ns/op)
90 % read: elapsed 527.433ms; 90_000_000 reads; 10_000_000 writes (52.743 ns/op)
100 % read: elapsed 25.672ms; 100_000_000 reads; 0 writes (2.567 ns/op)
--- PASS: TestArtReadWrite_sync_RWMutex_readers_writers_on_own_goro (18.26s)

*/

func Test_Go_builtin_map_RWMutex_ReadWrite_readers_writers_on_own_goro(t *testing.T) {
	value := newValue(123)
	for i := 0; i <= 10; i++ {
		//readFrac := float32(i) / 10.0
		//fmt.Printf("frac_%d", i)

		//vv("top of Run func: i = %v", i)

		//tree := NewArtTree()
		//tree.SkipLocking = true // we do locking manually below
		m := make(map[string][]byte)
		var rwmut sync.RWMutex

		t0 := time.Now()

		const ops = 10_0000
		var wg sync.WaitGroup
		Ngoro := 100
		wg.Add(Ngoro)
		for j := range Ngoro {
			isReader := j < i*10
			//vv("on i=%v; j=%v; am reader? %v", i, j, isReader)
			go func(isReader bool) (count int) {
				defer wg.Done()

				rng := rand.New(rand.NewSource(seed))
				var rkey [8]byte

				if isReader {
					rwmut.RLock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						_, ok := m[string(rk)]
						// try to prevent compiler from eliding the map read.
						if ok {
							count++
						}
					}
					rwmut.RUnlock()
				} else {
					// is writer
					rwmut.Lock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						//tree.Insert(rk, value)
						m[string(rk)] = value
					}
					rwmut.Unlock()
				}
				return
			}(isReader)
		} // end j over all 10 goro
		wg.Wait()
		e0 := time.Since(t0).Truncate(time.Microsecond)
		fmt.Printf("%v %% read: elapsed %v; %v reads; %v writes (%0.3f ns/op)\n", i*10, e0, formatUnder(i*Ngoro*ops), formatUnder((10-i)*Ngoro*ops), float64(e0)/float64(Ngoro*ops))
	}
}

/*
Linux 48 core:

go test -v -run Test_Go_builtin_map_RWMutex_ReadWrite_readers_writers_on_own_goro
48/48 cpus found in 19.471938ms: map[0:0 1:24 2:1 3:25 4:2 5:26 8:3 9:27 10:4 11:28 12:5 13:29 16:6 17:30 18:7 19:31 20:8 21:32 24:9 25:33 26:10 27:34 28:11 29:35 32:12 33:36 34:13 35:37 36:14 37:38 40:15 41:39 42:16 43:40 44:17 45:41 48:18 49:42 50:19 51:43 52:20 53:44 56:21 57:45 58:22 59:46 60:23 61:47]
=== RUN   Test_Go_builtin_map_RWMutex_ReadWrite_readers_writers_on_own_goro
 0 % read: elapsed 805.037ms; 0 reads; 100_000_000 writes (80.504 ns/op)
10 % read: elapsed 732.843ms; 10_000_000 reads; 90_000_000 writes (73.284 ns/op)
20 % read: elapsed 663.252ms; 20_000_000 reads; 80_000_000 writes (66.325 ns/op)
30 % read: elapsed 541.312ms; 30_000_000 reads; 70_000_000 writes (54.131 ns/op)
40 % read: elapsed 509.684ms; 40_000_000 reads; 60_000_000 writes (50.968 ns/op)
50 % read: elapsed 425.301ms; 50_000_000 reads; 50_000_000 writes (42.530 ns/op)
60 % read: elapsed 327.03ms; 60_000_000 reads; 40_000_000 writes (32.703 ns/op)
70 % read: elapsed 261.054ms; 70_000_000 reads; 30_000_000 writes (26.105 ns/op)
80 % read: elapsed 188.654ms; 80_000_000 reads; 20_000_000 writes (18.865 ns/op)
90 % read: elapsed 124.279ms; 90_000_000 reads; 10_000_000 writes (12.428 ns/op)
100 % read: elapsed 5.461ms; 100_000_000 reads; 0 writes (0.546 ns/op)
--- PASS: Test_Go_builtin_map_RWMutex_ReadWrite_readers_writers_on_own_goro (4.58s)


darwin 8 core:

go test -v -run=Test_Go_builtin_map_RWMutex_ReadWrite_readers_writers_on_own_goro
8/8 cpus found in 4.026µs: map[0:0 1:1 2:2 3:3 4:4 5:5 6:6 7:7]
=== RUN   Test_Go_builtin_map_RWMutex_ReadWrite_readers_writers_on_own_goro
 0 % read: elapsed 758.259ms; 0 reads; 100_000_000 writes (75.826 ns/op)
10 % read: elapsed 642.936ms; 10_000_000 reads; 90_000_000 writes (64.294 ns/op)
20 % read: elapsed 610.454ms; 20_000_000 reads; 80_000_000 writes (61.045 ns/op)
30 % read: elapsed 543.044ms; 30_000_000 reads; 70_000_000 writes (54.304 ns/op)
40 % read: elapsed 464.226ms; 40_000_000 reads; 60_000_000 writes (46.423 ns/op)
50 % read: elapsed 413.726ms; 50_000_000 reads; 50_000_000 writes (41.373 ns/op)
60 % read: elapsed 326.546ms; 60_000_000 reads; 40_000_000 writes (32.655 ns/op)
70 % read: elapsed 296.936ms; 70_000_000 reads; 30_000_000 writes (29.694 ns/op)
80 % read: elapsed 222.586ms; 80_000_000 reads; 20_000_000 writes (22.259 ns/op)
90 % read: elapsed 161.941ms; 90_000_000 reads; 10_000_000 writes (16.194 ns/op)
100 % read: elapsed 20.997ms; 100_000_000 reads; 0 writes (2.100 ns/op)
--- PASS: Test_Go_builtin_map_RWMutex_ReadWrite_readers_writers_on_own_goro (4.46s)
*/

func Test_syncMap_ReadWrite_readers_writers_on_own_goro(t *testing.T) {
	value := newValue(123)
	for i := 0; i <= 10; i++ {

		var m sync.Map

		t0 := time.Now()

		const ops = 10_0000
		var wg sync.WaitGroup
		Ngoro := 100
		wg.Add(Ngoro)
		for j := range Ngoro {
			isReader := j < i*10
			//vv("on i=%v; j=%v; am reader? %v", i, j, isReader)
			go func(isReader bool) (count int) {
				defer wg.Done()

				rng := rand.New(rand.NewSource(seed))
				var rkey [8]byte

				if isReader {
					for range ops {
						rk := randomKey(rng, rkey[:])
						_, ok := m.Load(string(rk))
						// try to prevent compiler from eliding the map read.
						if ok {
							count++
						}
					}
				} else {
					// is writer
					for range ops {
						rk := randomKey(rng, rkey[:])
						//tree.Insert(rk, value)
						m.Swap(string(rk), value)
					}
				}
				return
			}(isReader)
		} // end j over all 10 goro
		wg.Wait()
		e0 := time.Since(t0).Truncate(time.Microsecond)
		fmt.Printf("%v %% read: elapsed %v; %v reads; %v writes (%0.3f ns/op)\n", i*10, e0, formatUnder(i*Ngoro*ops), formatUnder((10-i)*Ngoro*ops), float64(e0)/float64(Ngoro*ops))
	}
}

/*

Linux 48 core:

go test -v -run Test_syncMap_ReadWrite_readers_writers_on_own_goro
48/48 cpus found in 18.284692ms: map[0:0 1:24 2:1 3:25 4:2 5:26 8:3 9:27 10:4 11:28 12:5 13:29 16:6 17:30 18:7 19:31 20:8 21:32 24:9 25:33 26:10 27:34 28:11 29:35 32:12 33:36 34:13 35:37 36:14 37:38 40:15 41:39 42:16 43:40 44:17 45:41 48:18 49:42 50:19 51:43 52:20 53:44 56:21 57:45 58:22 59:46 60:23 61:47]
=== RUN   Test_syncMap_ReadWrite_readers_writers_on_own_goro
 0 % read: elapsed 470.695ms; 0 reads; 100_000_000 writes (47.069 ns/op)
10 % read: elapsed 427.604ms; 10_000_000 reads; 90_000_000 writes (42.760 ns/op)
20 % read: elapsed 377.811ms; 20_000_000 reads; 80_000_000 writes (37.781 ns/op)
30 % read: elapsed 334.283ms; 30_000_000 reads; 70_000_000 writes (33.428 ns/op)
40 % read: elapsed 281.496ms; 40_000_000 reads; 60_000_000 writes (28.150 ns/op)
50 % read: elapsed 246.451ms; 50_000_000 reads; 50_000_000 writes (24.645 ns/op)
60 % read: elapsed 197.775ms; 60_000_000 reads; 40_000_000 writes (19.777 ns/op)
70 % read: elapsed 162.997ms; 70_000_000 reads; 30_000_000 writes (16.300 ns/op)
80 % read: elapsed 123.637ms; 80_000_000 reads; 20_000_000 writes (12.364 ns/op)
90 % read: elapsed 92.63ms; 90_000_000 reads; 10_000_000 writes (9.263 ns/op)
100 % read: elapsed 9.635ms; 100_000_000 reads; 0 writes (0.964 ns/op)
--- PASS: Test_syncMap_ReadWrite_readers_writers_on_own_goro (2.73s)

Darwin 8 core:

go test -v -run Test_syncMap_ReadWrite_readers_writers_on_own_goro
8/8 cpus found in 9.929µs: map[0:0 1:1 2:2 3:3 4:4 5:5 6:6 7:7]
=== RUN   Test_syncMap_ReadWrite_readers_writers_on_own_goro
 0 % read: elapsed 810.695ms; 0 reads; 100_000_000 writes (81.070 ns/op)
10 % read: elapsed 699.752ms; 10_000_000 reads; 90_000_000 writes (69.975 ns/op)
20 % read: elapsed 645.209ms; 20_000_000 reads; 80_000_000 writes (64.521 ns/op)
30 % read: elapsed 606.54ms; 30_000_000 reads; 70_000_000 writes (60.654 ns/op)
40 % read: elapsed 525.811ms; 40_000_000 reads; 60_000_000 writes (52.581 ns/op)
50 % read: elapsed 464.182ms; 50_000_000 reads; 50_000_000 writes (46.418 ns/op)
60 % read: elapsed 400.324ms; 60_000_000 reads; 40_000_000 writes (40.032 ns/op)
70 % read: elapsed 328.285ms; 70_000_000 reads; 30_000_000 writes (32.828 ns/op)
80 % read: elapsed 277.041ms; 80_000_000 reads; 20_000_000 writes (27.704 ns/op)
90 % read: elapsed 186.255ms; 90_000_000 reads; 10_000_000 writes (18.625 ns/op)
100 % read: elapsed 45.823ms; 100_000_000 reads; 0 writes (4.582 ns/op)
--- PASS: Test_syncMap_ReadWrite_readers_writers_on_own_goro (4.99s)

*/

// compare google/btree using DRWMutex
func Test_btree_DRWMutex_ReadWrite_readers_writers_on_own_goro(t *testing.T) {
	//value := newValue(123)
	for i := 0; i <= 10; i++ {
		//readFrac := float32(i) / 10.0
		//fmt.Printf("frac_%d", i)

		//vv("top of Run func: i = %v", i)

		degree := 3000
		tree := googbtree.NewG[string](degree, googbtree.Less[string]())

		DRWmut := drwmutex.NewDRWMutex()

		t0 := time.Now()

		const ops = 10_0000
		var wg sync.WaitGroup
		Ngoro := 100
		wg.Add(Ngoro)
		for j := range Ngoro {
			isReader := j < i*10
			//vv("on i=%v; j=%v; am reader? %v", i, j, isReader)
			go func(isReader bool) {
				defer wg.Done()

				rng := rand.New(rand.NewSource(seed))
				var rkey [8]byte

				if isReader {
					rlock := DRWmut.RLocker()
					rlock.RLock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						//tree.FindExact(rk)
						_, ok := tree.Get(string(rk))
						_ = ok
					}
					rlock.RUnlock()
				} else {
					// is writer
					DRWmut.Lock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						// not storing value... gives btree a little advantage?
						tree.ReplaceOrInsert(string(rk))
						//tree.Insert(rk, value)
					}
					DRWmut.Unlock()
				}
			}(isReader)
		} // end j over all 10 goro
		wg.Wait()
		e0 := time.Since(t0).Truncate(time.Microsecond)
		fmt.Printf("%v %% read: elapsed %v; %v reads; %v writes (%0.3f ns/op)\n", i*10, e0, formatUnder(i*Ngoro*ops), formatUnder((10-i)*Ngoro*ops), float64(e0)/float64(Ngoro*ops))
	}
}

/*

Linux 48 core:

go test -v -run Test_btree_DRWMutex_ReadWrite_readers_writers_on_own_goro
48/48 cpus found in 21.300156ms: map[0:0 1:24 2:1 3:25 4:2 5:26 8:3 9:27 10:4 11:28 12:5 13:29 16:6 17:30 18:7 19:31 20:8 21:32 24:9 25:33 26:10 27:34 28:11 29:35 32:12 33:36 34:13 35:37 36:14 37:38 40:15 41:39 42:16 43:40 44:17 45:41 48:18 49:42 50:19 51:43 52:20 53:44 56:21 57:45 58:22 59:46 60:23 61:47]
=== RUN   Test_btree_DRWMutex_ReadWrite_readers_writers_on_own_goro
0 % read: elapsed 3.084138s; 0 reads; 100_000_000 writes (308.414 ns/op)
10 % read: elapsed 2.839515s; 10_000_000 reads; 90_000_000 writes (283.952 ns/op)
20 % read: elapsed 2.495739s; 20_000_000 reads; 80_000_000 writes (249.574 ns/op)
30 % read: elapsed 2.220878s; 30_000_000 reads; 70_000_000 writes (222.088 ns/op)
40 % read: elapsed 1.90089s; 40_000_000 reads; 60_000_000 writes (190.089 ns/op)
50 % read: elapsed 1.596246s; 50_000_000 reads; 50_000_000 writes (159.625 ns/op)
60 % read: elapsed 1.356848s; 60_000_000 reads; 40_000_000 writes (135.685 ns/op)
70 % read: elapsed 1.049441s; 70_000_000 reads; 30_000_000 writes (104.944 ns/op)
80 % read: elapsed 739.494ms; 80_000_000 reads; 20_000_000 writes (73.949 ns/op)
90 % read: elapsed 463.687ms; 90_000_000 reads; 10_000_000 writes (46.369 ns/op)
100 % read: elapsed 60.333ms; 100_000_000 reads; 0 writes (6.033 ns/op)
--- PASS: Test_btree_DRWMutex_ReadWrite_readers_writers_on_own_goro (17.81s)
*/

// compare google/btree using sync.RWMutex
func Test_btree_sync_RWMutex_ReadWrite_readers_writers_on_own_goro(t *testing.T) {
	//value := newValue(123)
	for i := 0; i <= 10; i++ {
		//readFrac := float32(i) / 10.0
		//fmt.Printf("frac_%d", i)

		//vv("top of Run func: i = %v", i)

		degree := 3000
		tree := googbtree.NewG[string](degree, googbtree.Less[string]())

		var rwmut sync.RWMutex

		t0 := time.Now()

		const ops = 10_0000
		var wg sync.WaitGroup
		Ngoro := 100
		wg.Add(Ngoro)
		for j := range Ngoro {
			isReader := j < i*10
			//vv("on i=%v; j=%v; am reader? %v", i, j, isReader)
			go func(isReader bool) {
				defer wg.Done()

				rng := rand.New(rand.NewSource(seed))
				var rkey [8]byte

				if isReader {
					rwmut.RLock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						//tree.FindExact(rk)
						_, ok := tree.Get(string(rk))
						_ = ok
					}
					rwmut.RUnlock()
				} else {
					// is writer
					rwmut.Lock()
					for range ops {
						rk := randomKey(rng, rkey[:])
						// not storing value... gives btree a little advantage?
						tree.ReplaceOrInsert(string(rk))
						//tree.Insert(rk, value)
					}
					rwmut.Unlock()
				}
			}(isReader)
		} // end j over all 10 goro
		wg.Wait()
		e0 := time.Since(t0).Truncate(time.Microsecond)
		fmt.Printf("%v %% read: elapsed %v; %v reads; %v writes (%0.3f ns/op)\n", i*10, e0, formatUnder(i*Ngoro*ops), formatUnder((10-i)*Ngoro*ops), float64(e0)/float64(Ngoro*ops))
	}
}

/*
Linux 48 core:

go test -v -run Test_btree_sync_RWMutex_ReadWrite_readers_writers_on_own_goro
48/48 cpus found in 21.335641ms: map[0:0 1:24 2:1 3:25 4:2 5:26 8:3 9:27 10:4 11:28 12:5 13:29 16:6 17:30 18:7 19:31 20:8 21:32 24:9 25:33 26:10 27:34 28:11 29:35 32:12 33:36 34:13 35:37 36:14 37:38 40:15 41:39 42:16 43:40 44:17 45:41 48:18 49:42 50:19 51:43 52:20 53:44 56:21 57:45 58:22 59:46 60:23 61:47]
=== RUN   Test_btree_sync_RWMutex_ReadWrite_readers_writers_on_own_goro
0 % read: elapsed 3.06308s; 0 reads; 100_000_000 writes (306.308 ns/op)
10 % read: elapsed 2.792493s; 10_000_000 reads; 90_000_000 writes (279.249 ns/op)
20 % read: elapsed 2.479586s; 20_000_000 reads; 80_000_000 writes (247.959 ns/op)
30 % read: elapsed 2.162393s; 30_000_000 reads; 70_000_000 writes (216.239 ns/op)
40 % read: elapsed 1.920269s; 40_000_000 reads; 60_000_000 writes (192.027 ns/op)
50 % read: elapsed 1.637927s; 50_000_000 reads; 50_000_000 writes (163.793 ns/op)
60 % read: elapsed 1.354885s; 60_000_000 reads; 40_000_000 writes (135.488 ns/op)
70 % read: elapsed 1.066109s; 70_000_000 reads; 30_000_000 writes (106.611 ns/op)
80 % read: elapsed 774.336ms; 80_000_000 reads; 20_000_000 writes (77.434 ns/op)
90 % read: elapsed 491.9ms; 90_000_000 reads; 10_000_000 writes (49.190 ns/op)
100 % read: elapsed 62.952ms; 100_000_000 reads; 0 writes (6.295 ns/op)
--- PASS: Test_btree_sync_RWMutex_ReadWrite_readers_writers_on_own_goro (17.81s)


Darwin 8 core:

go test -v -run Test_btree_sync_RWMutex_ReadWrite_readers_writers_on_own_goro
8/8 cpus found in 4.987µs: map[0:0 1:1 2:2 3:3 4:4 5:5 6:6 7:7]
=== RUN   Test_btree_sync_RWMutex_ReadWrite_readers_writers_on_own_goro
0 % read: elapsed 3.091518s; 0 reads; 100_000_000 writes (309.152 ns/op)
10 % read: elapsed 2.752677s; 10_000_000 reads; 90_000_000 writes (275.268 ns/op)
20 % read: elapsed 2.538319s; 20_000_000 reads; 80_000_000 writes (253.832 ns/op)
30 % read: elapsed 2.28622s; 30_000_000 reads; 70_000_000 writes (228.622 ns/op)
40 % read: elapsed 2.039747s; 40_000_000 reads; 60_000_000 writes (203.975 ns/op)
50 % read: elapsed 1.799679s; 50_000_000 reads; 50_000_000 writes (179.968 ns/op)
60 % read: elapsed 1.584123s; 60_000_000 reads; 40_000_000 writes (158.412 ns/op)
70 % read: elapsed 1.35866s; 70_000_000 reads; 30_000_000 writes (135.866 ns/op)
80 % read: elapsed 1.105797s; 80_000_000 reads; 20_000_000 writes (110.580 ns/op)
90 % read: elapsed 872.747ms; 90_000_000 reads; 10_000_000 writes (87.275 ns/op)
100 % read: elapsed 79.896ms; 100_000_000 reads; 0 writes (7.990 ns/op)
--- PASS: Test_btree_sync_RWMutex_ReadWrite_readers_writers_on_own_goro (19.51s)

*/

func BenchmarkArtLinuxPaths(b *testing.B) {

	paths := loadTestFile("assets/linux.txt")
	n := len(paths)
	_ = n

	//for i := 0; i <= 1; i++ {
	for i := 0; i <= 10; i++ {
		readFrac := float32(i) / 10.0
		_ = readFrac
		b.Run(fmt.Sprintf("frac_%d", i), func(b *testing.B) {
			l := NewArtTree()
			b.ResetTimer()
			//var count int
			b.RunParallel(func(pb *testing.PB) {
				rng := rand.New(rand.NewSource(seed))
				for pb.Next() {
					for k := range paths {
						if rng.Float32() < readFrac {
							//l.FindExact(randomKey(rng))
							l.FindExact(paths[k])
							//l.Remove(paths[k])
						} else {
							//l.Insert(randomKey(rng), value)
							l.Insert(paths[k], paths[k])
						}
					}
				}
			})
		})
	}
}

// Standard test. Some fraction is read. Some fraction is write. Writes have
// to go through mutex lock.
func BenchmarkReadWrite_map_RWMutex_wrapped(b *testing.B) {
	value := newValue(123)
	for i := 0; i <= 10; i++ {
		readFrac := float32(i) / 10.0
		b.Run(fmt.Sprintf("frac_%d", i), func(b *testing.B) {
			m := make(map[string][]byte)
			var mutex sync.RWMutex
			b.ResetTimer()
			var count int
			b.RunParallel(func(pb *testing.PB) {
				rng := rand.New(rand.NewSource(seed))
				var rkey [8]byte
				for pb.Next() {
					rk := randomKey(rng, rkey[:])
					if rng.Float32() < readFrac {
						mutex.RLock()
						_, ok := m[string(rk)]
						mutex.RUnlock()
						if ok {
							count++
						}
					} else {
						mutex.Lock()
						m[string(rk)] = value
						mutex.Unlock()
					}
				}
			})
		})
	}
}

// bah. will crash the tester if run in parallel.
// so don't run in parallel.
func BenchmarkReadWrite_Map_NoMutex_NoParallel(b *testing.B) {
	value := newValue(123)
	for i := 0; i <= 10; i++ {
		readFrac := float32(i) / 10.0
		b.Run(fmt.Sprintf("frac_%d", i), func(b *testing.B) {
			m := make(map[string][]byte)
			b.ResetTimer()
			var count int

			rng := rand.New(rand.NewSource(seed))
			var rkey [8]byte

			for range b.N {
				rk := randomKey(rng, rkey[:])
				if rng.Float32() < readFrac {
					_, ok := m[string(rk)]
					if ok {
						count++
					}
				} else {
					m[string(rk)] = value
				}
			}
		})
	}
}

func BenchmarkArtReadWrite_NoLocking_NoParallel(b *testing.B) {
	value := newValue(123)
	for i := 0; i <= 10; i++ {
		readFrac := float32(i) / 10.0
		b.Run(fmt.Sprintf("frac_%d", i), func(b *testing.B) {
			l := NewArtTree()
			l.SkipLocking = true
			b.ResetTimer()

			rng := rand.New(rand.NewSource(seed))
			var rkey [8]byte

			for range b.N {
				rk := randomKey(rng, rkey[:])
				if rng.Float32() < readFrac {
					l.FindExact(rk)
				} else {
					l.Insert(rk, value)
				}
			}
		})
	}
}

type kvs struct {
	key string
	val string
}

// Standard test. Some fraction is read. Some fraction is write. Writes have
// to go through mutex lock.
func BenchmarkReadWriteSyncMap(b *testing.B) {
	value := newValue(123)
	for i := 0; i <= 10; i++ {
		readFrac := float32(i) / 10.0
		b.Run(fmt.Sprintf("frac_%d", i), func(b *testing.B) {
			var m sync.Map
			b.ResetTimer()
			var count int
			b.RunParallel(func(pb *testing.PB) {
				rng := rand.New(rand.NewSource(seed))
				for pb.Next() {
					if rng.Float32() < readFrac {
						_, ok := m.Load(string(randomKey2(rng)))
						if ok {
							count++
						}
					} else {
						m.Store(string(randomKey2(rng)), value)
					}
				}
			})
		})
	}
}

/*
// commented out to avoid any other package dependencies.

func BenchmarkReadWrite_RedBlackTree(b *testing.B) {

	tree := newRBtree()

	//value := newValue(123)
	for i := 0; i <= 10; i++ {
		readFrac := float32(i) / 10.0
		b.Run(fmt.Sprintf("frac_%d", i), func(b *testing.B) {

			b.ResetTimer()
			var count int

			rng := rand.New(rand.NewSource(seed))
			var rkey [8]byte

			for range b.N {
				v := randomKey(rng, rkey[:])
				str := string(v)
				if rng.Float32() < readFrac {
					query := &kvs{
						key: str,
					}
					it := tree.FindGE(query)
					ok := !it.Limit()
					if ok {
						count++
					}
				} else {
					pay := &kvs{
						key: str,
						val: str,
					}
					tree.Insert(pay)
					//m[string(randomKey(rng))] = value
				}
			}
		})
		//vv("count = %v", count)
		//_ = count
	}
}
*/
