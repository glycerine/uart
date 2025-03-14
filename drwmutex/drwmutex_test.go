package drwmutex_test

/*
import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	//"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/glycerine/uart/drwmutex"
)

const (
	BOTH int = 0
	SYNC int = 1
	DRWM int = 2
	END  int = 3
)

type L interface {
	Lock()
	Unlock()
	RLocker() sync.Locker
}

var cost []time.Duration // on average 150 nanoseconds Cpu2() call.


	// func intercept_SIGINT() {
	// 	//vv("intercept_SIGINT installing")
	// 	c := make(chan os.Signal, 100)
	// 	signal.Notify(c, os.Interrupt)
	// 	go func() {
	// 		for sig := range c {
	// 			// sig is a ^C, ctrl-c, handle it
	// 			_ = sig
	// 			fmt.Printf("got SIGINT: cost='%#v'\n", cost)

	// 		}
	// 	}()
	// }

// just main(), started but not finished conversion to test...
func TestDrwmutex(t *testing.T) {
	//intercept_SIGINT()
	if false {
		fmt.Printf("runtime.NumCPU() = %v\n", runtime.NumCPU()) // 8 with is logical cores.
		for i := range 10 {
			go func(i int) {
				for {
					time.Sleep(time.Second)
					fmt.Printf("i=%v LogicalCPUID from klauspost/cpuid: %v\n", i, drwmutex.Cpu2())
				}
			}(i)
		}
		select {}
	}
	cpuprofile := flag.Bool("cpuprofile", false, "enable CPU profiling")
	locks := flag.Uint64("i", 10000, "Number of iterations to perform")
	write := flag.Float64("p", 0.0001, "Probability of write locks")
	wwork := flag.Int("w", 1, "Amount of work for each writer")
	rwork := flag.Int("r", 100, "Amount of work for each reader")
	readers := flag.Int("n", runtime.GOMAXPROCS(0), "Total number of readers")
	checkcpu := flag.Uint64("c", 100, "Update CPU estimate every n iterations")

	strat := flag.Int("strat", 0, "Only loop this (1 or 2) strategy for perf measurement")

	flag.Parse()

	readers_per_core := *readers / runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	var mx L

	for repeat := 0; ; repeat++ {
		for l := 1; l < END; l++ {
			if *strat != 0 {
				if l != *strat {
					continue
				}
			}
			var o *os.File
			if *cpuprofile {
				if o != nil {
					pprof.StopCPUProfile()
					o.Close()
				}

				o, _ := os.Create(fmt.Sprintf("rw%d.out", l))
				pprof.StartCPUProfile(o)
			}

			switch l {
			case SYNC:
				mx = new(sync.RWMutex)
			case DRWM:
				mx = drwmutex.NewDRWMutex()
			}

			start := time.Now()
			for n := 0; n < runtime.GOMAXPROCS(0); n++ {
				for r := 0; r < readers_per_core; r++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						rmx := mx.RLocker()
						r := rand.New(rand.NewSource(rand.Int63()))
						for n := uint64(0); n < *locks; n++ {
							if l != SYNC && *checkcpu != 0 && n%*checkcpu == 0 {
								//t0 := time.Now()
								rmx = mx.RLocker()
								// 150 nanoseconds on average.
								//cost = append(cost, time.Since(t0))
							}
							if r.Float64() < *write {
								mx.Lock()
								x := 0
								for i := 0; i < *wwork; i++ {
									x++
								}
								_ = x
								mx.Unlock()
							} else {
								rmx.Lock()
								x := 0
								for i := 0; i < *rwork; i++ {
									x++
								}
								_ = x
								rmx.Unlock()
							}
						}
					}()
				}
			}
			wg.Wait()
			t := time.Since(start)
			name := "sync.RWMutex"
			if l == DRWM {
				name = "DRWMutex    "
			}

			if repeat == 0 {
				// 4-5x faster on 48 core linux box.
				// 2-3x faster on 8 core mac.
				fmt.Println(name, runtime.GOMAXPROCS(0), *readers, *locks, *write, *wwork, *rwork, *checkcpu, t.Seconds(), t)
			}
		}
		if *strat == 0 {
			break
		}
	}
}
*/
