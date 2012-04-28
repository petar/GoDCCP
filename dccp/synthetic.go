package dccp

import (
	"fmt"
	"runtime"
	"sort"
	"time"
	"os"
)

// SyntheticRuntime is a Runtime implementation that simulates real time without performing real sleeping
type SyntheticRuntime struct {
	reqch  chan interface{}
	donech chan int
}

// request message types
type requestSleep struct {
	duration int64
	resp     chan int
}

type requestNow struct {
	resp chan int64
}

// GoSynthetic runs g inside a synthetic time runtime.
// Access to the runtime is given to g via the its singleton argument.
// GoSynthetic "blocks" until g and any goroutines started by g complete.
// Since g executes inside a synthetic runtime, GoSynthetic really only blocks
// for the duration of time required to execute all non-blocking (in the
// traditional sense of the word) code in g.
func GoSynthetic(g func(Runtime)) {
	s := &SyntheticRuntime{
		reqch:  make(chan interface{}, 1),
		donech: make(chan int),
	}
	go g(s)
	s.loop()
}

// NewSyntheticRuntime creates a new synthetic time Runtime
func NewSyntheticRuntime() *SyntheticRuntime {
	s := &SyntheticRuntime{
		reqch:  make(chan interface{}, 1),
		donech: make(chan int),
	}
	go s.loop()
	return s
}

type scheduledToSleep struct {
	wake int64
	resp chan int
}

const (
	syntheticSleep         = time.Millisecond
	syntheticSleepCount    = 2

)

func (x *SyntheticRuntime) loop() {
	var now int64
	var sleepers sleeperQueue
	var nidle  int
	var nsleep int
ForLoop:
	for {
		runtime.Gosched()
		var req interface{}
		select {
		case req = <-x.reqch:
		default:
		}
		if req != nil {
			switch t := req.(type) {
			case requestSleep:
				if t.duration < 0 {
					panic("sleeping for negative time")
				}
				sleepers.Add(&scheduledToSleep{ wake: now + t.duration, resp: t.resp })
				//fmt.Fprintf(os.Stderr, "=>sleep %d\n", sleepers.Len())
			case requestNow:
				t.resp <- now
				//fmt.Fprintf(os.Stderr, " now = %d\n", now)
			default:
				panic("unknown request")
			} 
			continue ForLoop
		}

		nidle++
		if nidle < runtime.NumGoroutine()*2 {
			continue ForLoop
		}
		nidle = 0
		if nsleep < syntheticSleepCount {
			time.Sleep(syntheticSleep)
			nsleep++
			continue ForLoop
		}
		nsleep = 0

		nextToWake := sleepers.DeleteMin()

		// If no goroutines are left running, then quit the loop
		if nextToWake == nil {
			break
		}

		// Otherwise set clock forward and wake goroutine
		if nextToWake.wake < now {
			panic("waking in the past")
		}
		//fmt.Fprintf(os.Stderr, "=>waking %d\n", sleepers.Len())
		now = nextToWake.wake
		//fmt.Fprintf(os.Stderr, "wake = %d\n", now)
		close(nextToWake.resp)
	}
	fmt.Fprintf(os.Stderr, "=>out-of-time %d\n", now)
	close(x.donech)
	// If there are lingering goroutines that think the runtime is still alive,
	// when they call into the runtime, they will send a message to x.reqch,
	// which will cause a panic.
	close(x.reqch)
}

func (x *SyntheticRuntime) Sleep(nsec int64) {
	//log.Printf("sleep: %s", StackTrace(nil, 0, "", 0))
	resp := make(chan int)
	x.reqch <- requestSleep{
		duration: nsec,
		resp:     resp,
	}
	<-resp
}

// Join blocks until all goroutines running inside the synthetic runtime complete
func (x *SyntheticRuntime) Join() {
	<-x.donech
}

// Now returns the current time inside the synthetic runtime
func (x *SyntheticRuntime) Now() int64 {
	//log.Printf("now: %s", StackTrace(nil, 0, "", 0))
	resp := make(chan int64)
	x.reqch <- requestNow{
		resp: resp,
	}
	return <-resp
}

// sleeperQueue sorts scheduledToSleep instances ascending by timestamp
type sleeperQueue []*scheduledToSleep

func (t sleeperQueue) Len() int {
	return len(t)
}

func (t sleeperQueue) Less(i, j int) bool {
	return t[i].wake < t[j].wake
}

func (t sleeperQueue) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t *sleeperQueue) Add(a *scheduledToSleep) {
	*t = append(*t, a)
	sort.Sort(t)
}

func (t *sleeperQueue) DeleteMin() *scheduledToSleep {
	if len(*t) == 0 {
		return nil
	}
	q := (*t)[0]
	*t = (*t)[1:]
	return q
}
