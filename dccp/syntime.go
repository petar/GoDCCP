package dccp

import (
	"sort"
	"time"
)

// Runtime represents an environment of execution, and in particular the notion of time.
// Its methods provide access to that environment both in way of querying (e.g.
// "What time is it?") as well as in way of asking for actions (e.g. "Fork a
// new goroutine!") being taken.
type Runtime interface {

	// Sleep blocks for ns nanoseconds 
	Sleep(nsec int64)

	// Now returns the current time in nanoseconds since an abstract 0-moment
	Now() int64

	// Go executes f in a new goroutine
	Go(f func())
}

// RealTime is an implementation of Runtime that represents real time
type realTime struct {}

var RealTime realTime

func (realTime) Now() int64 {
	return time.Now().UnixNano()
}

func (realTime) Sleep(ns int64) {
	time.Sleep(time.Duration(ns))
}

func (realTime) Go(f func()) {
	go f()
}

// syntheticTime is a Runtime implementation that simulates real time without performing real sleeping
type syntheticTime struct {
	reqch chan interface{}
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
	s := &syntheticTime{
		reqch: make(chan interface{}, 1),
	}
	s.Go(func() { g(s) })
	s.loop()
}

type scheduledToSleep struct {
	wake int64
	resp chan int
}

func (x *syntheticTime) loop() {
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
}

func (x *syntheticTime) Sleep(nsec int64) {
	resp := make(chan int)
	x.reqch <- requestSleep{
		duration: nsec,
		resp:     resp,
	}
	<-resp
}

func (x *syntheticTime) Now() int64 {
	resp := make(chan int64)
	x.reqch <- requestNow{
		resp: resp,
	}
	return <-resp
}

func (x *syntheticTime) Go(f func()) {
	x.reqch <- requestGo{}
	go func() {
		f()
		x.die()
	}()
}

func (x *syntheticTime) die() {
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
