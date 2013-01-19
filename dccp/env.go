// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
	"time"
	"github.com/petar/GoGauge/filter"
)

// Env encapsulates the runtime environment of a DCCP endpoint.  It includes a pluggable
// time interface, in order to allow for use of real as well as synthetic (accelerated) time
// (for testing purposes), as well as a amb interface.
type Env struct {
	guzzle  Guzzle
	filter  *filter.Filter
	gojoin  *GoJoin

	sync.Mutex
	timeZero int64 // Time when execution started
	timeLast int64 // Time of last log message
}

func NewEnv(guzzle Guzzle) *Env {
	now := time.Now().UnixNano()
	r := &Env{
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

func (t *Env) Joiner() Joiner {
	return t.gojoin
}

func (t *Env) NewGoJoin(annotation string, group ...Joiner) *GoJoin {
	return NewGoJoin(annotation, group...)
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
	return time.Now().UnixNano()
}

func (t *Env) Sleep(ns int64) {
	time.Sleep(time.Duration(ns))
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
