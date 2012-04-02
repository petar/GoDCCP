// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
	gotime "time"
	"github.com/petar/GoGauge/filter"
)

// Runtime encapsulates the runtime environment of a DCCP endpoint.  It includes a pluggable
// time interface, in order to allow for use of real as well as synthetic (accelerated) time
// (for testing purposes), as well as a logger interface.
type Runtime struct {
	time   Time
	writer Guzzle
	filter *filter.Filter
	goconj *GoConjunction

	sync.Mutex
	timeZero int64 // Time when execution started
	timeLast int64 // Time of last log message
}

func NewRuntime(time Time, writer Guzzle) *Runtime {
	now := time.Nanoseconds()
	r := &Runtime{
		time:     time,
		writer:   writer,
		filter:   filter.NewFilter(),
		goconj:   NewGoConjunction("Runtime"),
		timeZero: now,
		timeLast: now,
	}
	return r
}

// Go runs f in a new GoRoutine, which is also added to the GoConj of the Runtime
func (t *Runtime) Go(f func(), afmt string, aargs ...interface{}) {
	t.goconj.Go(f, afmt, aargs...)
}

func (t *Runtime) Waiter() Waiter {
	return t.goconj
}

func (t *Runtime) Writer() Guzzle {
	return t.writer
}

func (t *Runtime) Filter() *filter.Filter {
	return t.filter
}

func (t *Runtime) Sync() error {
	return t.writer.Sync()
}

func (t *Runtime) Close() error {
	return t.writer.Close()
}

func (t *Runtime) Nanoseconds() int64 {
	return t.time.Nanoseconds()
}

func (t *Runtime) Sleep(ns int64) {
	t.time.Sleep(ns)
}

func (t *Runtime) After(ns int64) <-chan int64 {
	return t.time.After(ns)
}

func (t *Runtime) Snap() (sinceZero int64, sinceLast int64) {
	t.Lock()
	defer t.Unlock()

	logTime := t.Nanoseconds()
	timeLast := t.timeLast
	t.timeLast = logTime
	return logTime - t.timeZero, logTime - timeLast
}

// Expire periodically, on every interval duration, checks if the test condition has been met. If
// the condition is met within the timeout period, no further action is taken. Otherwise, the
// onexpire function is invoked.
func (t *Runtime) Expire(test func()bool, onexpire func(), timeout, interval int64, fmt_ string, args_ ...interface{}) {
	t.Go(func() {
		k := timeout / interval
		if k <= 0 {
			panic("frequency too small")
		}
		for i := int64(0); i < k; i++ {
			t.Sleep(interval)
			if test() {
				return
			}
		}
		onexpire()
	}, fmt_, args_...)
}

// Time is an interface for interacting time
type Time interface {
	// Nanoseconds returns the current time in nanoseconds since UTC zero
	Nanoseconds() int64

	// Sleep blocks for ns nanoseconds 
	Sleep(ns int64)

	// After returns a channel which sends the current time once, exactly ns later
	After(ns int64) <-chan int64

	// NewTicker creates a new time ticker that attemps to beat each ns
	NewTicker(ns int64) Ticker
}

// Ticker is an interface for representing a uniform time ticker
type Ticker interface {
	Chan() <-chan int64
	Stop()
}

// RealTime is an implementation of Time that represents real time
var RealTime realTime

type realTime struct {}


func (realTime) Nanoseconds() int64 {
	return gotime.Now().UnixNano()
}

func (realTime) Sleep(ns int64) {
	gotime.Sleep(gotime.Duration(ns))
}

func (realTime) After(ns int64) <-chan int64 {
	d := make(chan int64, 1)
	go func(){
		gotime.Sleep(gotime.Duration(ns))
		d <- gotime.Now().UnixNano()
		close(d)
	}()
	return d
}

func (realTime) NewTicker(ns int64) Ticker {
	return newTicker(gotime.NewTicker(gotime.Duration(ns)))
}

// realTicker proxies time.Ticker to the local interface Ticker
type realTicker struct {
	sync.Mutex
	tkr *gotime.Ticker
	c   chan int64
}

func newTicker(tkr *gotime.Ticker) *realTicker {
	t := &realTicker{ tkr: tkr, c : make(chan int64) }
	go func(){
		for {
			t.Lock()
			tkr := t.tkr
			t.Unlock()
			if tkr == nil {
				break
			}
			tm, ok := <-t.tkr.C
			if !ok {
				break
			}
			t.c <- tm.UnixNano()
		}
	}()
	return t
}

func (t *realTicker) Chan() <-chan int64 {
	return t.c
}

func (t *realTicker) Stop() {
	t.Lock()
	defer t.Unlock()
	t.tkr.Stop()
	t.tkr = nil
}
