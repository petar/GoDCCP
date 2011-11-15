// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"sync"
	"time"
)

func SetTime(time Time) {
	runtime.Lock()
	defer runtime.Unlock()

	runtime.Time = time
	runtime.timeZero = time.Nanoseconds()
}

func GetTime() Time {
	runtime.Lock()
	defer runtime.Unlock()

	return runtime.Time
}

func Nanoseconds() int64 {
	runtime.Lock()
	t := runtime.Time
	runtime.Unlock()

	return t.Nanoseconds()
}

func Sleep(ns int64) {
	runtime.Lock()
	t := runtime.Time
	runtime.Unlock()

	t.Sleep(ns)
}

func SnapLog() (sinceZero int64, sinceLast int64) {
	runtime.Lock()
	defer runtime.Unlock()

	logTime := runtime.Time.Nanoseconds()
	lastTime := runtime.timeLastLog
	if lastTime == 0 {
		lastTime = logTime
	}
	runtime.timeLastLog = logTime
	return logTime - runtime.timeZero, logTime - lastTime
}

// runtime ...
var runtime struct {
	sync.Mutex
	Time
	timeZero    int64 // Time when execution started
	timeLastLog int64 // Time of last log message
}

// Time is an interface for interacting time
type Time interface {
	// Nanoseconds returns the current time in nanoseconds since UTC zero
	Nanoseconds() int64

	// Sleep blocks for ns nanoseconds 
	Sleep(ns int64)
}

type realTime struct {}

var RealTime realTime

func (realTime) Nanoseconds() int64 {
	return time.Nanoseconds()
}

func (realTime) Sleep(ns int64) {
	<-time.NewTimer(ns).C
}
