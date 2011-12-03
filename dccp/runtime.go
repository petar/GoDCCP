// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
	"time"
	"github.com/petar/GoGauge/filter"
)

// Runtime encapsulates the runtime environment of a DCCP endpoint.  It includes a pluggable
// time interface, in order to allow for use of real as well as synthetic (accelerated) time
// (for testing purposes), as well as a logger interface.
type Runtime struct {
	time   Time
	writer LogWriter
	filter *filter.Filter

	sync.Mutex
	timeZero int64 // Time when execution started
	timeLast int64 // Time of last log message
}

func NewRuntime(time Time, writer LogWriter) *Runtime {
	now := time.Nanoseconds()
	return &Runtime{
		time:     time,
		writer:   writer,
		filter:   filter.NewFilter(),
		timeZero: now,
		timeLast: now,
	}
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
	// ? XXX
	return t.writer.Close()
}

func (t *Runtime) Nanoseconds() int64 {
	return t.time.Nanoseconds()
}

func (t *Runtime) Sleep(ns int64) {
	t.time.Sleep(ns)
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
}

// RealTime is an implementation of Time that represents real time
var RealTime realTime

type realTime struct {}


func (realTime) Nanoseconds() int64 {
	return time.Now().UnixNano()
}

func (realTime) Sleep(ns int64) {
	<-time.NewTimer(time.Duration(ns)).C
}
