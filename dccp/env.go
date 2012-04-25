// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
	"github.com/petar/GoGauge/filter"
)

// Env encapsulates the runtime environment of a DCCP endpoint.  It includes a pluggable
// time interface, in order to allow for use of real as well as synthetic (accelerated) time
// (for testing purposes), as well as a amb interface.
type Env struct {
	runtime Runtime
	guzzle  Guzzle
	filter  *filter.Filter
	gojoin  *GoJoin

	sync.Mutex
	timeZero int64 // Time when execution started
	timeLast int64 // Time of last log message
}

func NewEnv(runtime Runtime, guzzle Guzzle) *Env {
	now := runtime.Now()
	r := &Env{
		runtime:  runtime,
		guzzle:   guzzle,
		filter:   filter.NewFilter(),
		gojoin:   NewGoJoin("Env"),
		timeZero: now,
		timeLast: now,
	}
	return r
}

// Go runs f in a new GoRoutine. The GoRoutine is also added to the GoJoin of the Env.
func (t *Env) Go(f func(), fmt_ string, args_ ...interface{}) {
	t.gojoin.Go(f, fmt_, args_...)
}

func (t *Env) Waiter() Waiter {
	return t.gojoin
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
	return t.runtime.Now()
}

func (t *Env) Sleep(ns int64) {
	t.runtime.Sleep(ns)
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
