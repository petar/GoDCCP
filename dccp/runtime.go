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
	writer LogWriter
	filter *filter.Filter
	waiter *GoGroup

	sync.Mutex
	timeZero int64 // Time when execution started
	timeLast int64 // Time of last log message
}

func NewRuntime(time Time, writer LogWriter) *Runtime {
	now := time.Nanoseconds()
	r := &Runtime{
		time:     time,
		writer:   writer,
		filter:   filter.NewFilter(),
		waiter:   NewGoGroup(),
		timeZero: now,
		timeLast: now,
	}
	return r
}

// Go forks f in a new goroutine
func (t *Runtime) Go(f func()) {
	t.waiter.Go(f)
}

func (t *Runtime) Waiter() Waiter {
	return t.waiter
}

func (t *Runtime) Writer() LogWriter {
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
	d := make(chan int64)
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
