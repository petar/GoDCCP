// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
	gotime "time"
	"github.com/petar/GoGauge/filter"
)

// Env encapsulates the runtime environment of a DCCP endpoint.  It includes a pluggable
// time interface, in order to allow for use of real as well as synthetic (accelerated) time
// (for testing purposes), as well as a amb interface.
type Env struct {
	time   Time
	guzzle Guzzle
	filter *filter.Filter
	goconj *GoConjunction

	sync.Mutex
	timeZero int64 // Time when execution started
	timeLast int64 // Time of last log message
}

func NewEnv(time Time, guzzle Guzzle) *Env {
	now := time.Now()
	r := &Env{
		time:     time,
		guzzle:   guzzle,
		filter:   filter.NewFilter(),
		goconj:   NewGoConjunction("Env"),
		timeZero: now,
		timeLast: now,
	}
	return r
}

// Go runs f in a new GoRoutine, which is also added to the GoConj of the Env
func (t *Env) Go(f func(), afmt string, aargs ...interface{}) {
	t.goconj.Go(f, afmt, aargs...)
}

func (t *Env) Waiter() Waiter {
	return t.goconj
}

func (t *Env) Guzzle() Guzzle {
	return t.guzzle
}

func (t *Env) Filter() *filter.Filter {
	return t.filter
}

func (t *Env) Sync() error {
	return t.guzzle.Sync()
}

func (t *Env) Close() error {
	return t.guzzle.Close()
}

func (t *Env) Now() int64 {
	return t.time.Now()
}

func (t *Env) Sleep(ns int64) {
	t.time.Sleep(ns)
}

func (t *Env) After(ns int64) <-chan int64 {
	return t.time.After(ns)
}

func (t *Env) Snap() (sinceZero int64, sinceLast int64) {
	t.Lock()
	defer t.Unlock()

	logTime := t.Now()
	timeLast := t.timeLast
	t.timeLast = logTime
	return logTime - t.timeZero, logTime - timeLast
}

// Expire periodically, on every interval duration, checks if the test condition has been met. If
// the condition is met within the timeout period, no further action is taken. Otherwise, the
// onexpire function is invoked.
func (t *Env) Expire(test func()bool, onexpire func(), timeout, interval int64, fmt_ string, args_ ...interface{}) {
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

// Time is an interface for interacting with time
type Time interface {
	// Now returns the current time in nanoseconds since UTC zero
	Now() int64

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


func (realTime) Now() int64 {
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
