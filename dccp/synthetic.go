package dccp

import (
	"sort"
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

type requestGo  struct{}
type requestDie struct{}

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
	s.Go(func() { g(s) })
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

func (x *SyntheticRuntime) loop() {
	var sleepers sleeperQueue
	var now int64
	var ntogo int = 0
	for {
		req := <-x.reqch
		switch t := req.(type) {
		case requestSleep:
			if t.duration < 0 {
				panic("sleeping negative time")
			}
			sleepers.Add(&scheduledToSleep{ wake: now + t.duration, resp: t.resp })
		case requestNow:
			t.resp <- now
		case requestGo:
			ntogo++
		case requestDie:
			if ntogo < 1 {
				panic("die before birth")
			}
			ntogo--
		default:
			panic("unknown")
		} 

		if sleepers.Len() < ntogo {
			continue
		}

		nextToWake := sleepers.DeleteMin()
		if nextToWake == nil {
			break
		}
		if nextToWake.wake < now {
			panic("waking in the past")
		}
		now = nextToWake.wake
		close(nextToWake.resp)
	}
	close(x.donech)
}

func (x *SyntheticRuntime) Sleep(nsec int64) {
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
	resp := make(chan int64)
	x.reqch <- requestNow{
		resp: resp,
	}
	return <-resp
}

func (x *SyntheticRuntime) Go(f func()) {
	x.reqch <- requestGo{}
	go func() {
		// REMARK: Here we intentionally don't recover from panic in f, since proper program
		// logic demands that no subroutine ever panics
		f()
		x.die()
	}()
}

func (x *SyntheticRuntime) die() {
	x.reqch <- requestDie{}
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
